package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/Jonathan-A-White/sermon-index/internal/sources"
)

// Record is a line of tools/subsplash-notes's records file: a sermon's date
// (its Sunday), the title and passage Subsplash's notes gave it, and the id
// of the YouTube row they were matched to.
type Record struct {
	ID    string `json:"id"`
	Date  string `json:"date"`
	Title string `json:"title"`
	Ref   string `json:"ref"`
}

// ReadRecords reads the records file; a file that does not exist holds none.
func ReadRecords(path string) ([]Record, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var recs []Record
	for n, line := range strings.Split(string(b), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var r Record
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			return nil, fmt.Errorf("%s line %d: %w", path, n+1, err)
		}
		recs = append(recs, r)
	}
	return recs, nil
}

var (
	apostropheRE = regexp.MustCompile(`['’‘ʼ]`)
	notSlugRE    = regexp.MustCompile(`[^a-z0-9]+`)
)

func slugOf(s string) string {
	return strings.Trim(notSlugRE.ReplaceAllString(strings.ToLower(s), "-"), "-")
}

// Slugs are the slugs a title can have on Subsplash: lower case, letters
// and digits, any other run of characters a single hyphen. Subsplash's
// treatment of an apostrophe is not known, so a title with one has two: the
// apostrophe dropped (the first) and the apostrophe a hyphen.
func Slugs(title string) []string {
	first := slugOf(apostropheRE.ReplaceAllString(title, ""))
	if !apostropheRE.MatchString(title) {
		return []string{first}
	}
	second := slugOf(apostropheRE.ReplaceAllString(title, "-"))
	if second == first {
		return []string{first}
	}
	return []string{first, second}
}

func hasSlug(title, slug string) bool {
	for _, s := range Slugs(title) {
		if s == slug && s != "" {
			return true
		}
	}
	return false
}

// Result is what Link decided: the address to write to each row (by row
// id), how many rows already hold the address they would be given, and one
// line for each link that was not written and why.
type Result struct {
	Set     map[string]string
	Already int
	Report  []string
}

type claim struct {
	order int
	addr  Address
}

// Link matches each address to the one row it belongs to, by name and then
// by date:
//
//  1. a row whose title has the address's slug. Two such rows are reported,
//     not guessed.
//  2. else a record of the notes whose title has the slug, and the one row
//     whose date is the record's date. Two rows on that date, or none, are
//     reported.
//
// A row whose alt_url holds another address, or that two links claim with
// different addresses, is a conflict: reported, not written. An address met
// twice (same code) counts once.
func Link(rows []sources.Row, recs []Record, addrs []Address) Result {
	res := Result{Set: map[string]string{}}
	type line struct {
		order int
		text  string
	}
	var lines []line
	say := func(order int, a Address, format string, args ...any) {
		lines = append(lines, line{order, fmt.Sprintf("%s %s: %s", a.Code, a.Slug, fmt.Sprintf(format, args...))})
	}

	claims := map[int][]claim{} // row index -> links that claim it
	seen := map[string]bool{}
	for i, a := range addrs {
		if seen[a.Code] {
			continue
		}
		seen[a.Code] = true
		idx, why := find(rows, recs, a.Slug)
		if why != "" {
			say(i, a, "%s", why)
			continue
		}
		claims[idx] = append(claims[idx], claim{i, a})
	}

	idxs := make([]int, 0, len(claims))
	for idx := range claims {
		idxs = append(idxs, idx)
	}
	sort.Ints(idxs)
	for _, idx := range idxs {
		row, cs := rows[idx], claims[idx]
		distinct := map[string]bool{}
		for _, c := range cs {
			distinct[c.addr.URL] = true
		}
		if len(distinct) > 1 {
			for _, c := range cs {
				say(c.order, c.addr, "conflict: row %s is claimed by %d links with different addresses; none written", row.ID, len(cs))
			}
			continue
		}
		c := cs[0]
		switch {
		case row.AltURL == nil:
			res.Set[row.ID] = c.addr.URL
		case *row.AltURL == c.addr.URL:
			res.Already++
		default:
			say(c.order, c.addr, "conflict: row %s already has alt_url %s; not replaced", row.ID, *row.AltURL)
		}
	}
	sort.SliceStable(lines, func(i, j int) bool { return lines[i].order < lines[j].order })
	for _, l := range lines {
		res.Report = append(res.Report, l.text)
	}
	return res
}

// find is the index of the row a slug belongs to, or why not.
func find(rows []sources.Row, recs []Record, slug string) (int, string) {
	var byTitle []int
	for i, r := range rows {
		if hasSlug(r.Title, slug) {
			byTitle = append(byTitle, i)
		}
	}
	switch len(byTitle) {
	case 1:
		return byTitle[0], ""
	case 0:
	default:
		return 0, fmt.Sprintf("%d rows have this title (%s): not guessed", len(byTitle), rowIDs(rows, byTitle))
	}

	var matched []Record
	for _, rec := range recs {
		if hasSlug(rec.Title, slug) {
			matched = append(matched, rec)
		}
	}
	if len(matched) == 0 {
		return 0, "no row has this title and no note has it"
	}
	if len(matched) > 1 {
		return 0, fmt.Sprintf("%d notes have this title (dated %s and %s): not guessed", len(matched), matched[0].Date, matched[1].Date)
	}
	rec := matched[0]
	var onDate []int
	for i, r := range rows {
		if r.Date != nil && *r.Date == rec.Date {
			onDate = append(onDate, i)
		}
	}
	switch len(onDate) {
	case 1:
		return onDate[0], ""
	case 0:
		return 0, fmt.Sprintf("the note '%s' is dated %s but no row has that date", rec.Title, rec.Date)
	}
	return 0, fmt.Sprintf("the note '%s' is dated %s but %d rows have that date (%s): not guessed", rec.Title, rec.Date, len(onDate), rowIDs(rows, onDate))
}

func rowIDs(rows []sources.Row, idxs []int) string {
	ids := make([]string, len(idxs))
	for i, idx := range idxs {
		ids[i] = rows[idx].ID
	}
	return strings.Join(ids, ", ")
}
