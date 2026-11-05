// Command ccel-spurgeon writes the sermon source sources/spurgeon-ccel from
// CCEL's ThML edition of Spurgeon's sermons (63 volumes).
//
//	go run ./tools/ccel-spurgeon --cache <dir> --out sources/spurgeon-ccel [--report tools/ccel-spurgeon/REPORT.md]
//
// It fetches https://ccel.org/ccel/spurgeon/sermons01.xml to sermons63.xml
// into the cache directory (a file already there is not fetched again; one
// request at a time, a second apart), reads each sermon's number, title,
// section id, preaching text, cited references and the date of its head, and
// writes rows.jsonl. Nothing of a sermon's words is kept: not in the rows,
// not in the report, and the cache is for the tool's own re-runs and is never
// committed. Two runs over the same cache write the same bytes.
package main

import (
	"bytes"
	"context"
	"encoding/json"
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
	volumes   = 63
	userAgent = "sermon-index-ccel-spurgeon (+https://github.com/Jonathan-A-White/sermon-index)"
)

func main() {
	cache := flag.String("cache", "", "directory for the fetched ThML volumes (not committed)")
	out := flag.String("out", "", "the source directory to write rows.jsonl in")
	report := flag.String("report", "", "write the report of numbers with two preaching texts or none to this file")
	flag.Parse()
	if *cache == "" || *out == "" || flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "usage: ccel-spurgeon --cache <dir> --out <source dir> [--report <file>]")
		os.Exit(2)
	}
	if err := run(context.Background(), *cache, *out, *report, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "ccel-spurgeon:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, cache, out, reportPath string, stdout io.Writer) error {
	client := &http.Client{Timeout: 5 * time.Minute}
	if err := fetchVolumes(ctx, client, VolumeURL, cache, time.Second, stdout); err != nil {
		return err
	}
	var entries []Entry
	for v := 1; v <= volumes; v++ {
		f, err := os.Open(volumePath(cache, v))
		if err != nil {
			return err
		}
		es, err := ParseVolume(f, v)
		f.Close()
		if err != nil {
			return err
		}
		entries = append(entries, es...)
	}
	rows, rep := BuildRows(entries)

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	for _, r := range rows {
		if err := enc.Encode(r); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(out, "rows.jsonl"), buf.Bytes(), 0o644); err != nil {
		return err
	}
	if reportPath != "" {
		if err := os.WriteFile(reportPath, []byte(rep.Markdown()), 0o644); err != nil {
			return err
		}
	}
	fmt.Fprint(stdout, rep.Summary())
	fmt.Fprintf(stdout, "rows.jsonl: %d rows, %d bytes\n", len(rows), buf.Len())
	return nil
}

func volumePath(cache string, v int) string {
	return filepath.Join(cache, fmt.Sprintf("sermons%02d.xml", v))
}

// fetchVolumes fetches the volumes not yet in the cache, one request at a
// time with delay between them.
func fetchVolumes(ctx context.Context, client *http.Client, urlOf func(int) string, cache string, delay time.Duration, log io.Writer) error {
	if err := os.MkdirAll(cache, 0o755); err != nil {
		return err
	}
	fetched := 0
	for v := 1; v <= volumes; v++ {
		path := volumePath(cache, v)
		if st, err := os.Stat(path); err == nil && st.Size() > 0 {
			continue
		}
		if fetched > 0 {
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		fetched++
		u := urlOf(v)
		fmt.Fprintf(log, "fetching %s\n", u)
		if err := fetchOne(ctx, client, u, path); err != nil {
			return err
		}
	}
	return nil
}

func fetchOne(ctx context.Context, client *http.Client, u, path string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: %s", u, resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("%s: %w", u, err)
	}
	// CCEL answers 200 to anything: a volume is ThML.
	if !strings.Contains(string(body[:min(len(body), 2000)]), "<ThML") && !strings.Contains(string(body[:min(len(body), 2000)]), "<?xml") {
		return fmt.Errorf("%s: not ThML", u)
	}
	tmp := path + ".part"
	if err := os.WriteFile(tmp, body, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Summary is the counts, as printed by the tool.
func (r *Report) Summary() string {
	var b strings.Builder
	fmt.Fprintf(&b, "sermon entries read:                   %d\n", r.Entries)
	fmt.Fprintf(&b, "distinct sermon numbers:               %d\n", r.Numbers)
	fmt.Fprintf(&b, "rows:                                  %d\n", r.Rows)
	fmt.Fprintf(&b, "rows with a preaching text that resolves: %d\n", r.WithText)
	fmt.Fprintf(&b, "  of them in Romans 5:                 %d\n", r.Romans5)
	fmt.Fprintf(&b, "rows of kind exposition:               %d\n", r.Exposits)
	fmt.Fprintf(&b, "rows with a date:                      %d\n", r.Dated)
	fmt.Fprintf(&b, "numbers with two preaching texts:      %d\n", len(r.TwoTexts))
	fmt.Fprintf(&b, "entries with none (not an exposition): %d\n", len(r.NoText))
	return b.String()
}

// Markdown is REPORT.md: the numbers a build or a recheck should look at.
func (r *Report) Markdown() string {
	var b strings.Builder
	b.WriteString("# CCEL Spurgeon: what the ThML does not settle\n\n<!-- toc: none -->\n\n")
	b.WriteString("Written by `go run ./tools/ccel-spurgeon --cache <dir> --out sources/spurgeon-ccel --report tools/ccel-spurgeon/REPORT.md`; do not edit by hand. Only numbers, titles and references are listed.\n\n")
	b.WriteString("## Counts\n\n```\n" + r.Summary() + "```\n\n")
	list := func(title, why string, items []string) {
		fmt.Fprintf(&b, "## %s (%d)\n\n%s\n\n", title, len(items), why)
		for _, it := range items {
			fmt.Fprintf(&b, "- %s\n", it)
		}
		if len(items) == 0 {
			b.WriteString("None.\n")
		}
		b.WriteString("\n")
	}
	list("Two or more preaching texts", "The ThML tags more than one `scripCom type=\"Sermon\"`. The row's `ref` is the first; the others are `cited`.", r.TwoTexts)
	list("No preaching text that resolves", "Not an exposition, and the ThML has no `scripCom type=\"Sermon\"` with a reading Lampas's book table accepts (a meeting, an address, a dedication, an empty or misspelt field). The row has no `ref`.", r.NoText)
	list("A preaching text given but not resolved", "The `scripCom` is there and its `passage` is shown (empty, or misspelt); these are among the list above.", r.Unresolved)
	list("Tagged references the checker refuses", "The ThML tags a reference that does not exist in Lampas's numbering (a typing slip in the source: a verse past the chapter, a range that ends before it starts). It is left out of the row.", r.Refused)
	list("Combined entries", "One CCEL page for several numbers: one row each, the same `url`.", r.Combined)
	list("A number with more than one entry", "CCEL has two entries with the number; each has its own row, the id told here.", r.Repeated)
	return b.String()
}
