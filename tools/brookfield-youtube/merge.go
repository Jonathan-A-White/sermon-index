package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Jonathan-A-White/sermon-index/internal/passage"
	"github.com/Jonathan-A-White/sermon-index/internal/sources"
)

// The church's own list of its sermons: a tab-separated table typed by hand
// from the Governor's screenshots of the church's app (no tool reads the app),
// kept in sources/brookfield-first/passages.tsv. Columns, with a header line:
//
//	series  date  title  speaker  passage
//
// The date is the Sunday. A line that starts with "#" is a comment, except
// "# unsure:", whose text REPORT.md prints as a sermon the list could not pin
// down and which is never merged.
//
// MergeList sets, on the one YouTube row of each entry's date, the sermon's
// own title, its preacher and its passage; the row keeps its id, link and
// date. What it cannot match it names and leaves alone, and never guesses.

// Entry is one sermon of the list.
type Entry struct {
	Series, Date, Title, Speaker, Passage string
	Line                                  int
}

// List is the church's list.
type List struct {
	Entries []Entry
	Unsure  []string // the "# unsure:" lines, without the marker
}

// ReadList reads the table at path.
func ReadList(path string) (*List, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	l := &List{}
	header := false
	for i, line := range strings.Split(string(b), "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			if rest, ok := strings.CutPrefix(line, "# unsure:"); ok {
				l.Unsure = append(l.Unsure, strings.TrimSpace(rest))
			}
			continue
		}
		f := strings.Split(line, "\t")
		if !header {
			want := []string{"series", "date", "title", "speaker", "passage"}
			if len(f) < len(want) {
				return nil, fmt.Errorf("%s:%d: the header names %d columns, want series, date, title, speaker, passage", path, i+1, len(f))
			}
			for j, w := range want {
				if strings.TrimSpace(f[j]) != w {
					return nil, fmt.Errorf("%s:%d: column %d is %q, want %q", path, i+1, j+1, f[j], w)
				}
			}
			header = true
			continue
		}
		if len(f) < 5 {
			return nil, fmt.Errorf("%s:%d: %d columns, want series, date, title, speaker, passage", path, i+1, len(f))
		}
		e := Entry{
			Series: strings.TrimSpace(f[0]), Date: strings.TrimSpace(f[1]), Title: strings.TrimSpace(f[2]),
			Speaker: strings.TrimSpace(f[3]), Passage: strings.TrimSpace(f[4]), Line: i + 1,
		}
		if _, err := time.Parse("2006-01-02", e.Date); err != nil {
			return nil, fmt.Errorf("%s:%d: date %q is not YYYY-MM-DD", path, i+1, e.Date)
		}
		if e.Title == "" {
			return nil, fmt.Errorf("%s:%d: no title", path, i+1)
		}
		l.Entries = append(l.Entries, e)
	}
	if !header {
		return nil, fmt.Errorf("%s: no header line", path)
	}
	return l, nil
}

// Problem is an entry that was not merged, and why.
type Problem struct {
	Entry Entry
	Why   string
}

// Merge is what MergeList did.
type Merge struct {
	Merged   int
	Problems []Problem
	Unsure   []string
	Entries  int // entries in the list
}

// MergeList applies the list to rows, in place.
func MergeList(rows []sources.Row, l *List) *Merge {
	m := &Merge{Unsure: l.Unsure, Entries: len(l.Entries)}
	perDate := map[string]int{}
	for _, e := range l.Entries {
		perDate[e.Date]++
	}
	byDate := map[string][]int{}
	for i, r := range rows {
		if r.Date != nil {
			byDate[*r.Date] = append(byDate[*r.Date], i)
		}
	}
	for _, e := range l.Entries {
		fail := func(why string) { m.Problems = append(m.Problems, Problem{e, why}) }
		if perDate[e.Date] > 1 {
			fail(fmt.Sprintf("the list has %d entries on this date, a date is matched by one entry only", perDate[e.Date]))
			continue
		}
		ref, why := listPassage(e.Passage)
		if why != "" {
			fail(why)
			continue
		}
		idx := byDate[e.Date]
		switch len(idx) {
		case 0:
			fail("no row of the YouTube list has this date")
			continue
		case 1:
		default:
			fail(fmt.Sprintf("%d rows of the YouTube list have this date, so none is chosen", len(idx)))
			continue
		}
		r := &rows[idx[0]]
		r.Title = e.Title
		if e.Speaker != "" {
			r.Preacher = ptr(e.Speaker)
		}
		r.Ref = ptr(ref)
		r.Cited = nil
		r.Series = nil
		note := "Title, preacher and passage from the church's own list of its sermons (passages.tsv), matched to the video by date."
		if e.Series != "" {
			note = fmt.Sprintf("Series: %s. %s", e.Series, note)
		}
		r.Notes = ptr(note)
		m.Merged++
	}
	return m
}

// listPassage reads a passage cell ("1 Peter 1:22-2:3") as an OSIS range. A
// cell that is not exactly one passage the book table accepts, or that spans
// more than 3 chapters (a series, not a text), is refused with the reason.
func listPassage(cell string) (string, string) {
	refs, bad := textPassages(cell)
	switch {
	case len(bad) > 0:
		return "", fmt.Sprintf("the passage %q is refused by the book table", cell)
	case len(refs) != 1:
		return "", fmt.Sprintf("the passage %q is not exactly one passage", cell)
	}
	if loc := passageRE.FindStringIndex(cell); loc == nil || strings.TrimSpace(cell[:loc[0]]+cell[loc[1]:]) != "" {
		return "", fmt.Sprintf("the passage %q holds more than one passage", cell)
	}
	r, err := passage.Parse(refs[0])
	if err != nil {
		return "", fmt.Sprintf("the passage %q is refused: %v", cell, err)
	}
	if r.Chapters() > 3 {
		return "", fmt.Sprintf("the passage %q spans %d chapters, which is a series and not a text", cell, r.Chapters())
	}
	return r.OSIS, ""
}

// recount sets the preacher, passage and series counts from the rows, after
// the list has changed them.
func (r *Result) recount() {
	r.WithPreacher, r.WithPassage, r.SeriesRows = 0, 0, 0
	for _, x := range r.Rows {
		if x.Preacher != nil {
			r.WithPreacher++
		}
		if x.Ref != nil {
			r.WithPassage++
		}
		if x.IsSeries() {
			r.SeriesRows++
		}
	}
}
