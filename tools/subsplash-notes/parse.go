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

// Alert is a sermon-notes alert as Postern lists it (GET /api/alerts): parsed
// facts only. It has no place for a mail's body or an attachment, and a reply
// that carries a field not here is refused, so neither can be held by
// mistake.
type Alert struct {
	ID        string `json:"id"`
	Seq       int    `json:"seq"`
	Kind      string `json:"kind"`
	Title     string `json:"title"`
	Reference string `json:"reference"`
	When      string `json:"when"`
}

// Note is what one alert says about a sermon: its title, its passage as an
// OSIS reference, and the date the alert gives (YYYY-MM-DD).
type Note struct {
	AlertID string
	Title   string
	Ref     string
	Date    string
}

// Problem is an alert the reader dropped, and why. Why never repeats a
// field's value other than the reference the alert printed.
type Problem struct {
	AlertID string
	Why     string
}

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
			problems = append(problems, Problem{fmt.Sprintf("#%d", i+1), "not an alert object"})
			continue
		}
		if head.Kind != KindSermonNotes {
			continue
		}
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.DisallowUnknownFields()
		var a Alert
		if err := dec.Decode(&a); err != nil {
			problems = append(problems, Problem{head.ID, "dropped: " + err.Error() + " (the reader takes only title, reference and when)"})
			continue
		}
		n, err := a.note()
		if err != nil {
			problems = append(problems, Problem{a.ID, "dropped: " + err.Error()})
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
	return Note{AlertID: a.ID, Title: title, Ref: ref, Date: t.Format("2006-01-02")}, nil
}

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
	return regexp.MustCompile(`(?i)^(` + strings.Join(list, "|") + `)\.?\s+(\d{1,3})(?::(\d{1,3}))?(?:\s*[-–—]\s*(\d{1,3})(?::(\d{1,3}))?)?$`)
}()

var bookOSIS = func() map[string]string {
	m := map[string]string{"psalm": "Ps", "song of songs": "Song"}
	for _, b := range passage.Books() {
		m[strings.ToLower(b.Name)] = b.OSIS
	}
	return m
}()

// ReferenceToOSIS reads one passage as Subsplash prints it ("2 Peter
// 3:10-18", "Psalm 23", "John 3:16-4:3") as an OSIS reference. Anything else,
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
	c1, v1, x, y := m[2], m[3], m[4], m[5]
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
