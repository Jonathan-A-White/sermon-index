package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Jonathan-A-White/sermon-index/internal/sources"
)

func loadFixtureList(t *testing.T) *List {
	t.Helper()
	l, err := ReadList(filepath.Join("testdata", "series.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func TestReadListKeepsEntriesAndUnsureLines(t *testing.T) {
	l := loadFixtureList(t)
	if len(l.Entries) != 8 {
		t.Fatalf("%d entries, want 8", len(l.Entries))
	}
	e := l.Entries[0]
	if e.Date != "2026-11-01" || e.Title != "Gleaning in the Fields" || e.Speaker != "Ann Example" || e.Passage != "Ruth 1:22-2:3" || e.Series != "Ruth: A Made-Up Series" {
		t.Errorf("first entry %+v", e)
	}
	if len(l.Unsure) != 1 || !strings.Contains(l.Unsure[0], "Maybe Sermon") || strings.HasPrefix(l.Unsure[0], "#") {
		t.Errorf("unsure %q", l.Unsure)
	}
}

func TestReadListRefusesABadTable(t *testing.T) {
	dir := t.TempDir()
	write := func(body string) string {
		p := filepath.Join(dir, "t.tsv")
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	if _, err := ReadList(write("series\tdate\ttitle\n")); err == nil {
		t.Error("a table without the speaker and passage columns was read")
	}
	if _, err := ReadList(write("series\tdate\ttitle\tspeaker\tpassage\nS\t2026-01-04\tT\tSp\n")); err == nil {
		t.Error("a short line was read")
	}
	if _, err := ReadList(write("series\tdate\ttitle\tspeaker\tpassage\nS\tJan 4\tT\tSp\tRuth 1:1\n")); err == nil {
		t.Error("a date that is not YYYY-MM-DD was read")
	}
	if _, err := ReadList(filepath.Join(dir, "missing.tsv")); err == nil {
		t.Error("a missing file was read")
	}
}

func rowsForMerge() []sources.Row {
	mk := func(id, date, title string) sources.Row {
		return sources.Row{ID: id, Title: title, URL: watchURL + id, Kind: ptr("link"), Date: ptr(date)}
	}
	rows := []sources.Row{
		mk("vidA", "2026-11-01", "Online Service | November 1, 2026"),
		mk("vidB", "2020-08-23", "Online Service | August 23, 2020"),
		mk("vidC", "2020-07-12", "Online Service | July 12, 2020"),
		mk("vidD", "2026-09-15", "Online Service | September 15, 2026"),
		mk("vidE", "2022-03-03", "Online Service | March 3, 2022"),
		mk("vidF", "2021-02-01", "Online Service | February 1, 2021"),
		mk("vidG", "2020-01-05", "Online Service | January 5, 2020"),
	}
	// a stale YouTube-derived preacher and passage, which the list overrides
	rows[1].Preacher = ptr("Somebody Else")
	rows[1].Ref = ptr("Gen.1.1")
	rows[1].Notes = ptr("Preacher named in the video's description.")
	return rows
}

func TestMergeSetsTitlePreacherAndPassageByDate(t *testing.T) {
	rows := rowsForMerge()
	m := MergeList(rows, loadFixtureList(t))

	a := rowByID(rows, "vidA")
	if a.Title != "Gleaning in the Fields" || str(a.Preacher) != "Ann Example" {
		t.Errorf("row A: title %q preacher %q", a.Title, str(a.Preacher))
	}
	// A range over a chapter break is a valid OSIS range.
	if str(a.Ref) != "Ruth.1.22-Ruth.2.3" {
		t.Errorf("row A ref %q", str(a.Ref))
	}
	if a.URL != watchURL+"vidA" || str(a.Date) != "2026-11-01" || str(a.Kind) != "link" || a.ID != "vidA" {
		t.Errorf("row A lost its link, id, date or kind: %+v", a)
	}
	if !strings.Contains(str(a.Notes), "Ruth: A Made-Up Series") {
		t.Errorf("notes %q do not name the series", str(a.Notes))
	}

	b := rowByID(rows, "vidB")
	if b.Title != "The Second Reaper" || str(b.Preacher) != "Ben Sample" || str(b.Ref) != "Ruth.2.4-Ruth.2.9" {
		t.Errorf("row B %+v", b)
	}
	if strings.Contains(str(b.Notes), "description") {
		t.Errorf("row B keeps a note about the old source: %q", str(b.Notes))
	}
	if m.Merged != 2 {
		t.Errorf("merged %d, want 2", m.Merged)
	}
	rows2 := rowsForMerge()
	// untouched rows are unchanged
	for _, id := range []string{"vidC", "vidD", "vidE", "vidF", "vidG"} {
		got, want := rowByID(rows, id), rowByID(rows2, id)
		if got.Title != want.Title || got.Ref != nil || got.Preacher != nil || got.Notes != nil {
			t.Errorf("row %s changed: %+v", id, got)
		}
	}
}

func TestMergeReportsWhatItDidNotMatch(t *testing.T) {
	rows := rowsForMerge()
	// two rows on one date: reported, both left alone
	rows = append(rows, sources.Row{ID: "vidA2", Title: "Second video that Sunday", URL: watchURL + "vidA2", Date: ptr("2026-11-01")})
	m := MergeList(rows, loadFixtureList(t))

	if a := rowByID(rows, "vidA"); a.Title == "Gleaning in the Fields" || a.Ref != nil {
		t.Errorf("a date with two rows was merged: %+v", a)
	}
	if m.Merged != 1 {
		t.Errorf("merged %d, want only the 2020-08-23 entry", m.Merged)
	}
	why := map[string]string{}
	for _, p := range m.Problems {
		why[p.Entry.Date+"|"+p.Entry.Title] = p.Why
	}
	cases := []struct{ key, want string }{
		{"2026-11-01|Gleaning in the Fields", "2 rows"},
		{"2020-07-12|A Bad Chapter", "passage"},     // refused
		{"2019-01-06|Nobody Posted This", "no row"}, // none on the date
		{"2026-09-15|Twice On One Day", "entries"},
		{"2026-09-15|The Other One", "entries"},
		{"2022-03-03|Not A Passage", "passage"}, // not a passage at all
		{"2021-02-01|Too Wide", "chapters"},     // a series, not a text
	}
	for _, c := range cases {
		if w, ok := why[c.key]; !ok || !strings.Contains(w, c.want) {
			t.Errorf("%s: reported %q (found %v), want a reason with %q", c.key, w, ok, c.want)
		}
	}
	if len(m.Problems) != 7 {
		t.Errorf("%d problems: %+v", len(m.Problems), m.Problems)
	}
	// the refused row keeps its title and gets no passage
	if c := rowByID(rows, "vidC"); c.Title != "Online Service | July 12, 2020" || c.Ref != nil {
		t.Errorf("a refused passage changed the row: %+v", c)
	}
}

func TestRunKeepsTheMergedFieldsOnARerun(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "sources", "brookfield-first")
	rephil := filepath.Join(dir, "data", "morning-rephil.jsonl")
	report := filepath.Join(out, "REPORT.md")
	list := filepath.Join("testdata", "series.tsv")

	var first []byte
	for i := 0; i < 2; i++ {
		if _, err := runFixtureWith(t, out, rephil, report, list); err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(filepath.Join(out, "rows.jsonl"))
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			first = b
		} else if !bytes.Equal(first, b) {
			t.Error("the second run wrote other bytes")
		}
	}
	s := string(first)
	for _, w := range []string{`"title":"Gleaning in the Fields"`, `"ref":"Ruth.1.22-Ruth.2.3"`, `"preacher":"Ann Example"`, `"url":"https://www.youtube.com/watch?v=onlSvc0001A"`} {
		if !strings.Contains(s, w) {
			t.Errorf("rows.jsonl lacks %s", w)
		}
	}
	faults, warnings := sources.CheckReport(out)
	if len(faults) != 0 || len(warnings) != 0 {
		t.Errorf("check: faults %v warnings %v", faults, warnings)
	}

	rep, _ := os.ReadFile(report)
	for _, w := range []string{"Merged into a row: 2", "Nobody Posted This", "Twice On One Day", "Maybe Sermon", "Unsure", "Ruth 99:1-3"} {
		if !strings.Contains(string(rep), w) {
			t.Errorf("REPORT.md lacks %q\n%s", w, rep)
		}
	}
	toml, _ := os.ReadFile(filepath.Join(out, "source.toml"))
	if !strings.Contains(string(toml), "passages.tsv") {
		t.Error("source.toml does not say where the church's list comes from")
	}

	// With no list a run is the YouTube list alone.
	out2 := filepath.Join(dir, "plain")
	if _, err := runFixtureWith(t, out2, filepath.Join(dir, "r2.jsonl"), filepath.Join(out2, "REPORT.md"), ""); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(out2, "rows.jsonl"))
	if strings.Contains(string(b), "Gleaning") {
		t.Error("merged without a list")
	}
}
