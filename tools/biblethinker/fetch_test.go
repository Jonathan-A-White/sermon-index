package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// fakeSite serves the testdata the way biblethinker.org does: a series
// archive with a rel="next" link, the WordPress REST category and posts.
type fakeSite struct {
	srv      *httptest.Server
	requests []string
	inFlight int32
	overlap  bool
	badUA    bool
	robots   string
}

func newFakeSite(t *testing.T) *fakeSite {
	t.Helper()
	f := &fakeSite{robots: mustRead(t, "testdata/robots.txt")}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&f.inFlight, 1) > 1 {
			f.overlap = true
		}
		defer atomic.AddInt32(&f.inFlight, -1)
		time.Sleep(2 * time.Millisecond)
		f.requests = append(f.requests, r.URL.RequestURI())
		if !strings.Contains(r.UserAgent(), "sermon-index") {
			f.badUA = true
		}
		switch {
		case r.URL.Path == "/robots.txt":
			io.WriteString(w, f.robots)
		case r.URL.Path == "/1-peter/":
			io.WriteString(w, strings.Replace(mustRead(t, "testdata/archive-1-peter-1.html"), "https://biblethinker.org", f.srv.URL, -1))
		case r.URL.Path == "/1-peter/page/2/":
			io.WriteString(w, mustRead(t, "testdata/archive-1-peter-2.html"))
		case r.URL.Path == "/wp-json/wp/v2/categories":
			io.WriteString(w, mustRead(t, "testdata/category-1-peter.json"))
		case r.URL.Path == "/wp-json/wp/v2/posts" && r.URL.Query().Get("categories") == "95":
			io.WriteString(w, mustRead(t, "testdata/posts-1-peter.json"))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func mustRead(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestFetchSeriesReadsEveryPageAndTheDates(t *testing.T) {
	site := newFakeSite(t)
	f := &Fetcher{Base: site.srv.URL, HTTP: site.srv.Client(), Cache: t.TempDir()}
	got, err := f.FetchSeries(context.Background(), Series{Slug: "1-peter", Book: "1Pet", Name: "1 Peter"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Episodes) != 7 {
		t.Errorf("episodes = %d, want 7", len(got.Episodes))
	}
	if len(got.Pages) != 2 || got.Pages[1] != site.srv.URL+"/1-peter/page/2/" {
		t.Errorf("pages = %v", got.Pages)
	}
	if got.Dates[8336] != "2015-09-13" {
		t.Errorf("dates[8336] = %q", got.Dates[8336])
	}
	if got.Count != 6 {
		t.Errorf("category count = %d, want 6", got.Count)
	}
	if site.overlap {
		t.Error("two requests were in flight at once")
	}
	if site.badUA {
		t.Error("a request had no sermon-index User-Agent")
	}
}

func TestFetcherCachesWhatItFetched(t *testing.T) {
	site := newFakeSite(t)
	cache := t.TempDir()
	f := &Fetcher{Base: site.srv.URL, HTTP: site.srv.Client(), Cache: cache}
	if _, err := f.FetchSeries(context.Background(), Series{Slug: "1-peter", Book: "1Pet", Name: "1 Peter"}); err != nil {
		t.Fatal(err)
	}
	before := len(site.requests)
	f2 := &Fetcher{Base: site.srv.URL, HTTP: site.srv.Client(), Cache: cache}
	if _, err := f2.FetchSeries(context.Background(), Series{Slug: "1-peter", Book: "1Pet", Name: "1 Peter"}); err != nil {
		t.Fatal(err)
	}
	if len(site.requests) != before {
		t.Errorf("a second run fetched %v again", site.requests[before:])
	}
}

func TestFetcherPausesBetweenRequests(t *testing.T) {
	site := newFakeSite(t)
	f := &Fetcher{Base: site.srv.URL, HTTP: site.srv.Client(), Cache: t.TempDir(), Delay: 30 * time.Millisecond}
	start := time.Now()
	if _, err := f.FetchSeries(context.Background(), Series{Slug: "1-peter", Book: "1Pet", Name: "1 Peter"}); err != nil {
		t.Fatal(err)
	}
	// 2 archive pages, the category and the posts: 4 requests, 3 pauses.
	if d := time.Since(start); d < 90*time.Millisecond {
		t.Errorf("4 requests took %v, want at least 3 pauses of 30ms", d)
	}
}

func TestFetcherObeysRobots(t *testing.T) {
	site := newFakeSite(t)
	site.robots = "User-agent: *\nDisallow: /1-peter/\n"
	f := &Fetcher{Base: site.srv.URL, HTTP: site.srv.Client(), Cache: t.TempDir()}
	if err := f.LoadRobots(context.Background()); err != nil {
		t.Fatal(err)
	}
	_, err := f.FetchSeries(context.Background(), Series{Slug: "1-peter", Book: "1Pet", Name: "1 Peter"})
	if err == nil || !strings.Contains(err.Error(), "robots.txt") {
		t.Fatalf("err = %v, want a refusal naming robots.txt", err)
	}
	for _, r := range site.requests {
		if strings.HasPrefix(r, "/1-peter/") {
			t.Errorf("fetched %s though robots.txt disallows it", r)
		}
	}
}

func TestRobotsAllowsWhenDisallowIsEmpty(t *testing.T) {
	r := ParseRobots(mustRead(t, "testdata/robots.txt"))
	if !r.Allowed("/1-peter/") || !r.Allowed("/wp-json/wp/v2/posts") {
		t.Error("an empty Disallow forbids nothing")
	}
	r = ParseRobots("User-agent: Googlebot\nDisallow: /\n\nUser-agent: *\nDisallow: /private/\n")
	if !r.Allowed("/mark/") || r.Allowed("/private/x") {
		t.Error("only the * group's rules apply")
	}
}

func TestFetcherRefusesAnErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()
	f := &Fetcher{Base: srv.URL, HTTP: srv.Client(), Cache: t.TempDir()}
	if _, err := f.FetchSeries(context.Background(), Series{Slug: "1-peter", Book: "1Pet", Name: "1 Peter"}); err == nil {
		t.Fatal("a 404 archive page gave no error")
	}
}

func TestFetcherTriesAgainAfterAServerError(t *testing.T) {
	n := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n++
		if n < 3 {
			http.Error(w, "bad gateway", http.StatusBadGateway)
			return
		}
		io.WriteString(w, "<html><div class=\"entry-content\"><p>Romans 8:1</p></div></html>")
	}))
	defer srv.Close()
	f := &Fetcher{Base: srv.URL, HTTP: srv.Client(), Cache: t.TempDir(), Retry: time.Millisecond}
	got, err := f.FetchBody(context.Background(), Episode{URL: "https://biblethinker.org/a-post/"})
	if err != nil || got != "Romans 8:1" {
		t.Fatalf("got %q, %v after %d requests", got, err, n)
	}
	// a 404 is an answer, not a fault to try again
	n = 0
	nf := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { n++; http.NotFound(w, r) }))
	defer nf.Close()
	f = &Fetcher{Base: nf.URL, HTTP: nf.Client(), Cache: t.TempDir(), Retry: time.Millisecond}
	if _, err := f.FetchBody(context.Background(), Episode{URL: "https://biblethinker.org/gone/"}); err == nil || n != 1 {
		t.Errorf("404: err %v after %d requests", err, n)
	}
}
