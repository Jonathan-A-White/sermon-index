package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Jonathan-A-White/sermon-index/internal/sources"
)

// withSeries is a sermon page that also prints the series it belongs to, as
// the app does after its buttons ("More from <series>").
func withSeries(p Page, series string) Page {
	p.Text = strings.Replace(p.Text, "Related by Topic", "More from "+series+"\nAnother Sermon\nNov 1, 2026 • Pastor Example\nRelated by Topic", 1)
	return p
}

func TestParseSermonSeries(t *testing.T) {
	s, err := ParseSermon(withSeries(sermonPage("3nxf2bb", "The Invented Sermon", "Nov 8, 2026 • Pastor Example • Acts 1:8", true), "A Made-Up Series"))
	if err != nil {
		t.Fatal(err)
	}
	if s.Series != "A Made-Up Series" {
		t.Errorf("series %q", s.Series)
	}
	s, _ = ParseSermon(sermonPage("3nxf2bb", "The Invented Sermon", "Nov 8, 2026", false))
	if s.Series != "" {
		t.Errorf("a page with no 'More from' has series %q", s.Series)
	}
}

func inventoryWorld(t *testing.T) (*fakeBrowser, Config) {
	t.Helper()
	f, cfg := olderWorld(t)
	cfg.Inventory = filepath.Join(filepath.Dir(cfg.Out), "inventory.jsonl")
	cfg.Series = ""
	f.pages[sermonURL("3nxf2bb", "the-invented-sermon")] = withSeries(f.pages[sermonURL("3nxf2bb", "the-invented-sermon")], "Acts Series")
	// A sermon of a day no row has.
	f.pages[sermonURL("ooooooo", "the-older-sermon")] = sermonPage("ooooooo", "The Older Sermon", "Oct 25, 2026 • Pastor Example • Romans 8:1-4", true)
	return f, cfg
}

func TestInventoryWritesEverySermonNewestFirst(t *testing.T) {
	f, cfg := inventoryWorld(t)
	var out bytes.Buffer
	res, err := Browse(context.Background(), f, cfg, &out)
	if err != nil {
		t.Fatalf("%v\n%s", err, out.String())
	}
	entries, err := ReadInventory(cfg.Inventory)
	if err != nil {
		t.Fatal(err)
	}
	// 5 sermons: the list's one, shown again by a series, and four more; no episode.
	if len(entries) != 5 || res.Inventoried != 5 {
		t.Fatalf("%d entries, %d counted: %+v", len(entries), res.Inventoried, entries)
	}
	for i := 1; i < len(entries); i++ {
		if entries[i-1].Date < entries[i].Date {
			t.Errorf("not newest first: %s before %s", entries[i-1].Date, entries[i].Date)
		}
	}
	by := map[string]InvEntry{}
	for _, e := range entries {
		by[e.Code] = e
	}
	e := by["3nxf2bb"]
	want := InvEntry{Code: "3nxf2bb", Address: sermonURL("3nxf2bb", "the-invented-sermon"), Share: "https://brookfieldfirst.subspla.sh/3nxf2bb",
		Date: "2026-11-08", Title: "The Invented Sermon", Preacher: "Pastor Example", Passage: "Acts 1:8", Ref: "Acts.1.8", Series: "Acts Series",
		HasNotes: true, NotesLink: "https://notes.subsplash.com/fill-in/view?page=abc", Row: "ROW1"}
	if e.Code != want.Code || e.Address != want.Address || e.Share != want.Share || e.Date != want.Date || e.Title != want.Title ||
		e.Preacher != want.Preacher || e.Passage != want.Passage || e.Ref != want.Ref || e.Series != want.Series ||
		e.HasNotes != want.HasNotes || e.NotesLink != want.NotesLink || e.Row != want.Row {
		t.Errorf("got %+v\nwant %+v", e, want)
	}
	// No notes: no notes link, and its row is the one of its day.
	n := by["nnnnnnn"]
	if n.HasNotes || n.NotesLink != "" || n.Row != "ROW2" || n.Passage != "John 3:16" {
		t.Errorf("no-notes sermon: %+v", n)
	}
	// A sermon of a day no row has is listed with no row and says why.
	if o := by["zzzzzzz"]; o.Row != "" || o.NoRow == "" {
		t.Errorf("a sermon with no row: %+v", o)
	}
	// The episodes are not sermons.
	for _, l := range f.loads {
		if strings.Contains(l, "rrrrrr") {
			t.Errorf("an episode page was loaded: %s", l)
		}
	}
	// The pace.
	for i := 1; i < len(f.times); i++ {
		if gap := f.times[i].Sub(f.times[i-1]); gap < 10*time.Second {
			t.Errorf("load %d (%s) came %v after the one before", i, f.loads[i], gap)
		}
	}
	// The sermons' addresses are kept for subsplash-links, as browse does.
	if links := read(t, cfg.LinksOut); !strings.Contains(links, sermonURL("3nxf2bb", "the-invented-sermon")) {
		t.Errorf("no link kept:\n%s", links)
	}
}

// The sermon page's passage is kept when the notes' is the wider one.
func TestInventoryKeepsTheSermonPagesPassage(t *testing.T) {
	f, cfg := inventoryWorld(t)
	f.notes[sermonURL("ooooooo", "the-older-sermon")] = notesPage("The Older Sermon", "Romans 8", "October 25, 2026")
	if _, err := Browse(context.Background(), f, cfg, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	entries, _ := ReadInventory(cfg.Inventory)
	for _, e := range entries {
		if e.Code == "ooooooo" && (e.Passage != "Romans 8:1-4" || e.Ref != "Rom.8.1-Rom.8.4") {
			t.Errorf("got %q %q", e.Passage, e.Ref)
		}
	}
}

func TestInventoryResumesAndChangesNothingWritten(t *testing.T) {
	f, cfg := inventoryWorld(t)
	// The first run stops at the third sermon with a 429.
	blocked := sermonURL("yyyyyyy", "the-oldest-sermon")
	good := f.pages[blocked]
	bad := good
	bad.Status = 429
	f.pages[blocked] = bad
	_, err := Browse(context.Background(), f, cfg, &bytes.Buffer{})
	var st Stop
	if !errors.As(err, &st) || !strings.Contains(st.Why, "429") {
		t.Fatalf("got %v", err)
	}
	first, _ := ReadInventory(cfg.Inventory)
	if len(first) == 0 || len(first) >= 5 {
		t.Fatalf("%d entries after a stop", len(first))
	}
	for _, e := range first {
		if e.Code == "yyyyyyy" || e.Code == "zzzzzzz" {
			t.Errorf("an entry at or after the stop: %+v", e)
		}
	}
	before := read(t, cfg.Inventory)
	// The second run, the 429 gone, loads no page that is already inventoried and
	// keeps every line it had.
	f.pages[blocked] = good
	f.loads, f.times = nil, nil
	if _, err := Browse(context.Background(), f, cfg, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	for _, e := range first {
		for _, l := range f.loads {
			if l == e.Address || l == "notes of "+e.Address {
				t.Errorf("a second run loaded %s, already inventoried", l)
			}
		}
	}
	all, _ := ReadInventory(cfg.Inventory)
	if len(all) != 5 {
		t.Fatalf("%d entries after the second run", len(all))
	}
	for _, line := range strings.Split(strings.TrimSpace(before), "\n") {
		if !strings.Contains(read(t, cfg.Inventory), line+"\n") {
			t.Errorf("a line written before was changed or lost: %s", line)
		}
	}
	// A third run loads no sermon at all and writes the same file.
	again := read(t, cfg.Inventory)
	f.loads = nil
	if _, err := Browse(context.Background(), f, cfg, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	for _, l := range f.loads {
		if strings.HasPrefix(l, base) {
			t.Errorf("a third run loaded %s", l)
		}
	}
	if read(t, cfg.Inventory) != again {
		t.Error("a third run changed the inventory")
	}
}

func TestInventoryStopsAtALoginPrompt(t *testing.T) {
	f, cfg := inventoryWorld(t)
	f.pages[sermonURL("ooooooo", "the-older-sermon")] = Page{URL: sermonURL("ooooooo", "the-older-sermon"), Status: 200, Text: "Sign in\nPassword\n"}
	_, err := Browse(context.Background(), f, cfg, &bytes.Buffer{})
	var st Stop
	if !errors.As(err, &st) || !strings.Contains(st.Why, "login") {
		t.Fatalf("got %v", err)
	}
}

// ---- fill

func strp(s string) *string { return &s }

func TestFillWritesALinePerRowThatLacksWhatSubsplashHas(t *testing.T) {
	rows := []sources.Row{
		{ID: "BARE", Title: "Brookfield First - Online Service | March 15, 2020", Date: strp("2020-03-15")},
		{ID: "TITLED", Title: "Kept Title", Date: strp("2020-03-22"), Preacher: strp("Phil Morgan")},
		{ID: "FULL", Title: "Full Row", Date: strp("2020-03-29"), Preacher: strp("Phil Morgan"), Ref: strp("Acts.1.8")},
		{ID: "DONE", Title: "Brookfield First - Online Service | April 5, 2020", Date: strp("2020-04-05")},
	}
	entries := []InvEntry{
		{Code: "aaaaaaa", Date: "2020-03-15", Title: "Still the Creator", Preacher: "Phil Morgan", Passage: "Genesis 1:1-3", Ref: "Gen.1.1-Gen.1.3", Series: "Beginnings", Row: "BARE"},
		{Code: "bbbbbbb", Date: "2020-03-22", Title: "Another Title", Preacher: "Sam Morgan", Passage: "John 3:16", Ref: "John.3.16", Row: "TITLED"},
		{Code: "ccccccc", Date: "2020-03-29", Title: "Full Row", Preacher: "Phil Morgan", Passage: "Acts 1:8", Ref: "Acts.1.8", Row: "FULL"},
		{Code: "ddddddd", Date: "2020-04-05", Title: "Done Already", Preacher: "Phil Morgan", Passage: "John 1:1", Ref: "John.1.1", Row: "DONE"},
		{Code: "eeeeeee", Date: "1999-01-03", Title: "No Row", NoRow: "no row"},
	}
	dir := t.TempDir()
	tsv := filepath.Join(dir, "passages.tsv")
	header := "series\tdate\ttitle\tspeaker\tpassage\trow\tfrom\n"
	existing := "Some Series\t2020-04-05\tDone Already\tPhil Morgan\tJohn 1:1\tDONE\tnotes\n"
	if err := os.WriteFile(tsv, []byte("# a comment\n"+header+existing), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := Fill(entries, rows, FillFiles{Passages: tsv})
	if err != nil {
		t.Fatal(err)
	}
	got := read(t, tsv)
	// BARE gets all three; TITLED has a title and a preacher, so only its passage is added.
	wantBare := "Beginnings\t2020-03-15\tStill the Creator\tPhil Morgan\tGenesis 1:1-3\tBARE\tbrowse\n"
	wantTitled := "\t2020-03-22\tKept Title\t\tJohn 3:16\tTITLED\tbrowse\n"
	for _, w := range []string{wantBare, wantTitled, existing} {
		if !strings.Contains(got, w) {
			t.Errorf("missing line %q in:\n%s", w, got)
		}
	}
	for _, id := range []string{"\tFULL\t", "No Row"} {
		if strings.Contains(got, id) {
			t.Errorf("a line was written for %s:\n%s", id, got)
		}
	}
	if strings.Count(got, "\tDONE\t") != 1 {
		t.Errorf("a row already in the list was written again:\n%s", got)
	}
	if res.Lines != 2 || res.Title != 1 || res.Preacher != 1 || res.Passage != 2 {
		t.Errorf("counts %+v", res)
	}
	// A second run changes nothing.
	if _, err := Fill(entries, rows, FillFiles{Passages: tsv}); err != nil {
		t.Fatal(err)
	}
	if read(t, tsv) != got {
		t.Errorf("a second fill changed the list:\n%s", read(t, tsv))
	}
}

func TestFillReportCountsWhatSubsplashHasAndTheIndexLacks(t *testing.T) {
	rows := []sources.Row{
		{ID: "BOTH", Title: "Both", Date: strp("2026-11-08"), Ref: strp("Acts.1.8")},
		{ID: "ONLYROW", Title: "Only In Index", Date: strp("2026-11-01")},
		{ID: "NOREF", Title: "Brookfield First Service", Date: strp("2026-10-25")},
	}
	entries := []InvEntry{
		{Code: "aaaaaaa", Date: "2026-11-08", Title: "Both", Passage: "Acts 1:8", Ref: "Acts.1.8", Row: "BOTH"},
		{Code: "bbbbbbb", Date: "2026-10-25", Title: "Named", Preacher: "Pastor Example", Row: "NOREF"},
		{Code: "ccccccc", Date: "2019-01-06", Title: "Only On Subsplash", Passage: "John 3:16", Ref: "John.3.16", NoRow: "no row within one day of Sunday 2019-01-06"},
	}
	dir := t.TempDir()
	tsv := filepath.Join(dir, "passages.tsv")
	os.WriteFile(tsv, []byte("series\tdate\ttitle\tspeaker\tpassage\trow\tfrom\n"), 0o644)
	report := filepath.Join(dir, "report.md")
	topics := map[string]bool{"BOTH": true}
	res, err := Fill(entries, rows, FillFiles{Passages: tsv, Report: report, RowsWithThemes: topics})
	if err != nil {
		t.Fatal(err)
	}
	if res.OnlySubsplash != 1 || res.OnlyIndex != 1 || res.Matched != 2 {
		t.Errorf("counts %+v", res)
	}
	// BOTH has verses and themes, NOREF has neither after the fill that gives it only a preacher, the unmatched has verses.
	if res.WithVerses != 2 || res.WithThemes != 1 || res.WithNeither != 1 {
		t.Errorf("verses/themes/neither %+v", res)
	}
	r := read(t, report)
	for _, w := range []string{"3 sermons", "1 lines have `from` = `browse`", "ONLYROW", "ccccccc", "Only On Subsplash", "2026-11-01"} {
		if !strings.Contains(r, w) {
			t.Errorf("the report lacks %q:\n%s", w, r)
		}
	}
}
