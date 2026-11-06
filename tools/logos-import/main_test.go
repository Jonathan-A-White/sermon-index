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
		if err := run([]string{"testdata/sample-sermons.jsonl"}, "sample-logos", out, "", false, &log); err != nil {
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
	if err := run([]string{"testdata/sample-sermons.jsonl"}, "sample-logos", out2, "", false, b); err != nil {
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

func readFile(t *testing.T, name string) []Input {
	t.Helper()
	f, err := os.Open(name)
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

// The book-archive record form (bin/logos-sermons-parse.mjs fallbackRecord):
// ids like 2~14522 (a character the format refuses), no date, no series, a
// cited list, and offset, textFromArticle, multipart and length.
func TestFallbackRecordForms(t *testing.T) {
	rows, rep := BuildRows(readFile(t, "testdata/notes-sermons.jsonl"))
	if len(rows) != 7 || rep.Rows != 7 || len(rep.Dropped) != 0 {
		t.Fatalf("rows %d (report %d), dropped %v; want 7 and none", len(rows), rep.Rows, rep.Dropped)
	}

	// A row id is the resource and the article id with the ~ made a -; the
	// link is the ref.ly address of the offset.
	r := byID(rows, "MYSRMNOTES01.2-14522")
	if r == nil {
		t.Fatalf("no row MYSRMNOTES01.2-14522 among %+v", rows)
	}
	if str(r.Ref) != "Gen.19.15" || r.Title != "Genesis 19:15" || str(r.Kind) != "link" ||
		str(r.Preacher) != "Charles H. Spurgeon" || r.Date != nil ||
		r.URL != "https://ref.ly/logosres/LLS:MYSRMNOTES01?off=14522" {
		t.Fatalf("Gen. 19:15: %+v", r)
	}
	// cited: the cited list, with a trailing ++ (and the following verses)
	// read as the verse, and the ref itself not repeated.
	if got := strings.Join(r.Cited, " "); got != "Gen.19.3 1John.3.5 Isa.44.25" {
		t.Errorf("cited %q", got)
	}
	if !strings.Contains(str(r.Notes), "Chapter I.") {
		t.Errorf("the book's chapter is in the notes: %q", str(r.Notes))
	}

	// A range from Logos, and the cited range.
	r = byID(rows, "MYSRMNOTES03.2-27593")
	if r == nil || str(r.Ref) != "Matt.7.21-Matt.7.23" || r.Title != "Matthew 7:21–23" || strings.Join(r.Cited, " ") != "Acts.19.13-Acts.19.17" {
		t.Fatalf("Matthew 7:21-23: %+v", r)
	}
	// Phil. is Philippians or Philemon: the row's own reference settles it.
	if r := byID(rows, "MYSRMNOTES03.2-30011"); r == nil || r.Title != "Philippians 2:5" {
		t.Errorf("Phil. 2:5: %+v", r)
	}
	if r := byID(rows, "MYSRMNOTES03.2-31001"); r == nil || r.Title != "1 Corinthians 15:3" {
		t.Errorf("1 Cor. 15:3: %+v", r)
	}
	// Where the book gives a heading, it is the title.
	r = byID(rows, "WAYGODMOODY.1-1639")
	if r == nil || r.Title != "“Love that Passeth Knowledge”" || str(r.Ref) != "Eph.3.19" || strings.Join(r.Cited, " ") != "Eph.3.18" {
		t.Fatalf("Love that Passeth Knowledge: %+v", r)
	}
	// No ref, nothing cited: the title stands, the row stays.
	if r := byID(rows, "OVRCMNGLFMOODY.1-485"); r == nil || r.Ref != nil || r.Title != "The Overcoming Life" || r.Cited != nil {
		t.Errorf("The Overcoming Life: %+v", r)
	}
	// No ref but cited passages: they stay cited, none is guessed to be the
	// text. The two books outside the sixty-six are refused and listed.
	r = byID(rows, "OVRCMNGLFMOODY.1-71840")
	if r == nil || r.Ref != nil || strings.Join(r.Cited, " ") != "Dan.12.3" {
		t.Fatalf("True Wisdom: %+v", r)
	}
	if len(rep.Refused) != 1 || !strings.Contains(rep.Refused[0], "OVRCMNGLFMOODY.1-71840") {
		t.Errorf("refused %v, want True Wisdom's book 45", rep.Refused)
	}
	if len(rep.Multipart) != 1 || rep.Retitled != 5 {
		t.Errorf("multipart %d (want 1), retitled %d (want 5)", len(rep.Multipart), rep.Retitled)
	}
}

func TestFallbackRowsPassTheChecker(t *testing.T) {
	files := []File{
		{Name: "notes", Inputs: readFile(t, "testdata/notes-sermons.jsonl")},
		{Name: "a", Inputs: readFile(t, "testdata/wesley-a-sermons.jsonl")},
		{Name: "b", Inputs: readFile(t, "testdata/wesley-b-sermons.jsonl")},
	}
	rows, _ := BuildSource(files, true)
	out := t.TempDir()
	if err := writeRows(filepath.Join(out, "rows.jsonl"), rows); err != nil {
		t.Fatal(err)
	}
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

func TestReadableTitle(t *testing.T) {
	for _, c := range []struct{ title, ref, want, note string }{
		{"Chapter I: Gen. 19:15", "Gen.19.15", "Genesis 19:15", "Chapter I"},
		{"Chapter CXXX: Matthew 4:3", "Matt.4.3", "Matthew 4:3", "Chapter CXXX"},
		{"Chapter II: Ex. 3:2", "Exod.3.2", "Exodus 3:2", "Chapter II"},
		{"Chapter V: Prov. 3:5", "", "Proverbs 3:5", "Chapter V"},
		{"Chapter VI: Job. 34:33", "Job.34.33", "Job 34:33", "Chapter VI"},
		{"Chapter VII: Song of Solomon 2:1", "Song.2.1", "Song of Solomon 2:1", "Chapter VII"},
		{"Chapter VIII: Psalm 23", "Ps.23", "Psalm 23", "Chapter VIII"},
		{"Chapter IX: Phil. 2:5", "Phil.2.5", "Philippians 2:5", "Chapter IX"},
		{"Chapter IX: Phil. 2:5", "Phlm.1.5", "Philemon 2:5", "Chapter IX"},
		{"Chapter X: Phil. 2:5", "", "Phil. 2:5", "Chapter X"}, // ambiguous, no ref: left as printed
		{"Chapter XI: 2 Tim. 3:16", "2Tim.3.16", "2 Timothy 3:16", "Chapter XI"},
		{"Chapter XII: Romans 6:11, 12", "Rom.6.11", "Romans 6:11, 12", "Chapter XII"},
		{"Chapter I: “Love that Passeth Knowledge”", "Eph.3.19", "“Love that Passeth Knowledge”", "Chapter I"},
		{"Chapter 4: Words of Counsel", "", "Words of Counsel", "Chapter 4"},
		{"Sermon 1: Salvation by Faith", "Eph.2.8", "Sermon 1: Salvation by Faith", ""},
		{"The Overcoming Life", "", "The Overcoming Life", ""},
	} {
		got, note := ReadableTitle(c.title, c.ref)
		if got != c.want || note != c.note {
			t.Errorf("ReadableTitle(%q, %q) = %q, %q; want %q, %q", c.title, c.ref, got, note, c.want, c.note)
		}
	}
}

func TestNormTitle(t *testing.T) {
	for _, c := range []struct{ a, b string }{
		{"Sermon 1: Salvation by Faith", "Sermon I: Salvation by Faith"},
		{"Sermon 42: Satan’s Devices", "Sermon XLII: Satan's Devices"},
		{"Sermon 133: The Death of the Rev. Mr. John Fletcher", "Sermon CXXXIII.: The Death of the Rev. Mr. John Fletcher"},
		{"Sermon 2: The Almost Christian", "XLI. the almost christian"},
		{"Sermon 3: Awake, Thou that Sleepest", "Awake Thou That Sleepest"},
	} {
		if x, y := NormTitle(c.a), NormTitle(c.b); x != y || x == "" {
			t.Errorf("NormTitle(%q) = %q, NormTitle(%q) = %q; want the same", c.a, x, c.b, y)
		}
	}
	for _, c := range []struct{ a, b string }{
		{"Sermon 52: The Reformation of Manners", "Sermon LII: Preached before the Society for Reformation of Manners"},
		{"Sermon 42: Satan’s Devices", "Discourse I"},
	} {
		if NormTitle(c.a) == NormTitle(c.b) {
			t.Errorf("NormTitle(%q) = NormTitle(%q) = %q; want different", c.a, c.b, NormTitle(c.a))
		}
	}
	if got := NormTitle("Sermon XLII:"); got != "" {
		t.Errorf("a title that is only a prefix normalises to %q, want empty", got)
	}
}

func TestDuplicatesOfAnEarlierInputAreDropped(t *testing.T) {
	a, b := readFile(t, "testdata/wesley-a-sermons.jsonl"), readFile(t, "testdata/wesley-b-sermons.jsonl")
	files := []File{{Name: "sermons-on-several-occasions", Inputs: a}, {Name: "works-5-6-7", Inputs: b}}

	// Without --dedupe-titles every line stays; the ids of the two files
	// differ (the resource is in the id), so 2~45431 of both is no repeat.
	rows, rep := BuildSource(files, false)
	if len(rows) != 11 || len(rep.Dropped) != 0 || len(rep.Duplicates) != 0 {
		t.Fatalf("no dedupe: %d rows, dropped %v, duplicates %v; want 11, none, none", len(rows), rep.Dropped, rep.Duplicates)
	}

	rows, rep = BuildSource(files, true)
	// Salvation by Faith, The Almost Christian and Satan's Devices are in the
	// first book: three rows of the second are dropped; the other four stay.
	if len(rows) != 8 || rep.Rows != 8 || len(rep.Duplicates) != 3 {
		t.Fatalf("dedupe: %d rows (report %d), duplicates %v; want 8 and 3", len(rows), rep.Rows, rep.Duplicates)
	}
	for _, id := range []string{"WORKSWESLEY05.2-138047", "WORKSWESLEY05.2-160727", "WORKSWESLEY06.2-78220"} {
		if byID(rows, id) != nil {
			t.Errorf("%s is a duplicate and was written", id)
		}
	}
	for _, id := range []string{"36.0.11.2-45431", "WORKSWESLEY06.3-52318", "WORKSWESLEY07.2-45431", "WORKSWESLEY07.2-46000", "WORKSWESLEY05.2-47000"} {
		if byID(rows, id) == nil {
			t.Errorf("%s was dropped", id)
		}
	}
	// The report names each dropped row and the row it repeats, and counts
	// before and after for each file.
	if !strings.Contains(strings.Join(rep.Duplicates, "\n"), "WORKSWESLEY05.2-138047: \"Sermon I: Salvation by Faith\" repeats 36.0.11.2-45431") {
		t.Errorf("duplicates list: %v", rep.Duplicates)
	}
	if len(rep.Files) != 2 {
		t.Fatalf("files: %+v", rep.Files)
	}
	if f := rep.Files[0]; f.Lines != 4 || f.Before != 4 || f.Duplicates != 0 || f.After != 4 {
		t.Errorf("first file %+v", f)
	}
	if f := rep.Files[1]; f.Lines != 7 || f.Before != 7 || f.Duplicates != 3 || f.After != 4 {
		t.Errorf("second file %+v", f)
	}
	md := rep.Markdown("wesley-logos", []string{"a.jsonl", "b.jsonl"})
	for _, want := range []string{"sermons-on-several-occasions", "works-5-6-7", "Duplicates dropped (3)", "11", "8"} {
		if !strings.Contains(md, want) {
			t.Errorf("REPORT.md lacks %q:\n%s", want, md)
		}
	}
	// Two titles in one file that read the same are both kept: only an
	// earlier file counts.
	one := []Input{b[3], b[3]}
	one[1].ID = "3~99"
	rows, _ = BuildSource([]File{{Name: "x", Inputs: one}}, true)
	if len(rows) != 2 {
		t.Errorf("a repeat within one file: %d rows, want 2", len(rows))
	}
}

func TestRunManyInputs(t *testing.T) {
	out := t.TempDir()
	var log bytes.Buffer
	ins := []string{"testdata/wesley-a-sermons.jsonl", "testdata/wesley-b-sermons.jsonl"}
	if err := run(ins, "wesley-test-logos", out, "", true, &log); err != nil {
		t.Fatal(err)
	}
	a, _ := os.ReadFile(filepath.Join(out, "rows.jsonl"))
	rep, _ := os.ReadFile(filepath.Join(out, "REPORT.md"))
	if n := bytes.Count(a, []byte("\n")); n != 8 {
		t.Errorf("rows.jsonl has %d lines, want 8", n)
	}
	for _, want := range []string{"wesley-a-sermons.jsonl", "wesley-b-sermons.jsonl", "--dedupe-titles", "Duplicates dropped (3)"} {
		if !strings.Contains(string(rep), want) {
			t.Errorf("REPORT.md lacks %q:\n%s", want, rep)
		}
	}
	if !strings.Contains(log.String(), "duplicates dropped") {
		t.Errorf("the tool's own summary lacks the duplicates:\n%s", log.String())
	}
	// Same bytes again.
	out2 := t.TempDir()
	if err := run(ins, "wesley-test-logos", out2, "", true, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	a2, _ := os.ReadFile(filepath.Join(out2, "rows.jsonl"))
	rep2, _ := os.ReadFile(filepath.Join(out2, "REPORT.md"))
	if !bytes.Equal(a, a2) || !bytes.Equal(rep, rep2) {
		t.Error("two runs wrote different bytes")
	}
}
