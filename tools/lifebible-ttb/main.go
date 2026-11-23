// Command lifebible-ttb writes the sermon source sources/lifebible-ttb from
// Life Bible Ministry's Through the Bible (Pastor Paul LeBoutillier).
//
//	go run ./tools/lifebible-ttb --cache .lifebible-cache --out sources/lifebible-ttb
//
// It reads https://www.lifebibleministry.com/sitemap.xml, then each
// /through-the-bible/<book>/<...> page it lists, one request at a time, a
// second or more apart, with a User-Agent that names the project, never a
// path robots.txt disallows. A page is kept in the git-ignored cache
// directory (a page already there is not fetched again). From a page it keeps
// the title (whose opening is the passage), the address, and the id of the
// YouTube video it embeds: no audio, video, transcript or description. It
// writes rows.jsonl and REPORT.md in --out; two runs over the same cache
// write the same bytes.
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
	"time"
)

func main() {
	cache := flag.String("cache", "", "directory for the fetched pages (not committed)")
	out := flag.String("out", "", "the source directory to write rows.jsonl and REPORT.md in")
	delay := flag.Duration("delay", 1500*time.Millisecond, "pause between two requests")
	flag.Parse()
	if *cache == "" || *out == "" || flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "usage: lifebible-ttb --cache <dir> --out <source dir> [--delay 1.5s]")
		os.Exit(2)
	}
	if err := run(context.Background(), site+"/sitemap.xml", *cache, *out, *delay, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "lifebible-ttb:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, sitemapURL, cache, out string, delay time.Duration, stdout io.Writer) error {
	client := &http.Client{Timeout: time.Minute}
	if err := os.MkdirAll(cache, 0o755); err != nil {
		return err
	}
	body, err := fetchSitemap(ctx, client, sitemapURL, cache)
	if err != nil {
		return err
	}
	urls, err := ParseSitemap(bytes.NewReader(body))
	if err != nil {
		return err
	}
	pages, err := FetchPages(ctx, client, urls, cache, delay, stdout)
	if err != nil {
		return err
	}
	rows, rep := BuildRows(urls, pages)

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
	if err := os.WriteFile(filepath.Join(out, "REPORT.md"), []byte(rep.Markdown()), 0o644); err != nil {
		return err
	}
	fmt.Fprint(stdout, rep.Summary())
	fmt.Fprintf(stdout, "rows.jsonl: %d rows, %d bytes\n", len(rows), buf.Len())
	return nil
}
