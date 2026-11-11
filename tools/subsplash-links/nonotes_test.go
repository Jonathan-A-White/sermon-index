package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const (
	govKey   = "02aaaa00000000000000000000000000000000000000000000000000000000000001"
	otherKey = "02bbbb00000000000000000000000000000000000000000000000000000000000002"
)

func TestParseNoNotesReadsTheEasterMessage(t *testing.T) {
	got, bad, cand := ParseNoNotes("direct:t1", "The Running Man\n\nEaster Sunday 2026\n\nApr 5, 2026 Phil Morgan Luke 24:33-46\n\n(No notes)")
	if !cand || len(bad) != 0 {
		t.Fatalf("candidate %v, unread %v", cand, bad)
	}
	want := []Sermon{{Series: "Easter Sunday 2026", Date: "2026-04-05", Title: "The Running Man", Speaker: "Phil Morgan", Passage: "Luke 24:33-46", Tx: "direct:t1"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

// Each of the shapes he sent (a fixture each), by the sermons they hold.
func TestParseNoNotesTakesTheShapesHeSent(t *testing.T) {
	cases := []struct {
		name, text string
		want       []Sermon
	}{
		{"no series, No notes without brackets",
			"Will He Find Faith?\n\nNov 16, 2025 Phil Morgan Luke 18:1-8\n\nNo notes",
			[]Sermon{{"", "2025-11-16", "Will He Find Faith?", "Phil Morgan", "Luke 18:1-8", "t"}}},
		{"no preacher, last line only begins No notes",
			"Putting Hope Within Reach\n\n2025 Missions Convention\n\nNov 9, 2025 2 Kings 5\n\nNo notes, no pastor attribution (still Brookfield First)",
			[]Sermon{{"2025 Missions Convention", "2025-11-09", "Putting Hope Within Reach", "", "2 Kings 5", "t"}}},
		{"Psalms 103",
			"The Grace of Thanksgiving\n\nNov 24, 2024 Sam Morgan Psalms 103\n\nNo notes",
			[]Sermon{{"", "2024-11-24", "The Grace of Thanksgiving", "Sam Morgan", "Psalms 103", "t"}}},
		{"the date alone",
			"Run the Race\n\n2024 Missions Convention\n\nNov 10, 2024\n\nNo notes",
			[]Sermon{{"2024 Missions Convention", "2024-11-10", "Run the Race", "", "", "t"}}},
		{"date and preacher each alone, no passage",
			"Four Things The Bible Says About Persecution and One Thing It Does Not Say\n\nInternational Day of Prayer for Persecuted Christians 2024\n\nNov 3, 2024\n\nTodd Nettleton\n\nNo notes",
			[]Sermon{{"International Day of Prayer for Persecuted Christians 2024", "2024-11-03", "Four Things The Bible Says About Persecution and One Thing It Does Not Say", "Todd Nettleton", "", "t"}}},
		{"a trailing dot after the date, preacher and passage on the next line",
			"The Sound of Rain\n\nJul 14, 2024.\n\nPhil Morgan 1 Kings 18:41-46\n\nNo notes",
			[]Sermon{{"", "2024-07-14", "The Sound of Rain", "Phil Morgan", "1 Kings 18:41-46", "t"}}},
		{"nine sermons in one message",
			"A Colony of Heaven in a Country of Death\n\nSep 3, 2023.\n\nSam Morgan Philippians 1:27-30\n\nGideon's Pentecost\n\nMay 28, 2023 Phil Morgan Judges 6\n\nEaster Sunday 2023\n\nApr 9, 2023 Phil Morgan Luke 24:13-35\n\nThe Cross of Christ\n\nGood Friday 2023\n\nApr 7, 2023\n\nSam Morgan\n\nMatthew 27:11-50\n\nThe Joy of Abiding\n\nJan 15, 2023 Sam Morgan John 15:1-11\n\nThe Locksmith and the Key\n\nJan 8, 2023 Phil Morgan Psalms 65\n\nWhen the Battle Gets Tough\n\nJan 1, 2023\n\nPhil Morgan •\n\n1\n\nTimothy 1:18-20\n\nHe Is Worthy\n\nOct 23, 2022\n\nSelwyn Bodley Matthew 28:19-20\n\nCome Away\n\nOct 9, 2022 Phil Morgan Song of Songs\n\n2:10-13\n\nNo notes. \n\nStopping for now.\n\nThis is where I'm at.",
			[]Sermon{
				{"", "2023-09-03", "A Colony of Heaven in a Country of Death", "Sam Morgan", "Philippians 1:27-30", "t"},
				{"", "2023-05-28", "Gideon's Pentecost", "Phil Morgan", "Judges 6", "t"},
				{"", "2023-04-09", "Easter Sunday 2023", "Phil Morgan", "Luke 24:13-35", "t"},
				{"Good Friday 2023", "2023-04-07", "The Cross of Christ", "Sam Morgan", "Matthew 27:11-50", "t"},
				{"", "2023-01-15", "The Joy of Abiding", "Sam Morgan", "John 15:1-11", "t"},
				{"", "2023-01-08", "The Locksmith and the Key", "Phil Morgan", "Psalms 65", "t"},
				{"", "2023-01-01", "When the Battle Gets Tough", "Phil Morgan", "1 Timothy 1:18-20", "t"},
				{"", "2022-10-23", "He Is Worthy", "Selwyn Bodley", "Matthew 28:19-20", "t"},
				{"", "2022-10-09", "Come Away", "Phil Morgan", "Song of Songs 2:10-13", "t"},
			}},
	}
	for _, c := range cases {
		got, bad, cand := ParseNoNotes("t", c.text)
		if !cand || len(bad) != 0 {
			t.Errorf("%s: candidate %v, unread %v", c.name, cand, bad)
			continue
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s:\n got %+v\nwant %+v", c.name, got, c.want)
		}
	}
}

func TestParseNoNotesIgnoresAMessageWithoutNoNotes(t *testing.T) {
	got, bad, cand := ParseNoNotes("t", "The Running Man\n\nApr 5, 2026 Phil Morgan Luke 24:33-46\n\nsee you Sunday")
	if cand || len(got) != 0 || len(bad) != 0 {
		t.Errorf("candidate %v, read %+v, unread %+v", cand, got, bad)
	}
}

func TestParseNoNotesNamesWhatItCannotRead(t *testing.T) {
	for name, text := range map[string]string{
		"no date":                    "The Running Man\n\nPhil Morgan Luke 24:33-46\n\n(No notes)",
		"too many title rows":        "A\n\nB\n\nC\n\nApr 5, 2026 Phil Morgan Luke 24:33-46\n\n(No notes)",
		"no title":                   "Apr 5, 2026 Phil Morgan Luke 24:33-46\n\n(No notes)",
		"unsure where preacher ends": "A Title\n\nApr 5, 2026\n\nSam Morgan\n\nAnother Title\n\nApr 12, 2026 Phil Morgan Luke 24:33-46\n\n(No notes)",
	} {
		got, bad, cand := ParseNoNotes("direct:t9", text)
		if !cand || len(bad) == 0 {
			t.Errorf("%s: candidate %v, read %+v, unread %+v", name, cand, got, bad)
			continue
		}
		if bad[0].Tx != "direct:t9" || bad[0].Why == "" {
			t.Errorf("%s: unread %+v", name, bad[0])
		}
		if name == "unsure where preacher ends" && (len(got) != 0 || len(bad) != 2) {
			t.Errorf("%s: both entries are unread (the preacher and the next title cannot be told apart), got %+v and %+v", name, got, bad)
		}
	}
}

// --- the run over an inbox ---

const nnHeader = "series\tdate\ttitle\tspeaker\tpassage\trow\n"

type nnRig struct {
	t                     *testing.T
	inbox, rows, pass, nn string
}

func newNNRig(t *testing.T) *nnRig {
	t.Helper()
	d := t.TempDir()
	r := &nnRig{t: t, inbox: filepath.Join(d, "inbox"), rows: filepath.Join(d, "rows.jsonl"), pass: filepath.Join(d, "passages.tsv"), nn: filepath.Join(d, "no-notes.tsv")}
	if err := os.MkdirAll(r.inbox, 0o755); err != nil {
		t.Fatal(err)
	}
	rows := `{"id":"yt1","title":"Brookfield First - Online Service | Easter 2031","url":"https://www.youtube.com/watch?v=yt1","kind":"link","date":"2031-04-06"}
{"id":"yt2","title":"Brookfield First - Online Service | April 13, 2031","url":"https://www.youtube.com/watch?v=yt2","kind":"link","date":"2031-04-13"}
{"id":"yt3","title":"Brookfield First - Online Service | April 20, 2031","url":"https://www.youtube.com/watch?v=yt3","kind":"link","date":"2031-04-20"}
{"id":"yt4","title":"Brookfield First - Online Service | April 21, 2031","url":"https://www.youtube.com/watch?v=yt4","kind":"link","date":"2031-04-21"}
`
	must(t, os.WriteFile(r.rows, []byte(rows), 0o644))
	must(t, os.WriteFile(r.pass, []byte(nnHeader+"# a comment\nOld Series\t2030-01-06\tAn Old One\tSam Morgan\tJohn 3:1-8\t\n"), 0o644))
	return r
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func (r *nnRig) message(name, tx, from string, signed bool, ts, text string) {
	r.t.Helper()
	b, _ := json.Marshal(map[string]any{"seq": 1, "txid": tx, "class": "message", "from": from, "verified": signed, "signer_checked": signed, "ts": ts, "thread": "general", "text": text})
	must(r.t, os.WriteFile(filepath.Join(r.inbox, name+".message.json"), b, 0o644))
}

func (r *nnRig) run() (*NoNotesResult, error) {
	return RunNoNotes(r.inbox, govKey, r.pass, r.nn, r.rows)
}

func (r *nnRig) read(p string) string {
	b, _ := os.ReadFile(p)
	return string(b)
}

func TestRunNoNotesWritesTheListOnceAndMatchesThePublishDay(t *testing.T) {
	r := newNNRig(t)
	r.message("a", "direct:tx-easter", govKey, true, "2031-04-07T10:00:00Z", "The Running Man\n\nEaster Sunday 2031\n\nApr 5, 2031 Phil Morgan Luke 24:33-46\n\n(No notes)")
	res, err := r.run()
	if err != nil {
		t.Fatal(err)
	}
	if res.Added != 1 || len(res.Problems) != 0 {
		t.Fatalf("%+v", res)
	}
	wantPass := nnHeader + "# a comment\nOld Series\t2030-01-06\tAn Old One\tSam Morgan\tJohn 3:1-8\t\nEaster Sunday 2031\t2031-04-05\tThe Running Man\tPhil Morgan\tLuke 24:33-46\tyt1\n"
	if got := r.read(r.pass); got != wantPass {
		t.Errorf("passages.tsv:\n%q\nwant\n%q", got, wantPass)
	}
	wantNN := "date\trow\ttxid\n2031-04-05\tyt1\tdirect:tx-easter\n"
	if got := r.read(r.nn); got != wantNN {
		t.Errorf("no-notes.tsv:\n%q\nwant\n%q", got, wantNN)
	}
	// A second run adds nothing and writes nothing.
	res, err = r.run()
	if err != nil {
		t.Fatal(err)
	}
	if res.Added != 0 || len(res.Problems) != 0 || r.read(r.pass) != wantPass || r.read(r.nn) != wantNN {
		t.Errorf("second run: %+v", res)
	}
}

func TestRunNoNotesWithNoSeriesLineLeavesTheSeriesEmpty(t *testing.T) {
	r := newNNRig(t)
	r.message("a", "direct:tx", govKey, true, "2031-04-14T10:00:00Z", "Will He Find Faith?\n\nApr 13, 2031 Phil Morgan Luke 18:1-8\n\nNo notes")
	if _, err := r.run(); err != nil {
		t.Fatal(err)
	}
	if got := r.read(r.pass); !strings.HasSuffix(got, "\t2031-04-13\tWill He Find Faith?\tPhil Morgan\tLuke 18:1-8\tyt2\n") || !strings.Contains(got, "\n\t2031-04-13\t") {
		t.Errorf("passages.tsv:\n%s", got)
	}
}

func TestRunNoNotesIgnoresAnUnsignedAndAnotherKeysMessage(t *testing.T) {
	r := newNNRig(t)
	text := "The Running Man\n\nApr 5, 2031 Phil Morgan Luke 24:33-46\n\n(No notes)"
	r.message("a", "direct:unsigned", govKey, false, "2031-04-07T10:00:00Z", text)
	r.message("b", "direct:other", otherKey, true, "2031-04-07T10:01:00Z", text)
	r.message("c", "direct:hello", govKey, true, "2031-04-07T10:02:00Z", "no news here")
	res, err := r.run()
	if err != nil {
		t.Fatal(err)
	}
	if res.Added != 0 || len(res.Problems) != 0 {
		t.Errorf("%+v", res)
	}
	if _, err := os.Stat(r.nn); err == nil {
		t.Error("no-notes.tsv was made for messages that were ignored")
	}
	if strings.Contains(r.read(r.pass), "Running Man") {
		t.Error("passages.tsv was written")
	}
}

func TestRunNoNotesReportsWhatItCannotPlaceAndWritesNothingForIt(t *testing.T) {
	r := newNNRig(t)
	// An unparseable passage: "Luke 99:1" is refused by the book table.
	r.message("a", "direct:bad-passage", govKey, true, "2031-04-07T10:00:00Z", "A Title\n\nApr 13, 2031 Phil Morgan Luke 99:1\n\nNo notes")
	// A date within a day of no row.
	r.message("b", "direct:no-row", govKey, true, "2031-04-07T10:01:00Z", "Another\n\nMay 11, 2031 Phil Morgan Luke 1:1-4\n\nNo notes")
	// A date within a day of two rows (Apr 20 and Apr 21 both).
	r.message("c", "direct:two-rows", govKey, true, "2031-04-07T10:02:00Z", "Third\n\nApr 20, 2031 Phil Morgan Luke 1:1-4\n\nNo notes")
	res, err := r.run()
	if err != nil {
		t.Fatal(err)
	}
	if res.Added != 0 || len(res.Problems) != 3 {
		t.Fatalf("%+v", res)
	}
	for _, tx := range []string{"direct:bad-passage", "direct:no-row", "direct:two-rows"} {
		found := false
		for _, p := range res.Problems {
			found = found || strings.Contains(p, tx)
		}
		if !found {
			t.Errorf("no problem names %s: %v", tx, res.Problems)
		}
	}
	if got := r.read(r.pass); strings.Contains(got, "A Title") || strings.Contains(got, "Another") || strings.Contains(got, "Third") {
		t.Errorf("passages.tsv was written:\n%s", got)
	}
	if _, err := os.Stat(r.nn); err == nil {
		t.Error("no-notes.tsv was written")
	}
}

func TestRunNoNotesNeverReplacesALineAlreadyThere(t *testing.T) {
	r := newNNRig(t)
	r.message("a", "direct:one", govKey, true, "2031-04-07T10:00:00Z", "The Running Man\n\nApr 5, 2031 Phil Morgan Luke 24:33-46\n\nNo notes")
	if _, err := r.run(); err != nil {
		t.Fatal(err)
	}
	before := r.read(r.pass)
	// The same date, other words.
	r.message("b", "direct:two", govKey, true, "2031-04-08T10:00:00Z", "The Running Woman\n\nApr 5, 2031 Phil Morgan Luke 24:33-46\n\nNo notes")
	res, err := r.run()
	if err != nil {
		t.Fatal(err)
	}
	if res.Added != 0 || len(res.Problems) != 1 || !strings.Contains(res.Problems[0], "direct:two") {
		t.Errorf("%+v", res)
	}
	if r.read(r.pass) != before {
		t.Error("passages.tsv changed")
	}
}

func TestRunNoNotesRefusesTwoSermonsOnOneRow(t *testing.T) {
	r := newNNRig(t)
	// Apr 12 and Apr 13 are both within a day of the one row dated Apr 13.
	r.message("a", "direct:one", govKey, true, "2031-04-07T10:00:00Z", "One\n\nApr 12, 2031 Phil Morgan Luke 1:1-4\n\nNo notes")
	r.message("b", "direct:two", govKey, true, "2031-04-07T10:01:00Z", "Two\n\nApr 13, 2031 Phil Morgan Luke 1:5-9\n\nNo notes")
	res, err := r.run()
	if err != nil {
		t.Fatal(err)
	}
	if res.Added != 0 || len(res.Problems) != 2 {
		t.Errorf("%+v", res)
	}
}

func TestRunNoNotesNeedsTheRowColumn(t *testing.T) {
	r := newNNRig(t)
	must(t, os.WriteFile(r.pass, []byte("series\tdate\ttitle\tspeaker\tpassage\n"), 0o644))
	r.message("a", "direct:one", govKey, true, "2031-04-07T10:00:00Z", "One\n\nApr 13, 2031 Phil Morgan Luke 1:5-9\n\nNo notes")
	if _, err := r.run(); err == nil || !strings.Contains(err.Error(), "row") {
		t.Errorf("err %v", err)
	}
}

func TestReadNoNotesGivesTheRowIDs(t *testing.T) {
	p := filepath.Join(t.TempDir(), "no-notes.tsv")
	must(t, os.WriteFile(p, []byte("date\trow\ttxid\n2031-04-05\tyt1\tdirect:x\n"), 0o644))
	got, err := ReadNoNotes(p)
	if err != nil || !got["yt1"] || len(got) != 1 {
		t.Errorf("%v %v", got, err)
	}
	got, err = ReadNoNotes(filepath.Join(t.TempDir(), "none.tsv"))
	if err != nil || len(got) != 0 {
		t.Errorf("a missing file holds none: %v %v", got, err)
	}
}
