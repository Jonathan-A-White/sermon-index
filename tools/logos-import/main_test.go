package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
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
		if err := run([]string{"testdata/sample-sermons.jsonl"}, "sample-logos", out, "", false, "", false, &log); err != nil {
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
	if err := run([]string{"testdata/sample-sermons.jsonl"}, "sample-logos", out2, "", false, "", false, b); err != nil {
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
	if err := run(ins, "wesley-test-logos", out, "", true, "", false, &log); err != nil {
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
	if err := run(ins, "wesley-test-logos", out2, "", true, "", false, &bytes.Buffer{}); err != nil {
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
	if err := run([]string{"testdata/augustine-sermons.jsonl"}, "augustine-test-logos", out, "", false, "Augustine", true, &bytes.Buffer{}); err != nil {
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
