package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The four 'not recorded ... another sha256' alerts of 2026-11-11 were each a
// second mail for a sermon whose notes were already kept: the same title,
// passage and Sunday, the PDF made again a few seconds apart (so another
// sha256). A second mail for one sermon keeps the newer notes and the older
// beside them; a mail for another sermon never shares the row.

var pdf3 = []byte("%PDF-1.4\ninvented notes of the first Sunday, made again\n")

const garden = "2031-03-02T00:00:00-05:00"

func gardenAlert(id string, seq int, pdf []byte) string {
	return notesAlert(id, seq, "The Garden Gate", "Genesis 2:8-17", garden, hexSum(pdf), len(pdf))
}

// A second mail, in a later run, for the sermon whose notes are kept: the
// newer PDF is kept as <row>.pdf, the older beside it as <row>.prev.pdf, the
// record is unchanged and the alert is claimed.
func TestASecondMailForTheSameSermonKeepsTheNewerNotes(t *testing.T) {
	r := newNotesRun(t, alertsJSON(gardenAlert("n1", 1, pdf1)))
	if code, _, errOut := r.fetch(t); code != 0 {
		t.Fatalf("first fetch exited %d: %s", code, errOut)
	}
	recs1 := mustRead(t, r.records)

	r2 := newNotesRun(t, alertsJSON(gardenAlert("n3", 3, pdf2)))
	r2.notes, r2.records = r.notes, r.records
	code, out, errOut := r2.fetch(t)
	if code != 0 {
		t.Fatalf("second fetch exited %d: %s", code, errOut)
	}
	if got := mustRead(t, filepath.Join(r.notes, "v1.pdf")); got != string(pdf2) {
		t.Errorf("v1.pdf is %q, want the newer mail's notes", got)
	}
	if got := mustRead(t, filepath.Join(r.notes, "v1.prev.pdf")); got != string(pdf1) {
		t.Errorf("v1.prev.pdf is %q, want the older notes", got)
	}
	if strings.Join(r2.f.claimed, ",") != "n3" {
		t.Errorf("claimed %v, want n3", r2.f.claimed)
	}
	if mustRead(t, r.records) != recs1 {
		t.Error("the record changed")
	}
	if !strings.Contains(errOut, "n3") || !strings.Contains(errOut, "v1.prev.pdf") {
		t.Errorf("the replacement is not reported with the alert and where the older notes are: %s", errOut)
	}
	if !strings.Contains(out, "0 not replaced") {
		t.Errorf("the summary counts a replacement as a refusal: %s", out)
	}

	// The older mail is still in Postern (its claim had failed): its notes
	// are the ones kept beside, so it changes nothing and is claimed.
	r3 := newNotesRun(t, alertsJSON(gardenAlert("n1", 1, pdf1), gardenAlert("n3", 3, pdf2)))
	r3.notes, r3.records = r.notes, r.records
	if code, _, errOut := r3.fetch(t); code != 0 {
		t.Fatalf("third fetch exited %d: %s", code, errOut)
	}
	if mustRead(t, filepath.Join(r.notes, "v1.pdf")) != string(pdf2) || mustRead(t, filepath.Join(r.notes, "v1.prev.pdf")) != string(pdf1) {
		t.Error("the older mail replaced the newer notes")
	}
	if strings.Join(r3.f.claimed, ",") != "n1,n3" {
		t.Errorf("claimed %v, want n1,n3", r3.f.claimed)
	}
}

// Two mails for one sermon in one run: the later one's notes are kept, the
// earlier beside them; both are claimed.
func TestTwoMailsForOneSermonInOneRun(t *testing.T) {
	r := newNotesRun(t, alertsJSON(gardenAlert("n1", 1, pdf1), gardenAlert("n3", 3, pdf3)))
	r.f.notes["n3"] = pdf3
	code, _, errOut := r.fetch(t)
	if code != 0 {
		t.Fatalf("fetch exited %d: %s", code, errOut)
	}
	if mustRead(t, filepath.Join(r.notes, "v1.pdf")) != string(pdf3) || mustRead(t, filepath.Join(r.notes, "v1.prev.pdf")) != string(pdf1) {
		t.Error("the later mail's notes are not kept with the earlier beside them")
	}
	if strings.Join(r.f.claimed, ",") != "n1,n3" {
		t.Errorf("claimed %v", r.f.claimed)
	}
}

// A failed download of the newer notes leaves the kept notes as they are.
func TestAFailedDownloadOfNewerNotesKeepsTheOldOnes(t *testing.T) {
	r := newNotesRun(t, alertsJSON(gardenAlert("n1", 1, pdf1)))
	if code, _, errOut := r.fetch(t); code != 0 {
		t.Fatalf("first fetch exited %d: %s", code, errOut)
	}
	r2 := newNotesRun(t, alertsJSON(gardenAlert("n9", 3, pdf2))) // the fake has no notes for n9
	r2.notes, r2.records = r.notes, r.records
	if code, _, errOut := r2.fetch(t); code != 0 || !strings.Contains(errOut, "n9") {
		t.Fatalf("exit %d: %s", code, errOut)
	}
	if mustRead(t, filepath.Join(r.notes, "v1.pdf")) != string(pdf1) {
		t.Error("the kept notes changed")
	}
	if _, err := os.Stat(filepath.Join(r.notes, "v1.prev.pdf")); err == nil {
		t.Error("a v1.prev.pdf was made though nothing was replaced")
	}
	if len(r2.f.claimed) != 0 {
		t.Errorf("claimed %v", r2.f.claimed)
	}
}

// Two mails for different sermons that fall on one row never share it: the
// second is reported with both titles and its notes are neither downloaded nor
// written; the summary counts it.
func TestAMailForAnotherSermonNeverSharesTheRow(t *testing.T) {
	other := notesAlert("n3", 3, "A Different Sermon", "John 3:16", garden, hexSum(pdf2), len(pdf2))
	r := newNotesRun(t, alertsJSON(gardenAlert("n1", 1, pdf1), other))
	code, out, errOut := r.fetch(t)
	if code != 0 {
		t.Fatalf("fetch exited %d: %s", code, errOut)
	}
	if mustRead(t, filepath.Join(r.notes, "v1.pdf")) != string(pdf1) {
		t.Error("the other sermon's notes were written to the row")
	}
	if _, err := os.Stat(filepath.Join(r.notes, "v1.prev.pdf")); err == nil {
		t.Error("a v1.prev.pdf was made though nothing was replaced")
	}
	if !strings.Contains(errOut, "n3") || !strings.Contains(errOut, "The Garden Gate") || !strings.Contains(errOut, "A Different Sermon") {
		t.Errorf("the report does not name the alert and both titles: %s", errOut)
	}
	if strings.Join(r.f.claimed, ",") != "n1" || r.notesAsked() != 1 {
		t.Errorf("claimed %v, asked %d times", r.f.claimed, r.notesAsked())
	}
	if !strings.Contains(out, "1 not replaced") {
		t.Errorf("the summary does not count the refusal: %s", out)
	}
}

// Notes already on disk with no record to say they are the same sermon's are
// not replaced; the line says why and what to do, and the summary counts it.
func TestARefusedNotesFileSaysWhatToDo(t *testing.T) {
	r := newNotesRun(t, goodAlerts())
	if code, _, errOut := r.fetch(t); code != 0 {
		t.Fatalf("fetch exited %d: %s", code, errOut)
	}
	// The records were lost; the notes are still there.
	r2 := newNotesRun(t, alertsJSON(gardenAlert("n3", 3, pdf2)))
	r2.notes = r.notes
	code, out, errOut := r2.fetch(t)
	if code != 0 {
		t.Fatalf("second fetch exited %d: %s", code, errOut)
	}
	if mustRead(t, filepath.Join(r.notes, "v1.pdf")) != string(pdf1) {
		t.Error("the file was replaced")
	}
	if !strings.Contains(errOut, "delete") || !strings.Contains(errOut, "v1.pdf") {
		t.Errorf("the line does not say what to do: %s", errOut)
	}
	if !strings.Contains(out, "1 not replaced") {
		t.Errorf("the summary does not count it: %s", out)
	}
}
