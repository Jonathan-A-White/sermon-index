package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Jonathan-A-White/sermon-index/internal/sources"
)

// The job feed for Postern's Jobs page (Postern's docs/jobs.md): one item for
// each Brookfield row, newest first, held to three checks.
//
//	mail  a notes record (tools/subsplash-notes) names the row, or the row is
//	      in no-notes.tsv: the Governor read its sermon in the app and sent it
//	      as one that has no notes, so no mail will come, and the label ends
//	      " (no notes)"
//	link  the row holds a Subsplash address, which only a code he shared (or a
//	      search PDF he gave) can have put there
//	live  the sermons service serves that same address for the row, and with
//	      the passage the row holds (a row whose notes name a passage it lacks
//	      is not live)
//
// "here" is the oldest row with both mail and link: he works newest to oldest,
// so that is where his hands have got to. The feed holds titles, dates and the
// sermon's public Subsplash address; no notes text, no mail address, no
// preacher, no YouTube address.

const (
	feedID    = "subsplash-links"
	feedTitle = "Subsplash links"
)

// Feed is the file jobs/subsplash-links.json.
type Feed struct {
	ID      string     `json:"id"`
	Title   string     `json:"title"`
	Updated string     `json:"updated"`
	Checks  []string   `json:"checks"`
	Items   []FeedItem `json:"items"`
	Here    *int       `json:"here,omitempty"`
}

// FeedItem is one sermon.
type FeedItem struct {
	Label string          `json:"label"`
	Date  string          `json:"date,omitempty"`
	Link  string          `json:"link,omitempty"`
	Done  map[string]bool `json:"done"`
}

// recordRow is the index of the row a notes record names: the row its id
// names, else (a record with no id, or an id no row has) the one row dated
// the record's date. -1 when none or several.
func recordRow(rows []sources.Row, rec Record) int {
	if rec.ID != "" {
		for i, r := range rows {
			if r.ID == rec.ID {
				return i
			}
		}
	}
	found := -1
	for i, r := range rows {
		if rec.Date != "" && r.Date != nil && *r.Date == rec.Date {
			if found >= 0 {
				return -1
			}
			found = i
		}
	}
	return found
}

// isGenericTitle is a YouTube title that names the church's service and not
// the sermon ("Brookfield First - Online Service | March 9, 2031").
func isGenericTitle(t string) bool { return strings.HasPrefix(t, "Brookfield First") }

// BuildFeed is the feed for rows, as the notes records and the service's
// answer (served, by row id) stand at now; noNotes holds the ids of the rows of
// sermons that have no notes. A row with no passage cannot be
// asked about, so it is live when it has its link and the service holds every
// row it can serve as rows.jsonl has it.
func BuildFeed(rows []sources.Row, recs []Record, noNotes map[string]bool, served map[string]servedRow, now time.Time) Feed {
	mail := map[int]string{} // row index -> the title of its record
	noted := map[int]bool{}  // row index -> a record names a passage for it
	for _, rec := range recs {
		if i := recordRow(rows, rec); i >= 0 {
			if _, ok := mail[i]; !ok {
				mail[i] = rec.Title
			}
			if strings.TrimSpace(rec.Ref) != "" {
				noted[i] = true
			}
		}
	}
	allHeld := false
	if len(served) > 0 {
		ids, _ := Differing(rows, served)
		allHeld = len(ids) == 0
	}

	type entry struct {
		date string
		id   string
		item FeedItem
	}
	var entries []entry
	for i, r := range rows {
		date := orEmpty(r.Date)
		label := r.Title
		recTitle, hasMail := mail[i]
		if isGenericTitle(r.Title) {
			switch {
			case hasMail && strings.TrimSpace(recTitle) != "":
				label = recTitle
			case date != "":
				label = "Online Service " + date
			default:
				label = "Online Service"
			}
		}
		if noNotes[r.ID] {
			hasMail = true
			label += " (no notes)"
		}
		link := ""
		if r.AltURL != nil && strings.HasPrefix(*r.AltURL, addressPrefix) {
			link = *r.AltURL
		}
		live := false
		if link != "" {
			if s, ok := served[r.ID]; ok {
				// Served with its link, and with the passage the row holds.
				live = orEmpty(s.AltURL) == link
				if ref := orEmpty(r.Ref); ref != "" {
					live = live && s.Role == "text" && s.Ref == ref
				}
			} else {
				// A row with no passage cannot be served; but if the notes mail
				// names one the row lacks, the service is missing what it
				// should serve (mw-qtu3zvz.59.25), so it is not live.
				live = allHeld && !hasPassage(r) && !(noted[i] && !hasOwnPassage(r))
			}
		}
		entries = append(entries, entry{date, r.ID, FeedItem{
			Label: label, Date: date, Link: link,
			Done: map[string]bool{"mail": hasMail, "link": link != "", "live": live},
		}})
	}
	// Newest first; a row with no date last; the row id breaks a tie so the
	// same rows always give the same file.
	sort.SliceStable(entries, func(a, b int) bool {
		if entries[a].date != entries[b].date {
			return entries[a].date > entries[b].date
		}
		return entries[a].id < entries[b].id
	})

	f := Feed{ID: feedID, Title: feedTitle, Updated: now.UTC().Format(time.RFC3339), Checks: []string{"mail", "link", "live"}, Items: make([]FeedItem, len(entries))}
	for i, e := range entries {
		f.Items[i] = e.item
		if e.item.Done["mail"] && e.item.Done["link"] {
			here := i // the last such item is the oldest
			f.Here = &here
		}
	}
	return f
}

// hasPassage reports whether the service could serve the row.
func hasPassage(r sources.Row) bool {
	if r.Series != nil && *r.Series {
		return false
	}
	return (r.Ref != nil && *r.Ref != "") || len(r.Cited) > 0
}

// hasOwnPassage reports whether the row holds a passage at all, a series' first
// chapter included.
func hasOwnPassage(r sources.Row) bool {
	return (r.Ref != nil && *r.Ref != "") || len(r.Cited) > 0
}

// writeFeed writes the feed whole, to a temporary file beside path and then
// renames it over path, so that a reader never sees half a file.
func writeFeed(path string, f Feed) error {
	b, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".subsplash-links-*.tmp")
	if err != nil {
		return err
	}
	name := tmp.Name()
	if _, err := tmp.Write(append(b, '\n')); err != nil {
		tmp.Close()
		os.Remove(name)
		return err
	}
	if err := tmp.Chmod(0o644); err != nil {
		tmp.Close()
		os.Remove(name)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(name)
		return err
	}
	if err := os.Rename(name, path); err != nil {
		os.Remove(name)
		return fmt.Errorf("%w", err)
	}
	return nil
}

// feedService reads the rows, the records and the service, and writes the feed
// to path.
func feedService(rowsPath, notesPath, noNotesPath, base, source, path string, now time.Time) error {
	rows, err := readRows(rowsPath)
	if err != nil {
		return err
	}
	recs, err := ReadRecords(notesPath)
	if err != nil {
		return err
	}
	noNotes, err := ReadNoNotes(noNotesPath)
	if err != nil {
		return err
	}
	served, err := fetchServed(&http.Client{Timeout: 60 * time.Second}, base, source)
	if err != nil {
		return err
	}
	return writeFeed(path, BuildFeed(rows, recs, noNotes, served, now))
}
