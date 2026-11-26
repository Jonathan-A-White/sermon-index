package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// The sitemap's request is a request to the site too: the first page is asked
// for no sooner than delay after it.
func TestRunSpacesTheSitemapAndTheFirstPage(t *testing.T) {
	page, err := os.ReadFile("testdata/isaiah-1.html")
	if err != nil {
		t.Fatal(err)
	}
	const delay = 200 * time.Millisecond
	var mu sync.Mutex
	var paths []string
	var at []time.Time
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		paths = append(paths, r.URL.Path)
		at = append(at, time.Now())
		mu.Unlock()
		if r.URL.Path == "/sitemap.xml" {
			// The sitemap lists the site's own addresses, and ParseSitemap
			// keeps only those: the transport below sends them here.
			fmt.Fprintf(w, `<?xml version="1.0"?><urlset><url><loc>%s/isaiah/1</loc></url></urlset>`, ttbPrefix)
			return
		}
		w.Write(page)
	}))
	defer srv.Close()
	to, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	was := http.DefaultTransport
	http.DefaultTransport = roundTrip(func(r *http.Request) (*http.Response, error) {
		r = r.Clone(r.Context())
		r.URL.Scheme, r.URL.Host, r.Host = to.Scheme, to.Host, ""
		return was.RoundTrip(r)
	})
	defer func() { http.DefaultTransport = was }()

	if err := run(context.Background(), site+"/sitemap.xml", t.TempDir(), t.TempDir(), delay, io.Discard); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if want := []string{"/sitemap.xml", "/through-the-bible/isaiah/1"}; !slices.Equal(paths, want) {
		t.Fatalf("asked for %v, want %v", paths, want)
	}
	if gap := at[1].Sub(at[0]); gap < delay {
		t.Errorf("the first page was asked for %v after the sitemap, want at least %v", gap, delay)
	}
}

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestFetchPages(t *testing.T) {
	page, err := os.ReadFile("testdata/isaiah-1.html")
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	var paths, agents []string
	var inflight, maxInflight atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := inflight.Add(1)
		defer inflight.Add(-1)
		if n > maxInflight.Load() {
			maxInflight.Store(n)
		}
		mu.Lock()
		paths = append(paths, r.URL.Path)
		agents = append(agents, r.UserAgent())
		mu.Unlock()
		if strings.HasSuffix(r.URL.Path, "/gone") {
			http.NotFound(w, r)
			return
		}
		w.Write(page)
	}))
	defer srv.Close()

	urls := []string{
		srv.URL + "/through-the-bible/isaiah/1",
		srv.URL + "/through-the-bible/isaiah/gone",
		srv.URL + "/api/through-the-bible/x",  // robots.txt disallows /api/: never asked
		srv.URL + "/through-the-bible/isaiah", // a book page: not fetched
	}
	cache := t.TempDir()
	// A page already in the cache is not fetched again.
	if err := os.WriteFile(cachePath(cache, urls[0]), page, 0o644); err != nil {
		t.Fatal(err)
	}
	var log bytes.Buffer
	got, err := FetchPages(context.Background(), srv.Client(), urls, cache, 0, false, &log)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 1 || paths[0] != "/through-the-bible/isaiah/gone" {
		t.Fatalf("asked for %v, want only the page not cached", paths)
	}
	if got[urls[0]].Page.YouTube != "ynUYAm1-W6k" || got[urls[0]].Err != "" {
		t.Errorf("cached page: %+v", got[urls[0]])
	}
	if got[urls[1]].Err == "" || !strings.Contains(got[urls[1]].Err, "404") {
		t.Errorf("gone page: %+v", got[urls[1]])
	}
	if got[urls[2]].Err == "" || !strings.Contains(got[urls[2]].Err, "robots.txt") {
		t.Errorf("disallowed page: %+v", got[urls[2]])
	}
	if _, ok := got[urls[3]]; ok {
		t.Errorf("book page was read")
	}
	for _, a := range agents {
		if !strings.Contains(a, "sermon-index") {
			t.Errorf("User-Agent %q does not name the project", a)
		}
	}
	if maxInflight.Load() > 1 {
		t.Errorf("%d requests at once", maxInflight.Load())
	}
}
