package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Jonathan-A-White/sermon-index/internal/passage"
)

// KindSermonNotes is the alert kind Postern's alerts mail gives a forwarded
// Subsplash notes mail (docs/protocol.md section 26 there).
const KindSermonNotes = "sermon-notes"

// MaxNotesBytes is the most Postern keeps of one alert's notes PDF (5 MiB,
// docs/api.md there).
const MaxNotesBytes = 5 << 20

// Alert is a sermon-notes alert as Postern lists it (GET /api/alerts): parsed
// facts only. It has no place for a mail's body or an attachment's bytes (the
// list never carries them: an alert with notes names them by hash and size, and
// they are fetched on their own), and a reply that carries a field not here
// is refused, so neither can be held by mistake.
type Alert struct {
	ID          string `json:"id"`
	Seq         int    `json:"seq"`
	Kind        string `json:"kind"`
	Title       string `json:"title"`
	Reference   string `json:"reference"`
	When        string `json:"when"`
	NotesSHA256 string `json:"notes_sha256"`
	NotesSize   int    `json:"notes_size"`
}

// Note is what one alert says about a sermon: its title, its passage as an
// OSIS reference, and the date the alert gives (YYYY-MM-DD). NotesSHA256 and
// NotesSize name the notes PDF the alert holds at Postern, both empty/zero when
// it has none.
type Note struct {
	AlertID     string
	Title       string
	Ref         string
	Reference   string // the passage as the alert wrote it, for the report
	Date        string
	NotesSHA256 string
	NotesSize   int
}

// Problem is an alert the reader dropped, and why. Why never repeats a
// field's value other than the reference the alert printed. Says is what the
// alert itself says (see Says), empty when it says nothing readable.
type Problem struct {
	AlertID string
	Why     string
	Says    string
}

// Says is what an alert says of its sermon, for a reported line: its title,
// its passage as written and the date of its mail, each quoted so a stray
// newline cannot split the line. Subsplash's pages are not read, so this is
// the only way to tell which sermon a reported alert carries. An empty field
// is left out; the date is the day the alert's own offset prints.
func Says(title, reference, when string) string {
	var parts []string
	if t := strings.TrimSpace(title); t != "" {
		parts = append(parts, fmt.Sprintf("title %q", t))
	}
	if r := strings.TrimSpace(reference); r != "" {
		parts = append(parts, fmt.Sprintf("passage %q", r))
	}
	if when = strings.TrimSpace(when); when != "" {
		if t, err := time.Parse(time.RFC3339, when); err == nil {
			when = t.Format(dateLayout)
		}
		parts = append(parts, fmt.Sprintf("mail dated %q", when))
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, ", ")
}

// alertSays reads title, reference and when out of a raw alert as far as they
// are strings, whatever else the alert holds.
func alertSays(raw []byte) string {
	var a struct {
		Title     string `json:"title"`
		Reference string `json:"reference"`
		When      string `json:"when"`
	}
	if json.Unmarshal(raw, &a) != nil {
		return ""
	}
	return Says(a.Title, a.Reference, a.When)
}

// Says is what the note's alert says of it.
func (n Note) Says() string { return Says(n.Title, n.Reference, n.Date) }

// ReadAlerts reads the body of GET /api/alerts ({"alerts": [...], "next": n})
// and returns the notes of its sermon-notes alerts. An alert of any other kind
// is not the reader's and is left alone. A sermon-notes alert without a title,
// a valid reference or a date, or with a field the reader does not take, is
// dropped and reported.
func ReadAlerts(r io.Reader) ([]Note, []Problem, error) {
	var reply struct {
		Alerts []json.RawMessage `json:"alerts"`
	}
	if err := json.NewDecoder(r).Decode(&reply); err != nil {
		return nil, nil, fmt.Errorf("the alerts are not JSON of the form {\"alerts\": [...]}: %w", err)
	}
	var notes []Note
	var problems []Problem
	for i, raw := range reply.Alerts {
		var head struct {
			ID   string `json:"id"`
			Kind string `json:"kind"`
		}
		if err := json.Unmarshal(raw, &head); err != nil {
			problems = append(problems, Problem{fmt.Sprintf("#%d", i+1), "not an alert object", ""})
			continue
		}
		if head.Kind != KindSermonNotes {
			continue
		}
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.DisallowUnknownFields()
		var a Alert
		if err := dec.Decode(&a); err != nil {
			problems = append(problems, Problem{head.ID, "dropped: " + err.Error() + " (the reader takes only title, reference, when, notes_sha256 and notes_size)", alertSays(raw)})
			continue
		}
		n, err := a.note()
		if err != nil {
			problems = append(problems, Problem{a.ID, "dropped: " + err.Error(), Says(a.Title, a.Reference, a.When)})
			continue
		}
		notes = append(notes, n)
	}
	return notes, problems, nil
}

func (a Alert) note() (Note, error) {
	title := strings.TrimSpace(a.Title)
	if title == "" {
		return Note{}, errors.New("no title")
	}
	ref, err := ReferenceToOSIS(a.Reference)
	if err != nil {
		return Note{}, err
	}
	if a.When == "" {
		return Note{}, errors.New("no date")
	}
	t, err := time.Parse(time.RFC3339, a.When)
	if err != nil {
		return Note{}, errors.New("the date is not RFC 3339")
	}
	// The alert's own offset is Eastern: its date is the date the church's
	// mail printed, not the UTC date of that instant.
	n := Note{AlertID: a.ID, Title: title, Ref: ref, Reference: strings.TrimSpace(a.Reference), Date: t.Format("2006-01-02"), NotesSHA256: a.NotesSHA256, NotesSize: a.NotesSize}
	switch {
	case a.NotesSHA256 == "" && a.NotesSize == 0:
	case a.NotesSHA256 == "" || a.NotesSize == 0:
		return Note{}, errors.New("notes_sha256 and notes_size come together; one is missing")
	case !sha256RE.MatchString(a.NotesSHA256):
		return Note{}, errors.New("notes_sha256 is not 64 lower-case hex characters")
	case a.NotesSize < 0 || a.NotesSize > MaxNotesBytes:
		return Note{}, fmt.Errorf("notes_size %d is outside 1 to %d bytes", a.NotesSize, MaxNotesBytes)
	}
	return n, nil
}

var sha256RE = regexp.MustCompile(`^[0-9a-f]{64}$`)

var refRE = func() *regexp.Regexp {
	names := map[string]bool{"psalm": true, "song of songs": true}
	for _, b := range passage.Books() {
		names[strings.ToLower(b.Name)] = true
	}
	list := make([]string, 0, len(names))
	for n := range names {
		list = append(list, n)
	}
	sort.Slice(list, func(i, j int) bool { return len(list[i]) > len(list[j]) })
	for i, n := range list {
		list[i] = strings.ReplaceAll(regexp.QuoteMeta(n), `\ `, `\s+`)
	}
	return regexp.MustCompile(`(?i)^(` + strings.Join(list, "|") + `)\.?\s+(\d{1,3})(?::(\d{1,3})([a-z])?)?(?:\s*[-–—]\s*(\d{1,3})([a-z])?(?::(\d{1,3})([a-z])?)?)?$`)
}()

var bookOSIS = func() map[string]string {
	m := map[string]string{"psalm": "Ps", "song of songs": "Song"}
	for _, b := range passage.Books() {
		m[strings.ToLower(b.Name)] = b.OSIS
	}
	return m
}()

// ReferenceToOSIS reads one passage as Subsplash prints it ("2 Peter
// 3:10-18", "Psalm 23", "John 3:16-4:3", "2 Peter 2:4-10a") as an OSIS
// reference; a half verse is read as its verse. Anything else,
// a second passage included, is an error naming what was given.
func ReferenceToOSIS(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", errors.New("no reference")
	}
	m := refRE.FindStringSubmatch(s)
	if m == nil {
		return "", fmt.Errorf("reference '%s' is not one passage of a book, chapter and verses", s)
	}
	book := bookOSIS[strings.ToLower(strings.Join(strings.Fields(m[1]), " "))]
	// A letter part ("10a", "4b") names half a verse; the verse is the unit
	// here, so the letter is dropped. It is only a verse's: a chapter with a
	// letter ("Isaiah 40a", "Isaiah 40-41a") is still refused.
	c1, v1, x, y := m[2], m[3], m[5], m[7]
	if m[6] != "" && (v1 == "" || y != "") {
		return "", fmt.Errorf("reference '%s' is not one passage of a book, chapter and verses", s)
	}
	var ref string
	if v1 == "" {
		ref = book + "." + c1
		if x != "" {
			ref += "-" + book + "." + x
		}
	} else {
		ref = book + "." + c1 + "." + v1
		switch {
		case x != "" && y == "":
			ref += "-" + book + "." + c1 + "." + x
		case x != "":
			ref += "-" + book + "." + x + "." + y
		}
	}
	r, err := passage.Parse(ref)
	if err != nil {
		return "", fmt.Errorf("reference '%s' is refused: %w", s, err)
	}
	return r.OSIS, nil
}
