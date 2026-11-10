package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The invented notes PDF: not a real PDF, only bytes that begin like one. No
// test holds a sermon's words.
var (
	pdf1 = []byte("%PDF-1.4\ninvented notes of the first Sunday\n")
	pdf2 = []byte("%PDF-1.4\ninvented notes of the second Sunday\n")
)

func hexSum(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

// notesAlert is a sermon-notes alert for the Sunday with notes_sha256 and
// notes_size, which a test may make lie.
func notesAlert(id string, seq int, title, ref, when, sha string, size int) string {
	return fmt.Sprintf(`{"id":%q,"seq":%d,"kind":"sermon-notes","title":%q,"reference":%q,"when":%q,"notes_sha256":%q,"notes_size":%d}`,
		id, seq, title, ref, when, sha, size)
}

func plainAlert(id string, seq int, title, ref, when string) string {
	return fmt.Sprintf(`{"id":%q,"seq":%d,"kind":"sermon-notes","title":%q,"reference":%q,"when":%q}`, id, seq, title, ref, when)
}

func alertsJSON(alerts ...string) string {
	return `{"alerts":[` + strings.Join(alerts, ",") + `],"next":9}`
}

// The first Sunday (row v1) has notes, the second (row v2) has none.
func goodAlerts() string {
	return alertsJSON(
		notesAlert("n1", 1, "The Garden Gate", "Genesis 2:8-17", "2031-03-02T00:00:00-05:00", hexSum(pdf1), len(pdf1)),
		plainAlert("n2", 2, "Walking the Hill", "Psalm 23", "2031-03-09T00:00:00-05:00"),
	)
}

type notesRun struct {
	f       *fakePostern
	key     string
	postern string
	dir     string // holds notes/ and records
	notes   string
	records string
}

func newNotesRun(t *testing.T, alerts string) *notesRun {
	t.Helper()
	f, srv := newFakePostern(t, alerts)
	key, pub := newKey(t)
	f.licensed[pub] = true
	f.notes = map[string][]byte{"n1": pdf1, "n3": pdf2}
	dir := t.TempDir()
	return &notesRun{f: f, key: key, postern: srv.URL, dir: dir,
		notes: filepath.Join(dir, "notes"), records: filepath.Join(dir, "data", "subsplash-notes.jsonl")}
}

func (r *notesRun) fetch(t *testing.T, extra ...string) (int, string, string) {
	t.Helper()
	args := append([]string{"fetch", "--key", r.key, "--postern", r.postern, "--rows", rowsFile(t), "--out", r.records, "--notes", r.notes}, extra...)
	return cliRun(t, args...)
}

func (r *notesRun) notesAsked() int {
	n := 0
	for _, q := range r.f.requests {
		if strings.HasSuffix(q, "/notes") {
			n++
		}
	}
	return n
}

func mustRead(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// Acceptance 1 and 2: the PDF is fetched, checked and kept (dir 0700, file
// 0600, named by the row id) and the records are in place before the alert
// is claimed.
func TestFetchKeepsTheNotesThenTheRecordsThenClaims(t *testing.T) {
	r := newNotesRun(t, goodAlerts())
	r.f.onClaim = func(id string) {
		if got, err := os.ReadFile(filepath.Join(r.notes, "v1.pdf")); err != nil || string(got) != string(pdf1) {
			t.Errorf("alert %s claimed but the PDF is not on disk: %v", id, err)
		}
		if got, err := os.ReadFile(r.records); err != nil || !strings.Contains(string(got), `"id":"v1"`) {
			t.Errorf("alert %s claimed but the records are not in place: %v", id, err)
		}
	}
	code, out, errOut := r.fetch(t)
	if code != 0 {
		t.Fatalf("fetch exited %d: %s", code, errOut)
	}
	if strings.Join(r.f.claimed, ",") != "n1,n2" {
		t.Errorf("claimed %v", r.f.claimed)
	}
	if got := mustRead(t, filepath.Join(r.notes, "v1.pdf")); got != string(pdf1) {
		t.Errorf("the PDF kept is %q", got)
	}
	if _, err := os.Stat(filepath.Join(r.notes, "v2.pdf")); err == nil {
		t.Error("a PDF was kept for the alert that has no notes")
	}
	di, _ := os.Stat(r.notes)
	fi, _ := os.Stat(filepath.Join(r.notes, "v1.pdf"))
	if di.Mode().Perm() != 0o700 || fi.Mode().Perm() != 0o600 {
		t.Errorf("notes dir %v, file %v; want 0700 and 0600", di.Mode().Perm(), fi.Mode().Perm())
	}
	if entries, _ := os.ReadDir(r.notes); len(entries) != 1 {
		t.Errorf("the notes dir holds %d entries, want only v1.pdf", len(entries))
	}
	if !strings.Contains(out, "2 claimed") {
		t.Errorf("stdout: %q", out)
	}
	if strings.Contains(out+errOut, string(pdf1)) {
		t.Error("the notes' words are printed")
	}
}

// Acceptance 2: a failed download, a wrong size and a wrong sha256 write
// nothing for that alert and leave it unclaimed; the other alert goes on.
func TestAFailedNotesDownloadLeavesThatAlertAlone(t *testing.T) {
	cases := map[string]struct {
		alert string
		why   string
	}{
		"download refused": {notesAlert("n9", 1, "The Garden Gate", "Genesis 2:8-17", "2031-03-02T00:00:00-05:00", hexSum(pdf1), len(pdf1)), "404"},
		"wrong size":       {notesAlert("n1", 1, "The Garden Gate", "Genesis 2:8-17", "2031-03-02T00:00:00-05:00", hexSum(pdf1), len(pdf1)+1), "size"},
		"wrong sha256":     {notesAlert("n1", 1, "The Garden Gate", "Genesis 2:8-17", "2031-03-02T00:00:00-05:00", hexSum(pdf2), len(pdf1)), "sha256"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			r := newNotesRun(t, alertsJSON(c.alert, plainAlert("n2", 2, "Walking the Hill", "Psalm 23", "2031-03-09T00:00:00-05:00")))
			code, _, errOut := r.fetch(t)
			if code != 0 {
				t.Fatalf("fetch exited %d: %s", code, errOut)
			}
			if !strings.Contains(errOut, c.why) || !strings.Contains(errOut, strings.Split(c.alert, `"`)[3]) {
				t.Errorf("the report does not name the alert and %q: %s", c.why, errOut)
			}
			if strings.Join(r.f.claimed, ",") != "n2" {
				t.Errorf("claimed %v, want n2 only", r.f.claimed)
			}
			want := `{"id":"v2","date":"2031-03-09","title":"Walking the Hill","ref":"Ps.23"}` + "\n"
			if got := mustRead(t, r.records); got != want {
				t.Errorf("records:\n%s", got)
			}
			if entries, _ := os.ReadDir(r.notes); len(entries) != 0 {
				t.Errorf("a failed download left %v", entries)
			}
		})
	}
}

// Acceptance 4: a second run writes the same bytes and does not download
// what it holds; a file with another sha256 is reported and kept as it is.
func TestASecondRunDownloadsNothingAndANotesFileOfAnotherHashIsNotReplaced(t *testing.T) {
	r := newNotesRun(t, goodAlerts())
	if code, _, errOut := r.fetch(t); code != 0 {
		t.Fatalf("fetch exited %d: %s", code, errOut)
	}
	if r.notesAsked() != 1 {
		t.Fatalf("the first run asked for the notes %d times", r.notesAsked())
	}
	recs1, pdfBytes := mustRead(t, r.records), mustRead(t, filepath.Join(r.notes, "v1.pdf"))

	// The alerts are claimed and gone; a new Postern holds them again, as when
	// the claim had failed.
	r2 := newNotesRun(t, goodAlerts())
	r2.notes, r2.records = r.notes, r.records
	if code, _, errOut := r2.fetch(t); code != 0 {
		t.Fatalf("second fetch exited %d: %s", code, errOut)
	}
	if r2.notesAsked() != 0 {
		t.Errorf("the second run downloaded notes it already held (%d)", r2.notesAsked())
	}
	if mustRead(t, r.records) != recs1 || mustRead(t, filepath.Join(r.notes, "v1.pdf")) != pdfBytes {
		t.Error("a second run changed a file")
	}
	if strings.Join(r2.f.claimed, ",") != "n1,n2" {
		t.Errorf("the second run claimed %v", r2.f.claimed)
	}

	// A file already there with another sha256 is not replaced.
	r3 := newNotesRun(t, goodAlerts())
	r3.notes = filepath.Join(t.TempDir(), "notes")
	if err := os.MkdirAll(r3.notes, 0o700); err != nil {
		t.Fatal(err)
	}
	other := filepath.Join(r3.notes, "v1.pdf")
	if err := os.WriteFile(other, pdf2, 0o600); err != nil {
		t.Fatal(err)
	}
	code, _, errOut := r3.fetch(t)
	if code != 0 {
		t.Fatalf("fetch exited %d: %s", code, errOut)
	}
	if mustRead(t, other) != string(pdf2) {
		t.Error("a notes file of another sha256 was replaced")
	}
	if !strings.Contains(errOut, "n1") || !strings.Contains(errOut, "not replaced") {
		t.Errorf("not reported: %s", errOut)
	}
	if r3.notesAsked() != 0 || strings.Join(r3.f.claimed, ",") != "n2" {
		t.Errorf("asked %d times, claimed %v", r3.notesAsked(), r3.f.claimed)
	}
	if strings.Contains(mustRead(t, r3.records), `"id":"v1"`) {
		t.Error("a record was written for the alert whose notes file disagrees")
	}
}

// Acceptance 1: an alert with any OTHER unknown field is still dropped.
func TestAnAlertWithAnotherUnknownFieldIsStillDropped(t *testing.T) {
	odd := `{"id":"n1","seq":1,"kind":"sermon-notes","title":"The Garden Gate","reference":"Genesis 2:8-17","when":"2031-03-02T00:00:00-05:00","notes_sha256":"` + hexSum(pdf1) + `","notes_size":` + fmt.Sprint(len(pdf1)) + `,"attachment":"JVBERi0x"}`
	r := newNotesRun(t, alertsJSON(odd))
	code, _, errOut := r.fetch(t)
	if code != 0 || !strings.Contains(errOut, "n1") || len(r.f.claimed) != 0 || r.notesAsked() != 0 {
		t.Errorf("exit %d, claimed %v, asked %d: %s", code, r.f.claimed, r.notesAsked(), errOut)
	}
}

// Acceptance 5: the no-subcommand reader accepts the two fields, claims
// nothing and downloads nothing.
func TestTheReaderAcceptsTheNotesFieldsAndDownloadsNothing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := t.TempDir()
	alerts := filepath.Join(dir, "alerts.json")
	if err := os.WriteFile(alerts, []byte(goodAlerts()), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "records.jsonl")
	code, stdout, errOut := cliRun(t, "--alerts", alerts, "--rows", rowsFile(t), "--out", out)
	if code != 0 || errOut != "" {
		t.Fatalf("exit %d: %s", code, errOut)
	}
	if !strings.Contains(stdout, "2 records") {
		t.Errorf("stdout %q", stdout)
	}
	if strings.Contains(mustRead(t, out), "notes_") {
		t.Error("the records carry the notes' fields")
	}
	if entries, _ := os.ReadDir(home); len(entries) != 0 {
		t.Errorf("the reader wrote under HOME: %v", entries)
	}
}

// Acceptance 6: run in a temporary checkout, with the default notes
// directory (under HOME), nothing is written in the tree but the records.
func TestFetchWritesNothingInTheCheckoutButTheRecords(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	checkout := t.TempDir()
	if err := os.MkdirAll(filepath.Join(checkout, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(checkout, "data"), 0o755); err != nil {
		t.Fatal(err)
	}
	rows := filepath.Join(checkout, "rows.jsonl")
	if err := os.WriteFile(rows, []byte(mustRead(t, rowsFile(t))), 0o644); err != nil {
		t.Fatal(err)
	}
	snap := func() map[string]string {
		m := map[string]string{}
		filepath.WalkDir(checkout, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				t.Fatal(err)
			}
			rel, _ := filepath.Rel(checkout, p)
			if d.IsDir() {
				m[rel+"/"] = ""
			} else {
				m[rel] = mustRead(t, p)
			}
			return nil
		})
		return m
	}
	before := snap()
	r := newNotesRun(t, goodAlerts())
	t.Chdir(checkout)
	code, _, errOut := cliRun(t, "fetch", "--key", r.key, "--postern", r.postern, "--rows", "rows.jsonl", "--out", "data/subsplash-notes.jsonl")
	if code != 0 {
		t.Fatalf("fetch exited %d: %s", code, errOut)
	}
	after := snap()
	for k, v := range after {
		if bv, ok := before[k]; !ok && k != "data/subsplash-notes.jsonl" {
			t.Errorf("fetch added %s to the checkout", k)
		} else if ok && bv != v {
			t.Errorf("fetch changed %s", k)
		}
	}
	for k := range before {
		if _, ok := after[k]; !ok {
			t.Errorf("fetch removed %s", k)
		}
	}
	if _, ok := after["data/subsplash-notes.jsonl"]; !ok {
		t.Error("the records were not written")
	}
	kept := filepath.Join(home, ".local", "state", "sermon-index", "notes", "v1.pdf")
	if got := mustRead(t, kept); got != string(pdf1) {
		t.Errorf("the default notes path %s holds %q", kept, got)
	}

	// A notes directory inside a git work tree is refused: nothing is
	// downloaded or claimed.
	r2 := newNotesRun(t, goodAlerts())
	code, _, errOut = cliRun(t, "fetch", "--key", r2.key, "--postern", r2.postern, "--rows", "rows.jsonl", "--out", "data/other.jsonl", "--notes", "notes")
	if code == 0 || !strings.Contains(errOut, "git") || r2.notesAsked() != 0 || len(r2.f.claimed) != 0 {
		t.Errorf("exit %d, asked %d, claimed %v: %s", code, r2.notesAsked(), r2.f.claimed, errOut)
	}
	if _, err := os.Stat(filepath.Join(checkout, "notes")); err == nil {
		t.Error("a notes directory was made inside the checkout")
	}
}
