package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

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
	got, err := FetchPages(context.Background(), srv.Client(), urls, cache, 0, &log)
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
