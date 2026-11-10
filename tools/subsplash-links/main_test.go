package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Jonathan-A-White/sermon-index/internal/sources"
)

const testTOML = `id = "brookfield-first"
name = "An invented church's sermons"
kind = "sermons"
licence = "An invented licence."
credit = "Invented"
homepage = "https://example.org/"
terms_url = "https://example.org/terms"
added = 2031-01-01
on_by_default = false
`

const testRows = `{"id":"v1","title":"The Garden Gate","url":"https://www.youtube.com/watch?v=v1","kind":"link","preacher":"A. Preacher","date":"2031-03-02"}
{"id":"v2","title":"Brookfield First - Online Service | March 9, 2031","ref":"Luke.13.22-Luke.13.30","url":"https://www.youtube.com/watch?v=v2","kind":"link","date":"2031-03-09","notes":"Passage named in the video's description."}
{"id":"v3","title":"Brookfield First - Online Service | March 16, 2031","url":"https://www.youtube.com/watch?v=v3","kind":"link","date":"2031-03-16"}
{"id":"v4","title":"Brookfield First - Second Service | March 16, 2031","url":"https://www.youtube.com/watch?v=v4","kind":"link","date":"2031-03-16"}
{"id":"v5","title":"God's Quiet Work","url":"https://www.youtube.com/watch?v=v5","kind":"link","date":"2031-04-06"}
`

const testNotes = `{"id":"v2","date":"2031-03-09","title":"The Narrow Door","ref":"Luke.13.22-Luke.13.30"}
`

type env struct {
	dir, rows, notes, cache, links, pdf string
	f                                   *fake
}

func setup(t *testing.T, links string) *env {
	dir := t.TempDir()
	src := filepath.Join(dir, "sources", "brookfield-first")
	os.MkdirAll(src, 0o755)
	e := &env{
		dir:   dir,
		rows:  filepath.Join(src, "rows.jsonl"),
		notes: filepath.Join(dir, "notes.jsonl"),
		cache: filepath.Join(dir, "cache.jsonl"),
		links: filepath.Join(dir, "links.txt"),
		pdf:   filepath.Join(dir, "search.pdf"),
		f:     newFake(t),
	}
	os.WriteFile(filepath.Join(src, "source.toml"), []byte(testTOML), 0o644)
	os.WriteFile(e.rows, []byte(testRows), 0o644)
	os.WriteFile(e.notes, []byte(testNotes), 0o644)
	os.WriteFile(e.links, []byte(links), 0o644)
	os.WriteFile(e.pdf, []byte("%PDF invented"), 0o644)
	e.f.redirect["/aaa1111"] = "https://subsplash.com/u/brookfieldfirst/media/d/aaa1111-the-garden-gate"
	e.f.redirect["/bbb2222"] = "https://subsplash.com/u/brookfieldfirst/media/d/bbb2222-the-narrow-door"
	e.f.redirect["/ccc3333"] = "https://subsplash.com/u/brookfieldfirst/media/d/ccc3333-gods-quiet-work"
	e.f.redirect["/zzz9999"] = "https://subsplash.com/u/brookfieldfirst/media/s/zzz9999-a-series"
	e.f.redirect["/nnn0000"] = "https://subsplash.com/u/brookfieldfirst/media/d/nnn0000-nobody-has-this-title"
	return e
}

// run runs the tool against the fake, with a fake clock and a fake pdftohtml.
func (e *env) run(t *testing.T, extra ...string) (stdout, stderr string, code int) {
	t.Helper()
	var out, errb bytes.Buffer
	ex := func(cache *Cache) *Expander {
		x, _ := newExpander(t, e.f, cache)
		return x
	}
	args := append([]string{"--rows", e.rows, "--notes", e.notes, "--cache", e.cache}, extra...)
	code = cli(args, &out, &errb, deps{expander: ex, pdfToXML: func(path string) ([]byte, error) {
		if path != e.pdf {
			t.Errorf("pdftohtml asked for %s", path)
		}
		return []byte(fixtureXML), nil
	}})
	return out.String(), errb.String(), code
}

func (e *env) readRows(t *testing.T) []sources.Row {
	t.Helper()
	rs, err := readRows(e.rows)
	if err != nil {
		t.Fatal(err)
	}
	return rs
}

func alt(r sources.Row) string {
	if r.AltURL == nil {
		return ""
	}
	return *r.AltURL
}

const links1 = `https://brookfieldfirst.subspla.sh/aaa1111
https://brookfieldfirst.subspla.sh/bbb2222
https://brookfieldfirst.subspla.sh/ccc3333
https://brookfieldfirst.subspla.sh/zzz9999
https://brookfieldfirst.subspla.sh/nnn0000
https://example.org/never-asked
`

// Acceptance 1-6 end to end.
func TestRunWritesAltURLs(t *testing.T) {
	e := setup(t, links1)
	before, _ := os.ReadFile(e.rows)
	out, errs, code := e.run(t, "--links", e.links)
	if code != 0 {
		t.Fatalf("exit %d: %s%s", code, out, errs)
	}
	rs := e.readRows(t)
	want := map[string]string{
		"v1": "https://subsplash.com/u/brookfieldfirst/media/d/aaa1111-the-garden-gate",
		"v2": "https://subsplash.com/u/brookfieldfirst/media/d/bbb2222-the-narrow-door",
		"v5": "https://subsplash.com/u/brookfieldfirst/media/d/ccc3333-gods-quiet-work",
	}
	for _, r := range rs {
		if alt(r) != want[r.ID] {
			t.Errorf("row %s alt_url = %q, want %q", r.ID, alt(r), want[r.ID])
		}
		if !strings.HasPrefix(r.URL, "https://www.youtube.com/watch?v=") {
			t.Errorf("row %s url changed: %s", r.ID, r.URL)
		}
	}
	// Only the three matched lines changed; the rest are byte for byte.
	bl, al := strings.Split(string(before), "\n"), strings.Split(mustRead(t, e.rows), "\n")
	for i := range bl {
		changed := bl[i] != al[i]
		if changed != (i == 0 || i == 1 || i == 4) {
			t.Errorf("line %d changed = %v", i+1, changed)
		}
	}
	// 5 requests (the series and the unmatched are asked once each); the
	// off-host link never.
	if e.f.count() != 5 {
		t.Errorf("%d requests: %+v", e.f.count(), e.f.requests)
	}
	for _, r := range e.f.requests {
		if r.method != http.MethodHead || r.host != "brookfieldfirst.subspla.sh" {
			t.Errorf("request %+v", r)
		}
	}
	// The series is reported and written nowhere; the unmatched with its slug.
	if !strings.Contains(out, "zzz9999") || !strings.Contains(out, "series") {
		t.Errorf("no series report in:\n%s", out)
	}
	if !strings.Contains(out, "nobody-has-this-title") {
		t.Errorf("no unmatched slug in:\n%s", out)
	}
	if strings.Contains(mustRead(t, e.rows), "zzz9999") {
		t.Error("the series was written")
	}
	if !strings.Contains(out, "example.org/never-asked") {
		t.Errorf("the off-host line is not reported:\n%s", out)
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if last := lines[len(lines)-1]; last != "3 of 5 linked" {
		t.Errorf("last line = %q", last)
	}
}

func mustRead(t *testing.T, path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// Acceptance 2 and 5: running twice changes nothing and asks nothing.
func TestRunTwice(t *testing.T) {
	e := setup(t, links1)
	if _, errs, code := e.run(t, "--links", e.links); code != 0 {
		t.Fatalf("exit %d: %s", code, errs)
	}
	first := mustRead(t, e.rows)
	cache1 := mustRead(t, e.cache)
	asked := e.f.count()
	out, errs, code := e.run(t, "--links", e.links)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errs)
	}
	if mustRead(t, e.rows) != first || mustRead(t, e.cache) != cache1 {
		t.Error("the second run changed a file")
	}
	if e.f.count() != asked {
		t.Errorf("the second run made %d requests", e.f.count()-asked)
	}
	if !strings.HasSuffix(strings.TrimSpace(out), "3 of 5 linked") {
		t.Errorf("out = %s", out)
	}
}

// Acceptance 5: full addresses from a PDF go through the same matching with
// no request; ?autoplay duplicates count once.
func TestRunFromPDF(t *testing.T) {
	e := setup(t, "")
	// fixtureXML holds aaa1111-first-sermon and bbb2222-second-one: make rows for them.
	os.WriteFile(e.rows, []byte(`{"id":"p1","title":"First Sermon","url":"https://www.youtube.com/watch?v=p1","date":"2031-05-04"}
{"id":"p2","title":"Second One","url":"https://www.youtube.com/watch?v=p2","date":"2031-05-11"}
`), 0o644)
	out, errs, code := e.run(t, "--pdf", e.pdf)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errs)
	}
	if e.f.count() != 0 {
		t.Errorf("%d requests for addresses already in full", e.f.count())
	}
	rs := e.readRows(t)
	if alt(rs[0]) != "https://subsplash.com/u/brookfieldfirst/media/d/aaa1111-first-sermon" ||
		alt(rs[1]) != "https://subsplash.com/u/brookfieldfirst/media/d/bbb2222-second-one" {
		t.Errorf("rows = %+v %+v", alt(rs[0]), alt(rs[1]))
	}
	if !strings.Contains(out, "media/s/zzz9999") {
		t.Errorf("the series address in the PDF is not reported:\n%s", out)
	}
	if !strings.HasSuffix(strings.TrimSpace(out), "2 of 2 linked") {
		t.Errorf("out = %s", out)
	}
}

// A short link whose address a PDF already gave is not requested.
func TestRunPDFAddressSavesTheRequest(t *testing.T) {
	e := setup(t, "https://brookfieldfirst.subspla.sh/aaa1111\n")
	os.WriteFile(e.rows, []byte(`{"id":"p1","title":"First Sermon","url":"https://www.youtube.com/watch?v=p1","date":"2031-05-04"}`+"\n"), 0o644)
	if _, errs, code := e.run(t, "--links", e.links, "--pdf", e.pdf); code != 0 {
		t.Fatalf("exit %d: %s", code, errs)
	}
	if e.f.count() != 0 {
		t.Errorf("%d requests", e.f.count())
	}
}

// Acceptance 5: a different alt_url is a conflict and is not written.
func TestRunConflict(t *testing.T) {
	e := setup(t, "https://brookfieldfirst.subspla.sh/aaa1111\n")
	old := `{"id":"v1","title":"The Garden Gate","url":"https://www.youtube.com/watch?v=v1","alt_url":"https://subsplash.com/u/brookfieldfirst/media/d/old0000-the-garden-gate","kind":"link","date":"2031-03-02"}` + "\n"
	os.WriteFile(e.rows, []byte(old), 0o644)
	out, _, code := e.run(t, "--links", e.links)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if mustRead(t, e.rows) != old {
		t.Error("the row was rewritten")
	}
	if !strings.Contains(out, "conflict") || !strings.HasSuffix(strings.TrimSpace(out), "1 of 1 linked") {
		t.Errorf("out = %s", out)
	}
}

// Acceptance 7: --dry-run writes nothing, and asks nothing it has not been asked.
func TestRunDryRun(t *testing.T) {
	e := setup(t, links1)
	before := mustRead(t, e.rows)
	out, _, code := e.run(t, "--links", e.links, "--dry-run")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if mustRead(t, e.rows) != before {
		t.Error("dry run wrote the rows")
	}
	if _, err := os.Stat(e.cache); err == nil {
		t.Error("dry run wrote the cache")
	}
	if e.f.count() != 0 {
		t.Errorf("dry run made %d requests (it would make a link's one request twice)", e.f.count())
	}
	if !strings.Contains(out, "dry run") || !strings.HasSuffix(strings.TrimSpace(out), "0 of 5 linked") {
		t.Errorf("out = %s", out)
	}

	// With the cache filled by a real run, a dry run shows what would be written, and changes nothing.
	if _, _, code := e.run(t, "--links", e.links); code != 0 {
		t.Fatal("real run failed")
	}
	os.WriteFile(e.rows, []byte(before), 0o644)
	cache := mustRead(t, e.cache)
	asked := e.f.count()
	out, _, _ = e.run(t, "--links", e.links, "--dry-run")
	if mustRead(t, e.rows) != before || mustRead(t, e.cache) != cache || e.f.count() != asked {
		t.Error("the second dry run changed something")
	}
	if !strings.HasSuffix(strings.TrimSpace(out), "3 of 5 linked") {
		t.Errorf("a dry run over a full cache should say what would be linked:\n%s", out)
	}
}

// Acceptance 8: the source check passes on the written rows.
func TestRunWrittenRowsPassTheSourceCheck(t *testing.T) {
	e := setup(t, links1)
	if _, errs, code := e.run(t, "--links", e.links); code != 0 {
		t.Fatalf("exit %d: %s", code, errs)
	}
	if faults := sources.Check(filepath.Dir(e.rows)); len(faults) != 0 {
		t.Errorf("faults: %v", faults)
	}
}

func TestRunNeedsAnInput(t *testing.T) {
	e := setup(t, "")
	if _, _, code := e.run(t); code != 2 {
		t.Errorf("exit %d, want 2", code)
	}
}

func TestRunRecordsFileMayBeMissing(t *testing.T) {
	e := setup(t, "https://brookfieldfirst.subspla.sh/aaa1111\n")
	os.Remove(e.notes)
	_, errs, code := e.run(t, "--links", e.links)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errs)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(strings.Split(mustRead(t, e.rows), "\n")[0]), &m); err != nil || m["alt_url"] == nil {
		t.Errorf("row 1 = %v, %v", m, err)
	}
}
