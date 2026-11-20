package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Jonathan-A-White/sermon-index/internal/sources"
)

// The pages below are shaped like the ones Subsplash serves a signed-in
// reader, with invented titles and words: no test holds a sermon's words.

const base = "https://subsplash.com/u/brookfieldfirst/media/d/"

func sermonURL(code, slug string) string { return base + code + "-" + slug }

func listPage(pages int, items ...[2]string) Page {
	p := Page{URL: "https://subsplash.com/brookfieldfirst/media", Status: 200}
	var text strings.Builder
	text.WriteString("Brookfield First\nMEDIA\nMedia\n")
	for _, it := range items {
		text.WriteString(it[0] + "\n")
		p.Links = append(p.Links,
			Link{Text: it[0], Href: sermonURL(it[1], slugOf(it[0]))},
			Link{Text: "", Href: sermonURL(it[1], slugOf(it[0])) + "?autoplay"})
	}
	text.WriteString("1\n2\n3\n")
	if pages > 3 {
		text.WriteString("10\n")
	}
	text.WriteString("© 2026 Subsplash, Inc.\nLogged in as someone@example.org\n")
	p.Text = text.String()
	p.Pages = pages
	return p
}

func slugOf(title string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(title) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
			dash = false
		} else if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.TrimSuffix(b.String(), "-")
}

func sermonPage(code, title, meta string, notes bool) Page {
	text := "Media\nBrookfield First\nMedia\nJW\n" + title + "\n" + meta + "\nShare\nDownload\n"
	if notes {
		text += "Notes\n"
	}
	text += "Related by Topic\nMedia streaming powered by Subsplash\nLogged in as someone@example.org\n"
	return Page{URL: sermonURL(code, slugOf(title)), Status: 200, Title: title + " - Brookfield First", Text: text, HasNotes: notes}
}

func notesPage(title, ref, date string) Page {
	return Page{URL: "https://notes.subsplash.com/fill-in/view?page=abc", Status: 200,
		Text: "\"" + title + "\" — " + ref + "\nPart of A Made-Up Series  " + date + "\n\nFirst point: ____\nREVEAL ANSWER\n\nCLEAR EMAIL\n\nCreated by Subsplash. © 2009–2026 Subsplash, Inc.\n"}
}

func episodePage(code string, n int, date, extra string) Page {
	title := "Morning RePhil #" + itoa(n)
	text := "Media\nBrookfield First\nMedia\nJW\n" + title + "\n" + date + "\nShare\nDownload\n" + extra + "More from Morning RePhil\nMorning RePhil #1\nMedia streaming powered by Subsplash\nLogged in as someone@example.org\n"
	return Page{URL: sermonURL(code, slugOf(title)), Status: 200, Text: text}
}

func itoa(n int) string { return strconv.Itoa(n) }

func TestParseSermon(t *testing.T) {
	s, err := ParseSermon(sermonPage("3nxf2bb", "The Invented Sermon", "Nov 8, 2026 • Pastor Example • Acts 1:8", true))
	if err != nil {
		t.Fatal(err)
	}
	want := Sermon{Code: "3nxf2bb", Title: "The Invented Sermon", Date: "2026-11-08", Speaker: "Pastor Example", Reference: "Acts 1:8", HasNotes: true}
	if s != want {
		t.Errorf("got %+v, want %+v", s, want)
	}
	s, err = ParseSermon(sermonPage("3nxf2bb", "No Speaker", "Nov 8, 2026 • John 3:16-18", false))
	if err != nil || s.Speaker != "" || s.Reference != "John 3:16-18" || s.HasNotes {
		t.Errorf("no speaker: %+v, %v", s, err)
	}
	s, err = ParseSermon(sermonPage("3nxf2bb", "Date Only", "Nov 8, 2026", false))
	if err != nil || s.Speaker != "" || s.Reference != "" {
		t.Errorf("date only: %+v, %v", s, err)
	}
	if _, err := ParseSermon(Page{URL: sermonURL("3nxf2bb", "x"), Text: "Share\nDownload\n"}); err == nil {
		t.Error("a page with no date line was taken for a sermon")
	}
}

func TestParseNotes(t *testing.T) {
	n, err := ParseNotes(notesPage("The Invented Sermon", "Acts 1:8", "November 8, 2026"))
	if err != nil {
		t.Fatal(err)
	}
	if n.Title != "The Invented Sermon" || n.Reference != "Acts 1:8" || n.Date != "2026-11-08" {
		t.Errorf("got %+v", n)
	}
	if !strings.Contains(n.Text, "First point") {
		t.Errorf("notes text lost: %q", n.Text)
	}
	if _, err := ParseNotes(Page{URL: "https://notes.subsplash.com/fill-in/view?page=x", Text: "Something else\n"}); err == nil {
		t.Error("a page with no quoted title was taken for notes")
	}
}

// Some notes print the title with no quotes and a hyphen before the passage.
func TestParseNotesWithAnUnquotedTitle(t *testing.T) {
	page := func(first string) Page {
		return Page{URL: "https://notes.subsplash.com/fill-in/view?page=abc", Status: 200, Text: first + "\nPart of A Made-Up Series  August 18, 2024\n\nFirst point\n"}
	}
	n, err := ParseNotes(page("The Invented Gifts - 1 Corinthians 12:7-11"))
	if err != nil || n.Title != "The Invented Gifts" || n.Reference != "1 Corinthians 12:7-11" || n.Date != "2024-08-18" || n.URL == "" {
		t.Errorf("got %+v, %v", n, err)
	}
	// A hyphen in the title that no passage follows stays in the title.
	n, err = ParseNotes(page("Grace - and Truth"))
	if err != nil || n.Title != "Grace - and Truth" || n.Reference != "" {
		t.Errorf("got %+v, %v", n, err)
	}
}

func TestParseList(t *testing.T) {
	items, pages, err := ParseList(listPage(10,
		[2]string{"Morning RePhil #479", "fnnh79c"}, [2]string{"The Invented Sermon", "3nxf2bb"}, [2]string{"Morning RePhil #478", "b2kbz4b"}))
	if err != nil {
		t.Fatal(err)
	}
	if pages != 10 || len(items) != 3 {
		t.Fatalf("pages %d, items %v", pages, items)
	}
	if items[1].Code != "3nxf2bb" || items[1].Title != "The Invented Sermon" || items[1].Rephil != 0 {
		t.Errorf("sermon item: %+v", items[1])
	}
	if items[0].Rephil != 479 || items[2].Rephil != 478 {
		t.Errorf("episode numbers: %+v", items)
	}
	if _, _, err := ParseList(listPage(1)); err == nil {
		t.Error("a list with no item was taken for a list")
	}
}

func TestCheckStops(t *testing.T) {
	good := sermonPage("3nxf2bb", "The Invented Sermon", "Nov 8, 2026", false)
	if err := Check(good, KindSermon); err != nil {
		t.Fatalf("a sermon page stopped: %v", err)
	}
	cases := []struct {
		name string
		page Page
		kind Kind
		why  string
	}{
		{"429", func() Page { p := good; p.Status = 429; return p }(), KindSermon, "429"},
		{"403", func() Page { p := good; p.Status = 403; return p }(), KindSermon, "403"},
		{"login prompt", Page{URL: "https://subsplash.com/brookfieldfirst/media", Status: 200, Text: "Sign in\nEmail\nPassword\n"}, KindList, "login"},
		{"login url", Page{URL: "https://subsplash.com/login?next=x", Status: 200, Text: "Welcome\nLogged in as x\n"}, KindList, "login"},
		{"signed out", Page{URL: good.URL, Status: 200, Text: "Unfinished\nNov 8, 2026\nShare\n"}, KindSermon, "login"},
		{"captcha", Page{URL: good.URL, Status: 200, Text: "Please verify you are human\nLogged in as x\n"}, KindSermon, "captcha"},
		{"not found", Page{URL: good.URL, Status: 200, Text: "Page not found\n\nSorry, the page you are looking for doesn't exist\nLogged in as x\n"}, KindSermon, "unexpected"},
		{"wrong place", Page{URL: "https://example.org/x", Status: 200, Text: "Logged in as x\n"}, KindSermon, "unexpected"},
		{"notes elsewhere", Page{URL: "https://subsplash.com/x", Status: 200, Text: "x"}, KindNotes, "unexpected"},
	}
	for _, c := range cases {
		err := Check(c.page, c.kind)
		var st Stop
		if !errors.As(err, &st) || !strings.Contains(strings.ToLower(st.Why), c.why) {
			t.Errorf("%s: got %v, want a stop naming %q", c.name, err, c.why)
		}
	}
}

// clock is a fake time: sleeping moves it, so a test finishes at once and can
// read every gap between two loads.
type clock struct{ now time.Time }

func (c *clock) Now() time.Time        { return c.now }
func (c *clock) Sleep(d time.Duration) { c.now = c.now.Add(d) }

// fakeBrowser serves pages by address and records every load with its time.
type fakeBrowser struct {
	clk    *clock
	list   []Page // list pages 1..n
	pages  map[string]Page
	notes  map[string]Page // by the sermon's address
	cur    string
	loads  []string
	times  []time.Time
	failAt string // an address that answers failPage
	fail   Page
	series []Page // the series page, then after each View More
	more   int
	// moreBy holds, for any other series page by its address, the pages after
	// each View More there (the first is what Open gives).
	moreBy map[string][]Page
	moreAt map[string]int
}

func (f *fakeBrowser) note(what string) {
	f.loads = append(f.loads, what)
	f.times = append(f.times, f.clk.Now())
}

func (f *fakeBrowser) Open(_ context.Context, url string) (Page, error) {
	f.note(url)
	f.cur = url
	if url == f.failAt {
		return f.fail, nil
	}
	if url == listURL {
		return f.list[0], nil
	}
	if url == seriesURL {
		return f.series[0], nil
	}
	if q, ok := f.moreBy[url]; ok {
		f.moreAt[url] = 0
		return q[0], nil
	}
	p, ok := f.pages[url]
	if !ok {
		return Page{URL: url, Status: 200, Text: "Page not found\nLogged in as x\n"}, nil
	}
	return p, nil
}

func (f *fakeBrowser) OpenNotes(context.Context) (Page, error) {
	f.note("notes of " + f.cur)
	return f.notes[f.cur], nil
}

func (f *fakeBrowser) More(context.Context) (Page, error) {
	f.note("more")
	if q, ok := f.moreBy[f.cur]; ok {
		f.moreAt[f.cur]++
		return q[f.moreAt[f.cur]], nil
	}
	f.more++
	return f.series[f.more], nil
}

func (f *fakeBrowser) ListPage(_ context.Context, n int) (Page, error) {
	f.note("list page " + itoa(n))
	return f.list[n-1], nil
}

func newFake() *fakeBrowser {
	return &fakeBrowser{clk: &clock{now: time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)}, pages: map[string]Page{}, notes: map[string]Page{}, moreBy: map[string][]Page{}, moreAt: map[string]int{}}
}

func TestPacedNeverUnderTenSeconds(t *testing.T) {
	f := newFake()
	f.list = []Page{listPage(1, [2]string{"A Sermon", "aaaaaaa"})}
	// An interval under ten seconds is raised to ten.
	p := Paced(f, 2*time.Second, f.clk.Now, f.clk.Sleep)
	for i := 0; i < 4; i++ {
		if _, err := p.Open(context.Background(), listURL); err != nil {
			t.Fatal(err)
		}
	}
	for i := 1; i < len(f.times); i++ {
		if gap := f.times[i].Sub(f.times[i-1]); gap < 10*time.Second {
			t.Errorf("load %d came %v after the one before", i, gap)
		}
	}
}

func TestPacedRefusesOtherPages(t *testing.T) {
	f := newFake()
	p := Paced(f, 10*time.Second, f.clk.Now, f.clk.Sleep)
	for _, u := range []string{"https://subsplash.com/u/brookfieldfirst/account", "https://example.org/", "https://subsplash.com/brookfieldfirst/media/../../x", "http://subsplash.com/brookfieldfirst/media"} {
		if _, err := p.Open(context.Background(), u); err == nil {
			t.Errorf("%s was opened", u)
		}
	}
	if len(f.loads) != 0 {
		t.Errorf("loads made: %v", f.loads)
	}
}

// world is a church: two Sundays with notes, one without, and two episodes.
func world(t *testing.T) (*fakeBrowser, Config) {
	t.Helper()
	f := newFake()
	f.list = []Page{listPage(1,
		[2]string{"Morning RePhil #3", "rrrrrr3"},
		[2]string{"The Invented Sermon", "3nxf2bb"},
		[2]string{"Morning RePhil #2", "rrrrrr2"},
		[2]string{"A Sermon Without Notes", "nnnnnnn"},
		[2]string{"The Older Sermon", "ooooooo"},
	)}
	f.pages[sermonURL("3nxf2bb", "the-invented-sermon")] = sermonPage("3nxf2bb", "The Invented Sermon", "Nov 8, 2026 • Pastor Example • Acts 1:8", true)
	f.notes[sermonURL("3nxf2bb", "the-invented-sermon")] = notesPage("The Invented Sermon", "Acts 1:8", "November 8, 2026")
	f.pages[sermonURL("nnnnnnn", "a-sermon-without-notes")] = sermonPage("nnnnnnn", "A Sermon Without Notes", "Nov 1, 2026 • Pastor Example • John 3:16", false)
	f.pages[sermonURL("ooooooo", "the-older-sermon")] = sermonPage("ooooooo", "The Older Sermon", "Oct 25, 2026 • Pastor Example • Romans 8:1-4", true)
	f.notes[sermonURL("ooooooo", "the-older-sermon")] = notesPage("The Older Sermon", "Romans 8:1-4", "October 25, 2026")
	f.pages[sermonURL("rrrrrr3", "morning-rephil-3")] = episodePage("rrrrrr3", 3, "Nov 13, 2026", "A short thought for the day.\n")
	f.pages[sermonURL("rrrrrr2", "morning-rephil-2")] = episodePage("rrrrrr2", 2, "Nov 9, 2026", "")

	str := func(s string) *string { return &s }
	rows := []sources.Row{
		{ID: "ROW1", Title: "The Invented Sermon", Date: str("2026-11-08")},
		{ID: "ROW2", Title: "Brookfield First Service", Date: str("2026-11-01")},
		{ID: "ROW3", Title: "The Older Sermon", Date: str("2026-10-25")},
	}
	dir := t.TempDir()
	cfg := Config{
		Rows: rows, Out: filepath.Join(dir, "subsplash-notes.jsonl"), NotesDir: filepath.Join(dir, "notes"),
		LinksOut: filepath.Join(dir, "links.txt"), Rephil: filepath.Join(dir, "morning-rephil.jsonl"),
		Feed: filepath.Join(dir, "jobs", "subsplash-browse.json"), Interval: 10 * time.Second,
		Now: f.clk.Now, Sleep: f.clk.Sleep,
	}
	return f, cfg
}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	return string(b)
}

func TestBrowseWritesTheRecordsTheMailWould(t *testing.T) {
	f, cfg := world(t)
	var out bytes.Buffer
	res, err := Browse(context.Background(), f, cfg, &out)
	if err != nil {
		t.Fatalf("%v\n%s", err, out.String())
	}
	recs := read(t, cfg.Out)
	// The same line the mail-made job writes for that Sunday.
	want := `{"id":"ROW1","date":"2026-11-08","title":"The Invented Sermon","ref":"Acts.1.8","speaker":"Pastor Example"}`
	if !strings.Contains(recs, want+"\n") {
		t.Errorf("records:\n%s\nwant a line %s", recs, want)
	}
	// ROW3 is matched by date within a day: the older sermon, Oct 25, is a Sunday.
	if !strings.Contains(recs, `"id":"ROW3"`) {
		t.Errorf("the older sermon has no record:\n%s", recs)
	}
	// The notes text is kept beside the PDFs, mode 0600, and never in the records.
	txt := read(t, filepath.Join(cfg.NotesDir, "ROW1.txt"))
	if !strings.Contains(txt, "First point") {
		t.Errorf("notes text: %q", txt)
	}
	if fi, _ := os.Stat(filepath.Join(cfg.NotesDir, "ROW1.txt")); fi == nil || fi.Mode().Perm() != 0o600 {
		t.Errorf("notes text mode: %v", fi)
	}
	if strings.Contains(recs, "First point") {
		t.Error("the notes' words are in the records file")
	}
	// Every sermon's link, with notes or not, in the full form subsplash-links takes.
	links := read(t, cfg.LinksOut)
	for _, c := range []string{"3nxf2bb-the-invented-sermon", "nnnnnnn-a-sermon-without-notes", "ooooooo-the-older-sermon"} {
		if !strings.Contains(links, base+c+"\n") {
			t.Errorf("link %s missing from\n%s", c, links)
		}
	}
	if res.Recorded != 2 || res.Linked != 3 {
		t.Errorf("result %+v", res)
	}
	// Pace: ten seconds between any two loads.
	for i := 1; i < len(f.times); i++ {
		if gap := f.times[i].Sub(f.times[i-1]); gap < 10*time.Second {
			t.Errorf("load %d (%s) came %v after %s", i, f.loads[i], gap, f.loads[i-1])
		}
	}
}

func TestBrowseReadsOnlyWhatTheByHandJobReads(t *testing.T) {
	f, cfg := world(t)
	if _, err := Browse(context.Background(), f, cfg, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	for _, l := range f.loads {
		switch {
		case l == listURL, strings.HasPrefix(l, "list page "), strings.HasPrefix(l, "notes of "+base), strings.HasPrefix(l, base):
		default:
			t.Errorf("a load of %q is not the list, a sermon, its notes or an episode", l)
		}
	}
}

func TestBrowseSecondRunChangesNothing(t *testing.T) {
	f, cfg := world(t)
	if _, err := Browse(context.Background(), f, cfg, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	before := map[string]string{}
	for _, p := range []string{cfg.Out, cfg.LinksOut, cfg.Rephil, filepath.Join(cfg.NotesDir, "ROW1.txt")} {
		before[p] = read(t, p)
	}
	f.loads, f.times = nil, nil
	if _, err := Browse(context.Background(), f, cfg, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	for p, b := range before {
		if read(t, p) != b {
			t.Errorf("%s changed on the second run", p)
		}
	}
	// Only the list is read: nothing is visited twice.
	for _, l := range f.loads {
		if strings.HasPrefix(l, base) || strings.HasPrefix(l, "notes of") {
			t.Errorf("second run loaded %s", l)
		}
	}
}

func TestBrowseStopsAtTheFirstLoginPromptAndResumes(t *testing.T) {
	f, cfg := world(t)
	older := sermonURL("nnnnnnn", "a-sermon-without-notes")
	f.failAt = older
	f.fail = Page{URL: "https://subsplash.com/login", Status: 200, Text: "Sign in\nPassword\n"}
	var out bytes.Buffer
	_, err := Browse(context.Background(), f, cfg, &out)
	var st Stop
	if !errors.As(err, &st) {
		t.Fatalf("got %v, want a stop", err)
	}
	// What was read before the stop is written; nothing after it.
	recs := read(t, cfg.Out)
	if !strings.Contains(recs, `"id":"ROW1"`) || strings.Contains(recs, `"id":"ROW3"`) {
		t.Errorf("records after the stop:\n%s", recs)
	}
	if strings.Contains(read(t, cfg.LinksOut), "nnnnnnn") {
		t.Error("the sermon the stop came on has a link")
	}
	last := f.loads[len(f.loads)-1]
	if last != older {
		t.Errorf("a load was made after the stop: %v", f.loads)
	}
	kept := recs
	// He signs in again; the next run resumes at the sermon it stopped on.
	f.failAt = ""
	f.loads, f.times = nil, nil
	if _, err := Browse(context.Background(), f, cfg, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	for _, l := range f.loads {
		if l == sermonURL("3nxf2bb", "the-invented-sermon") {
			t.Error("the first run's sermon was visited again")
		}
	}
	if !strings.HasPrefix(read(t, cfg.Out), strings.Split(kept, "\n")[0]) && !strings.Contains(read(t, cfg.Out), strings.Split(kept, "\n")[0]) {
		t.Error("a record already written changed")
	}
	if !strings.Contains(read(t, cfg.Out), `"id":"ROW3"`) {
		t.Error("the resumed run did not write the rest")
	}
}

func TestBrowseStopsOn429(t *testing.T) {
	f, cfg := world(t)
	f.failAt = sermonURL("3nxf2bb", "the-invented-sermon")
	f.fail = Page{URL: f.failAt, Status: 429, Text: "Too many requests\n"}
	_, err := Browse(context.Background(), f, cfg, &bytes.Buffer{})
	var st Stop
	if !errors.As(err, &st) || !strings.Contains(st.Why, "429") {
		t.Fatalf("got %v", err)
	}
	if read(t, cfg.Out) != "" || read(t, cfg.LinksOut) != "" {
		t.Error("something was written after a 429 on the first sermon")
	}
}

func TestBrowseVerifyComparesAndWritesNothing(t *testing.T) {
	f, cfg := world(t)
	// The mail-made record and PDF for the Sunday are already there.
	if err := os.WriteFile(cfg.Out, []byte(`{"id":"ROW1","date":"2026-11-08","title":"The Invented Sermon","ref":"Acts.1.8"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg.Verify = 1
	cfg.PDFText = func(id string) (string, error) {
		return "\"The Invented\nSermon\" — Acts\n1:8\nPart of A Made-Up Series\nNovember 8, 2026\nFirst point: ____\n", nil
	}
	var out bytes.Buffer
	res, err := Browse(context.Background(), f, cfg, &out)
	if err != nil {
		t.Fatalf("%v\n%s", err, out.String())
	}
	if res.Verified != 1 || res.Differ != 0 {
		t.Errorf("result %+v\n%s", res, out.String())
	}
	if !strings.Contains(out.String(), "ROW1") || !strings.Contains(out.String(), "same") {
		t.Errorf("report: %s", out.String())
	}
	// Nothing written: the records are as they were, no links, no notes, no feed.
	if got := read(t, cfg.Out); !strings.HasPrefix(got, `{"id":"ROW1"`) || strings.Count(got, "\n") != 1 {
		t.Errorf("records changed: %s", got)
	}
	for _, p := range []string{cfg.LinksOut, cfg.Rephil, cfg.Feed} {
		if _, err := os.Stat(p); err == nil {
			t.Errorf("%s was written by a verify", p)
		}
	}
	if _, err := os.Stat(cfg.NotesDir); err == nil {
		t.Error("the notes directory was made by a verify")
	}
}

func TestBrowseVerifyNamesWhatDiffers(t *testing.T) {
	f, cfg := world(t)
	if err := os.WriteFile(cfg.Out, []byte(`{"id":"ROW1","date":"2026-11-08","title":"The Invented Sermon","ref":"Acts.1.9"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg.Verify = 1
	cfg.PDFText = func(string) (string, error) { return "other words entirely\n", nil }
	var out bytes.Buffer
	res, err := Browse(context.Background(), f, cfg, &out)
	if err != nil {
		t.Fatal(err)
	}
	if res.Differ != 1 || !strings.Contains(out.String(), "ref") || !strings.Contains(out.String(), "notes text") {
		t.Errorf("result %+v\n%s", res, out.String())
	}
}

func TestBrowseRephil(t *testing.T) {
	f, cfg := world(t)
	cfg.EpisodePages = true
	// Episode 1 is on file from YouTube, with its own date and no Subsplash link.
	if err := os.WriteFile(cfg.Rephil, []byte(`{"number":1,"title":"Morning RePhil #1","date":"2026-11-02","url":"https://www.youtube.com/watch?v=abc"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	f.list = []Page{listPage(1,
		[2]string{"Morning RePhil #3", "rrrrrr3"},
		[2]string{"Morning RePhil #2", "rrrrrr2"},
		[2]string{"Morning RePhil #1", "rrrrrr1"},
	)}
	if _, err := Browse(context.Background(), f, cfg, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(read(t, cfg.Rephil)), "\n")
	if len(lines) != 3 {
		t.Fatalf("rows:\n%s", strings.Join(lines, "\n"))
	}
	// Oldest first; the YouTube row keeps its fields and gains the share link.
	if !strings.HasPrefix(lines[0], `{"number":1,"title":"Morning RePhil #1","date":"2026-11-02","url":"https://www.youtube.com/watch?v=abc","share":"https://brookfieldfirst.subspla.sh/rrrrrr1"`) {
		t.Errorf("episode 1: %s", lines[0])
	}
	if !strings.HasPrefix(lines[1], `{"number":2,"title":"Morning RePhil #2","date":"2026-11-09","share":"https://brookfieldfirst.subspla.sh/rrrrrr2"`) {
		t.Errorf("episode 2: %s", lines[1])
	}
	if !strings.Contains(lines[2], `"number":3`) || !strings.Contains(lines[2], `"notes_file":"rephil-3.txt"`) {
		t.Errorf("episode 3: %s", lines[2])
	}
	if !strings.Contains(read(t, filepath.Join(cfg.NotesDir, "rephil-3.txt")), "A short thought") {
		t.Error("the episode's text is not kept")
	}
	// Episode 1's date is known, so its page is not loaded; 2 and 3 are loaded once.
	var visited []string
	for _, l := range f.loads {
		if strings.HasPrefix(l, base) {
			visited = append(visited, l)
		}
	}
	if len(visited) != 2 {
		t.Errorf("episode pages loaded: %v", visited)
	}
	// A re-run adds nothing and loads no episode.
	before := read(t, cfg.Rephil)
	f.loads = nil
	if _, err := Browse(context.Background(), f, cfg, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if read(t, cfg.Rephil) != before {
		t.Error("a re-run changed the episodes file")
	}
	for _, l := range f.loads {
		if strings.HasPrefix(l, base) {
			t.Errorf("a re-run loaded %s", l)
		}
	}
}

func TestBrowseRephilNumberByPosition(t *testing.T) {
	f, cfg := world(t)
	cfg.EpisodePages = true
	f.list = []Page{listPage(1,
		[2]string{"Morning RePhil #2", "rrrrrr2"},
		[2]string{"Morning RePhil Special", "sssssss"},
		[2]string{"Morning RePhil #1", "rrrrrr1"},
	)}
	f.pages[sermonURL("sssssss", "morning-rephil-special")] = Page{URL: sermonURL("sssssss", "morning-rephil-special"), Status: 200,
		Text: "Media\nJW\nMorning RePhil Special\nNov 5, 2026\nShare\nDownload\nMore from Morning RePhil\nLogged in as x\n"}
	f.pages[sermonURL("rrrrrr1", "morning-rephil-1")] = episodePage("rrrrrr1", 1, "Nov 2, 2026", "")
	if _, err := Browse(context.Background(), f, cfg, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	got := read(t, cfg.Rephil)
	// Two numbered, one not: by place among all, oldest = 1, so the special is 2
	// and collides with #2 (as brookfield-youtube's REPORT.md says it may); it is
	// kept, with a line saying so, never dropped.
	if strings.Count(got, `"title":"Morning RePhil Special"`) != 1 || !strings.Contains(got, `"number":2,"title":"Morning RePhil Special"`) {
		t.Errorf("rows:\n%s", got)
	}
}

func TestBrowseFeed(t *testing.T) {
	f, cfg := world(t)
	if _, err := Browse(context.Background(), f, cfg, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	feed := read(t, cfg.Feed)
	for _, w := range []string{`"id":"subsplash-browse"`, `"checks":["read"]`, `"The Invented Sermon"`, `"here":`} {
		if !strings.Contains(feed, w) {
			t.Errorf("feed lacks %s:\n%s", w, feed)
		}
	}
	if strings.Contains(feed, "someone@example.org") || strings.Contains(feed, "First point") {
		t.Errorf("feed holds an address or notes words:\n%s", feed)
	}
}

// YouTube's list has an episode with no number in its title, numbered by its
// place there; Subsplash prints its number. The row of that day is the same
// episode, and gets the share link instead of a second row.
func TestBrowseRephilFillsTheYouTubeRowWithNoNumber(t *testing.T) {
	f, cfg := world(t)
	cfg.EpisodePages = true
	if err := os.WriteFile(cfg.Rephil, []byte(`{"number":168,"title":"Morning RePhil","date":"2026-11-09","url":"https://www.youtube.com/watch?v=abc"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	f.list = []Page{listPage(1, [2]string{"Morning RePhil #2", "rrrrrr2"})}
	if _, err := Browse(context.Background(), f, cfg, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	got := strings.TrimSpace(read(t, cfg.Rephil))
	want := `{"number":168,"title":"Morning RePhil","date":"2026-11-09","url":"https://www.youtube.com/watch?v=abc","share":"https://brookfieldfirst.subspla.sh/rrrrrr2"}`
	if got != want {
		t.Errorf("got %s", got)
	}
}

// seriesPage is the series page showing the first n of episodes (newest
// first), with View More while there are more.
func seriesPage(episodes [][3]string, n int) Page {
	p := Page{URL: seriesURL, Status: 200, Total: len(episodes), More: n < len(episodes), Text: "Morning RePhil\nA live devotional\n"}
	for _, e := range episodes[:n] {
		p.Links = append(p.Links, Link{Text: e[0] + "\n" + e[1], Href: "https://brookfieldfirst.org/media/" + e[2] + "/" + slugOf(e[0])})
		p.Text += e[0] + "\n" + e[1] + "\n"
	}
	return p
}

func TestParseSeries(t *testing.T) {
	items, err := ParseSeries(seriesPage([][3]string{{"Morning RePhil #3", "Nov 13, 2026", "ccccccc"}, {"Morning RePhil", "Nov 9, 2026", "bbbbbbb"}}, 2))
	if err != nil || len(items) != 2 {
		t.Fatalf("%v %v", items, err)
	}
	if items[0] != (SeriesItem{Number: 3, Title: "Morning RePhil #3", Date: "2026-11-13", Code: "ccccccc", Slug: "morning-rephil-3"}) || items[1].Number != 0 || items[1].Date != "2026-11-09" {
		t.Errorf("items %+v", items)
	}
	more, err := ParseSeries(Page{URL: seriesURL, Links: []Link{{Text: "Morning RePhil #4\nJun 15, 2025\u00a0 \u2022 Pastor Example", Href: "https://brookfieldfirst.org/media/ddddddd/x"}}})
	if err != nil || more[0].Date != "2025-06-15" {
		t.Errorf("a date with a preacher after it: %v, %v", more, err)
	}
	if _, err := ParseSeries(Page{URL: seriesURL, Links: []Link{{Text: "Morning RePhil #3", Href: "https://brookfieldfirst.org/media/ccccccc/x"}}}); err == nil {
		t.Error("an episode with no date was taken")
	}
}

func TestBrowseSeriesWritesEveryEpisodeOldestFirst(t *testing.T) {
	f, cfg := world(t)
	cfg.Series = seriesURL
	cfg.Interval = 10 * time.Second
	eps := [][3]string{
		{"Morning RePhil #5", "Nov 13, 2026", "eeeeeee"},
		{"Morning RePhil", "Nov 9, 2026", "ddddddd"}, // no number: its place is 4 of 5
		{"Morning RePhil #3", "Nov 6, 2026", "ccccccc"},
		{"Morning RePhil #2", "Nov 2, 2026", "bbbbbbb"},
		{"Morning RePhil #1", "Oct 26, 2026", "aaaaaaa"},
	}
	f.series = []Page{seriesPage(eps, 2), seriesPage(eps, 4), seriesPage(eps, 5)}
	// Episode 2 is on file from YouTube with its own date and no share link.
	if err := os.WriteFile(cfg.Rephil, []byte(`{"number":2,"title":"Morning RePhil #2","date":"2026-11-03","url":"https://www.youtube.com/watch?v=y"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	f.list = []Page{listPage(1, [2]string{"A Sermon Without Notes", "nnnnnnn"})}
	var out bytes.Buffer
	if _, err := Browse(context.Background(), f, cfg, &out); err != nil {
		t.Fatalf("%v\n%s", err, out.String())
	}
	got := strings.Split(strings.TrimSpace(read(t, cfg.Rephil)), "\n")
	wantNums := []string{`"number":1,`, `"number":2,`, `"number":3,`, `"number":4,`, `"number":5,`}
	if len(got) != 5 {
		t.Fatalf("rows:\n%s", strings.Join(got, "\n"))
	}
	for i, w := range wantNums {
		if !strings.Contains(got[i], w) || !strings.Contains(got[i], `"share":"https://brookfieldfirst.subspla.sh/`) {
			t.Errorf("row %d: %s", i, got[i])
		}
	}
	// Episode 2 keeps its YouTube url and date and gains its share link.
	if !strings.Contains(got[1], `"date":"2026-11-03","url":"https://www.youtube.com/watch?v=y","share":"https://brookfieldfirst.subspla.sh/bbbbbbb"`) {
		t.Errorf("episode 2: %s", got[1])
	}
	if !strings.Contains(got[3], `"title":"Morning RePhil","date":"2026-11-09"`) {
		t.Errorf("the unnumbered episode: %s", got[3])
	}
	if !strings.Contains(out.String(), "no number in its title") {
		t.Errorf("the numbering by place is not said:\n%s", out.String())
	}
	// No episode page was loaded; the pace held.
	for i, l := range f.loads {
		if strings.HasPrefix(l, base) && strings.Contains(l, "rephil") {
			t.Errorf("episode page loaded: %s", l)
		}
		if i > 0 {
			if gap := f.times[i].Sub(f.times[i-1]); gap < 10*time.Second {
				t.Errorf("load %d came %v after the one before", i, gap)
			}
		}
	}
	if f.more != 2 {
		t.Errorf("View More pressed %d times", f.more)
	}
	// A re-run changes nothing and presses nothing: every episode is on file.
	before := read(t, cfg.Rephil)
	f.more, f.loads, f.times = 0, nil, nil
	if _, err := Browse(context.Background(), f, cfg, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if read(t, cfg.Rephil) != before || f.more != 0 {
		t.Errorf("a re-run changed the file or pressed View More %d times", f.more)
	}
}

func TestBrowseSeriesStopsOnA429(t *testing.T) {
	f, cfg := world(t)
	cfg.Series = seriesURL
	eps := [][3]string{{"Morning RePhil #2", "Nov 13, 2026", "bbbbbbb"}, {"Morning RePhil #1", "Nov 9, 2026", "aaaaaaa"}}
	limited := seriesPage(eps, 2)
	limited.Total, limited.More = 5, true
	blocked := seriesPage(eps, 2)
	blocked.Status = 429
	f.series = []Page{limited, blocked}
	f.list = []Page{listPage(1, [2]string{"A Sermon Without Notes", "nnnnnnn"})}
	_, err := Browse(context.Background(), f, cfg, &bytes.Buffer{})
	var st Stop
	if !errors.As(err, &st) || !strings.Contains(st.Why, "429") {
		t.Fatalf("got %v", err)
	}
	// What the first page showed is written, nothing after the 429.
	if got := read(t, cfg.Rephil); strings.Count(got, "\n") != 2 {
		t.Errorf("rows:\n%s", got)
	}
}
