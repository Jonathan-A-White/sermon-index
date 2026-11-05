package api

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Jonathan-A-White/sermon-index/internal/migrate"
	"github.com/Jonathan-A-White/sermon-index/internal/sources"
	"github.com/Jonathan-A-White/sermon-index/internal/store"
)

func TestMain(m *testing.M) { os.Exit(store.RunTests(m)) }

const (
	spurgeon20 = "../../sources/_fixtures/spurgeon-20"
	themesDemo = "../../sources/_fixtures/themes-demo"
	// testVersion is the version the tests' server says it is, the one
	// docs/api.md's /v1/health example shows.
	testVersion = "v0.2.0"
)

// syncBuffer is a bytes.Buffer safe for the server's log writes.
type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

// clock is a settable time for the rate limiter.
type clock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *clock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *clock) add(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

type testServer struct {
	h     http.Handler
	log   *syncBuffer
	clock *clock
}

// loadedDB is a migrated test database with the two fixtures loaded.
func loadedDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()
	db := store.TestDB(t)
	if _, err := migrate.Up(ctx, db); err != nil {
		t.Fatal(err)
	}
	for _, d := range []string{spurgeon20, themesDemo} {
		if _, err := sources.Load(ctx, db, d); err != nil {
			t.Fatalf("load %s: %v", d, err)
		}
	}
	return db
}

func newTestServer(t *testing.T, db *pgxpool.Pool) *testServer {
	t.Helper()
	ts := &testServer{log: &syncBuffer{}, clock: &clock{t: time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)}}
	ts.h = New(Config{
		DB:      db,
		Version: testVersion,
		Origins: ParseOrigins(DefaultOrigins),
		Log:     ts.log,
		Now:     ts.clock.now,
	})
	return ts
}

// get sends a request through the handler; headers are "Name: value".
func (ts *testServer) do(method, target string, headers ...string) *http.Response {
	r := httptest.NewRequest(method, target, nil)
	for _, h := range headers {
		k, v, _ := strings.Cut(h, ": ")
		r.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	ts.h.ServeHTTP(w, r)
	return w.Result()
}

func body(t *testing.T, res *http.Response) string {
	t.Helper()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// errorBody decodes an error response, failing unless it is exactly
// {error, detail}.
func errorBody(t *testing.T, res *http.Response) (string, string) {
	t.Helper()
	if ct := res.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("error Content-Type %q", ct)
	}
	var m map[string]string
	dec := json.NewDecoder(res.Body)
	if err := dec.Decode(&m); err != nil {
		t.Fatalf("error body: %v", err)
	}
	if len(m) != 2 || m["error"] == "" || m["detail"] == "" {
		t.Errorf("error body %v is not {error, detail}", m)
	}
	return m["error"], m["detail"]
}

// An example in docs/api.md is a line "Example: `GET <target>`", or
// "Example (<status>): `GET <target>`", followed by a ```json block: the
// exact body the server answers, byte for byte.
var exampleLine = regexp.MustCompile("^Example(?: \\((\\d{3})\\))?: `GET ([^`]+)`$")

type example struct {
	line   int
	status int
	target string
	body   string
}

func docExamples(t *testing.T) []example {
	t.Helper()
	f, err := os.Open("../../docs/api.md")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var out []example
	var cur *example
	inBlock := false
	var b strings.Builder
	sc := bufio.NewScanner(f)
	n := 0
	for sc.Scan() {
		n++
		line := sc.Text()
		switch {
		case inBlock && line == "```":
			cur.body = b.String()
			out = append(out, *cur)
			cur, inBlock = nil, false
		case inBlock:
			b.WriteString(line + "\n")
		case cur != nil && line == "```json":
			inBlock = true
			b.Reset()
		case cur != nil && strings.TrimSpace(line) != "":
			t.Fatalf("docs/api.md:%d: an example's line is not followed by a ```json block", cur.line)
		default:
			if m := exampleLine.FindStringSubmatch(line); m != nil {
				status := 200
				if m[1] != "" {
					status, _ = strconv.Atoi(m[1])
				}
				cur = &example{line: n, status: status, target: m[2]}
			}
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	if cur != nil {
		t.Fatalf("docs/api.md:%d: an example with no body", cur.line)
	}
	return out
}

// TestDocExamples: every example response in docs/api.md is what the server
// answers, byte for byte, over the two fixtures; and every route has one.
func TestDocExamples(t *testing.T) {
	ts := newTestServer(t, loadedDB(t))
	exs := docExamples(t)
	routes := map[string]bool{}
	for _, ex := range exs {
		res := ts.do("GET", ex.target)
		got := body(t, res)
		if res.StatusCode != ex.status {
			t.Errorf("docs/api.md:%d: GET %s: status %d, the doc says %d", ex.line, ex.target, res.StatusCode, ex.status)
		}
		if got != ex.body {
			t.Errorf("docs/api.md:%d: GET %s answers\n%s\nthe doc says\n%s", ex.line, ex.target, got, ex.body)
		}
		path, _, _ := strings.Cut(ex.target, "?")
		if ex.status == 200 {
			routes[path] = true
		}
	}
	for _, r := range []string{"/v1/health", "/v1/sources", "/v1/sermons", "/v1/themes", "/v1/search"} {
		if !routes[r] {
			t.Errorf("docs/api.md has no 200 example for %s", r)
		}
	}
	// The example the story names: rom.5.8 answers Love's Commendation first.
	found := false
	for _, ex := range exs {
		if ex.target == "/v1/sermons?ref=rom.5.8" {
			found = true
			var resp struct {
				Rows []struct{ Title string }
			}
			if err := json.Unmarshal([]byte(ex.body), &resp); err != nil || len(resp.Rows) == 0 || resp.Rows[0].Title != "Love's Commendation" {
				t.Errorf("the rom.5.8 example does not answer Love's Commendation first: %v %+v", err, resp)
			}
		}
	}
	if !found {
		t.Error("docs/api.md has no example for /v1/sermons?ref=rom.5.8")
	}
}

func TestHealth(t *testing.T) {
	ts := newTestServer(t, loadedDB(t))
	res := ts.do("GET", "/v1/health")
	var h struct {
		Status  string `json:"status"`
		Version string `json:"version"`
		Sources int    `json:"sources"`
		Sermons int    `json:"sermons"`
	}
	if err := json.NewDecoder(res.Body).Decode(&h); err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 200 || h.Status != "ok" || h.Version != testVersion || h.Sources != 2 || h.Sermons != 20 {
		t.Errorf("health: %d %+v", res.StatusCode, h)
	}
}

func TestHealthIs503WhenTheDatabaseIsUnreachable(t *testing.T) {
	db, err := pgxpool.New(context.Background(), "postgres://nobody:x@127.0.0.1:1/none?connect_timeout=2&sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ts := newTestServer(t, db)
	res := ts.do("GET", "/v1/health")
	if res.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("status %d", res.StatusCode)
	}
	if code, _ := errorBody(t, res); code != "unavailable" {
		t.Errorf("error %q", code)
	}
	if res.Header.Get("ETag") != "" || res.Header.Get("Cache-Control") != "no-store" {
		t.Errorf("a 503 is cached: ETag %q, Cache-Control %q", res.Header.Get("ETag"), res.Header.Get("Cache-Control"))
	}
	// Every other route says the same, not a 500.
	res = ts.do("GET", "/v1/sermons?ref=rom.5.8")
	if res.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("sermons with no database: %d", res.StatusCode)
	}
}

func TestCORS(t *testing.T) {
	ts := newTestServer(t, loadedDB(t))
	for _, origin := range []string{"https://lampas.allmymind.org", "http://localhost:5173"} {
		res := ts.do("GET", "/v1/sources", "Origin: "+origin)
		if got := res.Header.Get("Access-Control-Allow-Origin"); got != origin {
			t.Errorf("%s: Allow-Origin %q", origin, got)
		}
		if !strings.Contains(res.Header.Get("Vary"), "Origin") {
			t.Errorf("%s: Vary %q", origin, res.Header.Get("Vary"))
		}
		if !strings.Contains(res.Header.Get("Access-Control-Expose-Headers"), "ETag") {
			t.Errorf("%s: ETag not exposed", origin)
		}
	}
	for _, origin := range []string{"https://evil.example", "https://lampas.allmymind.org.evil.example", "null"} {
		res := ts.do("GET", "/v1/sources", "Origin: "+origin)
		if got := res.Header.Get("Access-Control-Allow-Origin"); got != "" {
			t.Errorf("%s: Allow-Origin %q", origin, got)
		}
	}
	// The preflight allows GET and HEAD only.
	res := ts.do("OPTIONS", "/v1/sermons?ref=rom.5.8", "Origin: https://lampas.allmymind.org", "Access-Control-Request-Method: GET")
	if res.StatusCode != http.StatusNoContent || res.Header.Get("Access-Control-Allow-Methods") != "GET, HEAD" ||
		res.Header.Get("Access-Control-Allow-Origin") != "https://lampas.allmymind.org" {
		t.Errorf("preflight: %d %v", res.StatusCode, res.Header)
	}
	res = ts.do("OPTIONS", "/v1/sermons", "Origin: https://evil.example", "Access-Control-Request-Method: GET")
	if res.Header.Get("Access-Control-Allow-Origin") != "" || res.Header.Get("Access-Control-Allow-Methods") != "" {
		t.Errorf("a preflight from an origin not allowed: %v", res.Header)
	}
	// Any other method is 405, as JSON, with Allow.
	for _, m := range []string{"POST", "PUT", "DELETE", "PATCH"} {
		res := ts.do(m, "/v1/sermons?ref=rom.5.8", "Origin: https://lampas.allmymind.org")
		if res.StatusCode != http.StatusMethodNotAllowed || res.Header.Get("Allow") != "GET, HEAD, OPTIONS" {
			t.Errorf("%s: %d Allow %q", m, res.StatusCode, res.Header.Get("Allow"))
		}
		if code, _ := errorBody(t, res); code != "method_not_allowed" {
			t.Errorf("%s: error %q", m, code)
		}
	}
}

func TestParseOrigins(t *testing.T) {
	got := ParseOrigins(" https://a.example , http://localhost:5173/,,")
	if strings.Join(got, "|") != "https://a.example|http://localhost:5173" {
		t.Errorf("%q", got)
	}
}

func TestETagAndCacheControl(t *testing.T) {
	ts := newTestServer(t, loadedDB(t))
	res := ts.do("GET", "/v1/sermons?ref=Rom.5")
	b := body(t, res)
	etag := res.Header.Get("ETag")
	if res.StatusCode != 200 || !regexp.MustCompile(`^W/"[0-9a-f]{32}"$`).MatchString(etag) {
		t.Fatalf("status %d, ETag %q", res.StatusCode, etag)
	}
	if cc := res.Header.Get("Cache-Control"); cc != "public, max-age=300" {
		t.Errorf("Cache-Control %q", cc)
	}
	if res.Header.Get("Content-Type") != "application/json" || res.Header.Get("Content-Length") != strconv.Itoa(len(b)) {
		t.Errorf("Content-Type %q, Content-Length %q for %d bytes", res.Header.Get("Content-Type"), res.Header.Get("Content-Length"), len(b))
	}
	// The ETag is the body's: the same body, the same tag; another body, another.
	if again := ts.do("GET", "/v1/sermons?ref=Rom.5"); again.Header.Get("ETag") != etag {
		t.Errorf("ETag changed: %q then %q", etag, again.Header.Get("ETag"))
	}
	if other := ts.do("GET", "/v1/sermons?ref=Rom.5.8"); other.Header.Get("ETag") == etag {
		t.Errorf("two bodies, one ETag")
	}
	for _, inm := range []string{etag, strings.TrimPrefix(etag, "W/"), `"x", ` + etag, "*"} {
		res := ts.do("GET", "/v1/sermons?ref=Rom.5", "If-None-Match: "+inm)
		if res.StatusCode != http.StatusNotModified || body(t, res) != "" || res.Header.Get("ETag") != etag ||
			res.Header.Get("Cache-Control") != "public, max-age=300" {
			t.Errorf("If-None-Match %s: %d %v", inm, res.StatusCode, res.Header)
		}
	}
	if res := ts.do("GET", "/v1/sermons?ref=Rom.5", `If-None-Match: W/"0000"`); res.StatusCode != 200 {
		t.Errorf("a stale If-None-Match: %d", res.StatusCode)
	}
	// An error is not cached.
	res = ts.do("GET", "/v1/sermons?ref=Rom.5.99")
	if res.StatusCode != 400 || res.Header.Get("ETag") != "" || res.Header.Get("Cache-Control") != "no-store" {
		t.Errorf("400: ETag %q Cache-Control %q", res.Header.Get("ETag"), res.Header.Get("Cache-Control"))
	}
	// HEAD answers the headers with no body.
	res = ts.do("HEAD", "/v1/sermons?ref=Rom.5")
	if res.StatusCode != 200 || res.Header.Get("ETag") != etag || body(t, res) != "" {
		t.Errorf("HEAD: %d %q", res.StatusCode, res.Header.Get("ETag"))
	}
}

func TestGzipWhenAccepted(t *testing.T) {
	ts := newTestServer(t, loadedDB(t))
	plain := body(t, ts.do("GET", "/v1/sources"))
	res := ts.do("GET", "/v1/sources", "Accept-Encoding: br, gzip")
	if res.Header.Get("Content-Encoding") != "gzip" || !strings.Contains(res.Header.Get("Vary"), "Accept-Encoding") {
		t.Fatalf("Content-Encoding %q, Vary %q", res.Header.Get("Content-Encoding"), res.Header.Get("Vary"))
	}
	zipped := body(t, res)
	if res.Header.Get("Content-Length") != strconv.Itoa(len(zipped)) || len(zipped) >= len(plain) {
		t.Errorf("Content-Length %q for %d bytes (plain %d)", res.Header.Get("Content-Length"), len(zipped), len(plain))
	}
	zr, err := gzip.NewReader(strings.NewReader(zipped))
	if err != nil {
		t.Fatal(err)
	}
	un, err := io.ReadAll(zr)
	if err != nil || string(un) != plain {
		t.Errorf("gunzipped body differs: %v", err)
	}
	if ts.do("GET", "/v1/sources", "Accept-Encoding: gzip").Header.Get("ETag") != ts.do("GET", "/v1/sources").Header.Get("ETag") {
		t.Errorf("the ETag depends on the encoding")
	}
	for _, ae := range []string{"", "identity", "br", "gzip;q=0"} {
		if res := ts.do("GET", "/v1/sources", "Accept-Encoding: "+ae); res.Header.Get("Content-Encoding") != "" {
			t.Errorf("Accept-Encoding %q: gzipped", ae)
		}
	}
}

func TestRateLimit(t *testing.T) {
	ts := newTestServer(t, loadedDB(t))
	from := func(addr string, headers ...string) *http.Response {
		r := httptest.NewRequest("GET", "/v1/health", nil)
		r.RemoteAddr = addr
		for _, h := range headers {
			k, v, _ := strings.Cut(h, ": ")
			r.Header.Set(k, v)
		}
		w := httptest.NewRecorder()
		ts.h.ServeHTTP(w, r)
		return w.Result()
	}
	for i := 0; i < 30; i++ {
		if res := from("203.0.113.5:4000"); res.StatusCode != 200 {
			t.Fatalf("request %d of the burst: %d", i+1, res.StatusCode)
		}
	}
	res := from("203.0.113.5:4001")
	if res.StatusCode != http.StatusTooManyRequests || res.Header.Get("Retry-After") != "1" {
		t.Errorf("past the burst: %d Retry-After %q", res.StatusCode, res.Header.Get("Retry-After"))
	}
	if code, _ := errorBody(t, res); code != "rate_limited" {
		t.Errorf("error %q", code)
	}
	if res := from("203.0.113.6:4000"); res.StatusCode != 200 {
		t.Errorf("another IP is limited too: %d", res.StatusCode)
	}
	// 60 a minute: one token a second.
	ts.clock.add(time.Second)
	if res := from("203.0.113.5:4000"); res.StatusCode != 200 {
		t.Errorf("a second later: %d", res.StatusCode)
	}
	if res := from("203.0.113.5:4000"); res.StatusCode != 429 {
		t.Errorf("the second's one token spent twice: %d", res.StatusCode)
	}
	ts.clock.add(30 * time.Second)
	for i := 0; i < 30; i++ {
		if res := from("203.0.113.5:4000"); res.StatusCode != 200 {
			t.Fatalf("after 30s, request %d: %d", i+1, res.StatusCode)
		}
	}

	// Behind the front door (a private or loopback peer), X-Real-IP is the
	// client; from a public peer it is not believed.
	for i := 0; i < 30; i++ {
		from("10.88.0.1:5000", "X-Real-IP: 198.51.100.7")
	}
	if res := from("10.88.0.1:5000", "X-Real-IP: 198.51.100.7"); res.StatusCode != 429 {
		t.Errorf("X-Real-IP behind the proxy not limited: %d", res.StatusCode)
	}
	if res := from("10.88.0.1:5000", "X-Real-IP: 198.51.100.8"); res.StatusCode != 200 {
		t.Errorf("another client behind the proxy is limited: %d", res.StatusCode)
	}
	if res := from("127.0.0.1:5000", "X-Forwarded-For: 192.0.2.99, 198.51.100.7"); res.StatusCode != 429 {
		t.Errorf("X-Forwarded-For's last hop behind the proxy not limited: %d", res.StatusCode)
	}
	if res := from("203.0.113.5:4000", "X-Real-IP: 198.51.100.9"); res.StatusCode != 429 {
		t.Errorf("a public peer escaped its limit by X-Real-IP: %d", res.StatusCode)
	}
}

func TestOneLogLineARequest(t *testing.T) {
	ts := newTestServer(t, loadedDB(t))
	res := ts.do("GET", "/v1/sermons?ref=rom.5.8")
	n := len(body(t, res))
	ts.do("GET", "/v1/sermons?ref=Rom.5.99")
	ts.do("GET", "/nowhere")
	lines := strings.Split(strings.TrimSuffix(ts.log.String(), "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("%d log lines: %q", len(lines), lines)
	}
	want := []*regexp.Regexp{
		regexp.MustCompile(`^GET /v1/sermons\?ref=rom\.5\.8 200 \d+ms ` + strconv.Itoa(n) + `B$`),
		regexp.MustCompile(`^GET /v1/sermons\?ref=Rom\.5\.99 400 \d+ms \d+B$`),
		regexp.MustCompile(`^GET /nowhere 404 \d+ms \d+B$`),
	}
	for i, re := range want {
		if !re.MatchString(lines[i]) {
			t.Errorf("log line %d: %q", i+1, lines[i])
		}
	}
}

func TestNotFoundIsJSON(t *testing.T) {
	ts := newTestServer(t, loadedDB(t))
	for _, p := range []string{"/nope", "/v1", "/v1/sermon", "/v2/sermons?ref=rom.5.8"} {
		res := ts.do("GET", p)
		if res.StatusCode != 404 {
			t.Errorf("%s: %d", p, res.StatusCode)
		}
		if code, _ := errorBody(t, res); code != "not_found" {
			t.Errorf("%s: error %q", p, code)
		}
	}
}

// The bare address is a short page for a person: the service's name, the
// counts /v1/health gives, the routes and one example link.
func TestBareAddressIsAPage(t *testing.T) {
	ts := newTestServer(t, loadedDB(t))
	res := ts.do("GET", "/")
	if res.StatusCode != 200 {
		t.Fatalf("GET /: %d", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("GET / Content-Type %q", ct)
	}
	b := body(t, res)
	for _, want := range []string{
		"Sermon index",
		"2 sources", "20 sermons",
		"/v1/health", "/v1/sources", "/v1/sermons", "/v1/themes", "/v1/search",
		`href="/v1/sermons?ref=rom.5.8"`,
		"docs/api.md",
	} {
		if !strings.Contains(b, want) {
			t.Errorf("GET / lacks %q:\n%s", want, b)
		}
	}
	if strings.Contains(b, "not_found") {
		t.Errorf("GET / is an error:\n%s", b)
	}
	// HEAD answers the headers alone.
	if h := ts.do("HEAD", "/"); h.StatusCode != 200 {
		t.Errorf("HEAD /: %d", h.StatusCode)
	}
}

func TestBareAddressIs503WhenTheDatabaseIsUnreachable(t *testing.T) {
	db := loadedDB(t)
	ts := newTestServer(t, db)
	db.Close()
	res := ts.do("GET", "/")
	if res.StatusCode != 503 {
		t.Errorf("GET / with the database down: %d", res.StatusCode)
	}
}
