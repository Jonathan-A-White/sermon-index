package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/Jonathan-A-White/sermon-index/internal/sources"
)

// The inventory: every sermon on the Governor's signed-in Subsplash account,
// one line each in data/subsplash-inventory.jsonl (git-ignored, as the links
// file is), read by `subsplash-notes inventory` through the same paced,
// stopping browser walk as browse (browse.go), and filled into the index by
// `subsplash-notes inventory-fill`, which needs no browser.

// InvEntry is a line of the inventory: what Subsplash says of one sermon.
type InvEntry struct {
	Code     string `json:"code"`
	Address  string `json:"address"` // the sermon's own page
	Share    string `json:"share"`   // the short share link its code makes
	Date     string `json:"date"`    // YYYY-MM-DD, as the page prints it
	Title    string `json:"title"`
	Preacher string `json:"preacher,omitempty"`
	Passage  string `json:"passage,omitempty"` // as printed
	Ref      string `json:"ref,omitempty"`     // its first place, as OSIS
	Also     string `json:"also,omitempty"`    // its other places, joined by AlsoSep
	Series   string `json:"series,omitempty"`
	// Themes are the tags or themes the app shows for the sermon. The sermon
	// page and the notes name none today, so this stays empty until one does.
	Themes    []string `json:"themes,omitempty"`
	HasNotes  bool     `json:"has_notes"`
	NotesLink string   `json:"notes_link,omitempty"` // the address the Notes button opens
	// Row is the id of the index row MatchNote gave it; NoRow says why none.
	Row   string `json:"row,omitempty"`
	Match string `json:"match,omitempty"` // empty for a match by date; else how
	NoRow string `json:"no_row,omitempty"`
	// Problem is a passage the book table could not read, kept as printed.
	Problem string `json:"problem,omitempty"`
}

// ReadInventory reads the inventory file; a missing file is an empty one.
func ReadInventory(path string) ([]InvEntry, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []InvEntry
	for n, l := range strings.Split(string(b), "\n") {
		if strings.TrimSpace(l) == "" {
			continue
		}
		var e InvEntry
		if err := json.Unmarshal([]byte(l), &e); err != nil {
			return nil, fmt.Errorf("%s line %d: %w", path, n+1, err)
		}
		out = append(out, e)
	}
	return out, nil
}

// writeInventory writes the entries newest to oldest, whole, to a temporary
// file renamed into place.
func writeInventory(path string, entries []InvEntry) error {
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Date != entries[j].Date {
			return entries[i].Date > entries[j].Date
		}
		return entries[i].Code < entries[j].Code
	})
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	for _, e := range entries {
		if err := enc.Encode(e); err != nil {
			return err
		}
	}
	return writeFileAtomic(path, buf.Bytes(), 0o644)
}

// inventorySermon reads one sermon's page and, when it has notes, its notes
// page's address, and puts the sermon in the inventory. A sermon already in it
// is not loaded again. Its row is the one MatchNote gives (the mail's matcher,
// the only one): none is not an error, the entry says why.
func (r *browseRun) inventorySermon(ctx context.Context, it Item) error {
	if _, ok := r.inv[it.Code]; ok {
		return nil
	}
	addr := it.Address()
	p, err := r.pb.Open(ctx, addr)
	if err != nil {
		return err
	}
	if err := Check(p, KindSermon); err != nil {
		return err
	}
	s, err := ParseSermon(p)
	if err != nil {
		return err
	}
	r.res.Visited++
	var np NotesPage
	if s.HasNotes {
		page, err := r.pb.OpenNotes(ctx)
		if err != nil {
			return err
		}
		if err := Check(page, KindNotes); err != nil {
			return err
		}
		if np, err = ParseNotes(page); err != nil {
			return err
		}
	}
	note, perr := r.note(s, np)
	if s.Reference != "" && np.Reference != "" && np.Reference != s.Reference {
		// The sermon page's passage can be the narrower ("Revelation 5:9-10"
		// where the notes say "Revelation 5"): the page is what is shown first.
		page, perr2 := r.note(s, NotesPage{})
		if perr2 == nil {
			note.Reference, note.Ref, note.Also, perr = page.Reference, page.Ref, page.Also, nil
		}
	}
	e := InvEntry{Code: it.Code, Address: addr, Share: it.Short(), Date: note.Date, Title: note.Title, Preacher: s.Speaker,
		Passage: note.Reference, Ref: note.Ref, Also: note.Also, Series: s.Series, HasNotes: s.HasNotes, NotesLink: np.URL}
	if perr != nil {
		e.Problem = perr.Error()
	}
	e.Row, e.Match, e.NoRow = matchEntry(e, r.cfg.Rows)
	r.inv[e.Code] = e
	all := make([]InvEntry, 0, len(r.inv))
	for _, v := range r.inv {
		all = append(all, v)
	}
	if err := writeInventory(r.cfg.Inventory, all); err != nil {
		return err
	}
	if err := r.addLink(addr); err != nil {
		return err
	}
	r.res.Inventoried++
	switch {
	case e.Row != "":
		r.say("%s (%s): inventoried, row %s", e.Title, e.Date, e.Row)
	default:
		r.say("%s (%s): inventoried, no row: %s", e.Title, e.Date, e.NoRow)
	}
	return r.writeFeed()
}

// matchEntry is the row MatchNote gives an entry, as the mail's alert would
// be matched, and how; none gives the reason.
func matchEntry(e InvEntry, rows []sources.Row) (row, how, noRow string) {
	rec, how, err := MatchNote(Note{Title: e.Title, Ref: e.Ref, Also: e.Also, Date: e.Date, Speaker: e.Preacher}, rows)
	if err != nil {
		return "", "", err.Error()
	}
	return rec.ID, how, ""
}

// ---- fill

// FillFiles names what Fill reads and writes. Passages is the church's list
// (sources/brookfield-first/passages.tsv); Report, when set, is the report
// written; RowsWithThemes holds the ids of rows that have a theme linked
// (sermon-topics.jsonl), beside the rows' own topics.
type FillFiles struct {
	Passages       string
	Report         string
	RowsWithThemes map[string]bool
}

// FillResult counts what Fill did and found.
type FillResult struct {
	Entries       int // sermons in the inventory
	Matched       int // of them with one row
	OnlySubsplash int // Subsplash has, the index has no row for
	OnlyIndex     int // rows with no sermon in the inventory
	Lines         int // lines written to passages.tsv
	Title         int // of them setting the title
	Preacher      int // the preacher
	Passage       int // the passage
	Listed        int // rows the list already has a line for
	Shared        int // entries left out because another claims the same row
	Differ        int // matched where the row's own title, preacher or passage is not Subsplash's
	WithVerses    int // sermons with a passage, from the entry or its row
	WithThemes    int // sermons with a theme
	WithNeither   int // with neither
}

// genericTitle is a title the church's list gives a service with no sermon
// title ("Brookfield First - ...", "... Online Service | <date>").
func genericTitle(t string) bool {
	t = strings.ToLower(strings.TrimSpace(t))
	return strings.HasPrefix(t, "brookfield first") || strings.Contains(t, "online service")
}

// Fill matches each entry to its row (MatchNote), writes to passages.tsv a line
// for each row that lacks a title, preacher or passage Subsplash has (a line
// with the row's id, from = "browse"; a row already in the list, by id, or by
// date for a line with no row, is left to it), and writes the report. A second
// run writes nothing new.
func Fill(entries []InvEntry, rows []sources.Row, files FillFiles) (FillResult, error) {
	res := FillResult{Entries: len(entries)}
	byID := map[string]sources.Row{}
	for _, r := range rows {
		byID[r.ID] = r
	}
	claims := map[string][]invClaim{}
	var unmatched []InvEntry
	for _, e := range entries {
		id, how, noRow := matchEntry(e, rows)
		if id == "" {
			e.NoRow = noRow
			unmatched = append(unmatched, e)
			continue
		}
		e.Row, e.Match = id, how
		claims[id] = append(claims[id], invClaim{e, how})
	}
	listed, listedDate, err := readPassagesKeys(files.Passages)
	if err != nil {
		return res, err
	}
	themed := func(r sources.Row) bool { return len(r.Topics) > 0 || files.RowsWithThemes[r.ID] }
	var lines []string
	var matchedIDs []string
	for id := range claims {
		matchedIDs = append(matchedIDs, id)
	}
	sort.Strings(matchedIDs)
	for _, id := range matchedIDs {
		cs := claims[id]
		row := byID[id]
		res.Matched += len(cs)
		if len(cs) > 1 {
			res.Shared += len(cs)
			for _, c := range cs {
				switch {
				case c.e.Ref != "" || (row.Ref != nil && *row.Ref != ""):
					res.WithVerses++
				case themed(row):
					res.WithThemes++
				default:
					res.WithNeither++
				}
			}
			continue
		}
		e := cs[0].e
		hasRef := e.Ref != "" || (row.Ref != nil && *row.Ref != "")
		hasTheme := len(e.Themes) > 0 || themed(row)
		if hasRef {
			res.WithVerses++
		}
		if hasTheme {
			res.WithThemes++
		}
		if !hasRef && !hasTheme {
			res.WithNeither++
		}
		if row.Preacher != nil && e.Preacher != "" && *row.Preacher != e.Preacher || row.Ref != nil && e.Ref != "" && *row.Ref != e.Ref || !genericTitle(row.Title) && foldTitle(row.Title) != foldTitle(e.Title) {
			res.Differ++
		}
		wantTitle := genericTitle(row.Title) && e.Title != "" && !genericTitle(e.Title)
		wantPreacher := (row.Preacher == nil || *row.Preacher == "") && e.Preacher != ""
		wantPassage := (row.Ref == nil || *row.Ref == "") && e.Passage != "" && e.Ref != ""
		if !wantTitle && !wantPreacher && !wantPassage {
			continue
		}
		if listed[id] || (row.Date != nil && listedDate[*row.Date]) || listedDate[e.Date] {
			res.Listed++
			continue
		}
		title, speaker, pass := row.Title, "", ""
		if wantTitle {
			title = e.Title
			res.Title++
		}
		if wantPreacher {
			speaker = e.Preacher
			res.Preacher++
		}
		if wantPassage {
			pass = e.Passage
			res.Passage++
		}
		lines = append(lines, strings.Join([]string{clean(e.Series), e.Date, clean(title), clean(speaker), clean(pass), id, "browse"}, "\t"))
		res.Lines++
	}
	for _, e := range unmatched {
		res.OnlySubsplash++
		if e.Ref != "" {
			res.WithVerses++
		} else {
			res.WithNeither++
		}
	}
	for _, r := range rows {
		if _, ok := claims[r.ID]; !ok {
			res.OnlyIndex++
		}
	}
	if len(lines) > 0 {
		if err := appendPassages(files.Passages, lines); err != nil {
			return res, err
		}
	}
	if files.Report != "" {
		if err := writeFileAtomic(files.Report, []byte(fillReport(res, entries, unmatched, claims, rows)), 0o644); err != nil {
			return res, err
		}
	}
	return res, nil
}

func clean(s string) string {
	return strings.Join(strings.Fields(strings.ReplaceAll(s, "\t", " ")), " ")
}

// readPassagesKeys gives the rows (by id) the list already has a line for, and
// the dates of its lines with no row. The header must name the row and from
// columns the lines written here use.
func readPassagesKeys(path string) (rows, dates map[string]bool, err error) {
	rows, dates = map[string]bool{}, map[string]bool{}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	header := false
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") {
			continue
		}
		f := strings.Split(line, "\t")
		if !header {
			if len(f) < 7 || strings.TrimSpace(f[5]) != "row" || strings.TrimSpace(f[6]) != "from" {
				return nil, nil, fmt.Errorf("%s: the header names no row and from columns (series, date, title, speaker, passage, row, from)", path)
			}
			header = true
			continue
		}
		if len(f) > 5 && strings.TrimSpace(f[5]) != "" {
			rows[strings.TrimSpace(f[5])] = true
		} else if len(f) > 1 {
			dates[strings.TrimSpace(f[1])] = true
		}
	}
	if !header {
		return nil, nil, fmt.Errorf("%s: no header line", path)
	}
	return rows, dates, nil
}

// appendPassages adds the lines at the end of the list, after a newline if it
// lacks one.
func appendPassages(path string, lines []string) error {
	b, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if len(b) > 0 && b[len(b)-1] != '\n' {
		b = append(b, '\n')
	}
	b = append(b, []byte(strings.Join(lines, "\n")+"\n")...)
	return writeFileAtomic(path, b, 0o644)
}

// invClaim is an entry matched to a row, and how.
type invClaim struct {
	e   InvEntry
	how string
}

func fillReport(res FillResult, entries, unmatched []InvEntry, claims map[string][]invClaim, rows []sources.Row) string {
	var b strings.Builder
	w := func(format string, a ...any) { fmt.Fprintf(&b, format, a...) }
	newest, oldest := "", ""
	if len(entries) > 0 {
		newest, oldest = entries[0].Date, entries[0].Date
		for _, e := range entries {
			if e.Date > newest {
				newest = e.Date
			}
			if e.Date < oldest {
				oldest = e.Date
			}
		}
	}
	w("# Brookfield First: the church's Subsplash account against the index (mw-qtu3zvz.59.38)\n\n")
	w("Written by `go run ./tools/subsplash-notes inventory-fill` from the inventory `go run ./tools/subsplash-notes inventory` read through the Governor's signed-in browser (his allowed use of his own account, polite pace; docs/operations.md, \"Reading Subsplash through the signed-in browser\"). The inventory itself (`data/subsplash-inventory.jsonl`: link, date, title, preacher, passage, series, themes, notes link) is git-ignored like the links file; this report holds counts and ids only. A re-run rewrites it.\n\n")
	w("**%d sermons** inventoried, %s back to %s (the Morning RePhil episodes are not sermons and are left out; `data/morning-rephil.jsonl` tracks them).\n\n", len(entries), newest, oldest)
	w("| | Sermons |\n|---|---|\n")
	w("| Matched to one index row (by date, `MatchNote`) | %d |\n", res.Matched)
	w("| On Subsplash, no row in the index | %d |\n", res.OnlySubsplash)
	w("| Rows of the index with no sermon on Subsplash | %d |\n", res.OnlyIndex)
	w("| Left to the church's list: two sermons claim one row | %d |\n", res.Shared)
	w("| Matched, where the row's own title, preacher or passage is not Subsplash's (kept, not changed) | %d |\n", res.Differ)
	w("\n## What went into passages.tsv\n\n")
	w("%d lines (`from` = `browse`; a row already in the list is not written again, %d left to it): %d set a title, %d a preacher, %d a passage. A row keeps a title, preacher and passage it already has.\n\n", res.Lines, res.Listed, res.Title, res.Preacher, res.Passage)
	w("## Verses and themes\n\n")
	w("| | Sermons |\n|---|---|\n")
	w("| With verses (a passage, from Subsplash or the row) | %d |\n", res.WithVerses)
	w("| With themes (a theme linked in `sermon-topics.jsonl`, or the row's own topics; Subsplash shows none) | %d |\n", res.WithThemes)
	w("| With neither | %d |\n", res.WithNeither)
	w("\nSubsplash's sermon page and its notes name no theme or tag, and the sermon-notes mails (Postern kind `sermon-notes`) carry a title, a passage, a date and sometimes a speaker, no theme; so every theme counted here is one the index already holds.\n\n")
	w("## On Subsplash, not in the index\n\n")
	if len(unmatched) == 0 {
		w("None.\n\n")
	} else {
		w("| Date | Code | Title | Passage | Why no row |\n|---|---|---|---|---|\n")
		for _, e := range unmatched {
			w("| %s | `%s` | %s | %s | %s |\n", e.Date, e.Code, cell(e.Title), cell(e.Passage), cell(e.NoRow))
		}
		w("\n")
	}
	w("## In the index, not on Subsplash\n\n")
	var only []sources.Row
	for _, r := range rows {
		if _, ok := claims[r.ID]; !ok {
			only = append(only, r)
		}
	}
	sort.SliceStable(only, func(i, j int) bool { return dateOf(only[i]) > dateOf(only[j]) })
	if len(only) == 0 {
		w("None.\n")
	} else {
		w("| Date | Row | Title |\n|---|---|---|\n")
		for _, r := range only {
			w("| %s | `%s` | %s |\n", dateOf(r), r.ID, cell(r.Title))
		}
	}
	return b.String()
}

func dateOf(r sources.Row) string {
	if r.Date == nil {
		return ""
	}
	return *r.Date
}

func cell(s string) string { return strings.ReplaceAll(clean(s), "|", "\\|") }
