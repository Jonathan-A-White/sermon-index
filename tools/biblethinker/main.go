// Command biblethinker writes the sermon source sources/biblethinker from
// Bible Thinker (https://biblethinker.org), Mike Winger's site: his book
// series (1 Peter, Hebrews, Mark, Romans) and every other teaching of his that
// names a passage (How to Find Jesus in the Old Testament, Wisdom in the Word,
// Understanding the Old Testament Law, Past Livestreams & Misc. Topical
// Teachings). Q&A sessions, debates, interviews, reactions and reviews and
// worship songs are not read.
//
//	go run ./tools/biblethinker --cache .biblethinker-cache --out sources/biblethinker [--report sources/biblethinker/REPORT.md]
//
// It reads each series' archive page and its later pages (one request at a
// time, two seconds apart, with a User-Agent that names the project, after
// robots.txt) and, for the dates, the WordPress REST API's id and date of the
// category's posts. A row is one post: the title, address and passage, the
// YouTube id of the video the post itself embeds, the preacher, the date and
// the length. The passage is read from the title, else from the digits ending
// the post's address, else from the post's own text (its page is fetched
// then); a post with none is named in the report. A post in two categories is
// the row of the first. Nothing of the media, the audio or a transcript is
// kept, and the cache is for the tool's own re-runs and is never committed.
// Two runs over the same cache write the same bytes.
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
)

func main() {
	cache := flag.String("cache", "", "directory for the fetched pages (not committed)")
	out := flag.String("out", "", "the source directory to write rows.jsonl in")
	report := flag.String("report", "", "write the report to this file")
	delay := flag.Duration("delay", 2*time.Second, "the pause between two requests to the site")
	flag.Parse()
	if *cache == "" || *out == "" || flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "usage: biblethinker --cache <dir> --out <source dir> [--report <file>]")
		os.Exit(2)
	}
	f := &Fetcher{Cache: *cache, Delay: *delay, Log: os.Stderr}
	if err := run(context.Background(), f, *out, *report); err != nil {
		fmt.Fprintln(os.Stderr, "biblethinker:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, f *Fetcher, out, reportPath string) error {
	if err := f.LoadRobots(ctx); err != nil {
		return err
	}
	robots, err := f.RobotsText(ctx)
	if err != nil {
		return err
	}
	var runs []SeriesRun
	var rows []Row
	seen := map[string]bool{}
	var bodyErr error
	bodyOf := func(e Episode) string {
		if bodyErr != nil {
			return ""
		}
		text, err := f.FetchBody(ctx, e)
		if err != nil {
			bodyErr = fmt.Errorf("the post %s: %v", e.URL, err)
		}
		return text
	}
	for _, s := range Categories {
		data, err := f.FetchSeries(ctx, s)
		if err != nil {
			return err
		}
		fresh, dups := SplitSeen(data.Episodes, seen)
		res := BuildSeries(s, fresh, data.Dates, bodyOf)
		if bodyErr != nil {
			return bodyErr
		}
		res.Seen += len(dups)
		res.Dups = len(dups)
		if res.Seen != data.Count {
			fmt.Fprintf(os.Stderr, "biblethinker: %s: the archive pages show %d posts, the category says %d\n", s.Slug, res.Seen, data.Count)
		}
		runs = append(runs, SeriesRun{Data: data, Res: res})
		rows = append(rows, res.Rows...)
	}
	// one list, in date order, a post of no date last
	sort.SliceStable(rows, func(i, j int) bool {
		a, b := rows[i].Date, rows[j].Date
		switch {
		case a != nil && b != nil && *a != *b:
			return *a < *b
		case (a == nil) != (b == nil):
			return a != nil
		}
		return false
	})
	var b bytes.Buffer
	if err := WriteRows(&b, rows); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(out, "rows.jsonl"), b.Bytes(), 0o644); err != nil {
		return err
	}
	if reportPath != "" {
		var r bytes.Buffer
		WriteReport(&r, runs, robots)
		if err := os.WriteFile(reportPath, r.Bytes(), 0o644); err != nil {
			return err
		}
	}
	return summary(os.Stdout, runs)
}

func summary(w io.Writer, runs []SeriesRun) error {
	for _, r := range runs {
		fmt.Fprintf(w, "%s: %d posts, %d rows, %d skipped, %d in an earlier category\n", r.Res.Series.Name, r.Res.Seen, len(r.Res.Rows), len(r.Res.Skips), r.Res.Dups)
	}
	return nil
}
