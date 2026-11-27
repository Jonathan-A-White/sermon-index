// Command allennolan-youtube writes the sermon source sources/allennolan from
// Pastor Allen Nolan's YouTube channel (@allennolan), by YouTube's official Data
// API (v3): his book-by-book preaching, one row a video of each playlist that
// is named by a Bible book.
//
//	YOUTUBE_API_KEY="$(mw secrets get youtube_api_key)" go run ./tools/allennolan-youtube
//
// It lists the channel's playlists with playlists.list, keeps those whose
// title names a Bible book and says it is a study of it (ClassifyPlaylist),
// walks each with playlistItems.list (50 a page), one request at a time and a
// second apart, and keeps, for each video, only facts: its title as YouTube
// prints it, its date, the watch link and the passage its title, its
// playlist's title or its description names (VideoRef). A video with no
// passage the tool can read is no row and is counted in REPORT.md. Nothing is
// downloaded: no video, no audio, no caption, no page. allennolan.com and the
// Subsplash pages it embeds are not read: Subsplash's terms forbid automated
// reading.
//
// The key is read from the environment variable YOUTUBE_API_KEY alone. It is
// sent in a request header, never written to a file, a log or the report, and
// any error text that would hold it has it blanked.
//
// Stop rule: if fewer than 30% of the videos of the book playlists carry a
// passage, the tool writes REPORT.md and no rows, and exits with an error.
//
// A second run over the same channel writes the same bytes (the order is
// publish order, oldest first), so a diff shows only new videos. YouTube's
// Developer Policies let this data be kept 30 calendar days, so the tool is to
// be run again at least that often.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	keyVar         = "YOUTUBE_API_KEY"
	defaultChannel = "UCSpQNTtLDeQ5Z571g9j0LRg" // @allennolan
	sourceID       = "allennolan"
	watchURL       = "https://www.youtube.com/watch?v="
	preacher       = "Allen Nolan"
	// minShare is the stop rule: under this share of the videos with a passage,
	// no rows are written.
	minShare = 0.30
)

func main() {
	channel := flag.String("channel", defaultChannel, "the channel's id")
	out := flag.String("out", "sources/"+sourceID, "the source directory to write source.toml, rows.jsonl and REPORT.md in")
	report := flag.String("report", "", "write the report here (default: REPORT.md in --out)")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "usage: "+keyVar+"=<key> allennolan-youtube [--channel UCSpQNTtLDeQ5Z571g9j0LRg] [--out sources/allennolan] [--report <file>]")
		os.Exit(2)
	}
	key, err := keyFromEnv(os.Getenv)
	if err != nil {
		fmt.Fprintln(os.Stderr, "allennolan-youtube:", err)
		os.Exit(1)
	}
	c := &Client{Key: key, HTTP: &http.Client{Timeout: time.Minute}}
	if err := run(context.Background(), c, *channel, *out, *report, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "allennolan-youtube:", err)
		os.Exit(1)
	}
}

// keyFromEnv reads the API key from the environment, or says plainly that it
// is not there. The key itself is never part of a message.
func keyFromEnv(getenv func(string) string) (string, error) {
	k := strings.TrimSpace(getenv(keyVar))
	if k == "" {
		return "", errors.New(keyVar + " is empty: run it as " + keyVar + `="$(mw secrets get youtube_api_key)" go run ./tools/allennolan-youtube`)
	}
	return k, nil
}

// discover reads the channel, sorts its playlists into book series and the
// rest, and lists the videos of each book series.
func discover(ctx context.Context, c *Client, channelID string) (*Listing, []SeriesItems, error) {
	ch, err := c.Channel(ctx, channelID)
	if err != nil {
		return nil, nil, err
	}
	pls, err := c.Playlists(ctx, ch.ID)
	if err != nil {
		return nil, nil, err
	}
	l := &Listing{Channel: ch, Playlists: pls}
	var series []SeriesItems
	for _, p := range pls {
		s, ok := ClassifyPlaylist(p)
		if !ok {
			if bookWordRE.MatchString(p.Title) {
				l.LeftOut = append(l.LeftOut, p)
			}
			continue
		}
		items, err := c.PlaylistItems(ctx, p.ID)
		if err != nil {
			return nil, nil, err
		}
		series = append(series, SeriesItems{Series: s, Items: items})
	}
	return l, series, nil
}

// run reads the channel and writes the source into out; reportPath is
// REPORT.md in out when empty.
func run(ctx context.Context, c *Client, channelID, out, reportPath string, stdout io.Writer) error {
	l, series, err := discover(ctx, c, channelID)
	if err != nil {
		return err
	}
	res := Build(l, series)
	res.Units = c.Units
	if reportPath == "" {
		reportPath = filepath.Join(out, "REPORT.md")
	}
	return write(res, out, reportPath, stdout)
}

// write writes the source's files, or, under the stop rule, only the report.
func write(res *Result, out, reportPath string, stdout io.Writer) error {
	if err := os.MkdirAll(out, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(reportPath), 0o755); err != nil {
		return err
	}
	if res.Share() < minShare {
		res.Stopped = true
		if err := os.WriteFile(reportPath, []byte(res.Markdown()), 0o644); err != nil {
			return err
		}
		fmt.Fprint(stdout, res.Summary())
		return fmt.Errorf("only %.0f%% of the %d listed videos of the book playlists carry a passage, under the 30%% the stop rule asks: REPORT.md written, no rows", 100*res.Share(), res.Videos-res.Private())
	}
	var rows bytes.Buffer
	for _, r := range res.Rows {
		enc := json.NewEncoder(&rows)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(r); err != nil {
			return err
		}
	}
	files := []struct {
		path string
		body []byte
	}{
		{filepath.Join(out, "rows.jsonl"), rows.Bytes()},
		{filepath.Join(out, "source.toml"), []byte(res.SourceTOML())},
		{reportPath, []byte(res.Markdown())},
	}
	for _, f := range files {
		if err := os.WriteFile(f.path, f.body, 0o644); err != nil {
			return err
		}
	}
	fmt.Fprint(stdout, res.Summary())
	return nil
}
