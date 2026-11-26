package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const maxPage = 8 << 20

// pagePath is the path of an address that is a page of a teaching
// (/through-the-bible/<book>/<more>), or "" for the list page, a book page or
// anything else.
func pagePath(u string) string {
	pu, err := url.Parse(u)
	if err != nil {
		return ""
	}
	rest, ok := strings.CutPrefix(strings.TrimRight(pu.Path, "/"), "/through-the-bible/")
	if !ok || !strings.Contains(rest, "/") {
		return ""
	}
	return pu.Path
}

// cachePath is the file a page's HTML is kept in, named by its path.
func cachePath(cache, u string) string {
	pu, _ := url.Parse(u)
	name := strings.ReplaceAll(strings.Trim(pu.Path, "/"), "/", "__")
	return filepath.Join(cache, name+".html")
}

// FetchPages reads each page address of urls (the list page and the book
// pages are not read), one request at a time with delay between requests.
// asked says the site has just been asked for something else (the sitemap),
// so the first page fetched waits delay too. A page already in the cache
// directory is read from it and not fetched again. A path robots.txt
// disallows is never asked for. A page that cannot be had is returned with
// its Err, and the run goes on.
func FetchPages(ctx context.Context, client *http.Client, urls []string, cache string, delay time.Duration, asked bool, log io.Writer) (map[string]PageResult, error) {
	if err := os.MkdirAll(cache, 0o755); err != nil {
		return nil, err
	}
	out := map[string]PageResult{}
	for _, u := range urls {
		pu, err := url.Parse(u)
		if err != nil {
			out[u] = PageResult{Err: err.Error()}
			continue
		}
		if Disallowed(pu.Path) {
			out[u] = PageResult{Err: "not fetched: robots.txt disallows " + pu.Path}
			continue
		}
		if pagePath(u) == "" {
			continue
		}
		path := cachePath(cache, u)
		if b, err := os.ReadFile(path); err == nil && len(b) > 0 {
			out[u] = PageResult{Page: ParsePage(string(b))}
			continue
		}
		if asked {
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
		asked = true
		fmt.Fprintf(log, "fetching %s\n", u)
		body, err := fetchOne(ctx, client, u)
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			out[u] = PageResult{Err: err.Error()}
			continue
		}
		tmp := path + ".part"
		if err := os.WriteFile(tmp, body, 0o644); err != nil {
			return nil, err
		}
		if err := os.Rename(tmp, path); err != nil {
			return nil, err
		}
		out[u] = PageResult{Page: ParsePage(string(body))}
	}
	return out, nil
}

func fetchOne(ctx context.Context, client *http.Client, u string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s", resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxPage))
	if err != nil {
		return nil, err
	}
	return body, nil
}

// fetchSitemap keeps the sitemap in the cache like a page. asked says the
// site was asked for it, false when it came from the cache.
func fetchSitemap(ctx context.Context, client *http.Client, u, cache string) (body []byte, asked bool, err error) {
	path := filepath.Join(cache, "sitemap.xml")
	if b, err := os.ReadFile(path); err == nil && len(b) > 0 {
		return b, false, nil
	}
	body, err = fetchOne(ctx, client, u)
	if err != nil {
		return nil, true, fmt.Errorf("%s: %w", u, err)
	}
	if !strings.Contains(string(body[:min(len(body), 400)]), "<urlset") && !strings.Contains(string(body[:min(len(body), 400)]), "<?xml") {
		return nil, true, fmt.Errorf("%s: not a sitemap", u)
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		return nil, true, err
	}
	return body, true, nil
}
