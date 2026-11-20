package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

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
	// Also is a second passage the notes name (a Governor-read "cited" text).
	Also string `json:"also,omitempty"`
	// Speaker is the preacher the notes mail's alert named, when it named one.
	Speaker string `json:"speaker,omitempty"`
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
	date  string // the sermon's date on Subsplash, or the notes record's; "" when unknown
}

// claimDate is the date of the sermon an address names: the date the
// inventory gave it, else the date of the one notes record whose title has its
// slug; "" when neither is known.
func claimDate(recs []Record, a Address) string {
	if a.Date != "" {
		return a.Date
	}
	date := ""
	for _, rec := range recs {
		if hasSlug(rec.Title, a.Slug) {
			if date != "" && date != rec.Date {
				return ""
			}
			date = rec.Date
		}
	}
	return date
}

// Link matches each address to the one row it belongs to, by name and then
// by date:
//
//  1. a row whose title has the address's slug. Two such rows are reported,
//     not guessed.
//  2. else a record of the notes whose title has the slug, and the row the
//     record's id names (the record's date is its Sunday, a row may be dated
//     the Saturday or the Monday).
//  3. else, for a record with no id or an id no row has, the one row whose
//     date is the record's date. Two rows on that date, or none, are
//     reported.
//  4. else, for an address whose date is known (Address.Date, from the
//     inventory), the one row dated within a day of it whose title is close
//     to the address's slug, or failing that the one generic-titled row.
//
// A row whose alt_url holds another address is a conflict: reported, not
// written. A row that two or more links claim with different addresses goes to
// the one whose date (the inventory's, else the notes record's) is the row's
// date; where none, or several, have it the row is left unwritten and named
// once, though a row that already holds one of several equally dated addresses
// keeps it. An address met twice (same code) counts once.
func Link(rows []sources.Row, recs []Record, addrs []Address) Result {
	res := Result{Set: map[string]string{}}
	type line struct {
		order int
		text  string
	}
	var lines []line
	var say sayFunc = func(order int, a Address, format string, args ...any) {
		lines = append(lines, line{order, fmt.Sprintf("%s %s: %s", a.Code, a.Slug, fmt.Sprintf(format, args...))})
	}

	claims := map[int][]claim{} // row index -> links that claim it
	seen := map[string]bool{}
	for i, a := range addrs {
		if seen[a.Code] {
			continue
		}
		seen[a.Code] = true
		date := claimDate(recs, a)
		idx, why := find(rows, recs, a.Slug)
		if why != "" && date != "" {
			if j, ok := findByDate(rows, a.Slug, date); ok {
				idx, why = j, ""
			}
		}
		if why != "" {
			say(i, a, "%s", why)
			continue
		}
		claims[idx] = append(claims[idx], claim{i, a, date})
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
			cs = resolveClaims(row, cs, say)
			if len(cs) == 0 {
				continue
			}
		}
		c := cs[0]
		switch {
		case row.AltURL == nil:
			res.Set[row.ID] = c.addr.URL
		case *row.AltURL == c.addr.URL || codeOf(*row.AltURL) == c.addr.Code:
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
	if rec.ID != "" {
		for i, r := range rows {
			if r.ID == rec.ID {
				return i, ""
			}
		}
	}
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

// sayFunc adds one line to the report, ordered by the link's place in the input.
type sayFunc = func(order int, a Address, format string, args ...any)

// resolveClaims chooses among links with different addresses that claim one
// row: the one whose date is the row's date. Where none has it, or several do
// (and the row holds none of them), nothing is chosen and the row is named
// once; the claims left to write (none, or one) are returned.
func resolveClaims(row sources.Row, cs []claim, say sayFunc) []claim {
	var onDate []claim
	if row.Date != nil {
		for _, c := range cs {
			if c.date == *row.Date {
				onDate = append(onDate, c)
			}
		}
	}
	byHeld := false
	if len(onDate) > 1 && row.AltURL != nil {
		var held []claim
		for _, c := range onDate {
			if c.addr.URL == *row.AltURL {
				held = append(held, c)
			}
		}
		if len(held) == 1 {
			onDate, byHeld = held, true
		}
	}
	if len(onDate) == 1 {
		for _, c := range cs {
			if c != onDate[0] {
				if byHeld {
					say(c.order, c.addr, "not written: row %s has the same date and already holds %s", row.ID, onDate[0].addr.Code)
				} else {
					say(c.order, c.addr, "not written: row %s is dated %s and %s is the link with that date", row.ID, *row.Date, onDate[0].addr.Code)
				}
			}
		}
		return onDate
	}
	codes := make([]string, len(cs))
	for i, c := range cs {
		d := c.date
		if d == "" {
			d = "undated"
		}
		codes[i] = c.addr.Code + " (" + d + ")"
	}
	why := "none has the row's date"
	if len(onDate) > 1 {
		why = fmt.Sprintf("%d have the row's date", len(onDate))
	}
	when := "undated"
	if row.Date != nil {
		when = *row.Date
	}
	say(cs[0].order, cs[0].addr, "conflict: row %s (dated %s) is claimed by %d links with different addresses, %s (%s); none written", row.ID, when, len(cs), strings.Join(codes, ", "), why)
	return nil
}

// findByDate is the one row dated within a day of date that an address with
// this slug names, for an address no row has the title of: a row whose title is
// close to the slug (as many words shared as not), else, with none close, the
// one row with a generic title ("Brookfield First Sunday Service (Live)").
func findByDate(rows []sources.Row, slug, date string) (int, bool) {
	d, err := time.Parse("2006-01-02", date)
	if err != nil {
		return 0, false
	}
	var near, generic []int
	for i, r := range rows {
		if r.Date == nil {
			continue
		}
		rd, err := time.Parse("2006-01-02", *r.Date)
		if err != nil {
			continue
		}
		if diff := rd.Sub(d); diff < -24*time.Hour || diff > 24*time.Hour {
			continue
		}
		switch {
		case closeTitle(r.Title, slug):
			near = append(near, i)
		case genericTitle(r.Title):
			generic = append(generic, i)
		}
	}
	if len(near) == 1 {
		return near[0], true
	}
	if len(near) == 0 && len(generic) == 1 {
		return generic[0], true
	}
	return 0, false
}

// genericTitle is the church's title for a service with no sermon title.
func genericTitle(t string) bool {
	t = strings.ToLower(strings.TrimSpace(t))
	return strings.HasPrefix(t, "brookfield first") || strings.Contains(t, "online service")
}

// closeTitle is whether a row's title shares at least half of its words with
// a slug's (the words of both together, counted once).
func closeTitle(title, slug string) bool {
	want := map[string]bool{}
	for _, w := range strings.Split(slug, "-") {
		want[w] = true
	}
	for _, s := range Slugs(title) {
		have := map[string]bool{}
		for _, w := range strings.Split(s, "-") {
			have[w] = true
		}
		both := 0
		for w := range have {
			if want[w] {
				both++
			}
		}
		if all := len(have) + len(want) - both; all > 0 && 2*both >= all {
			return true
		}
	}
	return false
}

// codeOf is the code of a sermon address (the part of its last path segment
// before the first "-"), or "" for an address that is not one of the church's.
func codeOf(u string) string {
	rest, ok := strings.CutPrefix(u, addressPrefix)
	if !ok {
		return ""
	}
	code, _, _ := strings.Cut(rest, "-")
	return code
}
