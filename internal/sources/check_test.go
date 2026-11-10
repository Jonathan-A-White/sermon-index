package sources

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const fixture = "../../sources/_fixtures/spurgeon-20"

const goodToml = `id = "demo"
name = "Demo"
kind = "sermons"
licence = "Public domain."
credit = "Someone"
homepage = "https://example.org"
terms_url = "https://example.org/terms"
added = 2026-11-05
on_by_default = false
`

const goodThemeToml = `id = "demo"
name = "Demo themes"
kind = "themes"
licence = "Public domain."
credit = "Someone"
homepage = "https://example.org"
terms_url = "https://example.org/terms"
added = 2026-11-05
on_by_default = true
`

const goodRow = `{"id": "1", "title": "One", "ref": "Rom.5.8", "url": "https://example.org/1"}`

// writeSource makes sources/demo in a temp directory with the files given.
func writeSource(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "demo")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func faultStrings(fs []Fault) []string {
	var out []string
	for _, f := range fs {
		out = append(out, f.String())
	}
	return out
}

// expectFault checks that the source's faults include want exactly.
func expectFault(t *testing.T, files map[string]string, want string) {
	t.Helper()
	got := faultStrings(Check(writeSource(t, files)))
	for _, g := range got {
		if g == want {
			return
		}
	}
	t.Errorf("faults %q\n  do not include %q", got, want)
}

func TestFixtureAndTemplatePass(t *testing.T) {
	for _, dir := range []string{fixture, "../../sources/_template"} {
		if fs := Check(dir); len(fs) != 0 {
			t.Errorf("%s: %q", dir, faultStrings(fs))
		}
	}
}

func TestAGoodSourcePasses(t *testing.T) {
	if fs := Check(writeSource(t, map[string]string{"source.toml": goodToml, "rows.jsonl": goodRow + "\n"})); len(fs) != 0 {
		t.Errorf("%q", faultStrings(fs))
	}
}

func TestFaultVersePastTheChapterNamesTheLine(t *testing.T) {
	rows := ""
	for i := 1; i <= 6; i++ {
		rows += strings.Replace(goodRow, `"1"`, fmt.Sprintf(`"%d"`, i), 1) + "\n"
	}
	rows += `{"id": "7", "title": "Seven", "ref": "Rom.5.99", "url": "https://example.org/7"}` + "\n"
	expectFault(t, map[string]string{"source.toml": goodToml, "rows.jsonl": rows},
		"rows.jsonl:7: ref 'Rom.5.99': verse past the chapter")
}

func TestCheckDotKnowsTheDirectorysName(t *testing.T) {
	dir := writeSource(t, map[string]string{"source.toml": goodToml, "rows.jsonl": goodRow + "\n"})
	t.Chdir(dir)
	if fs := Check("."); len(fs) != 0 {
		t.Errorf("%q", faultStrings(fs))
	}
}

func TestFaultsInSourceToml(t *testing.T) {
	expectFault(t, map[string]string{"rows.jsonl": goodRow}, "source.toml: missing")
	expectFault(t, map[string]string{"source.toml": strings.Replace(goodToml, `licence = "Public domain."`, `licence = "  "`, 1), "rows.jsonl": goodRow},
		"source.toml: licence is empty")
	expectFault(t, map[string]string{"source.toml": strings.Replace(goodToml, "credit = \"Someone\"\n", "", 1), "rows.jsonl": goodRow},
		"source.toml: credit is missing")
	expectFault(t, map[string]string{"source.toml": strings.Replace(goodToml, "added = 2026-11-05\n", "", 1), "rows.jsonl": goodRow},
		"source.toml: added is missing")
	expectFault(t, map[string]string{"source.toml": strings.Replace(goodToml, "on_by_default = false\n", "", 1), "rows.jsonl": goodRow},
		"source.toml: on_by_default is missing")
	expectFault(t, map[string]string{"source.toml": goodToml + "licnce = \"x\"\n", "rows.jsonl": goodRow},
		"source.toml: unknown key 'licnce'")
	expectFault(t, map[string]string{"source.toml": strings.Replace(goodToml, `kind = "sermons"`, `kind = "videos"`, 1), "rows.jsonl": goodRow},
		"source.toml: kind 'videos': not sermons or themes")
	expectFault(t, map[string]string{"source.toml": strings.Replace(goodToml, `id = "demo"`, `id = "other"`, 1), "rows.jsonl": goodRow},
		"source.toml: id 'other' is not the directory's name 'demo'")
	expectFault(t, map[string]string{"source.toml": strings.Replace(goodToml, `homepage = "https://example.org"`, `homepage = "http://example.org"`, 1), "rows.jsonl": goodRow},
		"source.toml: homepage 'http://example.org': not https")
	if got := faultStrings(Check(writeSource(t, map[string]string{"source.toml": "id = \n", "rows.jsonl": goodRow}))); len(got) != 1 || !strings.HasPrefix(got[0], "source.toml: not TOML: ") {
		t.Errorf("bad TOML: %q", got)
	}
}

func TestLanguageKey(t *testing.T) {
	with := func(lang string) map[string]string {
		return map[string]string{"source.toml": goodToml + lang, "rows.jsonl": goodRow}
	}
	for _, code := range []string{"en", "pt", "la", "de", "nl", "fr", "el", "he"} {
		if got := faultStrings(Check(writeSource(t, with("language = \""+code+"\"\n")))); len(got) != 0 {
			t.Errorf("language %q: %q", code, got)
		}
	}
	expectFault(t, with("language = \"xx\"\n"), "source.toml: language 'xx': not one of en, pt, la, de, nl, fr, el, he")
	expectFault(t, with("language = \"PT\"\n"), "source.toml: language 'PT': not one of en, pt, la, de, nl, fr, el, he")
	expectFault(t, with("language = \"\"\n"), "source.toml: language is empty: leave it out for English")
	d, fs := Read(writeSource(t, with("language = \"pt\"\n")))
	if len(fs) != 0 || d.Source.Language != "pt" || d.Source.LanguageOrDefault() != "pt" {
		t.Errorf("read: %q %+v", faultStrings(fs), d.Source)
	}
	d, _ = Read(writeSource(t, map[string]string{"source.toml": goodToml, "rows.jsonl": goodRow}))
	if d.Source.Language != "" || d.Source.LanguageOrDefault() != "en" {
		t.Errorf("no language: %+v", d.Source)
	}
}

func TestFaultsInRows(t *testing.T) {
	src := func(rows string) map[string]string {
		return map[string]string{"source.toml": goodToml, "rows.jsonl": rows}
	}
	expectFault(t, map[string]string{"source.toml": goodToml}, "rows.jsonl: missing")
	expectFault(t, src(""), "rows.jsonl: no rows")
	expectFault(t, src(goodRow+"\n{not json\n"), "rows.jsonl:2: not a JSON object: invalid character 'n' looking for beginning of object key string")
	expectFault(t, src(`{"id": "1", "titel": "One", "ref": "Rom.5.8", "url": "https://example.org/1"}`), `rows.jsonl:1: unknown field "titel"`)
	expectFault(t, src(`{"id": "1", "ref": "Rom.5.8", "url": "https://example.org/1"}`), "rows.jsonl:1: title is missing")
	expectFault(t, src(`{"title": "One", "ref": "Rom.5.8", "url": "https://example.org/1"}`), "rows.jsonl:1: id is missing")
	expectFault(t, src(`{"id": "1", "title": "One", "ref": "Rom.5.8"}`), "rows.jsonl:1: url is missing")
	expectFault(t, src(`{"id": "1", "title": "One", "ref": "Tob.1.1", "url": "https://example.org/1"}`), "rows.jsonl:1: ref 'Tob.1.1': unknown book 'Tob'")
	expectFault(t, src(`{"id": "1", "title": "One", "ref": "Rom.5.8", "cited": ["Rom.5.1", "Rom.17.1"], "url": "https://example.org/1"}`),
		"rows.jsonl:1: cited ref 'Rom.17.1': chapter past the book")
	expectFault(t, src(goodRow+"\n"+goodRow+"\n"), "rows.jsonl:2: id '1' repeats line 1")
	expectFault(t, src(`{"id": "1", "title": "One", "ref": "Rom.5.8", "url": "http://example.org/1"}`), "rows.jsonl:1: url 'http://example.org/1': not https")
	expectFault(t, src(`{"id": "1", "title": "One", "ref": "Rom.5.8", "url": "https://"}`), "rows.jsonl:1: url 'https://': not a well-formed URL")
	expectFault(t, src(`{"id": "1", "title": "One", "ref": "Rom.5.8", "url": "https://exa mple.org/1"}`), "rows.jsonl:1: url 'https://exa mple.org/1': not a well-formed URL")
	expectFault(t, src(`{"id": "1", "title": "One", "ref": "Rom.5.8", "url": "https://example.org/1", "alt_url": "ftp://example.org/1"}`), "rows.jsonl:1: alt_url 'ftp://example.org/1': not https")
	expectFault(t, src(`{"id": "1", "title": "One", "ref": "Rom.5.8", "url": "https://example.org/1", "kind": "video"}`), "rows.jsonl:1: kind 'video': not link, hosted or exposition")
	expectFault(t, src(`{"id": "1", "title": "One", "ref": "Rom.5.8", "url": "https://example.org/1", "date": "5 Nov 1855"}`), "rows.jsonl:1: date '5 Nov 1855': not YYYY, YYYY-MM or YYYY-MM-DD")
	expectFault(t, src(`{"id": "1", "title": "One", "ref": "Rom.5.8", "url": "https://example.org/1", "date": "1855-02-30"}`), "rows.jsonl:1: date '1855-02-30': not YYYY, YYYY-MM or YYYY-MM-DD")
	expectFault(t, src(`{"id": "1", "title": "One", "ref": "Rom.5.8", "url": "https://example.org/1", "duration_s": 0}`), "rows.jsonl:1: duration_s 0: not a positive number of seconds")
	expectFault(t, src(`{"id": "a b", "title": "One", "ref": "Rom.5.8", "url": "https://example.org/1"}`), "rows.jsonl:1: id 'a b': only letters, digits, '.', '_' and '-'")
	expectFault(t, src(`{"id": "1", "title": "One", "ref": "Rom.5.8", "url": "https://example.org/1", "preacher": ""}`), "rows.jsonl:1: preacher is empty: leave it out instead")
	expectFault(t, map[string]string{"source.toml": goodToml, "rows.jsonl": goodRow, "topics.jsonl": `{"id": "love", "label": "Love"}`},
		"topics.jsonl: a sermons source has no topics.jsonl")
}

func TestFaultsInAThemeSource(t *testing.T) {
	topics := `{"id": "love", "label": "Love", "parts": [{"n": 1, "label": "Of God"}, {"n": 2, "label": "Of man"}], "see": ["grace"]}
{"id": "grace", "label": "Grace"}
`
	src := func(topics, links string) map[string]string {
		return map[string]string{"source.toml": goodThemeToml, "topics.jsonl": topics, "links.jsonl": links}
	}
	good := `{"topic": "love", "part": 1, "ref": "Rom.5.8"}
{"topic": "grace", "ref": "Rom.5.15-Rom.5.17", "via": "bible"}
{"topic": "grace", "ref": "Eph.1.7", "via": "sermon", "from": "spurgeon-ccel:330"}
`
	if fs := Check(writeSource(t, src(topics, good))); len(fs) != 0 {
		t.Fatalf("a good theme source: %q", faultStrings(fs))
	}
	expectFault(t, src(topics, good+`{"topic": "hope", "ref": "Rom.5.5"}`+"\n"), "links.jsonl:4: topic 'hope': not in topics.jsonl")
	expectFault(t, src(topics, `{"topic": "love", "part": 3, "ref": "Rom.5.8"}`), "links.jsonl:1: part 3: topic 'love' has no such part")
	expectFault(t, src(topics, `{"topic": "love", "ref": "Rom.5.99"}`), "links.jsonl:1: ref 'Rom.5.99': verse past the chapter")
	expectFault(t, src(topics, `{"topic": "love", "ref": "Rom.5.8", "via": "dream"}`), "links.jsonl:1: via 'dream': not source, bible or sermon")
	expectFault(t, src(topics, `{"topic": "love", "ref": "Rom.5.8", "via": "sermon"}`), "links.jsonl:1: from is missing: a link via a sermon names it as <source>:<id>")
	expectFault(t, src(topics+`{"id": "love", "label": "Again"}`+"\n", good), "topics.jsonl:3: id 'love' repeats line 1")
	expectFault(t, src(`{"id": "love", "label": "Love", "see": ["hope"]}`, `{"topic": "love", "ref": "Rom.5.8"}`), "topics.jsonl:1: see 'hope': not in topics.jsonl")
	expectFault(t, src(`{"id": "love", "label": "Love", "parts": [{"n": 1, "label": "A"}, {"n": 1, "label": "B"}]}`, `{"topic": "love", "ref": "Rom.5.8"}`), "topics.jsonl:1: part 1 repeats")
	expectFault(t, src(`{"id": "love"}`, `{"topic": "love", "ref": "Rom.5.8"}`), "topics.jsonl:1: label is missing")
	expectFault(t, map[string]string{"source.toml": goodThemeToml, "links.jsonl": good}, "topics.jsonl: missing")
	expectFault(t, map[string]string{"source.toml": goodThemeToml, "topics.jsonl": topics}, "links.jsonl: missing")
	expectFault(t, map[string]string{"source.toml": goodThemeToml, "topics.jsonl": topics, "links.jsonl": good, "rows.jsonl": goodRow},
		"rows.jsonl: a themes source has no rows.jsonl")
}

func TestCheckURLsReportsAnyNot200(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodHead {
			t.Errorf("method %s, want HEAD", r.Method)
		}
		if r.URL.Path == "/gone" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	rows := `{"id": "1", "title": "One", "ref": "Rom.5.8", "url": "` + srv.URL + `/one"}
{"id": "2", "title": "Two", "ref": "Rom.5.8", "url": "` + srv.URL + `/gone"}
`
	dir := writeSource(t, map[string]string{"source.toml": goodToml, "rows.jsonl": rows})
	d, fs := Read(dir)
	if len(fs) != 0 {
		t.Fatal(faultStrings(fs))
	}
	got := faultStrings(CheckURLs(context.Background(), d, 10, srv.Client(), rand.New(rand.NewSource(1))))
	want := "rows.jsonl:2: url '" + srv.URL + "/gone': 404 Not Found"
	if len(got) != 1 || got[0] != want {
		t.Errorf("CheckURLs = %q, want [%q]", got, want)
	}

	// N smaller than the rows checks only N of them.
	calls := 0
	srv2 := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++ }))
	defer srv2.Close()
	d.Rows[0].URL, d.Rows[1].URL = srv2.URL+"/a", srv2.URL+"/b"
	CheckURLs(context.Background(), d, 1, srv2.Client(), rand.New(rand.NewSource(1)))
	if calls != 1 {
		t.Errorf("--urls 1 made %d requests", calls)
	}
}

// toServer is a client that sends every request, whatever its host, to srv.
type toServer struct {
	srv *httptest.Server
}

func (t toServer) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.URL.Scheme = "https"
	r.URL.Host = strings.TrimPrefix(t.srv.URL, "https://")
	return t.srv.Client().Transport.RoundTrip(r)
}

// CCEL answers 200 to any section id, so a CCEL row's title must be in the
// page it names.
func TestCheckURLsCCELTitleMustBeInThePage(t *testing.T) {
	var methods []string
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method+" "+r.URL.Path)
		switch {
		case strings.HasSuffix(r.URL.Path, ".l.html"):
			// the page of 104: the title in the navbar and, with a curly apostrophe, in the heading
			fmt.Fprint(w, `<html><td class="book_navbar_title">Sermon 104. Love&#39;s <i>Commendation</i></td><h1>Love’s Commendation </h1>`)
		default:
			fmt.Fprint(w, `<html><td class="book_navbar_title">Sermon 1. Something else</td></html>`) // 200 for anything
		}
	}))
	defer srv.Close()
	rows := `{"id": "104", "title": "Love's Commendation", "ref": "Rom.5.8", "url": "https://ccel.org/ccel/spurgeon/sermons02/sermons02.l.html", "alt_url": "https://www.spurgeongems.org/sermon/chs104.pdf"}
{"id": "105", "title": "A Made-Up Title", "ref": "Rom.5.8", "url": "https://ccel.org/ccel/spurgeon/sermons02/sermons02.zzz.html"}
{"id": "106", "title": "Elsewhere", "ref": "Rom.5.8", "url": "https://example.org/106"}
`
	dir := writeSource(t, map[string]string{"source.toml": goodToml, "rows.jsonl": rows})
	d, fs := Read(dir)
	if len(fs) != 0 {
		t.Fatal(faultStrings(fs))
	}
	client := &http.Client{Transport: toServer{srv}}
	got := faultStrings(CheckURLs(context.Background(), d, 10, client, rand.New(rand.NewSource(1))))
	want := "rows.jsonl:2: url 'https://ccel.org/ccel/spurgeon/sermons02/sermons02.zzz.html': 200 OK but the title 'A Made-Up Title' is not in the page"
	if len(got) != 1 || got[0] != want {
		t.Errorf("CheckURLs = %q, want [%q]", got, want)
	}
	// A CCEL page is fetched with GET (a HEAD has no title to read); the
	// other hosts are still only asked with HEAD.
	for _, m := range methods {
		ccel := strings.Contains(m, " /ccel/")
		if ccel != strings.HasPrefix(m, "GET ") {
			t.Errorf("request %q: only a CCEL page is fetched with GET", m)
		}
	}
	if len(methods) != 4 {
		t.Errorf("requests = %q, want one for each of the four urls", methods)
	}
}

func TestTitleInPage(t *testing.T) {
	for _, c := range []struct {
		title, page string
		want        bool
	}{
		{"Love's Commendation", `<h1>Love’s Commendation </h1>`, true},
		{"Love's Commendation", `<td>Sermon 104. Love&#39;s  <i>Commendation</i></td>`, true},
		{`"This Year Also"`, `<td>Sermon 1451A. &quot;This Year Also&quot;</td>`, true},
		{`"This Year Also"`, `<h1>&#8220;This Year Also&#8221;</h1>`, true},
		{"The Immutability of God", `<td>Sermon 1. Something else</td>`, false},
		{"A & B", `<h1>A &amp; B</h1>`, true},
		{"Trust In God—True Wisdom", `<h1>Trust In God&#8212;True Wisdom</h1>`, true},
	} {
		if got := titleInPage(c.title, c.page); got != c.want {
			t.Errorf("titleInPage(%q, %q) = %v, want %v", c.title, c.page, got, c.want)
		}
	}
}

// A row over more than 3 chapters is warned, never faulted: the source still
// loads. A `series` row is the way to keep it.
func TestAWideRefIsWarnedNotFaulted(t *testing.T) {
	rows := `{"id": "1", "title": "Three", "ref": "Rom.5-Rom.7", "url": "https://example.org/1"}
{"id": "2", "title": "Four", "ref": "Rom.5-Rom.8", "url": "https://example.org/2"}
{"id": "3", "title": "A series", "ref": "Acts.1.1-Jude.1.25", "url": "https://example.org/3", "series": true}
{"id": "4", "title": "A book", "ref": "Acts.1.1-Jude.1.25", "url": "https://example.org/4"}
`
	dir := writeSource(t, map[string]string{"source.toml": goodToml, "rows.jsonl": rows})
	faults, warnings := CheckReport(dir)
	if len(faults) != 0 {
		t.Fatalf("faults %q", faultStrings(faults))
	}
	got := faultStrings(warnings)
	want := []string{
		"rows.jsonl:2: warning: ref 'Rom.5-Rom.8' spans 4 chapters: a series over it would match every chapter: give the row a `series: true` and the narrowest text, or split it",
		"rows.jsonl:4: warning: ref 'Acts.1.1-Jude.1.25' spans 149 chapters: a series over it would match every chapter: give the row a `series: true` and the narrowest text, or split it",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("warnings\n got %q\nwant %q", got, want)
	}
	// Check itself still says no fault: a warning does not stop a load.
	if fs := Check(dir); len(fs) != 0 {
		t.Errorf("Check: %q", faultStrings(fs))
	}
	// A directory with a fault warns nothing about its rows.
	_, w := CheckReport(writeSource(t, map[string]string{"source.toml": goodToml, "rows.jsonl": rows + `{"id": "5", "title": "Bad", "ref": "Rom.99", "url": "https://example.org/5"}` + "\n"}))
	if len(w) != 2 {
		t.Errorf("warnings with a fault elsewhere: %q", faultStrings(w))
	}
}

func TestCheckAllReportsWarnings(t *testing.T) {
	root := t.TempDir()
	for _, id := range []string{"a", "b"} {
		dir := filepath.Join(root, id)
		os.Mkdir(dir, 0o755)
		os.WriteFile(filepath.Join(dir, "source.toml"), []byte(strings.Replace(goodToml, `id = "demo"`, `id = "`+id+`"`, 1)), 0o644)
		os.WriteFile(filepath.Join(dir, "rows.jsonl"), []byte(`{"id": "1", "title": "Wide", "ref": "Rom", "url": "https://example.org/1"}`+"\n"), 0o644)
	}
	dirs, faults, warnings := CheckAllReport(root)
	if len(dirs) != 2 || len(faults) != 0 || len(warnings) != 2 || !strings.HasPrefix(warnings[1].String(), "b/rows.jsonl:1: warning: ref 'Rom' spans 16 chapters") {
		t.Errorf("%v %q %q", dirs, faultStrings(faults), faultStrings(warnings))
	}
}

func TestSeriesAndExpectTitleAreCheckedFields(t *testing.T) {
	src := func(rows string) map[string]string {
		return map[string]string{"source.toml": goodToml, "rows.jsonl": rows}
	}
	if fs := Check(writeSource(t, src(`{"id": "1", "title": "One", "ref": "Rom.5", "url": "https://example.org/1", "series": true, "expect_title": "One (Part 1)"}`))); len(fs) != 0 {
		t.Errorf("%q", faultStrings(fs))
	}
	expectFault(t, src(`{"id": "1", "title": "One", "ref": "Rom.5", "url": "https://example.org/1", "series": false}`),
		"rows.jsonl:1: series is false: leave it out instead")
	expectFault(t, src(`{"id": "1", "title": "One", "ref": "Rom.5", "url": "https://example.org/1", "expect_title": " "}`),
		"rows.jsonl:1: expect_title is empty: leave it out instead")
}

func TestFaultsInHostRules(t *testing.T) {
	rules := func(toml string) map[string]string {
		return map[string]string{"source.toml": goodToml + toml, "rows.jsonl": goodRow}
	}
	good := "\n[host_rules.\"www.example.org\"]\ntitle_in_page = true\ndelay_ms = 500\n"
	if fs := Check(writeSource(t, rules(good))); len(fs) != 0 {
		t.Errorf("%q", faultStrings(fs))
	}
	expectFault(t, rules("\n[host_rules.\"https://example.org\"]\ntitle_in_page = true\n"),
		"source.toml: host_rules 'https://example.org': a host name only, lower case, with no scheme, path or port")
	expectFault(t, rules("\n[host_rules.\"Example.org\"]\ntitle_in_page = true\n"),
		"source.toml: host_rules 'Example.org': a host name only, lower case, with no scheme, path or port")
	expectFault(t, rules("\n[host_rules.\"example.org\"]\ndelay_ms = -1\n"),
		"source.toml: host_rules 'example.org': delay_ms -1: not 0 or more")
	expectFault(t, rules("\n[host_rules.\"example.org\"]\ntitle_in_pag = true\n"),
		`source.toml: unknown key 'host_rules."example.org".title_in_pag'`)
	expectFault(t, rules("\n[host_rules.\"example.org\"]\n"),
		"source.toml: host_rules 'example.org': no rule given (title_in_page or delay_ms)")
}

// titlePage answers a page with a title only for /ok, and 200 with another
// for everything else; it counts the requests by method.
func titleServer(t *testing.T, counts map[string]int) *httptest.Server {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		counts[r.Method]++
		if r.URL.Path == "/ok" {
			fmt.Fprint(w, `<h1>Sermon One: Part 1</h1>`)
			return
		}
		fmt.Fprint(w, `<h1>Not found, but 200</h1>`)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestHostRuleTitleInPage(t *testing.T) {
	counts := map[string]int{}
	srv := titleServer(t, counts)
	rows := `{"id": "1", "title": "Sermon One", "ref": "Rom.5.8", "url": "https://sermons.example.org/ok"}
{"id": "2", "title": "Sermon Two", "ref": "Rom.5.8", "url": "https://sermons.example.org/ok"}
{"id": "3", "title": "Elsewhere", "ref": "Rom.5.8", "url": "https://other.example.org/anything"}
`
	toml := goodToml + "\n[host_rules.\"sermons.example.org\"]\ntitle_in_page = true\ndelay_ms = 1\n"
	d, fs := Read(writeSource(t, map[string]string{"source.toml": toml, "rows.jsonl": rows}))
	if len(fs) != 0 {
		t.Fatal(faultStrings(fs))
	}
	client := &http.Client{Transport: toServer{srv}}
	got := faultStrings(CheckURLs(context.Background(), d, 10, client, rand.New(rand.NewSource(1))))
	want := "rows.jsonl:2: url 'https://sermons.example.org/ok': 200 OK but the title 'Sermon Two' is not in the page"
	if len(got) != 1 || got[0] != want {
		t.Errorf("CheckURLs = %q, want [%q]", got, want)
	}
	if counts["GET"] != 2 || counts["HEAD"] != 1 {
		t.Errorf("requests %v: want 2 GET (the ruled host) and 1 HEAD", counts)
	}
}

// A row's expect_title is what the page must hold instead of its title, on
// a host with no rule; and a source's rule can turn the CCEL default off.
func TestExpectTitleAndOverridingTheCCELDefault(t *testing.T) {
	counts := map[string]int{}
	srv := titleServer(t, counts)
	rows := `{"id": "1", "title": "Catalogue Name", "ref": "Rom.5.8", "url": "https://other.example.org/ok", "expect_title": "Sermon One"}
{"id": "2", "title": "Catalogue Name", "ref": "Rom.5.8", "url": "https://other.example.org/ok", "expect_title": "Sermon Nine"}
{"id": "3", "title": "Whatever", "ref": "Rom.5.8", "url": "https://ccel.org/ccel/x/y.html"}
`
	toml := goodToml + "\n[host_rules.\"ccel.org\"]\ntitle_in_page = false\n"
	d, fs := Read(writeSource(t, map[string]string{"source.toml": toml, "rows.jsonl": rows}))
	if len(fs) != 0 {
		t.Fatal(faultStrings(fs))
	}
	client := &http.Client{Transport: toServer{srv}}
	got := faultStrings(CheckURLs(context.Background(), d, 10, client, rand.New(rand.NewSource(1))))
	want := "rows.jsonl:2: url 'https://other.example.org/ok': 200 OK but the title 'Sermon Nine' is not in the page"
	if len(got) != 1 || got[0] != want {
		t.Errorf("CheckURLs = %q, want [%q]", got, want)
	}
	if counts["GET"] != 2 || counts["HEAD"] != 1 {
		t.Errorf("requests %v: want 2 GET (the expect_title rows) and 1 HEAD (ccel.org, rule off)", counts)
	}
}

// sourcesBeside makes a temp sources/ holding the directories given (each
// a map of file to text) and returns its path.
func sourcesBeside(t *testing.T, dirs map[string]map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for id, files := range dirs {
		dir := filepath.Join(root, id)
		if err := os.Mkdir(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		for name, body := range files {
			if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	return root
}

// demoWithTopics is sources/ with demo (two rows, the second a series, and
// the sermon-topics.jsonl given), the theme source themes-x (topics love
// and grace) and the sermon source other.
func demoWithTopics(t *testing.T, sermonTopics string) string {
	t.Helper()
	root := sourcesBeside(t, map[string]map[string]string{
		"demo": {
			"source.toml": goodToml,
			"rows.jsonl": goodRow + "\n" +
				`{"id": "s", "title": "Romans", "ref": "Rom.1.1-Rom.16.24", "url": "https://example.org/s", "series": true}` + "\n",
			"sermon-topics.jsonl": sermonTopics,
		},
		"themes-x": {
			"source.toml":  strings.Replace(goodThemeToml, `id = "demo"`, `id = "themes-x"`, 1),
			"topics.jsonl": `{"id": "love", "label": "Love"}` + "\n" + `{"id": "grace", "label": "Grace"}` + "\n",
			"links.jsonl":  `{"topic": "love", "ref": "Rom.5.8"}` + "\n" + `{"topic": "grace", "ref": "Eph.2.8"}` + "\n",
		},
		"other": {
			"source.toml": strings.Replace(goodToml, `id = "demo"`, `id = "other"`, 1),
			"rows.jsonl":  goodRow + "\n",
		},
	})
	return filepath.Join(root, "demo")
}

func expectTopicsFault(t *testing.T, sermonTopics, want string) {
	t.Helper()
	got := faultStrings(Check(demoWithTopics(t, sermonTopics)))
	if len(got) != 1 || got[0] != want {
		t.Errorf("faults %q\n  want only %q", got, want)
	}
}

func TestSermonTopicsGoodChecksClean(t *testing.T) {
	// Two topics of one theme source, and a link to a series row.
	dir := demoWithTopics(t, `{"row": "1", "theme": "themes-x", "topic": "love"}`+"\n"+
		`{"row": "1", "theme": "themes-x", "topic": "grace"}`+"\n"+
		`{"row": "s", "theme": "themes-x", "topic": "grace"}`+"\n")
	if fs := Check(dir); len(fs) != 0 {
		t.Errorf("faults: %q", faultStrings(fs))
	}
	d, fs := Read(dir)
	if len(fs) != 0 || !d.HasSermonTopics || len(d.SermonTopics) != 3 || d.SermonTopics[2] != (SermonTopic{Row: "s", Theme: "themes-x", Topic: "grace", Line: 3}) {
		t.Errorf("read: %+v %q", d.SermonTopics, faultStrings(fs))
	}
}

func TestSermonTopicsRowNotInRows(t *testing.T) {
	expectTopicsFault(t, `{"row": "1", "theme": "themes-x", "topic": "love"}`+"\n"+`{"row": "2", "theme": "themes-x", "topic": "love"}`+"\n",
		"sermon-topics.jsonl:2: row '2': not in rows.jsonl")
}

func TestSermonTopicsThemeNotBeside(t *testing.T) {
	expectTopicsFault(t, `{"row": "1", "theme": "themes-y", "topic": "love"}`+"\n",
		"sermon-topics.jsonl:1: theme 'themes-y': no theme source beside this directory (no ../themes-y/source.toml)")
}

func TestSermonTopicsThemeIsASermonSource(t *testing.T) {
	expectTopicsFault(t, `{"row": "1", "theme": "other", "topic": "love"}`+"\n",
		"sermon-topics.jsonl:1: theme 'other': ../other is a sermons source, not a themes source")
}

func TestSermonTopicsThemeNotASourceID(t *testing.T) {
	expectTopicsFault(t, `{"row": "1", "theme": "../themes-x", "topic": "love"}`+"\n",
		"sermon-topics.jsonl:1: theme '../themes-x': not a source id (lower-case letters, digits and '-')")
}

func TestSermonTopicsTopicNotInTopics(t *testing.T) {
	expectTopicsFault(t, `{"row": "1", "theme": "themes-x", "topic": "mercy"}`+"\n",
		"sermon-topics.jsonl:1: topic 'mercy': not in themes-x/topics.jsonl")
}

func TestSermonTopicsRepeatedLine(t *testing.T) {
	expectTopicsFault(t, `{"row": "1", "theme": "themes-x", "topic": "love"}`+"\n"+
		`{"row": "s", "theme": "themes-x", "topic": "love"}`+"\n"+
		`{"row": "1", "theme": "themes-x", "topic": "love"}`+"\n",
		"sermon-topics.jsonl:3: repeats line 1")
}

func TestSermonTopicsMissingFields(t *testing.T) {
	for field, line := range map[string]string{
		"row":   `{"theme": "themes-x", "topic": "love"}`,
		"theme": `{"row": "1", "topic": "love"}`,
		"topic": `{"row": "1", "theme": "themes-x"}`,
	} {
		expectTopicsFault(t, line+"\n", "sermon-topics.jsonl:1: "+field+" is missing")
	}
}

func TestSermonTopicsUnknownKey(t *testing.T) {
	expectTopicsFault(t, `{"row": "1", "theme": "themes-x", "topic": "love", "part": 1}`+"\n",
		`sermon-topics.jsonl:1: unknown field "part"`)
}

func TestSermonTopicsEmptyFile(t *testing.T) {
	expectTopicsFault(t, "\n", "sermon-topics.jsonl: no rows")
}

func TestSermonTopicsInAThemeSource(t *testing.T) {
	expectFault(t, map[string]string{
		"source.toml":         goodThemeToml,
		"topics.jsonl":        `{"id": "love", "label": "Love"}` + "\n",
		"links.jsonl":         `{"topic": "love", "ref": "Rom.5.8"}` + "\n",
		"sermon-topics.jsonl": `{"row": "1", "theme": "demo", "topic": "love"}` + "\n",
	}, "sermon-topics.jsonl: a themes source has no sermon-topics.jsonl")
}

// Faults come in file order: rows.jsonl's, then sermon-topics.jsonl's.
func TestSermonTopicsFaultsFollowRows(t *testing.T) {
	root := sourcesBeside(t, map[string]map[string]string{"demo": {
		"source.toml":         goodToml,
		"rows.jsonl":          `{"id": "1", "title": "One", "ref": "Rom.5.99", "url": "https://example.org/1"}` + "\n",
		"sermon-topics.jsonl": `{"row": "9", "theme": "themes-x", "topic": "love"}` + "\n",
	}})
	got := faultStrings(Check(filepath.Join(root, "demo")))
	want := []string{
		"rows.jsonl:1: ref 'Rom.5.99': verse past the chapter",
		"sermon-topics.jsonl:1: row '9': not in rows.jsonl",
		"sermon-topics.jsonl:1: theme 'themes-x': no theme source beside this directory (no ../themes-x/source.toml)",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("faults\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// The fixture's file names themes-demo beside it, and checks clean; a
// source without the file checks exactly as before (TestFixtureAndTemplatePass).
func TestTheFixturesSermonTopicsCheckClean(t *testing.T) {
	d, fs := checked(fixture)
	if len(fs) != 0 || !d.HasSermonTopics || len(d.SermonTopics) != 1 {
		t.Errorf("%q %+v", faultStrings(fs), d.SermonTopics)
	}
	d, fs = checked("../../sources/_template")
	if len(fs) != 0 || d.HasSermonTopics || d.SermonTopics != nil {
		t.Errorf("the template: %q %+v", faultStrings(fs), d)
	}
}
