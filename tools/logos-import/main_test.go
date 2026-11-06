package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Jonathan-A-White/sermon-index/internal/sources"
)

func sample(t *testing.T) []Input {
	t.Helper()
	f, err := os.Open("testdata/sample-sermons.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	in, err := ReadInput(f)
	if err != nil {
		t.Fatal(err)
	}
	return in
}

func byID(rows []sources.Row, id string) *sources.Row {
	for i := range rows {
		if rows[i].ID == id {
			return &rows[i]
		}
	}
	return nil
}

func str(p *string) string {
	if p == nil {
		return "<nil>"
	}
	return *p
}

func TestBuildRowsMapsTheFields(t *testing.T) {
	rows, rep := BuildRows(sample(t))

	// Who Is Man: bible.1.1 is Genesis 1; year only, so the date is the year.
	r := byID(rows, "YEAR.1996.6")
	if r == nil || str(r.Ref) != "Gen.1" || r.Title != "Who Is Man" || str(r.Kind) != "link" ||
		str(r.Preacher) != "Greg Laurie" || str(r.Date) != "1996" ||
		r.URL != "https://ref.ly/logosres/LLS:GRGLRSRMNRCH?art=YEAR.1996.6" {
		t.Fatalf("Who Is Man: %+v", r)
	}
	if r.Notes != nil || r.Cited != nil || r.Series != nil {
		t.Errorf("Who Is Man has no series, no cited, no notes: %+v", r)
	}

	// A series and a part go into the notes; a full date is kept.
	r = byID(rows, "YEAR.2002.2")
	if r == nil || str(r.Date) != "2002-07-12" || str(r.Notes) != "Series: Second Chances, part 2." {
		t.Fatalf("A Second Chance in Life: %+v", r)
	}

	// The other references are cited; the first is the ref.
	r = byID(rows, "CH51")
	if r == nil || str(r.Ref) != "2Sam.11" || strings.Join(r.Cited, " ") != "2Sam.12.1-2Sam.12.31 Ps.51.1-Ps.51.19" || str(r.Notes) != "Part 11." {
		t.Fatalf("Repentance: %+v", r)
	}

	// A label with no Logos reference is read from the ref text; the note says so.
	r = byID(rows, "YEAR.1996.17")
	if r == nil || str(r.Ref) != "Jonah" || !strings.Contains(str(r.Notes), "Series: A Second Chance for America, part 3.") ||
		!strings.Contains(str(r.Notes), `read from the label "Jonah"`) {
		t.Fatalf("Jonah: %+v", r)
	}
	// ... and Jonah is four chapters, so it is a series row (found by title, matched by no passage).
	if r.Series == nil || !*r.Series {
		t.Errorf("Jonah spans four chapters, want series: %+v", r)
	}

	// No reference at all: the row stays, with no ref.
	r = byID(rows, "YEAR.2001.36")
	if r == nil || r.Ref != nil || r.Series != nil {
		t.Errorf("Crusade July 4: %+v", r)
	}

	// The Revelation row's range of chapters to a verse.
	r = byID(rows, "SERIES8.14")
	if r == nil || str(r.Ref) != "Acts.2.47" || strings.Join(r.Cited, " ") != "Rev.4-Rev.5.14" {
		t.Fatalf("Hands up: %+v", r)
	}
	if rep.Rows != len(rows) {
		t.Errorf("report counts %d rows, %d written", rep.Rows, len(rows))
	}
}

func TestBuildRowsDropsAndReports(t *testing.T) {
	rows, rep := BuildRows(sample(t))
	// 15 lines: the row with neither ref nor title, the row with no
	// address and the repeat of an id are dropped.
	if rep.Input != 15 || len(rows) != 12 || rep.Rows != 12 {
		t.Fatalf("input %d, rows %d (report %d); want 15 in, 12 out", rep.Input, len(rows), rep.Rows)
	}
	for _, id := range []string{"NOTHING.1", "NOURL.1"} {
		if byID(rows, id) != nil {
			t.Errorf("%s was written", id)
		}
	}
	if r := byID(rows, "YEAR.1996.6"); r == nil || r.Title != "Who Is Man" {
		t.Errorf("the repeat replaced the first: %+v", r)
	}
	got := strings.Join(rep.Dropped, "\n")
	for _, want := range []string{"NOTHING.1: neither a ref nor a title", "NOURL.1: no url", "YEAR.1996.6: id repeats line 1"} {
		if !strings.Contains(got, want) {
			t.Errorf("dropped list lacks %q:\n%s", want, got)
		}
	}
	// A verse past the chapter is refused, the row kept without a ref; a
	// book outside the sixty-six likewise.
	if r := byID(rows, "BAD.1"); r == nil || r.Ref != nil {
		t.Errorf("BAD.1: %+v", r)
	}
	if r := byID(rows, "BAD.2"); r == nil || r.Ref != nil {
		t.Errorf("BAD.2: %+v", r)
	}
	if len(rep.Refused) != 2 {
		t.Errorf("refused %v, want the two", rep.Refused)
	}
	if len(rep.NoRef) != 4 { // Crusade, Exodus, and the two refused
		t.Errorf("no-ref rows %v, want 4", rep.NoRef)
	}
}

func TestBuildRowsPassTheChecker(t *testing.T) {
	rows, _ := BuildRows(sample(t))
	out := t.TempDir()
	if err := writeRows(filepath.Join(out, "rows.jsonl"), rows); err != nil {
		t.Fatal(err)
	}
	// A source.toml the checker accepts, to check the rows against.
	toml := `id = "` + filepath.Base(out) + `"
name = "t"
kind = "sermons"
licence = "x"
credit = "x"
homepage = "https://example.org"
terms_url = "https://example.org"
added = 2026-11-06
on_by_default = false
`
	if err := os.WriteFile(filepath.Join(out, "source.toml"), []byte(toml), 0o644); err != nil {
		t.Fatal(err)
	}
	faults, warnings := sources.CheckReport(out)
	if len(faults) != 0 || len(warnings) != 0 {
		t.Fatalf("faults %v, warnings %v", sources.Faults(faults), sources.Faults(warnings))
	}
}

func TestRunWritesTheSameBytesTwice(t *testing.T) {
	out := t.TempDir()
	var log bytes.Buffer
	for i := 0; i < 2; i++ {
		if err := run("testdata/sample-sermons.jsonl", "sample-logos", out, "", &log); err != nil {
			t.Fatal(err)
		}
	}
	a, err := os.ReadFile(filepath.Join(out, "rows.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	rep, err := os.ReadFile(filepath.Join(out, "REPORT.md"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Count(a, []byte("\n")) != 12 {
		t.Errorf("rows.jsonl has %d lines, want 12", bytes.Count(a, []byte("\n")))
	}
	// Every line is a JSON object the format knows; no HTML escaping of a quote or an ampersand.
	for _, line := range bytes.Split(bytes.TrimSpace(a), []byte("\n")) {
		var m map[string]any
		if err := json.Unmarshal(line, &m); err != nil {
			t.Fatalf("%s: %v", line, err)
		}
	}
	for _, want := range []string{"NOTHING.1", "neither a ref nor a title", "sample-logos"} {
		if !strings.Contains(string(rep), want) {
			t.Errorf("REPORT.md lacks %q", want)
		}
	}
	// A second run over the same input writes the same bytes.
	b := &bytes.Buffer{}
	out2 := t.TempDir()
	if err := run("testdata/sample-sermons.jsonl", "sample-logos", out2, "", b); err != nil {
		t.Fatal(err)
	}
	a2, _ := os.ReadFile(filepath.Join(out2, "rows.jsonl"))
	rep2, _ := os.ReadFile(filepath.Join(out2, "REPORT.md"))
	if !bytes.Equal(a, a2) || !bytes.Equal(rep, rep2) {
		t.Error("two runs wrote different bytes")
	}
}

func TestReadInputRefusesAnUnknownField(t *testing.T) {
	_, err := ReadInput(strings.NewReader(`{"id":"1","title":"t","url":"https://x","surprise":1}` + "\n"))
	if err == nil {
		t.Fatal("want an error for a field the tool does not know")
	}
}
