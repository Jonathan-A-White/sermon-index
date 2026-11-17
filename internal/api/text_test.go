package api

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Jonathan-A-White/sermon-index/internal/sources"
)

// ccelPage is a saved CCEL section page (Spurgeon's sermon 311, its words
// replaced by short stand-ins): the site's navigation, the sermon's head
// block, its body and the navigation again.
const ccelPage = "../../sources/_fixtures/ccel-pages/sermons06.xxiv.html"

// wantBody is the paragraphs of the saved page's sermon: not the head
// block (A Sermon, No., Delivered ..., REV., At ...), not the navigation.
var wantBody = []string{
	"“A placeholder for the preaching text.”—Job 8:7.",
	"First paragraph of the sermon, a few plain words standing in for CCEL’s page.",
	"Second paragraph, with an emphasis and a Job 8:7 reference & an ampersand.",
	"Third paragraph, which ends the body.",
}

// pageFetcher answers every address with the saved page and records what it
// was asked and when.
type pageFetcher struct {
	mu    sync.Mutex
	urls  []string
	times []time.Time
	page  []byte
	err   error
}

func newPageFetcher(t *testing.T) *pageFetcher {
	t.Helper()
	b, err := os.ReadFile(ccelPage)
	if err != nil {
		t.Fatal(err)
	}
	return &pageFetcher{page: b}
}

func (f *pageFetcher) fetch(ctx context.Context, url string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.urls = append(f.urls, url)
	f.times = append(f.times, time.Now())
	return f.page, f.err
}

func (f *pageFetcher) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.urls)
}

type textJSON struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	Source     string   `json:"source"`
	URL        string   `json:"url"`
	Paragraphs []string `json:"paragraphs"`
}

func textServer(t *testing.T, f *pageFetcher, dir string, delay time.Duration) *testServer {
	t.Helper()
	return newTestServerWith(t, loadedDB(t), func(c *Config) {
		c.TextFetch = f.fetch
		c.TextDir = dir
		c.TextDelay = delay
	})
}

func TestTextServesTheSermonBodyOnly(t *testing.T) {
	f := newPageFetcher(t)
	ts := textServer(t, f, t.TempDir(), time.Millisecond)
	res := ts.do("GET", "/v1/text?id=1")
	if res.StatusCode != 200 {
		t.Fatalf("status %d: %s", res.StatusCode, body(t, res))
	}
	var got textJSON
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	want := textJSON{ID: "1", Title: "The Immutability of God", Source: "spurgeon-20",
		URL: "https://ccel.org/ccel/spurgeon/sermons01/sermons01.i.html", Paragraphs: wantBody}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("answer\n%+v\nwant\n%+v", got, want)
	}
	if !reflect.DeepEqual(f.urls, []string{want.URL}) {
		t.Errorf("fetched %v, want the row's url", f.urls)
	}
	if ct := res.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type %q", ct)
	}
}

// CORS as the other /v1 routes.
func TestTextHasCORS(t *testing.T) {
	ts := textServer(t, newPageFetcher(t), t.TempDir(), time.Millisecond)
	res := ts.do("GET", "/v1/text?id=1", "Origin: https://lampas.allmymind.org")
	if got := res.Header.Get("Access-Control-Allow-Origin"); got != "https://lampas.allmymind.org" {
		t.Errorf("Access-Control-Allow-Origin %q", got)
	}
	pre := ts.do("OPTIONS", "/v1/text?id=1", "Origin: https://lampas.allmymind.org", "Access-Control-Request-Method: GET")
	if pre.StatusCode != 204 {
		t.Errorf("preflight %d", pre.StatusCode)
	}
}

// One fetch a sermon, kept on disk: a second request, and a new server over
// the same directory, do not fetch again.
func TestTextIsFetchedOnceAndCachedOnDisk(t *testing.T) {
	f := newPageFetcher(t)
	dir := t.TempDir()
	ts := textServer(t, f, dir, time.Millisecond)
	for i := 0; i < 3; i++ {
		if res := ts.do("GET", "/v1/text?id=1"); res.StatusCode != 200 {
			t.Fatalf("request %d: %d", i, res.StatusCode)
		}
	}
	if f.count() != 1 {
		t.Errorf("%d fetches for one sermon, want 1", f.count())
	}
	ts2 := textServer(t, f, dir, time.Millisecond)
	res := ts2.do("GET", "/v1/text?id=1")
	var got textJSON
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil || !reflect.DeepEqual(got.Paragraphs, wantBody) {
		t.Errorf("from the disk cache: %v %+v", err, got)
	}
	if f.count() != 1 {
		t.Errorf("a new server fetched again: %d", f.count())
	}
	var files []string
	filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			files = append(files, p)
		}
		return nil
	})
	if len(files) != 1 {
		t.Errorf("cache files %v, want one", files)
	}
	// Two requests for one sermon at once make one fetch.
	f2 := newPageFetcher(t)
	ts3 := textServer(t, f2, t.TempDir(), time.Millisecond)
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); ts3.do("GET", "/v1/text?id=12") }()
	}
	wg.Wait()
	if f2.count() != 1 {
		t.Errorf("four requests at once fetched %d times", f2.count())
	}
}

// The pace: requests to one host wait at least the delay between them.
func TestTextFetchesToOneHostAreSpacedByTheDelay(t *testing.T) {
	f := newPageFetcher(t)
	delay := 150 * time.Millisecond
	ts := textServer(t, f, t.TempDir(), delay)
	var wg sync.WaitGroup
	for _, id := range []string{"1", "12", "37"} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if res := ts.do("GET", "/v1/text?id="+id); res.StatusCode != 200 {
				t.Errorf("id %s: %d", id, res.StatusCode)
			}
		}()
	}
	wg.Wait()
	if len(f.times) != 3 {
		t.Fatalf("%d fetches", len(f.times))
	}
	for i := 1; i < 3; i++ {
		if gap := f.times[i].Sub(f.times[i-1]); gap < delay-5*time.Millisecond {
			t.Errorf("fetch %d came %v after the one before, want at least %v", i, gap, delay)
		}
	}
}

// CCEL's Crawl-delay is 10 seconds and the default is no shorter.
func TestTextDefaultPaceIsCCELsCrawlDelay(t *testing.T) {
	if DefaultTextDelay < 10*time.Second {
		t.Errorf("DefaultTextDelay %v, CCEL's robots.txt says Crawl-delay: 10", DefaultTextDelay)
	}
}

// A request that would wait too long for its turn is told to come back.
func TestTextAQueueTooLongAnswers503(t *testing.T) {
	f := newPageFetcher(t)
	ts := textServer(t, f, t.TempDir(), 40*time.Second)
	if res := ts.do("GET", "/v1/text?id=1"); res.StatusCode != 200 {
		t.Fatalf("first: %d", res.StatusCode)
	}
	if res := ts.do("GET", "/v1/text?id=12"); res.StatusCode != 200 {
		// the second would wait 40 s: over the longest wait
		code, _ := errorBody(t, res)
		if res.StatusCode != 503 || code != "busy" || res.Header.Get("Retry-After") == "" {
			t.Errorf("second: %d %s Retry-After %q", res.StatusCode, code, res.Header.Get("Retry-After"))
		}
	} else {
		t.Error("the second request did not wait or refuse")
	}
	if f.count() != 1 {
		t.Errorf("%d fetches", f.count())
	}
}

func TestTextFailuresAreNotCached(t *testing.T) {
	f := newPageFetcher(t)
	f.err = os.ErrDeadlineExceeded
	ts := textServer(t, f, t.TempDir(), time.Millisecond)
	res := ts.do("GET", "/v1/text?id=1")
	if code, _ := errorBody(t, res); res.StatusCode != 502 || code != "fetch_failed" {
		t.Errorf("a failed fetch: %d %s", res.StatusCode, code)
	}
	// CCEL answers 200 to a section it does not have: a page with no sermon.
	f.err = nil
	f.page = []byte("<html><body><h1>Not found</h1></body></html>")
	res = ts.do("GET", "/v1/text?id=1")
	if code, _ := errorBody(t, res); res.StatusCode != 502 || code != "no_text_in_page" {
		t.Errorf("a page with no sermon: %d %s", res.StatusCode, code)
	}
	b, _ := os.ReadFile(ccelPage)
	f.page = b
	if res := ts.do("GET", "/v1/text?id=1"); res.StatusCode != 200 {
		t.Errorf("after the page is right: %d", res.StatusCode)
	}
	if f.count() != 3 {
		t.Errorf("%d fetches, want 3 (the failures are not kept)", f.count())
	}
}

func TestTextRefusesWhatItDoesNotServe(t *testing.T) {
	ctx := context.Background()
	// A source with no `text`, and one whose host has no reader.
	closed := sermonSourceDir(t, "closed", "", `{"id": "1", "title": "Not public text", "url": "https://ccel.org/x/1.html"}`)
	odd := sermonSourceDir(t, "odd-host", `text = "public-domain"`, `{"id": "7", "title": "Elsewhere", "url": "https://example.org/7"}`)
	db := loadedDB(t)
	for _, d := range []string{closed, odd} {
		if _, err := sources.Load(ctx, db, d); err != nil {
			t.Fatal(err)
		}
	}
	f := newPageFetcher(t)
	ts := newTestServerWith(t, db, func(c *Config) { c.TextFetch = f.fetch; c.TextDir = t.TempDir(); c.TextDelay = time.Millisecond })
	for _, c := range []struct {
		target, code string
		status       int
	}{
		{"/v1/text?id=1&source=closed", "no_text", 404},
		{"/v1/text?id=7", "no_text", 404},
		{"/v1/text?id=nosuch", "not_found", 404},
		{"/v1/text?id=1&source=nosuch", "not_found", 404},
		{"/v1/text", "bad_id", 400},
		{"/v1/text?id=", "bad_id", 400},
	} {
		res := ts.do("GET", c.target)
		code, detail := errorBody(t, res)
		if res.StatusCode != c.status || code != c.code {
			t.Errorf("%s: %d %s (%s), want %d %s", c.target, res.StatusCode, code, detail, c.status, c.code)
		}
	}
	if f.count() != 0 {
		t.Errorf("a refused request fetched %d pages", f.count())
	}
	// id 1 is in spurgeon-20 (text) and in closed (not): only the first is served.
	res := ts.do("GET", "/v1/text?id=1")
	var got textJSON
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil || got.Source != "spurgeon-20" {
		t.Errorf("id 1 over a served and a closed source: %d %v %+v", res.StatusCode, err, got)
	}
}

// Two public-domain sources with one id: the caller names the source.
func TestTextAnIdInTwoServedSourcesNeedsASource(t *testing.T) {
	ctx := context.Background()
	other := sermonSourceDir(t, "other-ccel", `text = "public-domain"`, `{"id": "1", "title": "Another One", "url": "https://ccel.org/other/1.html"}`)
	db := loadedDB(t)
	if _, err := sources.Load(ctx, db, other); err != nil {
		t.Fatal(err)
	}
	ts := newTestServerWith(t, db, func(c *Config) {
		c.TextFetch = newPageFetcher(t).fetch
		c.TextDir = t.TempDir()
		c.TextDelay = time.Millisecond
	})
	res := ts.do("GET", "/v1/text?id=1")
	if code, detail := errorBody(t, res); res.StatusCode != 400 || code != "ambiguous_id" || !strings.Contains(detail, "spurgeon-20") || !strings.Contains(detail, "other-ccel") {
		t.Errorf("ambiguous: %d %s %s", res.StatusCode, code, detail)
	}
	res = ts.do("GET", "/v1/text?id=1&source=other-ccel")
	var got textJSON
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil || got.Source != "other-ccel" || got.Title != "Another One" {
		t.Errorf("named source: %d %v %+v", res.StatusCode, err, got)
	}
}

func sermonSourceDir(t *testing.T, id, extra string, rows ...string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), id)
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	toml := "id = \"" + id + "\"\nname = \"Source " + id + "\"\nkind = \"sermons\"\nlicence = \"Public domain.\"\ncredit = \"Someone\"\n" +
		"homepage = \"https://example.org\"\nterms_url = \"https://example.org/terms\"\nadded = 2026-11-05\non_by_default = true\n" + extra + "\n"
	if err := os.WriteFile(filepath.Join(dir, "source.toml"), []byte(toml), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "rows.jsonl"), []byte(strings.Join(rows, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestCCELParagraphs(t *testing.T) {
	page := func(inner string) []byte {
		return []byte(`<html><body><div id="content"><table class="book_navbar"><tr><td>« Prev</td></tr></table>` +
			`<div class="book-content"><p class="Body" id="a1"></p><h1 id="a2">The Title </h1>` + inner +
			`</div><table class="book_navbar" id="book_navbar_bottom"><tr><td>Next »</td></tr></table></div></body></html>`)
	}
	for _, c := range []struct {
		name, inner string
		want        []string
	}{
		{"head block then body",
			`<p>A Sermon</p><p>(No. 12)</p><p>Delivered on Sabbath Morning, April 29th, 1860, by the</p><p>REV. C.H. SPURGEON</p><p>At Exeter Hall, Strand.</p><p></p>` +
				`<p>“Text.”—Ps 1:1.</p><p>Body one.</p>`,
			[]string{"“Text.”—Ps 1:1.", "Body one."}},
		{"a body that begins like a head line is kept after the text",
			`<p>A Sermon</p><p>“Text.”—Ps 1:1.</p><p>At the first, we begin.</p>`,
			[]string{"“Text.”—Ps 1:1.", "At the first, we begin."}},
		{"entities, whitespace and tags",
			"<p>One &amp;\n   two&nbsp;three <i>four</i> &#8220;five&#8221;</p>",
			[]string{"One & two three four “five”"}},
		{"footnotes and notes are left out",
			`<p>Text<a class="fnref" href="#n1">1</a> goes on.</p><div class="Note"><p>A note of the editor.</p></div><p>Next.</p>`,
			[]string{"Text goes on.", "Next."}},
		{"a sub-heading is a paragraph", `<p>One.</p><h3>II. Second</h3><p>Two.</p>`, []string{"One.", "II. Second", "Two."}},
	} {
		got, err := ccelParagraphs(page(c.inner))
		if err != nil || !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: %q %v, want %q", c.name, got, err, c.want)
		}
	}
	if _, err := ccelParagraphs([]byte("<html>no sermon</html>")); err == nil {
		t.Error("a page with no book-content is not refused")
	}
	if _, err := ccelParagraphs(page("<p></p>")); err == nil {
		t.Error("a page with no paragraphs is not refused")
	}
}

func TestCCELParagraphsOfTheSavedPage(t *testing.T) {
	b, err := os.ReadFile(ccelPage)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ccelParagraphs(b)
	if err != nil || !reflect.DeepEqual(got, wantBody) {
		t.Errorf("%q %v", got, err)
	}
	for _, p := range got {
		for _, bad := range []string{"Prev", "Next", "Contents", "Delivered", "Sermon 311"} {
			if strings.Contains(p, bad) {
				t.Errorf("paragraph %q holds %q", p, bad)
			}
		}
	}
}

func TestTextRouteIsOnTheHomePage(t *testing.T) {
	ts := textServer(t, newPageFetcher(t), t.TempDir(), time.Millisecond)
	if b := body(t, ts.do("GET", "/")); !strings.Contains(b, "/v1/text") {
		t.Errorf("the home page does not list /v1/text:\n%s", b)
	}
	var _ http.Handler = ts.h
}
