package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Jonathan-A-White/sermon-index/internal/passage"
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
		if err := run([]string{"testdata/sample-sermons.jsonl"}, "sample-logos", out, "", Options{}, &log); err != nil {
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
	if err := run([]string{"testdata/sample-sermons.jsonl"}, "sample-logos", out2, "", Options{}, b); err != nil {
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
	if err := run(ins, "wesley-test-logos", out, "", Options{Dedupe: true}, &log); err != nil {
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
	if err := run(ins, "wesley-test-logos", out2, "", Options{Dedupe: true}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	a2, _ := os.ReadFile(filepath.Join(out2, "rows.jsonl"))
	rep2, _ := os.ReadFile(filepath.Join(out2, "REPORT.md"))
	if !bytes.Equal(a, a2) || !bytes.Equal(rep, rep2) {
		t.Error("two runs wrote different bytes")
	}
}

// Batch 3: Augustine, Chrysostom, Piper, Bunyan, Charles Wesley, Jimmy Evans.

func TestCleanLabel(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"Matthew, 5:20", "Matthew 5:20"},
		{"1 Kings, 18:2l", "1 Kings 18:21"},
		{"Song of Solomon, 2:1", "Song of Solomon 2:1"},
		{"Acts 1:12–15; John, 3:16", "Acts 1:12–15; John 3:16"},
		{"Psalm l9:1", "Psalm 19:1"},
		{"Acts 2:1, 7–13", "Acts 2:1, 7–13"},
		{"Philip. 3:13", "Philip. 3:13"},
		{"Genesis 19:15", "Genesis 19:15"},
		{"All Hallows", "All Hallows"},
	} {
		if got := CleanLabel(c.in); got != c.want {
			t.Errorf("CleanLabel(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestRefsInTitle(t *testing.T) {
	for _, c := range []struct {
		title string
		want  []string
	}{
		// Augustine's ref is in the tail of the title.
		{"Sermon 95: On the Words of the Gospel, Mark 8:1–9, Where the Miracle of the Seven Loaves Is Related", []string{"Mark.8.1-Mark.8.9"}},
		{"On the Words of the Gospel, Mark 8:1–9, Where the Miracle of the Seven Loaves Is Related", []string{"Mark.8.1-Mark.8.9"}},
		// More than one: the first is the text, the rest are cited.
		{"Sermon 96: On the Words of the Gospel, Mark 8:34: If Anyone Wishes to Follow Me, Etc., and on the Words of 1 John 2:15: Whoever Loves the World", []string{"Mark.8.34", "1John.2.15"}},
		{"Sermon 108: On the Words of the Gospel, Luke 12:35–36: Let Your Loins be Girt, Etc., and on the Words of Psalm 34:12–15: Who Is the One", []string{"Luke.12.35-Luke.12.36", "Ps.34.12-Ps.34.15"}},
		// Luke’s Gospel, 13:6–17; John’s Gospel 5:24–29; a whole chapter.
		{"Sermon 110: On the Words of Luke’s Gospel, 13:6–17, Where It Speaks of the Fig Tree", []string{"Luke.13.6-Luke.13.17"}},
		{"Sermon 127: On the Words of John’s Gospel 5:24–29: Amen, Amen", []string{"John.5.24-John.5.29"}},
		{"Sermon 136: On the Same Reading from the Gospel of John, Chapter 9: The Giving of Sight", []string{"John.9"}},
		{"Sermon 139A: On the Gospel according to John 11, on the Raising of Lazarus from the Dead", []string{"John.11"}},
		{"Sermon 125: Again on John 5:2–18: On the Five Arcades", []string{"John.5.2-John.5.18"}},
		{"Exodus 3–4", []string{"Exod.3-Exod.4"}},
		{"Sermon 145: On the Words of John’s Gospel 16:24: Until Now, and on the Words of Luke 10:17, 20: Lord, Behold", []string{"John.16.24", "Luke.10.17", "Luke.10.20"}},
		// A title with no passage in it, and a number that is not one.
		{"Sermon 94A: On the Martyrdom of John the Baptist and on the Persecution", nil},
		{"Sermon 112: Preached in the Restored Basilica, on Those Invited to the Dinner", nil},
		{"Sermon 113A: Preached at Hippo, on Sunday, 25 September, on the Gospel", nil},
		{"Sermon 111: On the Passage of the Gospel, Especially on the Words: Are They Few Who Are Saved? 13:23", nil},
		{"Homily XXVII", nil},
	} {
		if got := RefsInTitle(c.title); !reflect.DeepEqual(got, c.want) {
			t.Errorf("RefsInTitle(%q) = %v, want %v", c.title, got, c.want)
		}
	}
}

func TestNonSermon(t *testing.T) {
	for _, title := range []string{
		"Chronological Table",
		"New Testament",
		"Argument",
		"The Argument",
		"Argument, and Summary of the Epistle",
		"Argument of the Epistle",
		"Homilies on the Gospel of John: Index of Texts",
		"Homilies on the Epistle to the Hebrews: Index of Texts",
		"Index of Texts",
		"St. Chrysostom as a Homilist",
		"I. The Place of Chrysostom in the History of Exegesis",
		"II. Extent and Character of Chrysostom’s Exegetical Labors",
		"Introductory Discourse",
		"Prolegomena",
	} {
		if NonSermon(title) == "" {
			t.Errorf("NonSermon(%q) is empty, want a reason", title)
		}
	}
	for _, title := range []string{
		"Homily I", "Chapter II", "Sermon 94A: On the Martyrdom of John the Baptist",
		"Take Care How You Listen! (Part 1)", "Mr. Bunyan’s Last Sermon", "A Sermon on Ephesians 5:14",
		"Sermon 112A: On the Two Sons from the Gospel", "Salvation by Faith", "Arguing with God",
		"The New Testament Church", "Session 1—“The Blessed Marriage”",
	} {
		if why := NonSermon(title); why != "" {
			t.Errorf("NonSermon(%q) = %q, want none", title, why)
		}
	}
}

// Augustine: a ref in the title's tail becomes the row's ref (the archive's
// index has no Logos reference for it), the rest of the passages in the
// title are cited, and the Chronological Table and the Scripture index at
// the end of the book are not sermons.
func TestAugustineRefsFromTheTitleAndNonSermonsDropped(t *testing.T) {
	rows, rep := BuildSource([]File{{Inputs: readFile(t, "testdata/augustine-sermons.jsonl"), RefsFromTitle: true}}, false)
	if len(rows) != 8 || rep.Rows != 8 || len(rep.NonSermon) != 2 || len(rep.Dropped) != 0 {
		t.Fatalf("rows %d, non-sermons %v, dropped %v; want 8, two, none", len(rows), rep.NonSermon, rep.Dropped)
	}
	for _, id := range []string{"AGSTNSRMNS04.1-1278985", "AGSTNSRMNS04.2-1288444"} {
		if byID(rows, id) != nil {
			t.Errorf("%s is not a sermon but is a row", id)
		}
	}
	if !strings.Contains(strings.Join(rep.NonSermon, "\n"), "Chronological Table") || !strings.Contains(strings.Join(rep.NonSermon, "\n"), "New Testament") {
		t.Errorf("the report names the lines dropped: %v", rep.NonSermon)
	}

	r := byID(rows, "AGSTNSRMNS04.1-279517")
	if r == nil || str(r.Ref) != "Mark.8.1-Mark.8.9" || !strings.HasPrefix(r.Title, "Sermon 95: On the Words of the Gospel, Mark 8:1") ||
		str(r.Preacher) != "Augustine of Hippo" || len(r.Cited) != 12 || r.Cited[0] != "Ps.119.164" || contains(r.Cited, "Mark.8.1-Mark.8.9") {
		// The cited list is the fixture's own, with Mark 8:1-9 not repeated.
		t.Fatalf("Sermon 95: %+v", r)
	}
	if !strings.Contains(str(r.Notes), "read from the title") {
		t.Errorf("Sermon 95 notes: %q", str(r.Notes))
	}
	// Two passages in the title: the first is the text, the second cited.
	r = byID(rows, "AGSTNSRMNS04.1-291646")
	if r == nil || str(r.Ref) != "Mark.8.34" || !contains(r.Cited, "1John.2.15") {
		t.Fatalf("Sermon 96: %+v", r)
	}
	// Luke’s Gospel, 13:6–17 and a John chapter.
	if r := byID(rows, "AGSTNSRMNS04.1-541813"); r == nil || str(r.Ref) != "Luke.13.6-Luke.13.17" {
		t.Errorf("Sermon 110: %+v", r)
	}
	if r := byID(rows, "AGSTNSRMNS04.1-1156065"); r == nil || str(r.Ref) != "John.11" {
		t.Errorf("Sermon 139A: %+v", r)
	}
	// No passage in the title: no ref, what Logos cites stays cited.
	r = byID(rows, "AGSTNSRMNS04.1-270629")
	if r == nil || r.Ref != nil || len(r.Cited) == 0 {
		t.Errorf("Sermon 94A: %+v", r)
	}
	if rep.FromTitle != 7 {
		t.Errorf("refs read from a title: %d, want 7", rep.FromTitle)
	}
	// A ref read from the title is not also refused as a label ("no book");
	// only a cited book outside the sixty-six is refused.
	for _, x := range rep.Refused {
		if strings.Contains(x, "no book") {
			t.Errorf("refused as a label: %s", x)
		}
	}
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// Chrysostom: five volumes make one source; Schaff's prolegomena, the
// Arguments and the Indexes of Texts are not homilies; the preacher the book
// names (the editor) is replaced by --preacher.
func TestChrysostomVolumesDropNonSermons(t *testing.T) {
	files := []File{
		{Name: "vol-10", Inputs: readFile(t, "testdata/chrysostom-b-sermons.jsonl"), Preacher: "John Chrysostom"},
		{Name: "vol-11", Inputs: readFile(t, "testdata/chrysostom-d-sermons.jsonl"), Preacher: "John Chrysostom"},
		{Name: "vol-13", Inputs: readFile(t, "testdata/chrysostom-a-sermons.jsonl"), Preacher: "John Chrysostom"},
		{Name: "vol-14", Inputs: readFile(t, "testdata/chrysostom-c-sermons.jsonl"), Preacher: "John Chrysostom"},
	}
	rows, rep := BuildSource(files, false)
	// 3+2+4+3 lines; dropped: the place of Chrysostom, The Argument,
	// St. Chrysostom as a Homilist, Introductory Discourse, Argument and
	// Summary, Index of Texts.
	if rep.Input != 12 || len(rows) != 6 || len(rep.NonSermon) != 6 {
		t.Fatalf("lines %d, rows %d, non-sermons %d (%v); want 12, 6, 6", rep.Input, len(rows), len(rep.NonSermon), rep.NonSermon)
	}
	for _, r := range rows {
		if str(r.Preacher) != "John Chrysostom" {
			t.Errorf("%s: preacher %q", r.ID, str(r.Preacher))
		}
		if !strings.HasPrefix(r.Title, "Homily") && !strings.HasPrefix(r.Title, "Chapter") {
			t.Errorf("%s: %q is not a homily", r.ID, r.Title)
		}
	}
	if r := byID(rows, "6.60.10.3-267160"); r == nil || str(r.Ref) != "Matt.1.1" {
		t.Errorf("Homily II: %+v", r)
	}
	// The per-input counts leave the non-sermons out of "before".
	if rep.Files[2].Lines != 4 || rep.Files[2].Before != 2 || rep.Files[2].After != 2 {
		t.Errorf("vol 13 counts: %+v", rep.Files[2])
	}
	md := rep.Markdown("chrysostom-test-logos", []string{"a", "b"})
	for _, want := range []string{"Non-sermon lines dropped (6)", "Index of Texts", "The Argument", "Introductory Discourse"} {
		if !strings.Contains(md, want) {
			t.Errorf("REPORT.md lacks %q:\n%s", want, md)
		}
	}
}

// Charles Wesley: a comma after the book and an OCR l for a 1 are cleaned
// from the label; with no Logos reference the cleaned label is the ref.
func TestCharlesWesleyRefsCleaned(t *testing.T) {
	in := readFile(t, "testdata/charles-wesley-a-sermons.jsonl")
	rows, rep := BuildRows(in)
	if len(rows) != 5 {
		t.Fatalf("rows %d, want 5", len(rows))
	}
	if r := byID(rows, "SERMSCWESLEY.1-104394"); r == nil || str(r.Ref) != "Matt.5.20" {
		t.Errorf("Matthew, 5:20: %+v", r)
	}
	if len(rep.Disagree) != 0 {
		t.Errorf("a cleaned label agrees with Logos: %v", rep.Disagree)
	}
	// The same line with Logos's reference taken out: the label alone gives it.
	for i := range in {
		in[i].BibleRefs = nil
	}
	rows, rep = BuildRows(in)
	if r := byID(rows, "SERMSCWESLEY.1-104394"); r == nil || str(r.Ref) != "Matt.5.20" {
		t.Errorf("Matthew, 5:20 from the label: %+v", r)
	}
	r := byID(rows, "SERMSCWESLEY.1-247025")
	if r == nil || str(r.Ref) != "1Kgs.18.21" {
		t.Fatalf("1 Kings, 18:2l from the label: %+v", r)
	}
	if rep.FromText < 2 {
		t.Errorf("rows read from the label: %d", rep.FromText)
	}
}

func TestPiperAndBunyanKeepTheirCitedPassages(t *testing.T) {
	rows, rep := BuildRows(readFile(t, "testdata/piper-sermons.jsonl"))
	if len(rows) != 2 || len(rep.NonSermon) != 0 || rep.WithRef != 0 {
		t.Fatalf("Piper rows %d, non-sermons %v, with ref %d", len(rows), rep.NonSermon, rep.WithRef)
	}
	if r := byID(rows, "86BDFD4BE2316756AFB52F8937D86322.1-4922"); r == nil || r.Ref != nil || str(r.Preacher) != "John Piper" || !contains(r.Cited, "Luke.8.4-Luke.8.18") {
		t.Errorf("Take Care How You Listen (Part 1): %+v", r)
	}
	rows, rep = BuildRows(readFile(t, "testdata/bunyan-sermons.jsonl"))
	if len(rows) != 1 || len(rep.Multipart) != 1 || strings.Join(rows[0].Cited, " ") != "John.1.13" {
		t.Errorf("Bunyan: %+v %v", rows, rep.Multipart)
	}
}

// Jimmy Evans: an index-section archive with dates and series and no
// Scripture reference at all.
func TestRefLessSource(t *testing.T) {
	rows, rep := BuildRows(readFile(t, "testdata/jimmy-evans-sermons.jsonl"))
	if len(rows) != 4 || rep.WithRef != 0 || len(rep.NoRef) != 4 || rep.Dated != 2 || rep.YearOnly != 2 {
		t.Fatalf("rows %d, with ref %d, no ref %d, dated %d, year only %d", len(rows), rep.WithRef, len(rep.NoRef), rep.Dated, rep.YearOnly)
	}
	r := byID(rows, "SER2004.1.1")
	if r == nil || r.Ref != nil || str(r.Date) != "2004-06-11" || str(r.Notes) != "Series: Blessed Marriage Seminar, part 1." ||
		r.URL != "https://ref.ly/logosres/LLS:JIMMYEVANSSA?art=SER2004.1.1" {
		t.Fatalf("Session 1: %+v", r)
	}
	if r := byID(rows, "SER2008.3"); r == nil || str(r.Date) != "2008" || str(r.Notes) != "Series: The Cost of Oil." {
		t.Errorf("SER2008.3: %+v", r)
	}
	md := rep.Markdown("jimmy-evans-test-logos", []string{"jimmy-evans-sermons.jsonl"})
	for _, want := range []string{"rows written:                        4", "None of the 4 rows has a Scripture ref", "Rows with no date of their own (2)", "SER2008.3, SER2010.6"} {
		if !strings.Contains(md, want) {
			t.Errorf("REPORT.md lacks %q:\n%s", want, md)
		}
	}
	// And the checker accepts them.
	out := t.TempDir()
	if err := writeRows(filepath.Join(out, "rows.jsonl"), rows); err != nil {
		t.Fatal(err)
	}
	toml := "id = \"" + filepath.Base(out) + "\"\nname = \"t\"\nkind = \"sermons\"\nlicence = \"x\"\ncredit = \"x\"\nhomepage = \"https://example.org\"\nterms_url = \"https://example.org\"\nadded = 2026-11-06\non_by_default = false\n"
	if err := os.WriteFile(filepath.Join(out, "source.toml"), []byte(toml), 0o644); err != nil {
		t.Fatal(err)
	}
	if faults, warnings := sources.CheckReport(out); len(faults) != 0 || len(warnings) != 0 {
		t.Fatalf("faults %v, warnings %v", sources.Faults(faults), sources.Faults(warnings))
	}
}

func TestBatch3RowsPassTheChecker(t *testing.T) {
	files := []File{
		{Name: "aug", Inputs: readFile(t, "testdata/augustine-sermons.jsonl"), RefsFromTitle: true},
		{Name: "chr", Inputs: readFile(t, "testdata/chrysostom-a-sermons.jsonl"), Preacher: "John Chrysostom"},
		{Name: "pip", Inputs: readFile(t, "testdata/piper-sermons.jsonl")},
		{Name: "cw", Inputs: readFile(t, "testdata/charles-wesley-a-sermons.jsonl")},
		{Name: "cw2", Inputs: readFile(t, "testdata/charles-wesley-b-sermons.jsonl")},
	}
	rows, _ := BuildSource(files, false)
	out := t.TempDir()
	if err := writeRows(filepath.Join(out, "rows.jsonl"), rows); err != nil {
		t.Fatal(err)
	}
	toml := "id = \"" + filepath.Base(out) + "\"\nname = \"t\"\nkind = \"sermons\"\nlicence = \"x\"\ncredit = \"x\"\nhomepage = \"https://example.org\"\nterms_url = \"https://example.org\"\nadded = 2026-11-06\non_by_default = false\n"
	if err := os.WriteFile(filepath.Join(out, "source.toml"), []byte(toml), 0o644); err != nil {
		t.Fatal(err)
	}
	if faults, warnings := sources.CheckReport(out); len(faults) != 0 || len(warnings) != 0 {
		t.Fatalf("faults %v, warnings %v", sources.Faults(faults), sources.Faults(warnings))
	}
}

// run writes the flags that matter into REPORT.md's command, and rows.jsonl
// follows them.
func TestRunRecordsPreacherAndRefsFromTitle(t *testing.T) {
	out := t.TempDir()
	if err := run([]string{"testdata/augustine-sermons.jsonl"}, "augustine-test-logos", out, "", Options{Preacher: "Augustine", RefsFromTitle: true}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	rep, _ := os.ReadFile(filepath.Join(out, "REPORT.md"))
	rows, _ := os.ReadFile(filepath.Join(out, "rows.jsonl"))
	for _, want := range []string{"--refs-from-title", `--preacher "Augustine"`, "Non-sermon lines dropped (2)", "Chronological Table"} {
		if !strings.Contains(string(rep), want) {
			t.Errorf("REPORT.md lacks %q:\n%s", want, rep)
		}
	}
	if bytes.Count(rows, []byte(`"preacher":"Augustine"`)) != 8 || !bytes.Contains(rows, []byte(`"ref":"Mark.8.34"`)) {
		t.Errorf("rows.jsonl:\n%s", rows)
	}
}

// Batch 4a: front matter and tables of contents are not sermons.
func TestNonSermonFrontMatter(t *testing.T) {
	for _, title := range []string{
		"Preface", "The Preface", "Author’s Preface", "Translator’s Preface",
		"The Epistle to the Reader", "Epistle to the Reader", "To the Reader", "To the Christian Reader",
		"General Editors’ Introduction", "Foreword",
		"Contents", "Table of Contents",
		"The Contents of the Ensuing Treatise of Contentment",
		"The Contents of the Last Sermon, on Exodus, 14:13",
		"The Contents of the Foregoing Sermons",
		"An Exact Alphabetical Table of the Foregoing Treatise of Contentment",
		"An Exact Alphabetical Table of the Last Sermon on Exodus 13:14",
		"The Table of the Chief Heads Contained in This Book",
	} {
		if NonSermon(title) == "" {
			t.Errorf("NonSermon(%q) is empty, want a reason", title)
		}
	}
	for _, title := range []string{
		"Sermon I", "Sermon II. Ver. 2", "The Saints Dutie in Times of Extremitie",
		"Chapter I: The First Beatitude", "The Epistle of James", "A Preface to Preaching",
		"Contentment", "The Table of the Lord",
		"The First Sermon (July the 6th, 1645)",
	} {
		if why := NonSermon(title); why != "" {
			t.Errorf("NonSermon(%q) = %q, want none", title, why)
		}
	}
}

// Batch 4b: the World's Great Sermons end with a General Index, Strong's with
// a heading over the closing addresses; neither is a sermon.
func TestNonSermonIndexAndCatchAll(t *testing.T) {
	for _, title := range []string{"General Index", "L: Addresses to Successive Graduating Classes of the Rochester Theological Seminary"} {
		if NonSermon(title) == "" {
			t.Errorf("NonSermon(%q) is empty, want a reason", title)
		}
	}
	for _, title := range []string{"The Index of the Heart", "Addresses to the Class of 1900", "Indexing the Scriptures"} {
		if why := NonSermon(title); why != "" {
			t.Errorf("NonSermon(%q) = %q, want none", title, why)
		}
	}
}

// Burroughs, The Rare Jewel: two contents lines and two alphabetical tables
// are dropped and counted; "Sermon I" has no ref and no titled row covers
// it, so it stays.
func TestBurroughsFrontMatterDropped(t *testing.T) {
	rows, rep := BuildRows(readFile(t, "testdata/burroughs-a-sermons.jsonl"))
	if rep.Input != 7 || len(rows) != 3 || len(rep.NonSermon) != 4 || len(rep.Dropped) != 0 {
		t.Fatalf("lines %d, rows %d, non-sermons %d (%v), dropped %v; want 7, 3, 4", rep.Input, len(rows), len(rep.NonSermon), rep.NonSermon, rep.Dropped)
	}
	for _, id := range []string{"WRKSBURROUGHS01.1-15584", "WRKSBURROUGHS01.1-61480", "WRKSBURROUGHS01.1-478132"} {
		if byID(rows, id) == nil {
			t.Errorf("%s is not a row", id)
		}
	}
	if r := byID(rows, "WRKSBURROUGHS01.1-15584"); r == nil || r.Ref != nil || r.Title != "Sermon I" {
		t.Errorf("Sermon I: %+v", r)
	}
	md := rep.Markdown("burroughs-test-logos", []string{"burroughs-a-sermons.jsonl"})
	for _, want := range []string{"Non-sermon lines dropped (4)", "The Contents of the Ensuing Treatise of Contentment", "An Exact Alphabetical Table of the Last Sermon on Exodus 13:14", "lines read:                          7", "rows written:                        3"} {
		if !strings.Contains(md, want) {
			t.Errorf("REPORT.md lacks %q:\n%s", want, md)
		}
	}
}

// Keach and Kuyper: the table of heads, the Epistle to the Reader and the
// Author's Preface (which has a ref of its own, the text of the whole book)
// are front matter.
func TestKeachAndKuyperFrontMatterDropped(t *testing.T) {
	for _, c := range []struct {
		file     string
		rows     int
		dropped  string
		firstRow string
	}{
		{"testdata/keach-a-sermons.jsonl", 2, "The Table of the Chief Heads Contained in This Book", "Sermon I"},
		{"testdata/keach-b-sermons.jsonl", 1, "The Epistle to the Reader", "Sermon I"},
		{"testdata/kuyper-sermons.jsonl", 2, "Author’s Preface", "Sermon 1—The Need of the Hour: A Church that Is Both Organism and Institution"},
	} {
		rows, rep := BuildRows(readFile(t, c.file))
		if len(rows) != c.rows || len(rep.NonSermon) != 1 || !strings.Contains(rep.NonSermon[0], c.dropped) || rows[0].Title != c.firstRow {
			t.Errorf("%s: rows %d, non-sermons %v, first %q", c.file, len(rows), rep.NonSermon, rows[0].Title)
		}
	}
}

// A "Sermon I" row with no title of its own is dropped when a row titled
// "Sermon I: ..." in the same file covers it; alone it stays.
func TestBareSermonNumberCoveredByATitledRow(t *testing.T) {
	mk := func(id, title string) Input {
		return Input{ID: id, Title: title, Preacher: "P", URL: "https://ref.ly/logosres/LLS:X?off=" + id}
	}
	in := []Input{
		mk("1~10", "Sermon I"),
		mk("1~20", "Sermon I: The Fatherhood of God"),
		mk("1~30", "Sermon II"),
		mk("1~40", "Sermon III"),
		mk("1~50", "Sermon 3: Faith"),
		mk("1~60", "Sermon 4"),
	}
	rows, rep := BuildRows(in)
	var ids []string
	for _, r := range rows {
		ids = append(ids, r.ID)
	}
	// "Sermon III" is covered by "Sermon 3: Faith": the number is compared, not its form.
	if got := strings.Join(ids, " "); got != "X.1-20 X.1-30 X.1-50 X.1-60" {
		t.Fatalf("rows %s", got)
	}
	if len(rep.NonSermon) != 2 || !strings.Contains(rep.NonSermon[0], "X.1-10") || !strings.Contains(rep.NonSermon[0], "Sermon I") || !strings.Contains(rep.NonSermon[1], "X.1-40") {
		t.Errorf("non-sermons: %v", rep.NonSermon)
	}
}

func TestTitleDate(t *testing.T) {
	for _, c := range []struct{ title, want string }{
		{"The First Sermon (July the 6th, 1645)", "1645-07-06"},
		{"The Second Sermon (July the 20th, 1645)", "1645-07-20"},
		{"The Third Sermon (August the 10th, 1645)", "1645-08-10"},
		{"The Fourth Sermon (August the 23d, 1645)", "1645-08-23"},
		{"The Fifth Sermon (September the 1st, 1645)", "1645-09-01"},
		{"The Sixth Sermon (October the 2nd, 1645)", "1645-10-02"},
		{"A Sermon (April 4, 1742)", "1742-04-04"},
		{"A Sermon (Feb. the 3d, 1645)", "1645-02-03"},
		{"A Sermon (February the 31st, 1645)", ""},
		{"Sermon I", ""},
		{"A Sermon Preached on Sunday, April 4, 1742, Before the University", ""},
		{"Sermon 12 (1645)", ""},
	} {
		if got := TitleDate(c.title); got != c.want {
			t.Errorf("TitleDate(%q) = %q, want %q", c.title, got, c.want)
		}
	}
}

// Burroughs, The Spots of the Godly: the date in the title is the row's date
// when the index gives none; an index date is not overridden.
func TestDateInTheTitleBecomesTheRowsDate(t *testing.T) {
	in := readFile(t, "testdata/burroughs-b-sermons.jsonl")
	rows, rep := BuildRows(in)
	if len(rows) != 3 || rep.Dated != 3 || rep.DatedFromTitle != 3 || len(rep.NoDate) != 0 {
		t.Fatalf("rows %d, dated %d, from the title %d, no date %v", len(rows), rep.Dated, rep.DatedFromTitle, rep.NoDate)
	}
	r := byID(rows, "WRKSBURROUGHS07.1-1510")
	if r == nil || str(r.Date) != "1645-07-06" || r.Title != "The First Sermon (July the 6th, 1645)" || !strings.Contains(str(r.Notes), "date is read from the title") {
		t.Fatalf("First Sermon: %+v", r)
	}
	if r := byID(rows, "WRKSBURROUGHS07.1-109402"); r == nil || str(r.Date) != "1645-08-23" {
		t.Errorf("Fourth Sermon: %+v", r)
	}
	if !strings.Contains(rep.Summary(), "dates read from the title:") {
		t.Errorf("summary:\n%s", rep.Summary())
	}
	d := "1646-01-02"
	in[0].Date = &d
	rows, rep = BuildRows(in)
	if r := byID(rows, "WRKSBURROUGHS07.1-1510"); r == nil || str(r.Date) != d || rep.DatedFromTitle != 2 {
		t.Errorf("an index date wins: %+v, from the title %d", r, rep.DatedFromTitle)
	}
}

// Maclaren, Broadus, Shedd: "Chapter N: " is taken off a title and kept in
// the notes; "Sermon N:" titles are left as they are.
func TestChapterPrefixStripped(t *testing.T) {
	for _, c := range []struct{ file, id, title, note string }{
		{"testdata/maclaren-sermons.jsonl", "BTTDSTHRSRMNS.1-4253", "The First Beatitude", "Chapter I."},
		{"testdata/maclaren-sermons.jsonl", "BTTDSTHRSRMNS.1-110326", "Teacher or Saviour?", "Chapter VIII."},
		{"testdata/broadus-sermons.jsonl", "JABSERMADDR.1-8364", "Worship", "Chapter I."},
		{"testdata/broadus-sermons.jsonl", "JABSERMADDR.1-316930", "Ministerial Education", "Chapter XIII."},
		{"testdata/shedd-a-sermons.jsonl", "SRMNATMAN.1-43544", "The Future State a Self-Conscious State", "Chapter 1."},
		{"testdata/shedd-b-sermons.jsonl", "SRMSPRTMAN.1-13890", "Sermon I: Religious Meditation", ""},
	} {
		rows, rep := BuildRows(readFile(t, c.file))
		r := byID(rows, c.id)
		if r == nil || r.Title != c.title || (c.note != "" && !strings.Contains(str(r.Notes), c.note)) || strings.Contains(r.Title, "Chapter") {
			t.Errorf("%s %s: %+v", c.file, c.id, r)
		}
		if c.note == "" && rep.Retitled != 0 {
			t.Errorf("%s: retitled %d", c.file, rep.Retitled)
		}
	}
}

// All the batch 4a fixtures make rows the checker accepts.
func TestBatch4aRowsPassTheChecker(t *testing.T) {
	var files []File
	for _, n := range []string{"burroughs-a", "burroughs-b", "burroughs-c", "keach-a", "keach-b", "kuyper", "maclaren", "broadus", "shedd-a", "shedd-b"} {
		files = append(files, File{Name: n, Inputs: readFile(t, "testdata/"+n+"-sermons.jsonl")})
	}
	rows, _ := BuildSource(files, false)
	out := t.TempDir()
	if err := writeRows(filepath.Join(out, "rows.jsonl"), rows); err != nil {
		t.Fatal(err)
	}
	toml := "id = \"" + filepath.Base(out) + "\"\nname = \"t\"\nkind = \"sermons\"\nlicence = \"x\"\ncredit = \"x\"\nhomepage = \"https://example.org\"\nterms_url = \"https://example.org\"\nadded = 2026-11-06\non_by_default = false\n"
	if err := os.WriteFile(filepath.Join(out, "source.toml"), []byte(toml), 0o644); err != nil {
		t.Fatal(err)
	}
	if faults, warnings := sources.CheckReport(out); len(faults) != 0 || len(warnings) != 0 {
		t.Fatalf("faults %v, warnings %v", sources.Faults(faults), sources.Faults(warnings))
	}
}

// Maclaren: no row has a text, but each cites a passage; the report says so
// and does not claim the index gives no passage at all.
func TestRefLessReportWithCitedPassages(t *testing.T) {
	_, rep := BuildRows(readFile(t, "testdata/maclaren-sermons.jsonl"))
	md := rep.Markdown("maclaren-test-logos", []string{"maclaren-sermons.jsonl"})
	for _, want := range []string{"None of the 3 rows has a Scripture ref", "names no text for any of them", "the 3 that cite passages"} {
		if !strings.Contains(md, want) {
			t.Errorf("REPORT.md lacks %q:\n%s", want, md)
		}
	}
}

// Batch 4b: one test for each rule that reads more out of a title.

// "Basil (329–379): The Creation of the World" is the preacher Basil, the
// years 329–379 and the title alone; Volume 2 prints the preacher after the
// title, after a dash; a later volume gives "Born in 1851".
func TestParseTitlePreacherYears(t *testing.T) {
	for _, c := range []struct {
		title, words, preacher, years string
	}{
		{"Basil (329–379): The Creation of the World", "The Creation of the World", "Basil", "329–379"},
		{"H. W. Beecher (1813–1887): Immortality", "Immortality", "H. W. Beecher", "1813–1887"},
		{"Chapin (1814–1880): Nicodemus: The Seeker after Religion", "Nicodemus: The Seeker after Religion", "Chapin", "1814–1880"},
		{"Wagner (Born in 1851): I Am a Voice", "I Am a Voice", "Wagner", "born 1851"},
		{"MacArthur (Born in 1841): Christ—The Question of the Centuries", "Christ—The Question of the Centuries", "MacArthur", "born 1841"},
		{"The Activity of Faith; or, Abraham’s Imitators—Hooker (1586–1647)", "The Activity of Faith; or, Abraham’s Imitators", "Hooker", "1586–1647"},
		{"Christ’s Advent to Judgment—Jeremy Taylor (1613–1667)", "Christ’s Advent to Judgment", "Jeremy Taylor", "1613–1667"},
	} {
		p, ok := ParseTitle(FormPreacherYears, c.title)
		if !ok || p.Title != c.words || p.Preacher != c.preacher || p.Years != c.years || p.Label != "" || p.Numeral != "" {
			t.Errorf("ParseTitle(%q) = %+v, %v; want %q by %q, years %q", c.title, p, ok, c.words, c.preacher, c.years)
		}
	}
	for _, title := range []string{"General Index", "Excessive Grief at the Death of Friends", "Sermon 3 (1700–1750)"} {
		if p, ok := ParseTitle(FormPreacherYears, title); ok {
			t.Errorf("ParseTitle(%q) = %+v, want it refused", title, p)
		}
	}
}

// Princeton: "Christ’s Desire for His People—By William Henry Green (John
// 17:24)" is the title, the preacher and the passage; a stray comma in a
// name ("William, Paxton") is mended, a suffix ("Jr.") is not.
func TestParseTitleByPreacherRef(t *testing.T) {
	for _, c := range []struct {
		title, words, preacher, label string
	}{
		{"Christ’s Desire for His People—By William Henry Green (John 17:24)", "Christ’s Desire for His People", "William Henry Green", "John 17:24"},
		{"The Letter and the Spirit—By Francis L. Patton (2 Cor. 3:6)", "The Letter and the Spirit", "Francis L. Patton", "2 Cor. 3:6"},
		{"Salvation as a Work—By William, Paxton (Philippians 1:6)", "Salvation as a Work", "William Paxton", "Philippians 1:6"},
		{"A Title—With a Dash—By John Smith, Jr. (Psalms 90:9)", "A Title—With a Dash", "John Smith, Jr.", "Psalms 90:9"},
	} {
		p, ok := ParseTitle(FormByPreacherRef, c.title)
		if !ok || p.Title != c.words || p.Preacher != c.preacher || p.Label != c.label || p.Years != "" {
			t.Errorf("ParseTitle(%q) = %+v, %v; want %q by %q (%s)", c.title, p, ok, c.words, c.preacher, c.label)
		}
	}
	if p, ok := ParseTitle(FormByPreacherRef, "Drifting (Hebrews 2:1)"); ok {
		t.Errorf("a title with no By is read: %+v", p)
	}
}

// Strong: "XXIII: The Claims of ..." is numbered; the number is kept apart.
func TestCleanTitleLabel(t *testing.T) {
	for in, want := range map[string]string{"2 Cor. 3:6": "2 Corinthians 3:6", "1 Cor. 16:13": "1 Corinthians 16:13", "Psalms 90:9": "Psalms 90:9", "John 17:24": "John 17:24"} {
		if got := CleanTitleLabel(in); got != want {
			t.Errorf("CleanTitleLabel(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseTitleNumbered(t *testing.T) {
	for _, c := range []struct{ title, words, numeral string }{
		{"I: Philosophy and Religion", "Philosophy and Religion", "I"},
		{"VII: The Will in Theology, or, an Earlier View of the Will", "The Will in Theology, or, an Earlier View of the Will", "VII"},
		{"XLVIII: Dante and the Divine Comedy", "Dante and the Divine Comedy", "XLVIII"},
	} {
		p, ok := ParseTitle(FormNumbered, c.title)
		if !ok || p.Title != c.words || p.Numeral != c.numeral || p.Preacher != "" {
			t.Errorf("ParseTitle(%q) = %+v, %v; want %q numbered %s", c.title, p, ok, c.words, c.numeral)
		}
	}
	for _, title := range []string{"Philosophy and Religion", "Sermon III: Faith", "Mix: Not a Numeral"} {
		if p, ok := ParseTitle(FormNumbered, title); ok {
			t.Errorf("ParseTitle(%q) = %+v, want it refused", title, p)
		}
	}
}

// The World's Great Sermons: the preacher is the one the title names, the
// editor the index names on every line is the book's credit in the notes,
// and the General Index is not a sermon.
func TestWorldsGreatSermonsPreacherAndYearsFromTitle(t *testing.T) {
	var files []File
	for _, n := range []string{"worlds-great-a", "worlds-great-b", "worlds-great-c"} {
		files = append(files, File{Name: n, Inputs: readFile(t, "testdata/"+n+"-sermons.jsonl"), TitleForm: FormPreacherYears})
	}
	rows, rep := BuildSource(files, false)
	if rep.Input != 32 || len(rows) != 31 || len(rep.NonSermon) != 1 || !strings.Contains(rep.NonSermon[0], "General Index") || len(rep.Unparsed) != 0 {
		t.Fatalf("lines %d, rows %d, non-sermons %v, unparsed %v; want 32, 31, the General Index, none", rep.Input, len(rows), rep.NonSermon, rep.Unparsed)
	}
	r := byID(rows, "WRLDGRTSERM01.1-5589")
	if r == nil || r.Title != "The Creation of the World" || str(r.Preacher) != "Basil" || r.Ref != nil || str(r.Date) != "<nil>" {
		t.Fatalf("Basil: %+v", r)
	}
	if n := str(r.Notes); !strings.Contains(n, "Preacher’s years: 329–379.") || !strings.Contains(n, "Book credit: Grenville Kleiser.") {
		t.Errorf("Basil notes: %q", n)
	}
	if got := strings.Join(r.Cited, " "); got != "Gen.1.2" {
		t.Errorf("Basil cited: %q", got)
	}
	if r := byID(rows, "WRLDGRTSERM02.1-1004"); r == nil || str(r.Preacher) != "Hooker" || r.Title != "The Activity of Faith; or, Abraham’s Imitators" || !strings.Contains(str(r.Notes), "1586–1647") {
		t.Errorf("Hooker: %+v", r)
	}
	if r := byID(rows, "WRLDGRTSERM10.1-46019"); r == nil || str(r.Preacher) != "Wagner" || !strings.Contains(str(r.Notes), "Preacher born 1851.") {
		t.Errorf("Wagner: %+v", r)
	}
	for _, r := range rows {
		if str(r.Preacher) == "Grenville Kleiser" || r.Ref != nil {
			t.Errorf("%s: preacher %s, ref %s", r.ID, str(r.Preacher), str(r.Ref))
		}
	}
}

// Princeton: title, preacher, ref and the passages the sermon cites besides.
func TestPrincetonSermonsTitlePreacherRef(t *testing.T) {
	rows, rep := BuildSource([]File{{Name: "princeton-sermons", Inputs: readFile(t, "testdata/princeton-sermons.jsonl"), TitleForm: FormByPreacherRef}}, false)
	if len(rows) != 16 || len(rep.Unparsed) != 0 || len(rep.Refused) != 0 || rep.WithRef != 16 || rep.FromTitle != 16 {
		t.Fatalf("rows %d, unparsed %v, refused %v, with ref %d, from title %d", len(rows), rep.Unparsed, rep.Refused, rep.WithRef, rep.FromTitle)
	}
	r := byID(rows, "PRINCESERMON.1-5790")
	if r == nil || r.Title != "Christ’s Desire for His People" || str(r.Preacher) != "William Henry Green" || str(r.Ref) != "John.17.24" || len(r.Cited) != 0 {
		t.Fatalf("Green: %+v", r)
	}
	if n := str(r.Notes); !strings.Contains(n, "Book credit: Princeton Theological Seminary.") || !strings.Contains(n, "read from the title") {
		t.Errorf("Green notes: %q", n)
	}
	// Logos's first cited reference of "Religion in College" is 1 John 2:13,
	// the title's is John 2:13: the row takes the title's and the report lists it.
	if len(rep.TitleDisagree) != 1 || !strings.Contains(rep.TitleDisagree[0], "PRINCESERMON.1-170527") || !strings.Contains(rep.TitleDisagree[0], "1John.2.13") || !strings.Contains(rep.TitleDisagree[0], "John.2.13") {
		t.Errorf("title and Logos disagree: %v", rep.TitleDisagree)
	}
	rep.TitleForm = FormByPreacherRef
	if md := rep.Markdown("princeton-test-logos", []string{"princeton-sermons.jsonl"}); !strings.Contains(md, "The title and Logos disagree (1)") {
		t.Errorf("REPORT.md lacks the disagreement:\n%s", md)
	}
	if r := byID(rows, "PRINCESERMON.1-42127"); r == nil || str(r.Ref) != "John.16.12-John.16.15" || strings.Join(r.Cited, " ") != "John.14.25 John.14.26" {
		t.Errorf("Hodge: %+v", r)
	}
	if r := byID(rows, "PRINCESERMON.1-207437"); r == nil || str(r.Ref) != "2Cor.3.6" || str(r.Preacher) != "Francis L. Patton" {
		t.Errorf("Patton: %+v", r)
	}
	if r := byID(rows, "PRINCESERMON.1-100592"); r == nil || str(r.Preacher) != "William Paxton" || str(r.Ref) != "Phil.1.6" {
		t.Errorf("Paxton: %+v", r)
	}
	for _, r := range rows {
		if r.Ref == nil || strings.Contains(r.Title, "—By") {
			t.Errorf("%s: ref %s, title %q", r.ID, str(r.Ref), r.Title)
		}
	}
}

// Strong: the numbers come off the titles and are kept in the notes, with
// the kind of piece; the last line, a heading over the addresses to the
// graduating classes, is dropped; the essays and addresses stay.
func TestStrongEssaysKeptAndKindNoted(t *testing.T) {
	const kind = "An address, essay or sermon in a collection; the index does not say which."
	rows, rep := BuildSource([]File{{Name: "strong", Inputs: readFile(t, "testdata/strong-sermons.jsonl"), TitleForm: FormNumbered, Note: kind}}, false)
	if rep.Input != 50 || len(rows) != 49 || len(rep.NonSermon) != 1 || !strings.Contains(rep.NonSermon[0], "Addresses to Successive Graduating Classes") || len(rep.Unparsed) != 0 {
		t.Fatalf("lines %d, rows %d, non-sermons %v, unparsed %v; want 50, 49, the catch-all, none", rep.Input, len(rows), rep.NonSermon, rep.Unparsed)
	}
	r := byID(rows, "PHILANDREL.1-1485859")
	if r == nil || r.Title != "Christianity and Political Economy" || str(r.Preacher) != "Augustus Hopkins Strong" || r.Ref != nil {
		t.Fatalf("XLIV: %+v", r)
	}
	if n := str(r.Notes); !strings.Contains(n, "Number XLIV of the collection.") || !strings.Contains(n, kind) {
		t.Errorf("XLIV notes: %q", n)
	}
	if r := byID(rows, "PHILANDREL.1-1447259"); r == nil || strings.Join(r.Cited, " ") != "Matt.19.9 1Cor.7.15 1Cor.7.10 Mark.10.12 Rom.7.3" || r.Ref != nil {
		t.Errorf("XLIII: %+v", r)
	}
	for _, r := range rows {
		if strings.Contains(r.Title, ": ") && isRoman(strings.ToLower(strings.SplitN(r.Title, ":", 2)[0])) {
			t.Errorf("%s keeps its number: %q", r.ID, r.Title)
		}
	}
}

// A title that does not have the named form is kept as it is and listed.
func TestUnparsedTitleIsKeptAndListed(t *testing.T) {
	in := []Input{{ID: "1~10", Title: "Excessive Grief", Preacher: "Editor", URL: "https://ref.ly/logosres/LLS:X?off=10"}}
	rows, rep := BuildSource([]File{{Name: "x", Inputs: in, TitleForm: FormPreacherYears}}, false)
	if len(rows) != 1 || rows[0].Title != "Excessive Grief" || str(rows[0].Preacher) != "Editor" || len(rep.Unparsed) != 1 {
		t.Fatalf("rows %+v, unparsed %v", rows, rep.Unparsed)
	}
	rep.TitleForm = FormPreacherYears
	md := rep.Markdown("x-logos", []string{"x-sermons.jsonl"})
	for _, want := range []string{"--title-form \"preacher-years\"", "Titles that did not have the form (1)", "X.1-10"} {
		if !strings.Contains(md, want) {
			t.Errorf("REPORT.md lacks %q:\n%s", want, md)
		}
	}
}

// McPherson's Divine Healing Sermons needs no rule: eight lines, eight rows,
// the preacher as the index names her, the passages cited and no text.
func TestMcPhersonDivineHealingSermons(t *testing.T) {
	rows, rep := BuildRows(readFile(t, "testdata/mcpherson-sermons.jsonl"))
	if len(rows) != 8 || len(rep.NonSermon) != 0 || rep.WithRef != 0 || rep.Cited != 2 || len(rep.Multipart) != 6 {
		t.Fatalf("rows %d, non-sermons %v, with ref %d, cited %d, multipart %d", len(rows), rep.NonSermon, rep.WithRef, rep.Cited, len(rep.Multipart))
	}
	if r := byID(rows, "DVNHLNGSRMNS.1-21041"); r == nil || str(r.Preacher) != "Aimee Semple McPherson" || r.Title != "A Double Cure for a Double Curse" || r.Ref != nil || len(r.Cited) == 0 {
		t.Errorf("A Double Cure: %+v", r)
	}
}

// The batch 4b fixtures make rows the checker accepts, and the same bytes twice.
func TestBatch4bRunPassesTheChecker(t *testing.T) {
	for _, c := range []struct {
		ins  []string
		opts Options
		rows int
	}{
		{[]string{"testdata/worlds-great-a-sermons.jsonl", "testdata/worlds-great-b-sermons.jsonl", "testdata/worlds-great-c-sermons.jsonl"}, Options{TitleForm: FormPreacherYears}, 31},
		{[]string{"testdata/princeton-sermons.jsonl"}, Options{TitleForm: FormByPreacherRef}, 16},
		{[]string{"testdata/mcpherson-sermons.jsonl"}, Options{}, 8},
		{[]string{"testdata/strong-sermons.jsonl"}, Options{TitleForm: FormNumbered, Note: "A note."}, 49},
	} {
		out := t.TempDir()
		var log bytes.Buffer
		if err := run(c.ins, filepath.Base(out), out, "", c.opts, &log); err != nil {
			t.Fatal(err)
		}
		rows1, _ := os.ReadFile(filepath.Join(out, "rows.jsonl"))
		if n := bytes.Count(rows1, []byte("\n")); n != c.rows {
			t.Errorf("%v: %d rows, want %d", c.ins, n, c.rows)
		}
		out2 := t.TempDir()
		if err := run(c.ins, filepath.Base(out), out2, "", c.opts, &bytes.Buffer{}); err != nil {
			t.Fatal(err)
		}
		rows2, _ := os.ReadFile(filepath.Join(out2, "rows.jsonl"))
		if !bytes.Equal(rows1, rows2) {
			t.Errorf("%v: two runs differ", c.ins)
		}
		toml := "id = \"" + filepath.Base(out) + "\"\nname = \"t\"\nkind = \"sermons\"\nlicence = \"x\"\ncredit = \"x\"\nhomepage = \"https://example.org\"\nterms_url = \"https://example.org\"\nadded = 2026-11-06\non_by_default = false\n"
		if err := os.WriteFile(filepath.Join(out, "source.toml"), []byte(toml), 0o644); err != nil {
			t.Fatal(err)
		}
		if faults, warnings := sources.CheckReport(out); len(faults) != 0 || len(warnings) != 0 {
			t.Errorf("%v: faults %v, warnings %v", c.ins, sources.Faults(faults), sources.Faults(warnings))
		}
	}
}

func TestRunRefusesAnUnknownTitleForm(t *testing.T) {
	err := run([]string{"testdata/sample-sermons.jsonl"}, "x-logos", t.TempDir(), "", Options{TitleForm: "nonsense"}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "--title-form") {
		t.Errorf("err = %v", err)
	}
}

// A book of the order that gives no rows is named in REPORT.md with why.
func TestNotImportedBooksAreNamedInTheReport(t *testing.T) {
	out := t.TempDir()
	opts := Options{NotImported: []string{"Scofield, Things New and Old (LLS:NWOLDSCOFLD): lesson commentary", "McPherson, This Is That (LLS:THSSMCPHRVNGLST): autobiography"}}
	if err := run([]string{"testdata/mcpherson-sermons.jsonl"}, "mcpherson-test-logos", out, "", opts, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	rep, _ := os.ReadFile(filepath.Join(out, "REPORT.md"))
	for _, want := range []string{"## Books not imported (2)", "- Scofield, Things New and Old (LLS:NWOLDSCOFLD): lesson commentary", "- McPherson, This Is That (LLS:THSSMCPHRVNGLST): autobiography", `--not-imported "Scofield, Things New`} {
		if !strings.Contains(string(rep), want) {
			t.Errorf("REPORT.md lacks %q:\n%s", want, rep)
		}
	}
	// With none given the report has no such section (earlier sources are unchanged).
	out2 := t.TempDir()
	if err := run([]string{"testdata/mcpherson-sermons.jsonl"}, "mcpherson-test-logos", out2, "", Options{}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	rep2, _ := os.ReadFile(filepath.Join(out2, "REPORT.md"))
	if strings.Contains(string(rep2), "Books not imported") {
		t.Errorf("REPORT.md names books not imported with none given")
	}
}

// Batch 5a. Sedgwick prints his text in his title in the old spelling,
// "Reyel. 12:15, 16.", which Logos lists as two separate verses.
func TestTitleLabelToOSISReadsSedgwicksText(t *testing.T) {
	if got := CleanTitleLabel("Reyel. 12:15, 16."); got != "Revelation 12:15, 16" {
		t.Errorf("CleanTitleLabel = %q, want %q", got, "Revelation 12:15, 16")
	}
	for in, want := range map[string][]string{
		"Reyel. 12:15, 16.":  {"Rev.12.15-Rev.12.16"},
		"Reyel. 12:15,16":    {"Rev.12.15-Rev.12.16"},
		"Jerem . 4:3":        {"Jer.4.3"},
		"Luke 10:17, 20":     {"Luke.10.17", "Luke.10.20"},
		"John 3:14, 15, 16":  {"John.3.14-John.3.16"},
		"Psalms 90:9":        {"Ps.90.9"},
		"Romans 5:1-3, 6, 7": {"Rom.5.1-Rom.5.3", "Rom.5.6-Rom.5.7"},
	} {
		got, err := TitleLabelToOSIS(in)
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Errorf("TitleLabelToOSIS(%q) = %v, %v; want %v", in, got, err, want)
		}
	}
	if got, err := TitleLabelToOSIS("Reyel. 99:1"); err == nil {
		t.Errorf("a chapter past the end is read: %v", got)
	}
}

func TestParseTitleRefInBrackets(t *testing.T) {
	p, ok := ParseTitle(FormRefInBrackets, "The Nature and Danger of Heresies (Reyel. 12:15, 16.)")
	if !ok || p.Title != "The Nature and Danger of Heresies" || p.Label != "Reyel. 12:15, 16." || p.Preacher != "" {
		t.Errorf("ParseTitle = %+v, %v", p, ok)
	}
	for _, title := range []string{
		"A Sermon Preached at the Late Fast, to the Commons House of Parliament",
		"Burnt (July the 6th, 1645)",
		"Second Part: And Sow Not among Thornes (Part Two)",
	} {
		if p, ok := ParseTitle(FormRefInBrackets, title); ok {
			t.Errorf("ParseTitle(%q) = %+v, want it refused", title, p)
		}
	}
}

func TestReadableRef(t *testing.T) {
	for in, want := range map[string]string{
		"Matt.5.3":              "Matthew 5:3",
		"Matt.23.29-Matt.23.32": "Matthew 23:29–32",
		"Acts.5.12-Acts.5.14":   "Acts 5:12–14",
		"Rom.8":                 "Romans 8",
		"Ps.3.8":                "Psalms 3:8",
		"Matt.5.3-Matt.6.2":     "Matthew 5:3–6:2",
		"Rom.8-Rom.9":           "Romans 8–9",
		"1Cor.13.4":             "1 Corinthians 13:4",
		"Jude":                  "Jude",
	} {
		if got := ReadableRef(in); got != want {
			t.Errorf("ReadableRef(%q) = %q, want %q", in, got, want)
		}
	}
}

// AC2: Sedgwick's heresies sermon is Revelation 12:15-16, not 12:15 with 12:16
// cited; his other six lines are kept as they are.
func TestSedgwickRefsAreParsed(t *testing.T) {
	rows, rep := BuildSource([]File{{Name: "sedgwick", Inputs: readFile(t, "testdata/sedgwick-sermons.jsonl"), TitleForm: FormRefInBrackets}}, false)
	if rep.Input != 7 || len(rows) != 7 || len(rep.Dropped) != 0 || len(rep.NonSermon) != 0 {
		t.Fatalf("lines %d, rows %d, dropped %v, non-sermons %v; want 7, 7, none, none", rep.Input, len(rows), rep.Dropped, rep.NonSermon)
	}
	r := byID(rows, "NTRDNGRHRSS.1-4734")
	if r == nil || r.Title != "The Nature and Danger of Heresies" || str(r.Preacher) != "Obadiah Sedgwick" || str(r.Ref) != "Rev.12.15-Rev.12.16" {
		t.Fatalf("Heresies: %+v", r)
	}
	for _, c := range r.Cited {
		if c == "Rev.12.15" || c == "Rev.12.16" {
			t.Errorf("Heresies cites %s, which its text holds: %v", c, r.Cited)
		}
	}
	if !strings.Contains(str(r.Notes), "read from the title") {
		t.Errorf("Heresies notes: %q", str(r.Notes))
	}
	if len(rep.Disagree) != 0 || len(rep.TitleDisagree) != 0 || len(rep.Refused) != 0 {
		t.Errorf("disagreements %v %v, refused %v", rep.Disagree, rep.TitleDisagree, rep.Refused)
	}
	// The others keep Logos's ref and the title as it is.
	if r := byID(rows, "THNKSGVNGSRMNWS.1-3910"); r == nil || str(r.Ref) != "Ps.3.8" || !strings.HasPrefix(r.Title, "A Sermon Preached at the Thanksgiving") {
		t.Errorf("Thanksgiving: %+v", r)
	}
	if r := byID(rows, "NGLNDSPRJDGMNTS.1-6622"); r == nil || str(r.Ref) != "Jer.4.3" {
		t.Errorf("England's Preservation: %+v", r)
	}
	if r := byID(rows, "LSHHSTRNSLTNLJH.1-5309"); r == nil || r.Ref != nil || len(r.Cited) == 0 {
		t.Errorf("Elisha has no ref and cites passages: %+v", r)
	}
	// 4 with a Logos text (Thanksgiving, Arke, England's, Heresies), 3 without.
	if rep.WithRef != 4 || len(rep.NoRef) != 3 || rep.FromTitle != 1 || len(rep.Unparsed) != 6 {
		t.Errorf("with ref %d, no ref %v, from title %d, unparsed %d; want 4, 3 ids, 1, 6", rep.WithRef, rep.NoRef, rep.FromTitle, len(rep.Unparsed))
	}
}

// Without the form, Sedgwick's heresies sermon keeps what Logos lists.
func TestSedgwickWithoutTheFormKeepsLogosRef(t *testing.T) {
	rows, _ := BuildRows(readFile(t, "testdata/sedgwick-sermons.jsonl"))
	r := byID(rows, "NTRDNGRHRSS.1-4734")
	if r == nil || str(r.Ref) != "Rev.12.15" || r.Title != "The Nature and Danger of Heresies (Reyel. 12:15, 16.)" {
		t.Errorf("Heresies: %+v", r)
	}
}

// AC1: Moberly's titles are only "Sermon I" to "Sermon XV"; each takes its ref.
func TestMoberlyTitlesComeFromTheirRefs(t *testing.T) {
	rows, rep := BuildSource([]File{{Name: "moberly", Inputs: readFile(t, "testdata/moberly-sermons.jsonl"), TitleFromRef: true}}, false)
	if rep.Input != 15 || len(rows) != 15 || rep.WithRef != 15 || rep.TitledByRef != 15 || len(rep.BareNoRef) != 0 {
		t.Fatalf("lines %d, rows %d, with ref %d, titled by ref %d, bare with no ref %v; want 15, 15, 15, 15, none", rep.Input, len(rows), rep.WithRef, rep.TitledByRef, rep.BareNoRef)
	}
	for id, want := range map[string]string{
		"SRMNSBTTD.1-113776": "Sermon I: Matthew 5:3",
		"SRMNSBTTD.1-327410": "Sermon IX: Matthew 23:29–32",
		"SRMNSBTTD.1-353883": "Sermon X: Acts 5:12–14",
		"SRMNSBTTD.1-387332": "Sermon XI: John 13:18",
		"SRMNSBTTD.1-418201": "Sermon XII: Revelation 2:14",
		"SRMNSBTTD.1-444228": "Sermon XIII: Romans 8:9",
		"SRMNSBTTD.1-469088": "Sermon XIV: Revelation 5:9",
		"SRMNSBTTD.1-489559": "Sermon XV: Revelation 5:9",
	} {
		if r := byID(rows, id); r == nil || r.Title != want || str(r.Preacher) != "George Moberly" {
			t.Errorf("%s: %+v, want the title %q", id, r, want)
		}
	}
	for _, r := range rows {
		if !strings.Contains(r.Title, ": ") || r.Ref == nil {
			t.Errorf("%s is %q with ref %s", r.ID, r.Title, str(r.Ref))
		}
		if !strings.Contains(str(r.Notes), "titles it by the number alone") {
			t.Errorf("%s notes: %q", r.ID, str(r.Notes))
		}
	}
	// Off unless asked.
	rows, rep = BuildRows(readFile(t, "testdata/moberly-sermons.jsonl"))
	if r := byID(rows, "SRMNSBTTD.1-113776"); r == nil || r.Title != "Sermon I" || r.Notes != nil || rep.TitledByRef != 0 {
		t.Errorf("without the flag: %+v", r)
	}
}

// A bare "Sermon N" with no ref has nothing to take: kept, and listed.
func TestBareSermonTitleWithNoRefIsKeptAndListed(t *testing.T) {
	in := []Input{
		{ID: "1~10", Title: "Sermon III", Preacher: "X", URL: "https://ref.ly/logosres/LLS:X?off=10"},
		{ID: "1~20", Title: "Sermon IV: A Title of Its Own", Preacher: "X", Ref: ptr("John 3:16"), URL: "https://ref.ly/logosres/LLS:X?off=20"},
	}
	rows, rep := BuildSource([]File{{Name: "x", Inputs: in, TitleFromRef: true}}, false)
	if len(rows) != 2 || rows[0].Title != "Sermon III" || rows[1].Title != "Sermon IV: A Title of Its Own" || rep.TitledByRef != 0 || len(rep.BareNoRef) != 1 {
		t.Fatalf("rows %+v, titled %d, bare %v", rows, rep.TitledByRef, rep.BareNoRef)
	}
	rep.TitleFromRef = true
	md := rep.Markdown("x-logos", []string{"x-sermons.jsonl"})
	for _, want := range []string{"--title-from-ref", "Bare sermon numbers with no ref (1)", "X.1-10"} {
		if !strings.Contains(md, want) {
			t.Errorf("REPORT.md lacks %q:\n%s", want, md)
		}
	}
}

// Eyton, Chapin and Watson need no rule: their counts, and what each has.
func TestEytonChapinWatson(t *testing.T) {
	for _, c := range []struct {
		name                  string
		rows, withRef, cited  int
		first, firstTitle, pr string
	}{
		{"eyton", 17, 0, 17, "BEATITUDES.1-3676", "Sermon I: Christ as a Moral Teacher", "Robert Eyton"},
		{"chapin", 8, 0, 8, "DSCRSSBTTDS.1-2048", "Discourse I: The Blessing of the Poor in Spirit", "E. H. Chapin"},
		{"watson", 8, 8, 3, "SRMNSBTTDWATSON.1-3399", "Sermon I: Meekness the Rule of Advancement in Christ’s Kingdom", "Alexander Watson"},
	} {
		rows, rep := BuildRows(readFile(t, "testdata/"+c.name+"-sermons.jsonl"))
		if len(rows) != c.rows || rep.WithRef != c.withRef || rep.Cited != c.cited || len(rep.Dropped) != 0 || len(rep.NonSermon) != 0 {
			t.Errorf("%s: rows %d, with ref %d, cited %d, dropped %v, non-sermons %v", c.name, len(rows), rep.WithRef, rep.Cited, rep.Dropped, rep.NonSermon)
			continue
		}
		if r := byID(rows, c.first); r == nil || r.Title != c.firstTitle || str(r.Preacher) != c.pr {
			t.Errorf("%s first: %+v", c.name, r)
		}
	}
	rows, _ := BuildRows(readFile(t, "testdata/watson-sermons.jsonl"))
	if r := byID(rows, "SRMNSBTTDWATSON.1-3399"); r == nil || str(r.Ref) != "Matt.5.5" {
		t.Errorf("Watson I: %+v", r)
	}
}

// The batch 5a fixtures make rows the checker accepts, and the same bytes twice.
func TestBatch5aRunPassesTheChecker(t *testing.T) {
	for _, c := range []struct {
		in   string
		opts Options
		rows int
	}{
		{"testdata/sedgwick-sermons.jsonl", Options{TitleForm: FormRefInBrackets}, 7},
		{"testdata/moberly-sermons.jsonl", Options{TitleFromRef: true}, 15},
		{"testdata/eyton-sermons.jsonl", Options{}, 17},
		{"testdata/chapin-sermons.jsonl", Options{}, 8},
		{"testdata/watson-sermons.jsonl", Options{}, 8},
	} {
		out := t.TempDir()
		if err := run([]string{c.in}, filepath.Base(out), out, "", c.opts, &bytes.Buffer{}); err != nil {
			t.Fatal(err)
		}
		rows1, _ := os.ReadFile(filepath.Join(out, "rows.jsonl"))
		if n := bytes.Count(rows1, []byte("\n")); n != c.rows {
			t.Errorf("%s: %d rows, want %d", c.in, n, c.rows)
		}
		if c.opts.TitleFromRef && !bytes.Contains(rows1, []byte(`"title":"Sermon I: Matthew 5:3"`)) {
			t.Errorf("%s: run did not give the title from the ref", c.in)
		}
		if c.opts.TitleForm == FormRefInBrackets && !bytes.Contains(rows1, []byte(`"ref":"Rev.12.15-Rev.12.16"`)) {
			t.Errorf("%s: run did not read Revelation 12:15-16", c.in)
		}
		out2 := t.TempDir()
		if err := run([]string{c.in}, filepath.Base(out), out2, "", c.opts, &bytes.Buffer{}); err != nil {
			t.Fatal(err)
		}
		if rows2, _ := os.ReadFile(filepath.Join(out2, "rows.jsonl")); !bytes.Equal(rows1, rows2) {
			t.Errorf("%s: two runs differ", c.in)
		}
		toml := "id = \"" + filepath.Base(out) + "\"\nname = \"t\"\nkind = \"sermons\"\nlicence = \"x\"\ncredit = \"x\"\nhomepage = \"https://example.org\"\nterms_url = \"https://example.org\"\nadded = 2026-11-06\non_by_default = false\n"
		if err := os.WriteFile(filepath.Join(out, "source.toml"), []byte(toml), 0o644); err != nil {
			t.Fatal(err)
		}
		if faults, warnings := sources.CheckReport(out); len(faults) != 0 || len(warnings) != 0 {
			t.Errorf("%s: faults %v, warnings %v", c.in, sources.Faults(faults), sources.Faults(warnings))
		}
	}
}

// Batch 5b. Calvin's sermons are in Portuguese: the contents list of the book
// has a front-matter "Prefácio" and, after the thirteenth sermon, an appendix
// that is a reply to slanders.
func TestCalvinPtPrefaceAndAppendixDropped(t *testing.T) {
	for _, c := range []struct{ title, want string }{
		{"Prefácio", "front matter"},
		{"Prefácio à Edição Brasileira", "front matter"},
		{"Resposta a Certas Calúnias e Blasfêmias", "an appendix"},
		{"I. Primeiro Sermão: Gênesis 25.12–22", ""},
		{"Preface to the Sermons", ""},
	} {
		if got := NonSermon(c.title); !strings.HasPrefix(got, c.want) || (c.want == "") != (got == "") {
			t.Errorf("NonSermon(%q) = %q, want %q", c.title, got, c.want)
		}
	}
	rows, rep := BuildSource([]File{{Name: "calvin", Inputs: readFile(t, "testdata/calvin-pt-sermons.jsonl"), TitleForm: FormPtRef}}, false)
	if rep.Input != 15 || len(rows) != 13 || len(rep.NonSermon) != 2 || len(rep.Dropped) != 0 || rep.Rows != 13 {
		t.Fatalf("lines %d, rows %d, non-sermons %v, dropped %v; want 15, 13, two, none", rep.Input, len(rows), rep.NonSermon, rep.Dropped)
	}
	for _, id := range []string{"SRMSSBRLRPRV.1-4439", "SRMSSBRLRPRV.1-520149"} {
		if byID(rows, id) != nil {
			t.Errorf("%s was written", id)
		}
	}
	for _, r := range rows {
		if strings.Contains(r.Title, "Prefácio") || strings.Contains(r.Title, "Calúnias") {
			t.Errorf("%s: %q", r.ID, r.Title)
		}
	}
}

func TestPtBookNamesCoverTheSixtySixBooks(t *testing.T) {
	// One Portuguese name (as Almeida prints it) for each of the 66 books, in
	// the book table's order.
	names := []string{"Gênesis", "Êxodo", "Levítico", "Números", "Deuteronômio", "Josué", "Juízes", "Rute", "1 Samuel", "2 Samuel", "1 Reis", "2 Reis", "1 Crônicas", "2 Crônicas", "Esdras", "Neemias", "Ester", "Jó", "Salmos", "Provérbios", "Eclesiastes", "Cantares", "Isaías", "Jeremias", "Lamentações", "Ezequiel", "Daniel", "Oseias", "Joel", "Amós", "Obadias", "Jonas", "Miqueias", "Naum", "Habacuque", "Sofonias", "Ageu", "Zacarias", "Malaquias", "Mateus", "Marcos", "Lucas", "João", "Atos", "Romanos", "1 Coríntios", "2 Coríntios", "Gálatas", "Efésios", "Filipenses", "Colossenses", "1 Tessalonicenses", "2 Tessalonicenses", "1 Timóteo", "2 Timóteo", "Tito", "Filemom", "Hebreus", "Tiago", "1 Pedro", "2 Pedro", "1 João", "2 João", "3 João", "Judas", "Apocalipse"}
	var books []string
	for _, b := range passage.Books() {
		books = append(books, b.OSIS)
	}
	if len(names) != len(books) {
		t.Fatalf("%d names for %d books", len(names), len(books))
	}
	for i, n := range names {
		if refs, err := TitleLabelToOSIS(PtLabel(n + " 1")); err != nil || len(refs) != 1 || refs[0] != books[i]+".1" {
			t.Errorf("%s 1 = %v, %v; want %s.1", n, refs, err, books[i])
		}
	}
}

func TestPtLabel(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"Gênesis 25.12–22", "Genesis 25:12–22"},
		{"Gênesis 25.21,22", "Genesis 25:21,22"},
		{"Gênesis 26.1–5", "Genesis 26:1–5"},
		{"Salmos 119.1–8; Romanos 8.28", "Psalms 119:1–8; Romans 8:28"},
		{"1 Coríntios 13", "1 Corinthians 13"},
		{"Gênesis 25.12–26.3", "Genesis 25:12–26:3"},
		{"Sermão Primeiro", "Sermão Primeiro"},
	} {
		if got := PtLabel(c.in); got != c.want {
			t.Errorf("PtLabel(%q) = %q, want %q", c.in, got, c.want)
		}
	}
	for _, c := range []struct {
		in   string
		want []string
	}{
		{"Gênesis 25.12–22", []string{"Gen.25.12-Gen.25.22"}},
		{"Gênesis 25.21,22", []string{"Gen.25.21-Gen.25.22"}},
		{"Gênesis 25.12–26.3", []string{"Gen.25.12-Gen.26.3"}},
	} {
		got, err := TitleLabelToOSIS(PtLabel(c.in))
		if err != nil || !reflect.DeepEqual(got, c.want) {
			t.Errorf("%q = %v, %v; want %v", c.in, got, err, c.want)
		}
	}
}

func TestParseTitlePtRef(t *testing.T) {
	tp, ok := ParseTitle(FormPtRef, "I. Primeiro Sermão: Gênesis 25.12–22")
	if !ok || tp.Title != "I. Primeiro Sermão: Gênesis 25.12–22" || tp.Label != "Genesis 25:12–22" {
		t.Errorf("got %+v, %v: want the title kept whole and the label in English", tp, ok)
	}
	for _, bad := range []string{"Prefácio", "Sermão: sem passagem", "Capítulo 3: Resposta"} {
		if _, ok := ParseTitle(FormPtRef, bad); ok {
			t.Errorf("%q has no passage but was read", bad)
		}
	}
}

func TestCalvinPtRefsAreParsedFromTheTitles(t *testing.T) {
	rows, rep := BuildSource([]File{{Name: "calvin", Inputs: readFile(t, "testdata/calvin-pt-sermons.jsonl"), TitleForm: FormPtRef}}, false)
	if rep.WithRef != 13 || rep.FromTitle != 13 || len(rep.NoRef) != 0 || len(rep.Unparsed) != 0 || len(rep.TitleDisagree) != 0 || len(rep.Refused) != 0 || rep.Series != 0 {
		t.Fatalf("with ref %d, from the title %d, no ref %v, unparsed %v, disagree %v, refused %v, series %d", rep.WithRef, rep.FromTitle, rep.NoRef, rep.Unparsed, rep.TitleDisagree, rep.Refused, rep.Series)
	}
	for id, want := range map[string]struct{ title, ref string }{
		"SRMSSBRLRPRV.1-18335":  {"I. Primeiro Sermão: Gênesis 25.12–22", "Gen.25.12-Gen.25.22"},
		"SRMSSBRLRPRV.1-96169":  {"III. Terceiro Sermão: Gênesis 25.21,22", "Gen.25.21-Gen.25.22"},
		"SRMSSBRLRPRV.1-292198": {"VIII. Oitavo Sermão: Gênesis 26.11–21", "Gen.26.11-Gen.26.21"},
		"SRMSSBRLRPRV.1-481671": {"XIII. Décimo Terceiro Sermão: Gênesis 27.29–36", "Gen.27.29-Gen.27.36"},
	} {
		r := byID(rows, id)
		if r == nil || r.Title != want.title || str(r.Ref) != want.ref || str(r.Preacher) != "John Calvin" {
			t.Errorf("%s: %+v, want %q on %s", id, r, want.title, want.ref)
			continue
		}
		for _, c := range r.Cited {
			if rg, err := passage.Parse(c); err == nil && holdsAll(want.ref, []string{rg.OSIS}) {
				t.Errorf("%s: %s, a verse of the text %s, is also cited", id, c, want.ref)
			}
		}
		if len(r.Cited) == 0 {
			t.Errorf("%s cites nothing", id)
		}
		if !strings.Contains(str(r.Notes), "read from the title") {
			t.Errorf("%s notes: %q", id, str(r.Notes))
		}
	}
	// Off unless asked: Logos gives these rows no reference of their own.
	rows, rep = BuildRows(readFile(t, "testdata/calvin-pt-sermons.jsonl"))
	if r := byID(rows, "SRMSSBRLRPRV.1-18335"); r == nil || r.Ref != nil || rep.WithRef != 0 {
		t.Errorf("without the form: %+v, %d with a ref", r, rep.WithRef)
	}
}

// W. I. Evans and Smale print "Chapter N: " before every title.
func TestEvansAndSmaleChapterPrefixStripped(t *testing.T) {
	for _, c := range []struct{ file, id, title, note string }{
		{"testdata/wi-evans-sermons.jsonl", "THSRVRMSTFLW.1-506", "This River Must Flow!", "Chapter 1."},
		{"testdata/wi-evans-sermons.jsonl", "THSRVRMSTFLW.1-43383", "“Diversities of Operations”", "Chapter 5."},
		{"testdata/wi-evans-sermons.jsonl", "THSRVRMSTFLW.1-106740", "Had I but One Hour to Live", "Chapter 10."},
		{"testdata/smale-sermons.jsonl", "PNTCSTLBLSSNG.1-38362", "Some Misconceptions of the Gospel", "Chapter 1."},
		{"testdata/smale-sermons.jsonl", "PNTCSTLBLSSNG.1-134797", "The Secrets of Pentecostal Fullness", "Chapter 6."},
	} {
		rows, rep := BuildRows(readFile(t, c.file))
		r := byID(rows, c.id)
		if r == nil || r.Title != c.title || !strings.Contains(str(r.Notes), c.note) || rep.Retitled != len(rows) {
			t.Errorf("%s %s: %+v (retitled %d of %d)", c.file, c.id, r, rep.Retitled, len(rows))
		}
	}
	rows, _ := BuildRows(readFile(t, "testdata/wi-evans-sermons.jsonl"))
	if r := byID(rows, "THSRVRMSTFLW.1-38321"); r == nil || str(r.Ref) != "1Cor.14" || r.Title != "Speaking in Tongues" {
		t.Errorf("Evans 4: %+v", r)
	}
}

// Pope, Smale and Evans: the counts, and no text for any but Evans's fourth.
func TestPopeSmaleEvansCounts(t *testing.T) {
	for _, c := range []struct {
		name                 string
		rows, withRef, cited int
		first, firstTitle    string
		pr                   string
	}{
		{"pope", 17, 0, 16, "WITOTHERDISC.1-2961", "The Inward Witness of Life in the Son", "William Burt Pope"},
		{"smale", 6, 0, 5, "PNTCSTLBLSSNG.1-38362", "Some Misconceptions of the Gospel", "Joseph Smale"},
		{"wi-evans", 10, 1, 8, "THSRVRMSTFLW.1-506", "This River Must Flow!", "W. I. Evans"},
	} {
		rows, rep := BuildRows(readFile(t, "testdata/"+c.name+"-sermons.jsonl"))
		if len(rows) != c.rows || rep.WithRef != c.withRef || rep.Cited != c.cited || len(rep.Dropped) != 0 || len(rep.NonSermon) != 0 {
			t.Errorf("%s: rows %d, with ref %d, cited %d, dropped %v, non-sermons %v", c.name, len(rows), rep.WithRef, rep.Cited, rep.Dropped, rep.NonSermon)
			continue
		}
		if r := byID(rows, c.first); r == nil || r.Title != c.firstTitle || str(r.Preacher) != c.pr {
			t.Errorf("%s first: %+v", c.name, r)
		}
	}
}

// The batch 5b fixtures make rows the checker accepts, and the same bytes
// twice; REPORT.md's counts are the lines minus the drops.
func TestBatch5bRunPassesTheChecker(t *testing.T) {
	for _, c := range []struct {
		in           string
		opts         Options
		rows, nonSer int
	}{
		{"testdata/pope-sermons.jsonl", Options{}, 17, 0},
		{"testdata/smale-sermons.jsonl", Options{}, 6, 0},
		{"testdata/wi-evans-sermons.jsonl", Options{}, 10, 0},
		{"testdata/calvin-pt-sermons.jsonl", Options{TitleForm: FormPtRef}, 13, 2},
	} {
		out := t.TempDir()
		if err := run([]string{c.in}, filepath.Base(out), out, "", c.opts, &bytes.Buffer{}); err != nil {
			t.Fatal(err)
		}
		rows1, _ := os.ReadFile(filepath.Join(out, "rows.jsonl"))
		if n := bytes.Count(rows1, []byte("\n")); n != c.rows {
			t.Errorf("%s: %d rows, want %d", c.in, n, c.rows)
		}
		md, _ := os.ReadFile(filepath.Join(out, "REPORT.md"))
		in := readFile(t, c.in)
		for _, want := range []string{
			fmt.Sprintf("lines read:                          %d\n", len(in)),
			fmt.Sprintf("rows written:                        %d\n", c.rows),
		} {
			if !bytes.Contains(md, []byte(want)) {
				t.Errorf("%s: REPORT.md lacks %q", c.in, want)
			}
		}
		if c.nonSer > 0 && !bytes.Contains(md, []byte(fmt.Sprintf("Non-sermon lines dropped (%d)", c.nonSer))) {
			t.Errorf("%s: REPORT.md does not count %d non-sermon lines", c.in, c.nonSer)
		}
		if c.opts.TitleForm != "" && !bytes.Contains(md, []byte("--title-form \"pt-ref\"")) {
			t.Errorf("%s: REPORT.md does not repeat the command's form", c.in)
		}
		out2 := t.TempDir()
		if err := run([]string{c.in}, filepath.Base(out), out2, "", c.opts, &bytes.Buffer{}); err != nil {
			t.Fatal(err)
		}
		if rows2, _ := os.ReadFile(filepath.Join(out2, "rows.jsonl")); !bytes.Equal(rows1, rows2) {
			t.Errorf("%s: two runs differ", c.in)
		}
		toml := "id = \"" + filepath.Base(out) + "\"\nname = \"t\"\nkind = \"sermons\"\nlicence = \"x\"\ncredit = \"x\"\nhomepage = \"https://example.org\"\nterms_url = \"https://example.org\"\nadded = 2026-11-06\non_by_default = false\n"
		if err := os.WriteFile(filepath.Join(out, "source.toml"), []byte(toml), 0o644); err != nil {
			t.Fatal(err)
		}
		if faults, warnings := sources.CheckReport(out); len(faults) != 0 || len(warnings) != 0 {
			t.Errorf("%s: faults %v, warnings %v", c.in, sources.Faults(faults), sources.Faults(warnings))
		}
	}
}

// ---- batch 6a: the Sermon Outlines books (mw-qtu3zvz.28) ----

// Sermon Outlines on Acts is mixed: chapters, each an outline on a passage,
// and the numbered sub-points of other chapters. --chapter-outlines keeps one
// row for each chapter that has a passage and drops the rest.
func TestActsKeepsOneRowForEachChapterOutline(t *testing.T) {
	in := readFile(t, "testdata/sermon-outlines-on-acts-sermons.jsonl")
	if len(in) != 86 {
		t.Fatalf("fixture has %d lines, want 86", len(in))
	}
	files := []File{{Inputs: in, ChapterOutlines: true}}
	rows, rep := BuildSource(files, false)
	if len(rows) != 38 || rep.Rows != 38 || len(rep.NotOutline) != 48 || rep.Input != 86 {
		t.Fatalf("rows %d, dropped %d, lines %d: want 38, 48, 86", len(rows), len(rep.NotOutline), rep.Input)
	}
	for _, r := range rows {
		if r.Ref == nil {
			t.Errorf("%s %q has no ref; a kept chapter has a passage", r.ID, r.Title)
		}
		if len(r.Title) > 0 && r.Title[0] >= '0' && r.Title[0] <= '9' {
			t.Errorf("sub-point kept: %q", r.Title)
		}
	}
	// A chapter keeps its passage, loses its "Chapter N: ", and notes the chapter.
	r := byID(rows, "SOACTS.1-129186")
	if r == nil || r.Title != "The Holy Spirit in Conversion" || str(r.Ref) != "Acts.7.51-Acts.7.53" || !strings.Contains(str(r.Notes), "Chapter XIX.") {
		t.Errorf("Chapter XIX: %+v", r)
	}
	// A chapter whose passage is not in Acts is an outline of the book too.
	if r := byID(rows, "SOACTS.1-33015"); r == nil || str(r.Ref) != "Matt.3.11" {
		t.Errorf("Chapter V: %+v", r)
	}
	// What was dropped, and why: a sub-point, a chapter with no passage, the afterword.
	want := map[string]string{
		"SOACTS.2-88511":  "sub-point",
		"SOACTS.1-9251":   "no passage",
		"SOACTS.1-275413": "no passage",
		"SOACTS.1-372551": "not a chapter",
	}
	got := map[string]string{}
	for _, d := range rep.NotOutline {
		id := strings.SplitN(d, ":", 2)[0]
		got[id] = d
	}
	for id, why := range want {
		if d, ok := got[id]; !ok || !strings.Contains(d, why) {
			t.Errorf("%s: dropped as %q, want mention of %q", id, d, why)
		}
	}
	// Without the flag every line is a row, as before.
	all, rep2 := BuildRows(in)
	if len(all) != 86 || len(rep2.NotOutline) != 0 {
		t.Errorf("without --chapter-outlines: %d rows, %d dropped", len(all), len(rep2.NotOutline))
	}
	// The report says the rule and the count.
	rep.ChapterOutlines = true
	md := rep.Markdown("sermon-outlines-on-acts-logos", []string{"sermon-outlines-on-acts-sermons.jsonl"})
	for _, s := range []string{"## Chapter and sub-point lines dropped (48)", "lines dropped as not a chapter outline: 48\n", "- SOACTS.2-88511:"} {
		if !strings.Contains(md, s) && !strings.Contains(rep.Summary(), s) {
			t.Errorf("report lacks %q", s)
		}
	}
}

// The ref in each title parses (ref-in-brackets), and it is the ref the row
// carries: five rows across the series, one with a chapter-spanning text.
func TestOutlineTitleRefsAreParsed(t *testing.T) {
	for _, c := range []struct{ file, id, title, ref string }{
		{"growing-christians", "SOGROWCHR.1-1913", "So Help Me, Me", "Heb.6.13-Heb.6.20"},
		{"seekers", "SOSEEKERS.1-9766", "Seven Secrets to Spiritual Success", "Luke.5.1-Luke.5.11"},
		{"seekers", "SOSEEKERS.1-15704", "Prayer Pointers", "Matt.6"},
		{"special-days", "SOSPECDAYS.1-8396", "Maundy Thursday: The Example of Jesus at Communion", "Luke.22.7-Luke.22.16"},
		{"1-corinthians", "SO1COR.1-2867", "Paul and the Corinthians", "1Cor.1.1-1Cor.1.9"},
	} {
		name := "sermon-outlines-" + c.file
		if c.file == "growing-christians" || c.file == "seekers" || c.file == "special-days" {
			name = "sermon-outlines-for-" + c.file
		} else {
			name = "sermon-outlines-on-" + c.file
		}
		rows, _ := BuildSource([]File{{Inputs: readFile(t, "testdata/"+name+"-sermons.jsonl"), TitleForm: FormRefInBrackets}}, false)
		r := byID(rows, c.id)
		if r == nil || r.Title != c.title || str(r.Ref) != c.ref {
			t.Errorf("%s %s: %+v", name, c.id, r)
		}
	}
	// A bracketed text over two chapters: 'Title (Galatians 1:1–2:21)'.
	x := Input{ID: "1~10", Title: "Freedom in Christ (Galatians 1:1–2:21)", Preacher: "A. Preacher",
		Ref: ptr("Galatians 1:1–2:21"), BibleRefs: []string{"bible.69.1.1-69.2.21"}, URL: "https://ref.ly/logosres/LLS:SOGAL?off=10"}
	rows, rep := BuildSource([]File{{Inputs: []Input{x}, TitleForm: FormRefInBrackets}}, false)
	if len(rows) != 1 || rows[0].Title != "Freedom in Christ" || str(rows[0].Ref) != "Gal.1.1-Gal.2.21" || len(rep.Unparsed) != 0 {
		t.Errorf("Galatians: %+v %v", rows, rep.Unparsed)
	}
	// A title with no Logos reference takes the one it prints.
	x.BibleRefs, x.Ref = nil, nil
	rows, rep = BuildSource([]File{{Inputs: []Input{x}, TitleForm: FormRefInBrackets}}, false)
	if len(rows) != 1 || str(rows[0].Ref) != "Gal.1.1-Gal.2.21" || rep.FromTitle != 1 {
		t.Errorf("Galatians without Logos's ref: %+v (from title %d)", rows, rep.FromTitle)
	}
}

// A title that prints exactly the ref Logos has is not "read from the title":
// nothing was settled by it, so the row has no such note and the report no count.
func TestTitleRefThatEqualsLogosIsNotNoted(t *testing.T) {
	rows, rep := BuildSource([]File{{Inputs: readFile(t, "testdata/sermon-outlines-on-1-corinthians-sermons.jsonl"), TitleForm: FormRefInBrackets}}, false)
	if len(rows) != 28 || rep.FromTitle != 0 {
		t.Fatalf("%d rows, %d read from the title; want 28 and 0", len(rows), rep.FromTitle)
	}
	for _, r := range rows {
		if strings.Contains(str(r.Notes), "read from the title") {
			t.Errorf("%s: %s", r.ID, str(r.Notes))
		}
	}
	// Where Logos lists the verses one by one, the title's range still replaces them.
	x := Input{ID: "1~10", Title: "Sermon on Worry (Matthew 6:25–34)", Ref: ptr("Matthew 6:25"), URL: "https://ref.ly/logosres/LLS:SOX?off=10",
		BibleRefs: []string{"bible.61.6.25", "bible.61.6.26-61.6.33", "bible.61.6.34"}}
	rows, rep = BuildSource([]File{{Inputs: []Input{x}, TitleForm: FormRefInBrackets}}, false)
	if r := rows[0]; str(r.Ref) != "Matt.6.25-Matt.6.34" || !strings.Contains(str(r.Notes), "prints the whole text") || rep.FromTitle != 1 {
		t.Errorf("Worry: %+v (from title %d)", r, rep.FromTitle)
	}
}

// A book of the order that is not imported is named in a "Skipped" section
// with its titles.
func TestSkippedBookIsNamedWithItsTitles(t *testing.T) {
	_, rep := BuildRows(readFile(t, "testdata/evangelistic-sermon-outlines-sermons.jsonl"))
	rep.Skipped = []string{"Sermon Outlines: Soul Winning! Church Building! (LLS:SOSOULWIN): no title carries a ref, so none can be found by a verse || Preaching with Purpose || Authority"}
	md := rep.Markdown("evangelistic-sermon-outlines-logos", []string{"evangelistic-sermon-outlines-sermons.jsonl"})
	for _, s := range []string{
		"## Skipped (1)",
		"- Sermon Outlines: Soul Winning! Church Building! (LLS:SOSOULWIN): no title carries a ref, so none can be found by a verse\n  - Preaching with Purpose\n  - Authority\n",
		"--skipped \"Sermon Outlines: Soul Winning!",
	} {
		if !strings.Contains(md, s) {
			t.Errorf("REPORT.md lacks %q", s)
		}
	}
}

// The batch 6a fixtures make rows the checker accepts, with the counts the
// Mayor kept (the lines of each index, minus what the rules drop), and the
// same bytes twice.
func TestBatch6aRunPassesTheChecker(t *testing.T) {
	for _, c := range []struct {
		in           string
		opts         Options
		lines, rows  int
		reportHas    []string
		reportHasNot []string
	}{
		{"testdata/evangelistic-sermon-outlines-sermons.jsonl", Options{Skipped: []string{"Sermon Outlines: Soul Winning! Church Building! (LLS:SOSOULWIN): no ref || Authority"}}, 29, 29, []string{"## Skipped (1)", "  - Authority"}, nil},
		{"testdata/sermon-outlines-for-growing-christians-sermons.jsonl", Options{TitleForm: FormRefInBrackets}, 27, 27, nil, []string{"## Skipped"}},
		{"testdata/sermon-outlines-for-seekers-sermons.jsonl", Options{TitleForm: FormRefInBrackets}, 26, 26, nil, nil},
		{"testdata/sermon-outlines-for-special-days-sermons.jsonl", Options{TitleForm: FormRefInBrackets}, 28, 28, nil, nil},
		{"testdata/sermon-outlines-on-1-corinthians-sermons.jsonl", Options{TitleForm: FormRefInBrackets}, 28, 28, nil, nil},
		{"testdata/sermon-outlines-on-acts-sermons.jsonl", Options{ChapterOutlines: true}, 86, 38, []string{"lines dropped as not a chapter outline: 48\n", "--chapter-outlines"}, nil},
	} {
		out := t.TempDir()
		if err := run([]string{c.in}, filepath.Base(out), out, "", c.opts, &bytes.Buffer{}); err != nil {
			t.Fatal(err)
		}
		rows1, _ := os.ReadFile(filepath.Join(out, "rows.jsonl"))
		if n := bytes.Count(rows1, []byte("\n")); n != c.rows {
			t.Errorf("%s: %d rows, want %d", c.in, n, c.rows)
		}
		md, _ := os.ReadFile(filepath.Join(out, "REPORT.md"))
		for _, want := range append([]string{
			fmt.Sprintf("lines read:                          %d\n", c.lines),
			fmt.Sprintf("rows written:                        %d\n", c.rows),
		}, c.reportHas...) {
			if !bytes.Contains(md, []byte(want)) {
				t.Errorf("%s: REPORT.md lacks %q", c.in, want)
			}
		}
		for _, not := range c.reportHasNot {
			if bytes.Contains(md, []byte(not)) {
				t.Errorf("%s: REPORT.md has %q", c.in, not)
			}
		}
		out2 := t.TempDir()
		if err := run([]string{c.in}, filepath.Base(out), out2, "", c.opts, &bytes.Buffer{}); err != nil {
			t.Fatal(err)
		}
		if rows2, _ := os.ReadFile(filepath.Join(out2, "rows.jsonl")); !bytes.Equal(rows1, rows2) {
			t.Errorf("%s: two runs differ", c.in)
		}
		toml := "id = \"" + filepath.Base(out) + "\"\nname = \"t\"\nkind = \"sermons\"\nlicence = \"x\"\ncredit = \"x\"\nhomepage = \"https://example.org\"\nterms_url = \"https://example.org\"\nadded = 2026-11-06\non_by_default = false\n"
		if err := os.WriteFile(filepath.Join(out, "source.toml"), []byte(toml), 0o644); err != nil {
			t.Fatal(err)
		}
		if faults, warnings := sources.CheckReport(out); len(faults) != 0 || len(warnings) != 0 {
			t.Errorf("%s: faults %v, warnings %v", c.in, sources.Faults(faults), sources.Faults(warnings))
		}
	}
}
