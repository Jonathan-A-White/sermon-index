// Command biblethinker writes the sermon source sources/biblethinker from
// Bible Thinker (https://biblethinker.org), Mike Winger's site: his
// book-by-book series through the Bible (1 Peter, Hebrews, Mark, Romans).
//
//	go run ./tools/biblethinker --cache .biblethinker-cache --out sources/biblethinker [--report sources/biblethinker/REPORT.md]
//
// It reads each series' archive page and its later pages (one request at a
// time, two seconds apart, with a User-Agent that names the project, after
// robots.txt) and, for the dates, the WordPress REST API's id and date of the
// series' posts. A row is one episode: the post's title, address and
// passage, the YouTube id of the video the post itself embeds, the preacher,
// the date and the length. Topical posts, Q&A sessions and livestreams are
// never read. Nothing of the media, the audio or a transcript is kept, and
// the cache is for the tool's own re-runs and is never committed. Two runs
// over the same cache write the same bytes.
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
	for _, s := range BookSeries {
		data, err := f.FetchSeries(ctx, s)
		if err != nil {
			return err
		}
		res := BuildSeries(s, data.Episodes, data.Dates)
		if res.Seen != data.Count {
			fmt.Fprintf(os.Stderr, "biblethinker: %s: the archive pages show %d posts, the category says %d\n", s.Slug, res.Seen, data.Count)
		}
		runs = append(runs, SeriesRun{Data: data, Res: res})
		rows = append(rows, res.Rows...)
	}
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
		fmt.Fprintf(w, "%s: %d episodes, %d rows, %d skipped\n", r.Res.Series.Name, r.Res.Seen, len(r.Res.Rows), len(r.Res.Skips))
	}
	return nil
}
