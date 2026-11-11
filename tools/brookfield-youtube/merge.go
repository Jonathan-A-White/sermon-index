package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
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
//	series  date  title  speaker  passage  [row]
//
// The date is the Sunday. The sixth column, row, is optional (the header may
// stop at passage): a line from one of the Governor's messages names the id of
// the row it belongs to, because that row is dated the day the video was
// published, which need not be the Sunday; tools/subsplash-links writes those
// lines, with a line in no-notes.tsv for each. A line with a row is matched to
// that row by id and not by date, and an empty passage sets the title and the
// preacher and leaves the row's passage as it was. A line that starts with "#" is a comment, except
// "# unsure:", whose text REPORT.md prints as a sermon the list could not pin
// down and which is never merged.
//
// MergeList sets, on the one YouTube row of each entry's date, the sermon's
// own title, its preacher and its passage; the row keeps its id, link and
// date. What it cannot match it names and leaves alone, and never guesses.

// Entry is one sermon of the list.
type Entry struct {
	Series, Date, Title, Speaker, Passage string
	Row                                   string // the id of the row a message entry belongs to; empty = match by date
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
			want := []string{"series", "date", "title", "speaker", "passage", "row"}
			if len(f) < 5 {
				return nil, fmt.Errorf("%s:%d: the header names %d columns, want series, date, title, speaker, passage", path, i+1, len(f))
			}
			for j, w := range want {
				if j >= len(f) {
					break
				}
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
		if len(f) > 5 {
			e.Row = strings.TrimSpace(f[5])
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
	Merged int
	// FromMessages is how many of Merged are lines with a row: sermons the
	// Governor sent in a message, read by him in the church's app.
	FromMessages int
	Problems     []Problem
	// Counted is set by MergeRowsFile, which counts the rows after the merge:
	// the rows with a preacher, with a passage, and series rows.
	Counted                               bool
	RowsPreacher, RowsPassage, RowsSeries int
	RowsBoth, RowsNeither                 int
	Unsure                                []string
	Entries                               int // entries in the list
}

// MergeList applies the list to rows, in place.
func MergeList(rows []sources.Row, l *List) *Merge {
	m := &Merge{Unsure: l.Unsure, Entries: len(l.Entries)}
	// An entry is matched to one row: by id when it names one, else by date.
	// Two entries for one row (or one date) are each refused.
	key := func(e Entry) string {
		if e.Row != "" {
			return "row " + e.Row
		}
		return "date " + e.Date
	}
	perKey := map[string]int{}
	for _, e := range l.Entries {
		perKey[key(e)]++
	}
	byDate := map[string][]int{}
	byID := map[string]int{}
	for i, r := range rows {
		if r.Date != nil {
			byDate[*r.Date] = append(byDate[*r.Date], i)
		}
		byID[r.ID] = i
	}
	for _, e := range l.Entries {
		fail := func(why string) { m.Problems = append(m.Problems, Problem{e, why}) }
		if n := perKey[key(e)]; n > 1 {
			if e.Row != "" {
				fail(fmt.Sprintf("the list has %d entries for this row, a row is matched by one entry only", n))
			} else {
				fail(fmt.Sprintf("the list has %d entries on this date, a date is matched by one entry only", n))
			}
			continue
		}
		ref := ""
		if e.Passage != "" || e.Row == "" {
			var why string
			ref, why = listPassage(e.Passage)
			if why != "" {
				fail(why)
				continue
			}
		}
		var at int
		if e.Row != "" {
			i, ok := byID[e.Row]
			if !ok {
				fail(fmt.Sprintf("no row of the YouTube list has the id %q", e.Row))
				continue
			}
			at = i
		} else {
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
			at = idx[0]
		}
		r := &rows[at]
		r.Title = e.Title
		if e.Speaker != "" {
			r.Preacher = ptr(e.Speaker)
		}
		var note string
		switch {
		case e.Row == "":
			r.Ref = ptr(ref)
			r.Cited = nil
			r.Series = nil
			note = "Title, preacher and passage from the church's own list of its sermons (passages.tsv), matched to the video by date."
		case ref != "":
			r.Ref = ptr(ref)
			r.Cited = nil
			r.Series = nil
			note = "Title, preacher and passage from a message the Governor sent, read by him in the church's app (passages.tsv; no notes exist for this sermon), matched to the video by its date."
		default:
			note = "Title and preacher from a message the Governor sent, read by him in the church's app (passages.tsv; no notes exist for this sermon), matched to the video by its date; the message names no passage."
		}
		if e.Series != "" {
			note = fmt.Sprintf("Series: %s. %s", e.Series, note)
		}
		r.Notes = ptr(note)
		m.Merged++
		if e.Row != "" {
			m.FromMessages++
		}
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

// ReadAltURLs reads the alt_url of each row of the rows.jsonl at path, by row
// id. tools/subsplash-links writes alt_url, and this tool rewrites the file
// whole, so what is there is read before it is replaced. A file that is not
// there holds none; a line that cannot be read is an error, because carrying
// on would drop what the line holds.
func ReadAltURLs(path string) (map[string]string, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	alts := map[string]string{}
	for i, line := range bytes.Split(b, []byte("\n")) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var r struct {
			ID     string  `json:"id"`
			AltURL *string `json:"alt_url"`
		}
		if err := json.Unmarshal(line, &r); err != nil {
			return nil, fmt.Errorf("%s:%d: %v (the alt_url of its rows would be lost)", path, i+1, err)
		}
		if r.AltURL != nil && r.ID != "" {
			alts[r.ID] = *r.AltURL
		}
	}
	return alts, nil
}

// KeepAltURLs sets the alt_url of each row that has one in alts (by id). A row
// that is not in rows is not carried over, and a row that alts does not know
// has none.
func KeepAltURLs(rows []sources.Row, alts map[string]string) {
	for i := range rows {
		if u, ok := alts[rows[i].ID]; ok {
			rows[i].AltURL = ptr(u)
		}
	}
}

// MergeRowsFile applies the list at listPath to the rows already in the
// rows.jsonl at rowsPath, with no call to the API: the run that the hourly
// script makes after a message of the Governor's has added a line to the list.
// A row the list does not change is left byte for byte as it was, and the file
// is not written when no row changes. It returns the merge and how many rows
// changed.
func MergeRowsFile(rowsPath, listPath string) (*Merge, int, error) {
	l, err := ReadList(listPath)
	if err != nil {
		return nil, 0, err
	}
	b, err := os.ReadFile(rowsPath)
	if err != nil {
		return nil, 0, err
	}
	lines := strings.Split(string(b), "\n")
	var rows []sources.Row
	var at []int
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		dec := json.NewDecoder(strings.NewReader(line))
		dec.DisallowUnknownFields() // a rewritten line must lose nothing
		var r sources.Row
		if err := dec.Decode(&r); err != nil {
			return nil, 0, fmt.Errorf("%s:%d: %w", rowsPath, i+1, err)
		}
		rows = append(rows, r)
		at = append(at, i)
	}
	encode := func(r sources.Row) (string, error) {
		var buf bytes.Buffer
		if err := writeLine(&buf, r); err != nil {
			return "", err
		}
		return strings.TrimRight(buf.String(), "\n"), nil
	}
	before := make([]string, len(rows))
	for i, r := range rows {
		if before[i], err = encode(r); err != nil {
			return nil, 0, err
		}
	}
	m := MergeList(rows, l)
	m.Counted = true
	for _, r := range rows {
		if r.Preacher != nil {
			m.RowsPreacher++
		}
		if r.Ref != nil {
			m.RowsPassage++
		}
		if r.IsSeries() {
			m.RowsSeries++
		}
		if r.Ref != nil && r.Preacher != nil {
			m.RowsBoth++
		}
		if r.Ref == nil && r.Preacher == nil {
			m.RowsNeither++
		}
	}
	changed := 0
	for i, r := range rows {
		now, err := encode(r)
		if err != nil {
			return nil, 0, err
		}
		if now != before[i] {
			lines[at[i]] = now
			changed++
		}
	}
	if changed > 0 {
		if err := os.WriteFile(rowsPath, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
			return nil, 0, err
		}
	}
	return m, changed, nil
}
