package main

import (
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
