package main

import (
	"github.com/Jonathan-A-White/sermon-index/internal/passage"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The rows of his bug: 'Lightning in Clay Pots' (FhOTBXf3S90) reads like every
// other service on the channel, with no passage, so Lampas never offers it.
const notesRows = `{"id":"FhOTBXf3S90","title":"Brookfield First - Online Service | October 19, 2025","url":"y1","kind":"link","date":"2025-10-19"}
{"id":"also1","title":"Brookfield First - Online Service | June 21, 2026","url":"y2","kind":"link","date":"2026-06-21"}
{"id":"has1","title":"Growing in Grace","ref":"2Pet.1.5-2Pet.1.11","url":"y3","kind":"link","date":"2026-09-06","preacher":"Sam Morgan"}
{"id":"msg1","title":"Brookfield First - Online Service | November 2, 2025","url":"y4","kind":"link","date":"2025-11-02"}
{"id":"titled1","title":"A Title Of Its Own","url":"y5","kind":"link","date":"2025-12-07"}
{"id":"noref1","title":"Brookfield First - Online Service | January 4, 2026","url":"y6","kind":"link","date":"2026-01-04"}
`

const notesRecords = `{"id":"FhOTBXf3S90","date":"2025-10-19","title":"Lightning in Clay Pots","ref":"2Cor.13"}
{"id":"also1","date":"2026-06-21","title":"Pictures of Our Father","ref":"Matt.6.9","also":"Rom.8.14-Rom.8.16"}
{"id":"has1","date":"2026-09-06","title":"Growing in Grace","ref":"2Pet.1.5-2Pet.1.11"}
{"id":"msg1","date":"2025-11-02","title":"The Notes Title","ref":"Eph.4.1-Eph.5.14"}
{"id":"titled1","date":"2025-12-07","title":"Another","ref":"Ps.23.1-Ps.23.6"}
{"id":"noref1","date":"2026-01-04","title":"No Ref Here"}
{"id":"nowhere","date":"2024-01-07","title":"Nobody's","ref":"Gen.1.1-Gen.1.5"}
`

const passagesHeader = "series\tdate\ttitle\tspeaker\tpassage\trow\tfrom\n"

func notesRig(t *testing.T, passages string) (rows, recs, list string) {
	t.Helper()
	dir := t.TempDir()
	rows = filepath.Join(dir, "rows.jsonl")
	recs = filepath.Join(dir, "notes.jsonl")
	list = filepath.Join(dir, "passages.tsv")
	for p, b := range map[string]string{rows: notesRows, recs: notesRecords, list: passages} {
		if err := os.WriteFile(p, []byte(b), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return
}

func readFile(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// Acceptance 1: a notes record's title and passage reach a tracked file, the
// church's list, for a row that is only a generic title.
func TestReadNotesAddsTheRecordsTitleAndPassageToTheList(t *testing.T) {
	rows, recs, list := notesRig(t, passagesHeader+"\t2025-11-02\tFrom His Message\tPhil Morgan\tJohn 3:16\tmsg1\n")
	res, err := RunNotes(recs, list, rows)
	if err != nil {
		t.Fatal(err)
	}
	got := readFile(t, list)
	for _, want := range []string{
		"\t2025-10-19\tLightning in Clay Pots\t\t2 Corinthians 13\tFhOTBXf3S90\tnotes\n",
		"\t2026-06-21\tPictures of Our Father\t\tMatthew 6:9; Romans 8:14-16\talso1\tnotes\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("passages.tsv lacks %q:\n%s", want, got)
		}
	}
	if res.Added != 2 {
		t.Errorf("added %d, want 2: %+v", res.Added, res)
	}
	// A row with a passage already, a row his message has put a title on, a row
	// with a title of its own, a record with no passage and a record no row
	// holds are each left alone.
	for _, not := range []string{"has1", "Another", "No Ref Here", "Nobody", "The Notes Title"} {
		if strings.Contains(got, not) {
			t.Errorf("passages.tsv holds %q:\n%s", not, got)
		}
	}
	if !strings.Contains(got, "From His Message\tPhil Morgan\tJohn 3:16\tmsg1\n") {
		t.Errorf("his message's line was changed:\n%s", got)
	}
	// His message's words differ from the mail's: said, not replaced.
	if !strings.Contains(strings.Join(res.Problems, "\n"), "msg1") {
		t.Errorf("the difference with his message is not reported: %v", res.Problems)
	}
	// The ones not added are named with why.
	all := strings.Join(res.Problems, "\n")
	for _, id := range []string{"noref1", "nowhere", "titled1"} {
		if !strings.Contains(all, id) && !strings.Contains(all, "No Ref Here") && !strings.Contains(all, "Nobody") && !strings.Contains(all, "Another") {
			t.Errorf("%s is not named in %v", id, res.Problems)
		}
	}
}

func TestReadNotesTwiceAddsNothingTheSecondTime(t *testing.T) {
	rows, recs, list := notesRig(t, passagesHeader)
	if _, err := RunNotes(recs, list, rows); err != nil {
		t.Fatal(err)
	}
	first := readFile(t, list)
	res, err := RunNotes(recs, list, rows)
	if err != nil {
		t.Fatal(err)
	}
	if res.Added != 0 || readFile(t, list) != first {
		t.Errorf("second run added %d:\n%s", res.Added, readFile(t, list))
	}
}

// Words his message already put on the row are not replaced, and a difference is
// reported; the same words are quiet.
func TestReadNotesKeepsWhatHisMessageSaidAndReportsADifference(t *testing.T) {
	same := "\t2025-10-19\tLightning in Clay Pots\tPhil Morgan\t2 Corinthians 13\tFhOTBXf3S90\n"
	rows, recs, list := notesRig(t, passagesHeader+same)
	res, err := RunNotes(recs, list, rows)
	if err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, list); strings.Count(got, "Lightning in Clay Pots") != 1 {
		t.Errorf("a second line for the row:\n%s", got)
	}
	if strings.Contains(strings.Join(res.Problems, "\n"), "FhOTBXf3S90") {
		t.Errorf("the same words were reported: %v", res.Problems)
	}
	// A line by date (no row column) for the record's Sunday is also his.
	rows, recs, list = notesRig(t, passagesHeader+"\t2025-10-19\tOther Words\tPhil Morgan\t2 Corinthians 12\t\n")
	res, err = RunNotes(recs, list, rows)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(readFile(t, list), "Lightning in Clay Pots") || !strings.Contains(strings.Join(res.Problems, "\n"), "FhOTBXf3S90") {
		t.Errorf("list:\n%s\nproblems %v", readFile(t, list), res.Problems)
	}
}

func TestOSISTextWritesAPassageTheMergeReads(t *testing.T) {
	for in, want := range map[string]string{
		"2Cor.13":               "2 Corinthians 13",
		"2Cor.2.15":             "2 Corinthians 2:15",
		"Ps.23.1-Ps.23.6":       "Psalms 23:1-6",
		"Eph.4.1-Eph.5.14":      "Ephesians 4:1-5:14",
		"Rom.8.14-Rom.8.16":     "Romans 8:14-16",
		"1Sam.3-1Sam.4":         "1 Samuel 3-4",
		"Song.2.10-Song.2.13":   "Song of Solomon 2:10-13",
		"Luke.24.33-Luke.24.46": "Luke 24:33-46",
	} {
		got, err := osisText(in)
		if err != nil || got != want {
			t.Errorf("osisText(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "Nope.1.1", "Gen.1.1-Exod.1.1"} {
		if got, err := osisText(bad); err == nil {
			t.Errorf("osisText(%q) = %q, want an error", bad, got)
		}
	}
}

func TestReadNotesFlagPrintsItsLine(t *testing.T) {
	rows, recs, list := notesRig(t, passagesHeader)
	var out, errb strings.Builder
	code := cli([]string{"--read-notes", "--notes", recs, "--passages", list, "--rows", rows}, &out, &errb, deps{})
	if code != 0 || !strings.Contains(out.String(), "notes: 7 record(s), 3 added") {
		t.Errorf("exit %d\nout %s\nerr %s", code, out.String(), errb.String())
	}
}

// notesRecordsFile is where the sweep below finds the notes records: the
// git-ignored data/subsplash-notes.jsonl of the checkout, or the file
// SERMON_INDEX_NOTES names (a Builder's worktree has no data/ of its own).
func notesRecordsFile() string {
	if p := os.Getenv("SERMON_INDEX_NOTES"); p != "" {
		return p
	}
	return filepath.Join("..", "..", "data", "subsplash-notes.jsonl")
}

// Sweep (mw-qtu3zvz.59.28): every notes record's title, and its passage when
// the mail names one, is on its sermon's row in the tracked rows.jsonl. The
// row of L4mXkwT7tW4 stayed 'Brookfield First - Online Service | August 17,
// 2025' because its mail names no passage and --read-notes wrote nothing for
// a record with none. The records file is private, so a checkout without it
// skips this.
func TestEveryNotesRecordsTitleAndPassageIsOnItsRow(t *testing.T) {
	path := notesRecordsFile()
	if _, err := os.Stat(path); err != nil {
		t.Skipf("no notes records at %s (set SERMON_INDEX_NOTES)", path)
	}
	recs, err := ReadRecords(path)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := readRows(filepath.Join("..", "..", "sources", "brookfield-first", "rows.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) == 0 {
		t.Fatalf("%s holds no records", path)
	}
	for _, rec := range recs {
		at := recordRow(rows, rec)
		if at < 0 {
			t.Errorf("%s %q: no one row holds this record", rec.ID, rec.Title)
			continue
		}
		row := rows[at]
		// The church's own list may spell a title's case its own way ("How to
		// Spot a Fable"); the words are the same.
		if !strings.EqualFold(row.Title, rec.Title) {
			t.Errorf("%s: the row's title is %q, the mail's is %q", rec.ID, row.Title, rec.Title)
		}
		if rec.Ref == "" {
			continue
		}
		if row.Ref == nil || *row.Ref == "" {
			t.Errorf("%s %q: the row has no passage, the mail says %s", rec.ID, rec.Title, rec.Ref)
			continue
		}
		a, aerr := passage.Parse(*row.Ref)
		b, berr := passage.Parse(rec.Ref)
		if aerr != nil || berr != nil {
			t.Errorf("%s: row %q or mail %q passage unreadable: %v %v", rec.ID, *row.Ref, rec.Ref, aerr, berr)
		} else if a.OSIS != b.OSIS && (row.Series == nil || !*row.Series) {
			t.Errorf("%s: the row's passage is %s, the mail's is %s", rec.ID, a.OSIS, b.OSIS)
		}
	}
}

// A mail that names no passage (L4mXkwT7tW4, 'From Good Intentions to Godly
// Impact', whose notes PDF names 2 Corinthians 8:10-15) has its title and
// passage put on the list by hand; --read-notes then finds the line already
// holds the mail's title and has nothing to report. With no such line the
// record is still reported as having no passage.
func TestReadNotesOfAMailWithNoPassageFindsTheLineAlreadyHoldingItsTitle(t *testing.T) {
	rows, recs, list := notesRig(t, passagesHeader+"\t2026-01-04\tNo Ref Here\t\t2 Corinthians 8:10-15\tnoref1\tnotes\n")
	res, err := RunNotes(recs, list, rows)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range res.Problems {
		if strings.Contains(p, "noref1") {
			t.Errorf("reported though the list holds its title: %v", res.Problems)
		}
	}
	rows, recs, list = notesRig(t, passagesHeader)
	res, err = RunNotes(recs, list, rows)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(res.Problems, "\n"), "noref1") {
		t.Errorf("a record with no passage and no line is not reported: %v", res.Problems)
	}
}

// The text pdftotext gives of the first page of two real notes PDFs
// (L4mXkwT7tW4 and P-CduhO5oPo): a notes PDF prints its speaker under the
// title, and pdftotext breaks words in the middle ("Pas tor").
const headerWithPastor = "\"From Good\nIntentions to\nGodly Impact\"\n2 Corinthians 8:10-15\nPas tor Phil Morgan\n\nPart of 2 Corinthians Onward We Stumble\nAugus t 17, 2025\n\n\n2 Corinthians 8:10-15\n\n1. Our Accountability ____ (v. 10-11)\n"

func TestNotesSpeakerReadsThePastorOfAHeaderAndNothingElse(t *testing.T) {
	for text, want := range map[string]string{
		headerWithPastor: "Phil Morgan",
		"\"The Teachable\nHeart\" — Luke 8:1-\n15\nPastor Sam Morgan\nPart of Standalone Sermons\nDecember 28, 2025\n": "Sam Morgan",
		// No speaker in the header: nothing, and the body is never read.
		"\"Gospel Simplicity\"\n— 2 Corinthians\n11:1-15\nPart of 2 Corinthians Onward We Stumble\nSeptember 21, 2025\n\n2 Corinthians 11:1-15\n": "",
		"\"Title\"\nPart of X\nJuly 21, 2024\n\nAsk Pastor Phil Morgan about it\n\nPastor Phil Morgan\n":                                          "",
		"": "",
	} {
		if got := notesSpeaker(text); got != want {
			t.Errorf("notesSpeaker(%q) = %q, want %q", text, got, want)
		}
	}
}

// A line --read-notes writes takes its speaker from the row's notes PDF.
func TestReadNotesWritesTheSpeakerTheNotesPDFNames(t *testing.T) {
	rows, recs, list := notesRig(t, passagesHeader)
	pdfs := t.TempDir()
	text := func(path string) (string, error) {
		switch filepath.Base(path) {
		case "FhOTBXf3S90.pdf":
			return headerWithPastor, nil
		case "also1.pdf":
			return "\"x\"\nPart of Y\n", nil
		}
		return "", os.ErrNotExist
	}
	res, err := RunNotesWithSpeakers(recs, list, rows, pdfs, text)
	if err != nil {
		t.Fatal(err)
	}
	if res.Added == 0 {
		t.Fatalf("%+v", res)
	}
	got := readFile(t, list)
	if !strings.Contains(got, "\t2025-10-19\tLightning in Clay Pots\tPhil Morgan\t2 Corinthians 13\tFhOTBXf3S90\tnotes\n") {
		t.Errorf("no speaker on FhOTBXf3S90's line:\n%s", got)
	}
	if !strings.Contains(got, "\t2026-06-21\tPictures of Our Father\t\tMatthew 6:9; Romans 8:14-16\talso1\tnotes\n") {
		t.Errorf("also1 has a speaker or no line:\n%s", got)
	}
}

// Acceptance 1 (mw-qtu3zvz.59.29): a mail that names no passage (noref1, as
// L4mXkwT7tW4 was) has the passage its notes PDF's text holds written to the
// list with the mail's title, and 'notes' as its source.
func TestReadNotesOfAMailWithNoPassageReadsItFromTheNotesPDF(t *testing.T) {
	rows, recs, list := notesRig(t, passagesHeader)
	pdfs := t.TempDir()
	text := func(path string) (string, error) {
		if filepath.Base(path) == "noref1.pdf" {
			return headerWithPastor, nil
		}
		return "", os.ErrNotExist
	}
	res, err := RunNotesWithSpeakers(recs, list, rows, pdfs, text)
	if err != nil {
		t.Fatal(err)
	}
	got := readFile(t, list)
	want := "\t2026-01-04\tNo Ref Here\tPhil Morgan\t2 Corinthians 8:10-15\tnoref1\tnotes\n"
	if !strings.Contains(got, want) {
		t.Errorf("passages.tsv lacks %q:\n%s", want, got)
	}
	if strings.Contains(strings.Join(res.Problems, "\n"), "noref1") {
		t.Errorf("noref1 is reported though its passage was read: %v", res.Problems)
	}
	// A second run finds the line and adds nothing.
	first := got
	res, err = RunNotesWithSpeakers(recs, list, rows, pdfs, text)
	if err != nil {
		t.Fatal(err)
	}
	if readFile(t, list) != first || strings.Contains(strings.Join(res.Problems, "\n"), "noref1") {
		t.Errorf("second run changed the list or reported: %v\n%s", res.Problems, readFile(t, list))
	}
}

// Acceptance 2: neither the mail nor the PDF text holds a passage; the row is
// named in the report, with which of the two was missing.
func TestReadNotesOfAMailWithNoPassageNamesTheRowWhenThePDFHoldsNone(t *testing.T) {
	for name, text := range map[string]func(string) (string, error){
		"text with no passage": func(string) (string, error) {
			return "\"No Ref Here\"\nPart of Standalone Sermons\nJanuary 4, 2026\n\n1. Our ____ (v. 10-11)\n", nil
		},
		"no PDF": func(string) (string, error) { return "", os.ErrNotExist },
	} {
		rows, recs, list := notesRig(t, passagesHeader)
		res, err := RunNotesWithSpeakers(recs, list, rows, t.TempDir(), text)
		if err != nil {
			t.Fatal(err)
		}
		var line string
		for _, p := range res.Problems {
			if strings.Contains(p, "noref1") {
				line = p
			}
		}
		if line == "" || !strings.Contains(line, "no passage") || !strings.Contains(line, "notes PDF") {
			t.Errorf("%s: the row is not named with why: %v", name, res.Problems)
		}
		if strings.Contains(readFile(t, list), "No Ref Here") {
			t.Errorf("%s: a line was written:\n%s", name, readFile(t, list))
		}
	}
}

// The CLI prints that report line.
func TestReadNotesFlagReportsARecordWithNoPassageAnywhere(t *testing.T) {
	rows, recs, list := notesRig(t, passagesHeader)
	var out, errb strings.Builder
	code := cli([]string{"--read-notes", "--notes", recs, "--passages", list, "--rows", rows, "--notes-pdfs", "none"}, &out, &errb, deps{})
	if code != 0 || !strings.Contains(out.String(), "notes: not added: noref1") {
		t.Errorf("exit %d\nout %s\nerr %s", code, out.String(), errb.String())
	}
}

func TestPDFPassageReadsTheHeadersPassageAndNotABodyLine(t *testing.T) {
	for text, want := range map[string]string{
		headerWithPastor: "2Cor.8.10-2Cor.8.15",
		// The header wraps the passage and drops its dash; the body's heading is whole.
		"\"The Teachable\nHeart\" — Luke 8:115\nPas tor Phil Morgan\nPart of Standalone Sermons\nDecember 28, 2025\n\nLuke 8:1-15\n1. A ____ Heart\n":              "Luke.8.1-Luke.8.15",
		"\"Dear 21st\nCentury Church\" —\n1 Corinthians 1:13\nPart of 1 Corinthians Following\nSeptember 9, 2023\n\n1 Corinthians 1:1-3\n1. The ____ of Corinth\n": "1Cor.1.1-1Cor.1.3",
		// The header keeps its dash: taken as it is, with an abbreviated book.
		"\"The Cheerful\nGiver\" — 2 Cor.\n9:1-15\nPart of 2 Corinthians\nAugus t 31, 2025\n\n2 Corinthians 9\n1. God Loves ____\n": "2Cor.9.1-2Cor.9.15",
		// The header's passage lost a dash and no body line confirms it: none,
		// and the body's sub-passage and quoted verse are not taken for it.
		"\"Following Jesus\ninto 2024\" — Luke\n6:40\nPart of Standalone Sermons\nDecember 31, 2023\n\n\u201cA disciple is not above his teacher.\u201d (Luke 6:40)\nLuke 6:41\n": "",
		"\"Title\"\nPart of Y\nJanuary 4, 2026\n\nLuke 8:1-15\n": "",
		"": "",
	} {
		got, ok := pdfPassage(text)
		if want == "" {
			if ok {
				t.Errorf("pdfPassage(%q) = %q, want none", text, got)
			}
			continue
		}
		if !ok || got != want {
			t.Errorf("pdfPassage(%q) = %q, %v; want %q", text, got, ok, want)
		}
	}
}
