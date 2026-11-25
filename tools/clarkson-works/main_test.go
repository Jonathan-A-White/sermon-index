package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

// The fixtures are made up here, so that no sentence of a sermon is kept in
// the repository: a leaf is a list of lines, and body is the marker words every
// body line is written with. A test fails if a marker reaches any output.
const body = "zebrabody quillwort sentence of the made up page"

type fline struct {
	x0, top, height int
	text            string
}

// head is a running head line, large is a sermon's heading, plain is a line of
// the page's text.
func head(text string) fline { return fline{20, 150, 36, text} }
func large(top int, text string) fline {
	return fline{400, top, 64, text}
}
func plain(top int, text string) fline { return fline{30, top, 37, text} }

// opening is the leaf a sermon begins on: its heading, its text under it.
func opening(title, quote string) []fline {
	return []fline{large(900, title), plain(1200, body+". —"), plain(1260, quote), plain(1320, body), plain(1380, body)}
}

// page is a leaf inside a sermon: its running head and body.
func page(h string) []fline {
	return []fline{head(h), plain(300, body), plain(360, body), plain(420, body)}
}

// djvu writes the leaves as the Archive's _djvu.xml does: an OBJECT a leaf, a
// LINE of WORD elements with coords "x0,bottom,x1,top".
func djvu(leaves [][]fline) string {
	var b strings.Builder
	b.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<DjVuXML><BODY>\n")
	for _, lv := range leaves {
		b.WriteString("<OBJECT width=\"1800\" height=\"3200\"><HIDDENTEXT><PAGECOLUMN><REGION><PARAGRAPH>\n")
		for _, l := range lv {
			b.WriteString("<LINE>\n")
			x := l.x0
			for _, w := range strings.Fields(l.text) {
				x1 := x + 20*len(w)
				fmt.Fprintf(&b, "<WORD coords=\"%d,%d,%d,%d\">%s</WORD>\n", x, l.top+l.height, x1, l.top, strings.NewReplacer("&", "&amp;", "<", "&lt;").Replace(w))
				x = x1 + 20
			}
			b.WriteString("</LINE>\n")
		}
		b.WriteString("</PARAGRAPH></REGION></PAGECOLUMN></HIDDENTEXT></OBJECT>\n")
	}
	b.WriteString("</BODY></DjVuXML>\n")
	return b.String()
}

// fixtureVolume is a made-up volume. Page p is on leaf p+4. It has:
//   - page 1: "Of Faith", opened by a heading and a text, with running heads
//     one of which misreads the book (as OCR does);
//   - page 6: "Of Hope", whose contents page says 8: the heading is on 6;
//   - page 11: "The Sacrifice", with no heading of its own, only running
//     heads;
//   - page 15: "Of Silence", whose text no part of the scan reads;
//   - page 19: "Of Charity", whose heading and running heads read another
//     text than the contents page prints;
//   - page 22: "Of Patience", whose heading prints the contents page's text
//     and whose running heads print another (a printer's slip).
func fixtureVolume() [][]fline {
	leaves := make([][]fline, 30)
	set := func(p int, l []fline) { leaves[p+4] = l }
	set(1, opening("OF FAITH.", "Mark XVI. 16."))
	set(2, page("2 OF FAITH. [Mark XVI. 16."))
	set(3, page("Mauk XVI. 16.] OF FAITH. 3"))
	set(4, page("4 OF FAITH. [Mark XVI. 16."))
	set(5, append(page("Mark XVI. 16.] OF FAITH. 5"), plain(500, "the promise, 2 Pet. i. 4 ; and the like")))
	set(6, opening("OF HOPE.", "Heb. XI. 1."))
	set(7, page("Heb. XI. 1.] OF HOPE. 7"))
	set(8, page("8 OF HOPE. [Heb. XI. 1."))
	set(9, page("Heb. XI. 1.] OF HOPE. 9"))
	set(10, page("10 OF HOPE. [Heb. XI. 1."))
	set(11, page("Eph. V. 2, 3.] THE SACKIFICE. 11"))
	set(12, page("12 THE SACRIFICE. [Eph. V. 2, 3."))
	set(13, page("Eph. V. 2, 3.] THE SACRIFICE. 13"))
	set(14, page("14 THE SACRIFICE. [Eph. V. 2, 3."))
	set(15, []fline{large(900, "OF SILENCE."), plain(1200, body)})
	set(16, page("16 OF SILENCE. ["))
	set(17, page("OF SILENCE. 17"))
	set(18, page("18 OF SILENCE. ["))
	set(19, opening("OF CHARITY.", "John III. 16."))
	set(20, page("20 OF CHARITY. [John III. 16."))
	set(21, page("John III. 16.] OF CHARITY. 21"))
	set(22, opening("OF PATIENCE.", "James I. 4."))
	set(23, page("Luke XIII. 19.] OF PATIENCE. 23"))
	set(24, page("24 OF PATIENCE. [Luke XIII. 19."))
	set(25, page("Luke XIII. 19.] OF PATIENCE. 25"))
	return leaves
}

var fixtureCatalogue = []Entry{
	{Vol: 1, Page: 1, Title: "Of Faith", Text: "Mark.16.16"},
	{Vol: 1, Page: 8, Title: "Of Hope", Text: "Heb.11.1"},
	{Vol: 1, Page: 11, Title: "The Sacrifice", Text: "Eph.5.2-Eph.5.3"},
	{Vol: 1, Page: 15, Title: "Of Silence", Text: "Ps.4.4"},
	{Vol: 1, Page: 19, Title: "Of Charity", Text: "1Cor.13.1"},
	{Vol: 1, Page: 22, Title: "Of Patience", Text: "Jas.1.4"},
}

func fixtureLeaves(t *testing.T) map[int][]Leaf {
	t.Helper()
	leaves, err := ParseDjvu(strings.NewReader(djvu(fixtureVolume())))
	if err != nil {
		t.Fatal(err)
	}
	return map[int][]Leaf{1: leaves}
}

func TestReadRefs(t *testing.T) {
	for in, want := range map[string][]string{
		"Psalm LI. 5.":                                    {"Ps.51.5"},
		"6 OF ORIGINAL SIN. [Ps. LI. 5.":                  {"Ps.51.5"},
		"Mat. VII. 22, 23.] the conviction. 243":          {"Matt.7.22-Matt.7.23"},
		"332 MEN BY NATURE [JoHN V. 40. |":                {"John.5.40"},
		"[1 ChRON. XXIX, 11.":                             {"1Chr.29.11"},
		"1Chron.XXIX.11":                                  {"1Chr.29.11"},
		"LuKE XIII. 3.] op repentance. 21":                {"Luke.13.3"},
		"Mark XVl. 16.":                                   {"Mark.16.16"},
		"166 BELIEVERS COMMUNION WITH [1 John I. 3.":      {"1John.1.3"},
		"Ps. Lxxxvii. 2.] PREFERRED":                      {"Ps.87.2"},
		"Isa. XXVII. 9.]":                                 {"Isa.27.9"},
		"Mark XVI. IG.]":                                  nil, // the verse is misread
		"IVIakk XVI. 16.":                                 nil, // no such book
		"Ps. LL 5.":                                       nil, // not a roman numeral
		"Ps. LI. 99.":                                     nil, // no such verse
		"the first chapter of the book":                   nil,
		"[Philip. III. 8. and [Philip. III. 9. both read": {"Phil.3.8", "Phil.3.9"},
	} {
		if got := ReadRefs(in); !reflect.DeepEqual(got, want) {
			t.Errorf("ReadRefs(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestParseDjvu(t *testing.T) {
	leaves, err := ParseDjvu(strings.NewReader(djvu([][]fline{{}, {head("4 OF X. [Ps. I. 1."), large(900, "OF X.")}})))
	if err != nil {
		t.Fatal(err)
	}
	if len(leaves) != 2 || len(leaves[0].Lines) != 0 || len(leaves[1].Lines) != 2 {
		t.Fatalf("leaves: %+v", leaves)
	}
	l := leaves[1].Lines[1]
	if l.Text != "OF X." || l.Top != 900 || l.Height != 64 || l.X0 != 400 {
		t.Errorf("line: %+v", l)
	}
}

func TestBuildRows(t *testing.T) {
	rows, rep := BuildRows(fixtureLeaves(t), fixtureCatalogue)
	var ids []string
	for _, r := range rows {
		ids = append(ids, r.ID)
	}
	if want := []string{"v1-p1", "v1-p8", "v1-p11", "v1-p22"}; !reflect.DeepEqual(ids, want) {
		t.Fatalf("rows %v, want %v; left out %v", ids, want, rep.LeftOut)
	}
	faith, hope, sacrifice := rows[0], rows[1], rows[2]
	if faith.Title != "Of Faith" || faith.Ref == nil || *faith.Ref != "Mark.16.16" {
		t.Errorf("faith: %+v", faith)
	}
	if want := "https://archive.org/details/practicalworksof01clar/page/n5"; faith.URL != want {
		t.Errorf("faith url %s, want %s", faith.URL, want)
	}
	if faith.Preacher == nil || *faith.Preacher != "David Clarkson" {
		t.Errorf("faith preacher %v", faith.Preacher)
	}
	if faith.Notes == nil || *faith.Notes != "Volume 1, page 1." {
		t.Errorf("faith notes %v", faith.Notes)
	}
	// The contents page says 8; the heading is on page 6 (leaf 10).
	if !strings.HasSuffix(hope.URL, "/page/n10") || *hope.Notes != "Volume 1, page 6 (the contents page prints 8)." {
		t.Errorf("hope: %s %q", hope.URL, *hope.Notes)
	}
	if !strings.HasSuffix(sacrifice.URL, "/page/n15") || *sacrifice.Ref != "Eph.5.2-Eph.5.3" ||
		*sacrifice.Notes != "Volume 1, page 11. It begins with no heading of its own." {
		t.Errorf("sacrifice: %s %v %q", sacrifice.URL, *sacrifice.Ref, *sacrifice.Notes)
	}
	if rep.Read != 6 || rep.Rows != 4 || rep.WithText != 4 || len(rep.LeftOut) != 2 {
		t.Errorf("report: %+v", rep)
	}
	if len(rep.LeftOut) == 2 {
		if !strings.Contains(rep.LeftOut[0], "Of Silence") || !strings.Contains(rep.LeftOut[0], "under the heading: not read; running heads: none") {
			t.Errorf("silence left out as %q", rep.LeftOut[0])
		}
		if !strings.Contains(rep.LeftOut[1], "Of Charity") || !strings.Contains(rep.LeftOut[1], "John.3.16") || !strings.Contains(rep.LeftOut[1], "1Cor.13.1") {
			t.Errorf("charity left out as %q", rep.LeftOut[1])
		}
	}
}

// The heading and the contents page agree against the running heads: the row
// keeps their text and the report names what the running heads print.
func TestRunningHeadSlip(t *testing.T) {
	rows, rep := BuildRows(fixtureLeaves(t), fixtureCatalogue)
	last := rows[len(rows)-1]
	if last.ID != "v1-p22" || *last.Ref != "Jas.1.4" {
		t.Fatalf("patience: %+v", last)
	}
	if len(rep.Slips) != 1 || !strings.Contains(rep.Slips[0], "Of Patience") || !strings.Contains(rep.Slips[0], "Luke.13.19 ×3") {
		t.Errorf("slips: %q", rep.Slips)
	}
	if !strings.Contains(rep.Markdown(), "Luke.13.19") {
		t.Error("REPORT.md does not name the running heads' text")
	}
}

func TestReportSecondPeter(t *testing.T) {
	_, rep := BuildRows(fixtureLeaves(t), fixtureCatalogue)
	md := rep.Markdown()
	for _, want := range []string{
		"sermons read",
		"rows with a resolving text",
		"None: no sermon's text is 2 Peter 1:4",
		"Volume 1, page 5, in \"Of Faith\"",
		"Of Silence",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("REPORT.md lacks %q:\n%s", want, md)
		}
	}
	_, rep = BuildRows(fixtureLeaves(t), []Entry{{Vol: 1, Page: 1, Title: "Of Faith", Text: "Mark.16.16"}})
	if strings.Contains(rep.Markdown(), "None: no sermon's text is 2 Peter 1:4") == false {
		t.Error("a volume with no sermon on 2 Peter 1:4 must say none")
	}
}

// Facts only: nothing of a page's text but a title (from the catalogue), a
// reference and a page number reaches the rows or either report.
func TestNoSermonSentenceIsWritten(t *testing.T) {
	rows, rep := BuildRows(fixtureLeaves(t), fixtureCatalogue)
	j, err := json.Marshal(rows)
	if err != nil {
		t.Fatal(err)
	}
	all := string(j) + rep.Markdown() + rep.Evidence() + rep.Summary()
	for _, w := range []string{"zebrabody", "quillwort", "made up page", "the promise"} {
		if strings.Contains(all, w) {
			t.Errorf("%q, a word of the page's text, reaches the output", w)
		}
	}
}

// The catalogue is the contents pages' 34 sermons, in page order, each with a
// title and a text of the book table.
func TestCatalogue(t *testing.T) {
	perVol := map[int]int{}
	last := map[int]int{}
	var vols []int
	for _, e := range catalogue {
		perVol[e.Vol]++
		if e.Page <= last[e.Vol] {
			t.Errorf("%+v: not in page order", e)
		}
		last[e.Vol] = e.Page
		if e.Title == "" || strings.HasSuffix(e.Title, ".") {
			t.Errorf("%+v: a title with no final point", e)
		}
		if got := e.ref(); got == "" {
			t.Errorf("%+v: the text does not resolve", e)
		}
	}
	for v := range perVol {
		vols = append(vols, v)
	}
	sort.Ints(vols)
	if len(catalogue) != 34 || !reflect.DeepEqual(vols, []int{1, 2, 3}) || perVol[1] != 13 || perVol[2] != 13 || perVol[3] != 8 {
		t.Errorf("%d entries, %v", len(catalogue), perVol)
	}
}

func TestFetchVolumes(t *testing.T) {
	var mu sync.Mutex
	var got, agents []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		got = append(got, r.URL.Path)
		agents = append(agents, r.UserAgent())
		mu.Unlock()
		w.Write([]byte(djvu(nil)))
	}))
	defer srv.Close()
	urlOf := func(item string) string { return srv.URL + "/" + item }
	cache := t.TempDir()
	// A volume already in the cache is not fetched again.
	if err := os.WriteFile(volumePath(cache, volumes[1].Item), []byte(djvu(nil)), 0o644); err != nil {
		t.Fatal(err)
	}
	var log bytes.Buffer
	start := time.Now()
	if err := fetchVolumes(context.Background(), srv.Client(), urlOf, cache, 20*time.Millisecond, &log); err != nil {
		t.Fatal(err)
	}
	if want := []string{"/practicalworksof01clar", "/practicalworksof03clar"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("fetched %v, want %v", got, want)
	}
	if d := time.Since(start); d < 20*time.Millisecond {
		t.Errorf("two requests in %v: not spaced", d)
	}
	for _, a := range agents {
		if a == "" || strings.HasPrefix(a, "Go-http-client") {
			t.Errorf("User-Agent %q does not name a client", a)
		}
	}
	got = nil
	if err := fetchVolumes(context.Background(), srv.Client(), urlOf, cache, 0, &log); err != nil || len(got) != 0 {
		t.Fatalf("second fetch: %v, %d requests", err, len(got))
	}
}

func TestFetchRefusesNotDjvu(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("<html><body>Not here</body></html>"))
	}))
	defer srv.Close()
	cache := t.TempDir()
	err := fetchVolumes(context.Background(), srv.Client(), func(string) string { return srv.URL }, cache, 0, &bytes.Buffer{})
	if err == nil {
		t.Fatal("an answer that is not DjVu XML was taken")
	}
	if es, _ := os.ReadDir(cache); len(es) != 0 {
		t.Errorf("the cache keeps %d files of a refused answer", len(es))
	}
}

func TestRunWritesTheSameBytesTwice(t *testing.T) {
	cache := t.TempDir()
	for _, v := range volumes {
		if err := os.WriteFile(volumePath(cache, v.Item), []byte(djvu(fixtureVolume())), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	dir := t.TempDir()
	o := options{cache: cache, out: filepath.Join(dir, "src"), report: filepath.Join(dir, "src", "REPORT.md"), evidence: filepath.Join(dir, "EVIDENCE.md")}
	read := func() []byte {
		var all []byte
		for _, f := range []string{filepath.Join(o.out, "rows.jsonl"), o.report, o.evidence} {
			b, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			all = append(all, b...)
		}
		return all
	}
	var first []byte
	for i := 0; i < 2; i++ {
		var stdout bytes.Buffer
		if err := run(context.Background(), o, fixtureCatalogue, &stdout); err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			first = read()
			if !strings.Contains(stdout.String(), "sermons read") {
				t.Errorf("counts not printed: %s", stdout.String())
			}
		} else if !bytes.Equal(first, read()) {
			t.Error("the second run wrote other bytes")
		}
	}
}
