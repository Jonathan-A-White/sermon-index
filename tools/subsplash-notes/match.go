package main

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/Jonathan-A-White/sermon-index/internal/passage"
	"github.com/Jonathan-A-White/sermon-index/internal/sources"
)

const dateLayout = "2006-01-02"

// Record is a sermon's date, title and passage, kept against the YouTube row
// it was matched to. Date is the sermon's Sunday, or its own day for a sermon
// preached midweek (see SundayOf).
type Record struct {
	ID    string `json:"id"` // the YouTube row's id
	Date  string `json:"date"`
	Title string `json:"title"`
	Ref   string `json:"ref"`            // the first place the mail's passage names
	Also  string `json:"also,omitempty"` // its other places, joined by AlsoSep; empty for one place
	// Speaker is the preacher the alert named, when it named one.
	Speaker string `json:"speaker,omitempty"`
}

// SundayOf is the sermon's day for an alert date: a Sunday stays, a Saturday
// or a Monday is taken to the Sunday one day away (the alert is sent beside the
// service), and a Tuesday to a Friday stays as it is, since a sermon dated then
// is a midweek service, a funeral or Christmas Day, not a Sunday's sermon. Only
// a date that is not a date is an error.
func SundayOf(date string) (string, error) {
	t, err := time.Parse(dateLayout, date)
	if err != nil {
		return "", fmt.Errorf("'%s' is not a date (YYYY-MM-DD)", date)
	}
	switch t.Weekday() {
	case time.Sunday:
		return date, nil
	case time.Saturday:
		return t.AddDate(0, 0, 1).Format(dateLayout), nil
	case time.Monday:
		return t.AddDate(0, 0, -1).Format(dateLayout), nil
	}
	return date, nil
}

// Match gives the note the one YouTube row within one day of its Sunday (a
// video is dated by its publish instant in UTC, which can miss the Sunday by a
// day). No row, or more than one, is an error: the reader does not guess.
// The record's date is the Sunday; its id is the row's.
func Match(n Note, rows []sources.Row) (Record, error) {
	sunday, err := SundayOf(n.Date)
	if err != nil {
		return Record{}, err
	}
	st, _ := time.Parse(dateLayout, sunday)
	var hits, sameTitle []string
	for _, r := range rows {
		if r.Date == nil {
			continue
		}
		rt, err := time.Parse(dateLayout, *r.Date)
		if err != nil {
			continue
		}
		if d := rt.Sub(st); d >= -24*time.Hour && d <= 24*time.Hour {
			hits = append(hits, r.ID)
			if foldTitle(r.Title) == foldTitle(n.Title) {
				sameTitle = append(sameTitle, r.ID)
			}
		}
	}
	// A mail with no passage has only its date and its title to go by: of
	// several rows within the day, the one with its title is taken.
	if n.Ref == "" && len(hits) > 1 && len(sameTitle) == 1 {
		hits = sameTitle
	}
	switch len(hits) {
	case 0:
		return Record{}, noRowError{sunday}
	case 1:
		return Record{ID: hits[0], Date: sunday, Title: n.Title, Ref: n.Ref, Also: n.Also, Speaker: n.Speaker}, nil
	}
	sort.Strings(hits)
	return Record{}, fmt.Errorf("more than one row within one day of %s (%s): not guessed", dayName(sunday), strings.Join(hits, ", "))
}

// noRowError is Match's answer when no row is within one day of the Sunday:
// the one failure MatchNote tries a second way on.
type noRowError struct{ sunday string }

func (e noRowError) Error() string {
	return "no row within one day of " + dayName(e.sunday)
}

// dayName is "Sunday 2031-03-02" for a Sunday and "2031-03-05 (a Wednesday)"
// for a midweek day, so a reason names the day it looked at.
func dayName(day string) string {
	if t, err := time.Parse(dateLayout, day); err == nil && t.Weekday() != time.Sunday {
		return fmt.Sprintf("%s (a %s)", day, t.Weekday())
	}
	return "Sunday " + day
}

// TitleWindowDays is how far from the mail's Sunday a row may be dated for
// MatchNote to take it by its title and passage alone.
const TitleWindowDays = 14

// MatchNote is Match and, when no row is within one day of the Sunday, a
// second try: the one row within TitleWindowDays of the Sunday that has the
// note's title (case, punctuation and spacing folded) and the same passage.
// The church's list and the mail may differ by a week, and which is right is
// not known, so the date is not asked. A row's title may carry the passage and
// the preacher after " | ", and its passage may be a verse range inside the
// note's chapter (or the reverse): both count as the same. Two such rows, or only rows with
// another passage, are an error naming them. how is empty for a match by
// date, and says so for one by title and passage.
func MatchNote(n Note, rows []sources.Row) (rec Record, how string, err error) {
	rec, err = Match(n, rows)
	var nr noRowError
	if !errors.As(err, &nr) {
		return rec, "", err
	}
	st, _ := time.Parse(dateLayout, nr.sunday)
	want := foldTitle(n.Title)
	var same, otherPassage []string
	rowDate := map[string]string{}
	for _, r := range rows {
		if r.Date == nil || want == "" || !hasTitle(r.Title, want) {
			continue
		}
		rt, perr := time.Parse(dateLayout, *r.Date)
		if perr != nil {
			continue
		}
		if d := rt.Sub(st); d < -TitleWindowDays*24*time.Hour || d > TitleWindowDays*24*time.Hour {
			continue
		}
		rowDate[r.ID] = *r.Date
		if sharesPlace(n, r) {
			same = append(same, r.ID)
		} else {
			otherPassage = append(otherPassage, r.ID)
		}
	}
	name := func(ids []string) string {
		sort.Strings(ids)
		var parts []string
		for _, id := range ids {
			parts = append(parts, fmt.Sprintf("%s dated %s", id, rowDate[id]))
		}
		return strings.Join(parts, ", ")
	}
	switch {
	case len(same) == 1:
		return Record{ID: same[0], Date: nr.sunday, Title: n.Title, Ref: n.Ref, Also: n.Also, Speaker: n.Speaker},
			fmt.Sprintf("matched by title and passage, row dated %s, mail dated %s", rowDate[same[0]], n.Date), nil
	case len(same) > 1:
		return Record{}, "", fmt.Errorf("%w; more than one row within %d days has its title and passage (%s): not guessed", err, TitleWindowDays, name(same))
	case len(otherPassage) > 0:
		return Record{}, "", fmt.Errorf("%w; within %d days only a row with its title and another passage (%s): not guessed", err, TitleWindowDays, name(otherPassage))
	}
	return Record{}, "", err
}

// sharesPlace is whether the row's passage, or one it cites, is any of the
// places the note's passage names.
func sharesPlace(n Note, r sources.Row) bool {
	if n.Ref == "" {
		return false
	}
	places := []string{n.Ref}
	if n.Also != "" {
		places = append(places, strings.Split(n.Also, AlsoSep)...)
	}
	have := append([]string{}, r.Cited...)
	if r.Ref != nil {
		have = append(have, *r.Ref)
	}
	for _, h := range have {
		for _, p := range places {
			if h == p || passage.SamePlace(h, p) {
				return true
			}
		}
	}
	return false
}

// hasTitle is whether a row's title is the folded title want: the whole title,
// or its part before the first " | " (YouTube titles read "Title | Passage |
// Preacher").
func hasTitle(title, want string) bool {
	if foldTitle(title) == want {
		return true
	}
	head, _, found := strings.Cut(title, " | ")
	return found && foldTitle(head) == want
}

// foldTitle is a title with case, punctuation and spacing folded away.
func foldTitle(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(unicode.ToLower(r))
		}
	}
	return b.String()
}
