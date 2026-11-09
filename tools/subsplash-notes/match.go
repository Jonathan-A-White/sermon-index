package main

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Jonathan-A-White/sermon-index/internal/sources"
)

const dateLayout = "2006-01-02"

// Record is a sermon's date, title and passage, kept against the YouTube row
// it was matched to. Date is the sermon's Sunday.
type Record struct {
	ID    string `json:"id"` // the YouTube row's id
	Date  string `json:"date"`
	Title string `json:"title"`
	Ref   string `json:"ref"`
}

// SundayOf is the sermon's Sunday for an alert date: a Sunday stays, a
// Saturday or a Monday is taken to the Sunday one day away, and any other
// weekday is an error, not a guess.
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
	return "", fmt.Errorf("%s %s is a %s: only a Sunday, or a Saturday or Monday beside one, is taken", t.Weekday(), date, t.Weekday())
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
	var hits []string
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
		}
	}
	switch len(hits) {
	case 0:
		return Record{}, fmt.Errorf("no row within one day of Sunday %s", sunday)
	case 1:
		return Record{ID: hits[0], Date: sunday, Title: n.Title, Ref: n.Ref}, nil
	}
	sort.Strings(hits)
	return Record{}, fmt.Errorf("more than one row within one day of Sunday %s (%s): not guessed", sunday, strings.Join(hits, ", "))
}
