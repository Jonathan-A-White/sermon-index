package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestFetchVolumes(t *testing.T) {
	body, err := os.ReadFile("testdata/sermons02-two.xml")
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	var got []string
	var agents []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		got = append(got, r.URL.Path)
		agents = append(agents, r.UserAgent())
		mu.Unlock()
		w.Write(body)
	}))
	defer srv.Close()
	urlOf := func(v int) string { return fmt.Sprintf("%s/sermons%02d.xml", srv.URL, v) }

	cache := t.TempDir()
	// A volume already in the cache is not fetched again.
	if err := os.WriteFile(volumePath(cache, 5), body, 0o644); err != nil {
		t.Fatal(err)
	}
	var log bytes.Buffer
	if err := fetchVolumes(context.Background(), srv.Client(), urlOf, cache, 0, &log); err != nil {
		t.Fatal(err)
	}
	if len(got) != 62 {
		t.Fatalf("fetched %d volumes, want 62", len(got))
	}
	for _, p := range got {
		if p == "/sermons05.xml" {
			t.Error("fetched volume 5, which was in the cache")
		}
	}
	for _, a := range agents {
		if !strings.Contains(a, "sermon-index") {
			t.Fatalf("User-Agent %q does not name the project", a)
		}
	}
	// A second run fetches nothing.
	got = nil
	if err := fetchVolumes(context.Background(), srv.Client(), urlOf, cache, 0, &log); err != nil || len(got) != 0 {
		t.Fatalf("second fetch: %v, %d requests", err, len(got))
	}
}

func TestFetchRefusesNotThML(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("<html><body>Not here</body></html>"))
	}))
	defer srv.Close()
	err := fetchVolumes(context.Background(), srv.Client(), func(v int) string { return srv.URL }, t.TempDir(), 0, &bytes.Buffer{})
	if err == nil {
		t.Fatal("an answer that is not ThML was taken")
	}
}

func TestFetchSpacesRequests(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("<?xml version=\"1.0\"?><ThML/>"))
	}))
	defer srv.Close()
	start := time.Now()
	cache := t.TempDir()
	if err := fetchVolumes(context.Background(), srv.Client(), func(v int) string { return srv.URL }, cache, 5*time.Millisecond, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if d := time.Since(start); d < 62*5*time.Millisecond {
		t.Errorf("63 requests in %v: not spaced", d)
	}
}

func TestRunWritesTheSameBytesTwice(t *testing.T) {
	body, err := os.ReadFile("testdata/sermons02-two.xml")
	if err != nil {
		t.Fatal(err)
	}
	cache := t.TempDir()
	for v := 1; v <= volumes; v++ {
		if err := os.WriteFile(volumePath(cache, v), body, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	out := t.TempDir()
	var first []byte
	for i := 0; i < 2; i++ {
		var stdout bytes.Buffer
		if err := run(context.Background(), cache, out, filepath.Join(out, "REPORT.md"), &stdout); err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(filepath.Join(out, "rows.jsonl"))
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			first = b
			if !strings.Contains(stdout.String(), "distinct sermon numbers") {
				t.Errorf("counts not printed: %s", stdout.String())
			}
		} else if !bytes.Equal(first, b) {
			t.Error("the second run wrote other bytes")
		}
	}
	if bytes.Contains(first, []byte(`&`)) {
		t.Error("rows.jsonl escapes & (HTML escaping is on)")
	}
}
