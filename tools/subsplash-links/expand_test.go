package main

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// fake is a stand-in for brookfieldfirst.subspla.sh: every dial the client
// makes, whatever the host, reaches it, and it records each request.
type fake struct {
	srv      *httptest.Server
	mu       sync.Mutex
	requests []seen
	redirect map[string]string // path -> Location
	status   map[string]int    // path -> status when not a redirect
}

type seen struct {
	method, host, path, ua string
}

func newFake(t *testing.T) *fake {
	f := &fake{redirect: map[string]string{}, status: map[string]int{}}
	f.srv = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.requests = append(f.requests, seen{r.Method, r.Host, r.URL.Path, r.UserAgent()})
		f.mu.Unlock()
		if loc, ok := f.redirect[r.URL.Path]; ok {
			w.Header().Set("Location", loc)
			w.WriteHeader(http.StatusFound)
			w.Write([]byte("a body the tool must never read"))
			return
		}
		if st, ok := f.status[r.URL.Path]; ok {
			w.WriteHeader(st)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fake) client() *http.Client {
	addr := f.srv.Listener.Addr().String()
	return &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, network, addr)
		},
	}}
}

func (f *fake) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.requests)
}

// clock is a fake clock: Sleep moves it forward.
type clock struct{ t time.Time }

func (c *clock) Now() time.Time        { return c.t }
func (c *clock) Sleep(d time.Duration) { c.t = c.t.Add(d) }

func newExpander(t *testing.T, f *fake, cache *Cache) (*Expander, *clock) {
	ck := &clock{t: time.Date(2031, 3, 2, 12, 0, 0, 0, time.UTC)}
	e := NewExpander(cache)
	e.Client = f.client()
	e.Client.CheckRedirect = noRedirect
	e.Now, e.Sleep = ck.Now, ck.Sleep
	return e, ck
}

func emptyCache(t *testing.T) *Cache {
	c, err := LoadCache(filepath.Join(t.TempDir(), "cache.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

const dest = "https://subsplash.com/u/brookfieldfirst/media/d/aaa1111-the-garden-gate"

// Acceptance 1: HEAD only, no redirect followed, the body not read, the
// short host only, a generic User-Agent.
func TestExpandHeadOnly(t *testing.T) {
	f := newFake(t)
	f.redirect["/aaa1111"] = dest + "?autoplay=true"
	// A second destination that a followed redirect would request.
	f.redirect["/u/brookfieldfirst/media/d/aaa1111-the-garden-gate"] = "https://example.org/never"
	e, _ := newExpander(t, f, emptyCache(t))

	got, err := e.Expand("aaa1111")
	if err != nil {
		t.Fatal(err)
	}
	if got != dest {
		t.Errorf("Expand = %q, want %q (query stripped)", got, dest)
	}
	if f.count() != 1 {
		t.Fatalf("%d requests, want 1: %+v", f.count(), f.requests)
	}
	r := f.requests[0]
	if r.method != http.MethodHead {
		t.Errorf("method = %s, want HEAD", r.method)
	}
	if r.host != "brookfieldfirst.subspla.sh" || r.path != "/aaa1111" {
		t.Errorf("requested %s%s", r.host, r.path)
	}
	if r.ua == "" || strings.HasPrefix(r.ua, "Go-http-client") {
		t.Errorf("User-Agent = %q: want a plain name of our own", r.ua)
	}
	if strings.Contains(strings.ToLower(r.ua), "claude") || strings.Contains(r.ua, "@") {
		t.Errorf("User-Agent = %q is not generic", r.ua)
	}
}

func TestExpandRefusesAnythingButACode(t *testing.T) {
	f := newFake(t)
	e, _ := newExpander(t, f, emptyCache(t))
	for _, code := range []string{"", "a/b", "../x", "aaa1111?x=1", "a b", "https://example.org/x"} {
		if _, err := e.Expand(code); err == nil {
			t.Errorf("Expand(%q) succeeded", code)
		}
	}
	if f.count() != 0 {
		t.Errorf("%d requests made for refused codes", f.count())
	}
}

// Acceptance 1: at most one request a second.
func TestExpandSpacesRequests(t *testing.T) {
	f := newFake(t)
	for _, c := range []string{"aaa1111", "bbb2222", "ccc3333"} {
		f.redirect["/"+c] = "https://subsplash.com/u/brookfieldfirst/media/d/" + c + "-x"
	}
	e, ck := newExpander(t, f, emptyCache(t))
	e.Interval = time.Second
	var at []time.Time
	for _, c := range []string{"aaa1111", "bbb2222", "ccc3333"} {
		if _, err := e.Expand(c); err != nil {
			t.Fatal(err)
		}
		at = append(at, ck.Now())
	}
	for i := 1; i < len(at); i++ {
		if d := at[i].Sub(at[i-1]); d < time.Second {
			t.Errorf("requests %d and %d are %v apart", i, i+1, d)
		}
	}
	// A cached code costs no wait.
	before := ck.Now()
	if _, err := e.Expand("aaa1111"); err != nil || !ck.Now().Equal(before) {
		t.Errorf("cached expand waited: %v %v", err, ck.Now().Sub(before))
	}
}

func TestIntervalHasAFloor(t *testing.T) {
	e := NewExpander(nil)
	e.Interval = time.Millisecond
	if got := e.interval(); got < time.Second {
		t.Errorf("interval = %v: want at least a second", got)
	}
}

// Acceptance 2: the cache keeps code -> address; a second run asks nothing.
func TestExpandCaches(t *testing.T) {
	f := newFake(t)
	f.redirect["/aaa1111"] = dest
	path := filepath.Join(t.TempDir(), "cache.jsonl")
	c, _ := LoadCache(path)
	e, _ := newExpander(t, f, c)
	if _, err := e.Expand("aaa1111"); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Expand("aaa1111"); err != nil {
		t.Fatal(err)
	}
	if f.count() != 1 {
		t.Errorf("%d requests for one code", f.count())
	}
	// The cache is on disk after the request, for a run that is cut short.
	c2, err := LoadCache(path)
	if err != nil {
		t.Fatal(err)
	}
	e2, _ := newExpander(t, f, c2)
	if got, err := e2.Expand("aaa1111"); err != nil || got != dest {
		t.Errorf("second run = %q, %v", got, err)
	}
	if f.count() != 1 {
		t.Errorf("second run made a request: %d", f.count())
	}
}

// A series destination is kept in the cache (so it is not asked again) as
// the address it is; whether it is a sermon is for ParseAddress.
func TestExpandCachesSeriesToo(t *testing.T) {
	f := newFake(t)
	f.redirect["/zzz9999"] = "https://subsplash.com/u/brookfieldfirst/media/s/zzz9999-a-series"
	e, _ := newExpander(t, f, emptyCache(t))
	for i := 0; i < 2; i++ {
		got, err := e.Expand("zzz9999")
		if err != nil || !strings.Contains(got, "/media/s/zzz9999") {
			t.Fatalf("Expand = %q, %v", got, err)
		}
	}
	if f.count() != 1 {
		t.Errorf("%d requests", f.count())
	}
}

func TestExpandErrorsAreNotCached(t *testing.T) {
	f := newFake(t)
	f.status["/aaa1111"] = http.StatusNotFound
	f.status["/bbb2222"] = http.StatusOK // an answer without a Location
	f.redirect["/ccc3333"] = "https://example.org/elsewhere"
	f.redirect["/ddd4444"] = "/relative/path"
	e, _ := newExpander(t, f, emptyCache(t))
	for _, c := range []string{"aaa1111", "bbb2222", "ccc3333", "ddd4444"} {
		if got, err := e.Expand(c); err == nil {
			t.Errorf("Expand(%s) = %q, want an error", c, got)
		}
		if _, ok := e.Cache.Get(c); ok {
			t.Errorf("%s was cached", c)
		}
	}
}

// A 429 or 403 means stop asking.
func TestExpandStopsWhenRefused(t *testing.T) {
	f := newFake(t)
	f.status["/aaa1111"] = http.StatusTooManyRequests
	f.redirect["/bbb2222"] = dest
	e, _ := newExpander(t, f, emptyCache(t))
	if _, err := e.Expand("aaa1111"); err == nil {
		t.Fatal("no error for 429")
	}
	if _, err := e.Expand("bbb2222"); err == nil || !strings.Contains(err.Error(), "stopped") {
		t.Errorf("after a 429: %v", err)
	}
	if f.count() != 1 {
		t.Errorf("%d requests after a 429", f.count())
	}
}
