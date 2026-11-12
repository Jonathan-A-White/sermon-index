package main

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

func fixture(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "notes-invented.txt"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// Acceptance 1: a sermon-notes alert yields title, passage and date.
func TestReadAlertsYieldsTitlePassageDate(t *testing.T) {
	notes, _, err := ReadAlerts(strings.NewReader(fixture(t)))
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) < 2 {
		t.Fatalf("notes = %+v", notes)
	}
	want := Note{AlertID: "00000000000000a1", Title: "The Garden Gate", Ref: "Gen.2.8-Gen.2.17", Reference: "Genesis 2:8-17", Date: "2031-03-02"}
	if notes[0] != want {
		t.Errorf("first note = %+v, want %+v", notes[0], want)
	}
	// The date is the date the alert's own offset prints (midnight Eastern),
	// not the UTC date.
	if notes[1].Date != "2031-03-08" || notes[1].Ref != "Ps.23" || notes[1].Title != "Walking the Hill" {
		t.Errorf("second note = %+v", notes[1])
	}
}

// Acceptance 2: an alert without a valid reference is reported and dropped.
func TestReadAlertsDropsABadReference(t *testing.T) {
	notes, problems, err := ReadAlerts(strings.NewReader(fixture(t)))
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range notes {
		if n.AlertID == "00000000000000a4" {
			t.Errorf("an alert with a bad reference was kept: %+v", n)
		}
	}
	found := false
	for _, p := range problems {
		if p.AlertID == "00000000000000a4" && strings.Contains(p.Why, "Hezekiah") {
			found = true
		}
	}
	if !found {
		t.Errorf("no problem names the bad reference: %+v", problems)
	}
}

func TestReadAlertsRefusesMissingAndBadFields(t *testing.T) {
	cases := map[string]string{
		"past chapter": `{"id":"x2","kind":"sermon-notes","title":"T","reference":"Romans 99:1","when":"2031-03-02T00:00:00-05:00"}`,
		"no title":     `{"id":"x3","kind":"sermon-notes","reference":"Romans 5:1","when":"2031-03-02T00:00:00-05:00"}`,
		"no date":      `{"id":"x4","kind":"sermon-notes","title":"T","reference":"Romans 5:1"}`,
		"bad date":     `{"id":"x5","kind":"sermon-notes","title":"T","reference":"Romans 5:1","when":"last Sunday"}`,
	}
	for name, a := range cases {
		notes, problems, err := ReadAlerts(strings.NewReader(`{"alerts":[` + a + `]}`))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if len(notes) != 0 || len(problems) != 1 {
			t.Errorf("%s: notes %+v problems %+v", name, notes, problems)
		}
	}
}

// Only a sermon-notes alert is the reader's; any other kind is left alone
// (not read, not reported as a fault).
func TestReadAlertsLeavesOtherKindsAlone(t *testing.T) {
	notes, problems, err := ReadAlerts(strings.NewReader(fixture(t)))
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range notes {
		if n.AlertID == "00000000000000a3" {
			t.Errorf("a charge alert was read as a note: %+v", n)
		}
	}
	for _, p := range problems {
		if p.AlertID == "00000000000000a3" {
			t.Errorf("a charge alert was reported: %+v", p)
		}
	}
}

// Acceptance 5: the reader never receives or stores an attachment. An alert
// that carries a field the reader does not know (an attachment, a body) is
// refused whole, so its content is never held.
func TestReadAlertsNeverTakesAnAttachment(t *testing.T) {
	notes, problems, err := ReadAlerts(strings.NewReader(fixture(t)))
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range notes {
		if n.AlertID == "00000000000000a5" {
			t.Errorf("an alert with an attachment field was kept: %+v", n)
		}
	}
	found := false
	for _, p := range problems {
		if p.AlertID == "00000000000000a5" {
			found = true
			if strings.Contains(p.Why, "JVBERi0x") {
				t.Errorf("the report repeats the attachment: %q", p.Why)
			}
		}
	}
	if !found {
		t.Errorf("the alert with an attachment was not reported: %+v", problems)
	}
	// The types hold no place for one.
	for _, typ := range []any{Alert{}, Note{}, Record{}} {
		rt := reflect.TypeOf(typ)
		for i := 0; i < rt.NumField(); i++ {
			name := strings.ToLower(rt.Field(i).Name)
			for _, bad := range []string{"attach", "body", "pdf", "content"} {
				if strings.Contains(name, bad) {
					t.Errorf("%s has a field named like %q: %s", rt.Name(), bad, name)
				}
			}
		}
	}
}

func TestReadAlertsRefusesNonJSON(t *testing.T) {
	if _, _, err := ReadAlerts(strings.NewReader("not json")); err == nil {
		t.Error("no error for a reply that is not JSON")
	}
}

func TestReferenceToOSIS(t *testing.T) {
	cases := map[string]string{
		"2 Peter 3:10-18":     "2Pet.3.10-2Pet.3.18",
		"2 Peter 3:10–18":     "2Pet.3.10-2Pet.3.18",
		"Romans 5:8":          "Rom.5.8",
		"Psalm 23":            "Ps.23",
		"Psalms 23":           "Ps.23",
		"Genesis 2:8-17":      "Gen.2.8-Gen.2.17",
		"John 3:16-4:3":       "John.3.16-John.4.3",
		"Isaiah 40-41":        "Isa.40-Isa.41",
		"Ephesians 4-5:14":    "Eph.4.1-Eph.5.14",
		"Ephesians 4:1-5:14":  "Eph.4.1-Eph.5.14",
		"Ephesians 4-5":       "Eph.4-Eph.5",
		"Song of Songs 2:1-7": "Song.2.1-Song.2.7",
		"  1 John 1:5 - 2:2 ": "1John.1.5-1John.2.2",
		"2 Peter 2:4-10a":     "2Pet.2.4-2Pet.2.10",
		"1 Peter 4:12b-19":    "1Pet.4.12-1Pet.4.19",
		"2 Peter 2:10a":       "2Pet.2.10",
		"John 3:16b-4:3a":     "John.3.16-John.4.3",
		"2 Peter 2:4-10A":     "2Pet.2.4-2Pet.2.10",
	}
	for in, want := range cases {
		got, err := ReferenceToOSIS(in)
		if err != nil {
			t.Errorf("%q: %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("%q = %q, want %q", in, got, want)
		}
	}
	for _, bad := range []string{"", "Hezekiah 3:1", "Romans 99:1", "Romans", "3:10-18", "Romans 5:8 and more", "Romans 5:8; John 3:16",
		"Romans 5:8a; John 3:16", "Romans 5:8ab", "Romans 5:8-10a; John 3:16", "Isaiah 40a", "Isaiah 40-41a", "Ephesians 4-5a:14", "Romans 5:8 a", "Romans 5:8-"} {
		if got, err := ReferenceToOSIS(bad); err == nil {
			t.Errorf("%q accepted as %q", bad, got)
		}
	}
}

// Acceptance 7: no real email address is written in the tool or its fixture.
func TestNoRealEmailAddress(t *testing.T) {
	re := regexp.MustCompile(`[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}`)
	files, err := filepath.Glob("*")
	if err != nil {
		t.Fatal(err)
	}
	more, _ := filepath.Glob("testdata/*")
	for _, f := range append(files, more...) {
		b, err := os.ReadFile(f)
		if err != nil {
			continue // a directory
		}
		for _, m := range re.FindAllString(string(b), -1) {
			if !strings.HasSuffix(m, "@example.com") && !strings.HasSuffix(m, "@example.org") {
				t.Errorf("%s holds an address that is not an example one: %q", f, m)
			}
		}
	}
}

// A sermon-notes alert with notes carries notes_sha256 and notes_size, which
// the reader takes; one without carries neither. Half a pair, a malformed
// hash or a size past Postern's limit is dropped and reported.
func TestReadAlertsTakesTheNotesFields(t *testing.T) {
	sha := strings.Repeat("ab", 32)
	head := `{"id":"x1","kind":"sermon-notes","title":"T","reference":"Romans 5:1","when":"2031-03-02T00:00:00-05:00"`
	notes, problems, err := ReadAlerts(strings.NewReader(`{"alerts":[` + head + `,"notes_sha256":"` + sha + `","notes_size":1234}]}`))
	if err != nil || len(problems) != 0 || len(notes) != 1 {
		t.Fatalf("notes %+v problems %+v err %v", notes, problems, err)
	}
	if notes[0].NotesSHA256 != sha || notes[0].NotesSize != 1234 {
		t.Errorf("note = %+v", notes[0])
	}
	bad := map[string]string{
		"sha without size": `,"notes_sha256":"` + sha + `"`,
		"size without sha": `,"notes_size":10`,
		"short sha":        `,"notes_sha256":"abcd","notes_size":10`,
		"upper-case sha":   `,"notes_sha256":"` + strings.ToUpper(sha) + `","notes_size":10`,
		"zero size":        `,"notes_sha256":"` + sha + `","notes_size":0`,
		"too large":        `,"notes_sha256":"` + sha + `","notes_size":5242881`,
	}
	for name, extra := range bad {
		notes, problems, err := ReadAlerts(strings.NewReader(`{"alerts":[` + head + extra + `}]}`))
		if err != nil || len(notes) != 0 || len(problems) != 1 || problems[0].AlertID != "x1" {
			t.Errorf("%s: notes %+v problems %+v err %v", name, notes, problems, err)
		}
	}
}

// An alert may carry an optional "speaker" (the preacher the mail names). The
// reader takes it when it is there, keeps the other fields as they were, and
// still refuses a field it does not know.
func TestReadAlertsTakesAnOptionalSpeaker(t *testing.T) {
	head := `{"id":"x1","kind":"sermon-notes","title":"T","reference":"Romans 5:1","when":"2031-03-02T00:00:00-05:00"`
	notes, problems, err := ReadAlerts(strings.NewReader(`{"alerts":[` + head + `,"speaker":"  Phil Morgan "}]}`))
	if err != nil || len(problems) != 0 || len(notes) != 1 {
		t.Fatalf("with speaker: notes %+v problems %+v err %v", notes, problems, err)
	}
	if notes[0].Speaker != "Phil Morgan" {
		t.Errorf("speaker = %q, want %q", notes[0].Speaker, "Phil Morgan")
	}
	notes, problems, err = ReadAlerts(strings.NewReader(`{"alerts":[` + head + `}]}`))
	if err != nil || len(problems) != 0 || len(notes) != 1 || notes[0].Speaker != "" {
		t.Fatalf("without speaker: notes %+v problems %+v err %v", notes, problems, err)
	}
	// An unknown field is still refused (DisallowUnknownFields kept).
	notes, problems, err = ReadAlerts(strings.NewReader(`{"alerts":[` + head + `,"pastor":"X"}]}`))
	if err != nil || len(notes) != 0 || len(problems) != 1 {
		t.Errorf("unknown field: notes %+v problems %+v err %v", notes, problems, err)
	}
}
