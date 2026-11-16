package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Jonathan-A-White/sermon-index/internal/sources"
)

// Config is one run of browse. Paths are the files the by-hand job writes:
// Out is data/subsplash-notes.jsonl, NotesDir holds the notes beside the PDFs
// (<row id>.txt, mode 0600, never committed), LinksOut is the file of full
// sermon addresses subsplash-links takes with --links, Rephil is
// data/morning-rephil.jsonl, Feed the Jobs page's file.
type Config struct {
	Rows     []sources.Row
	Out      string
	NotesDir string
	LinksOut string
	Rephil   string
	Feed     string
	Interval time.Duration
	Limit    int // most sermon and episode pages to visit; 0 is no limit
	// Series is the Morning RePhil series page to read for every episode
	// (seriesURL); empty reads none. EpisodePages also loads each new
	// episode's own page, to keep any text it carries.
	Series       string
	EpisodePages bool
	SeriesOnly   bool // read the series page and not the sermon list
	Verify       int  // when over 0: compare the newest n sermons already recorded, write nothing
	Now          func() time.Time
	Sleep        func(time.Duration)
	// PDFText is the text of the notes PDF kept for a row, for Verify; nil
	// leaves the notes text uncompared.
	PDFText func(rowID string) (string, error)
}

// Result counts what a run did.
type Result struct {
	Visited  int // sermon and episode pages loaded
	Recorded int // records written
	Linked   int // sermon links written
	Episodes int // episodes written or given their share link
	Verified int
	Differ   int
	Loads    int
}

// RephilRow is a line of data/morning-rephil.jsonl: the YouTube fields of
// tools/brookfield-youtube, and the share link and the file of the page's
// text that this adds. The text is kept in NotesDir, not here: the repo is
// public.
type RephilRow struct {
	Number    int    `json:"number"`
	Title     string `json:"title"`
	Date      string `json:"date"`
	URL       string `json:"url,omitempty"`
	Share     string `json:"share,omitempty"`
	NotesFile string `json:"notes_file,omitempty"`
}

// feedFile is the Jobs page's feed (Postern's docs/jobs.md).
type feedFile struct {
	ID      string     `json:"id"`
	Title   string     `json:"title"`
	Updated string     `json:"updated"`
	Checks  []string   `json:"checks"`
	Items   []feedItem `json:"items"`
	Here    *int       `json:"here,omitempty"`
	Note    string     `json:"note,omitempty"`
}

type feedItem struct {
	Label string          `json:"label"`
	Link  string          `json:"link,omitempty"`
	Done  map[string]bool `json:"done"`
}

type browseRun struct {
	cfg      Config
	pb       *PacedBrowser
	out      io.Writer
	res      Result
	records  []Record
	byTitle  map[string]bool // folded titles with a record
	links    map[string]bool // full addresses in the links file
	altCodes map[string]bool // codes some row's alt_url holds
	rephil   []RephilRow
	seen     []Item // the list so far, newest first
	pages    int
}

// Browse walks the list newest to oldest through b and, for each sermon not
// sent by hand and not read before, reads its page and notes, writing the
// record the mail would have made, the notes text, and the sermon's address;
// for each Morning RePhil episode not yet on file it writes the episode.
// A Stop ends it where it is (the error), with nothing written after.
func Browse(ctx context.Context, b Browser, cfg Config, out io.Writer) (Result, error) {
	r := &browseRun{cfg: cfg, pb: Paced(b, cfg.Interval, cfg.Now, cfg.Sleep), out: out, byTitle: map[string]bool{}, links: map[string]bool{}, altCodes: map[string]bool{}}
	if cfg.Now == nil {
		r.cfg.Now = time.Now
	}
	var err error
	if r.records, err = readRecords(cfg.Out); err != nil {
		return r.res, err
	}
	for _, rec := range r.records {
		r.byTitle[foldTitle(rec.Title)] = true
	}
	for _, row := range cfg.Rows {
		if row.AltURL != nil {
			if code, _, ok := CodeOf(*row.AltURL); ok {
				r.altCodes[code] = true
			}
		}
	}
	if err := r.readLinks(); err != nil {
		return r.res, err
	}
	if r.rephil, err = readRephil(cfg.Rephil); err != nil {
		return r.res, err
	}
	if cfg.SeriesOnly {
		err = r.series(ctx)
	} else {
		err = r.walk(ctx)
	}
	r.res.Loads = r.pb.Loads()
	return r.res, err
}

func (r *browseRun) say(format string, a ...any) { fmt.Fprintf(r.out, format+"\n", a...) }

func (r *browseRun) walk(ctx context.Context) error {
	p, err := r.pb.Open(ctx, listURL)
	if err != nil {
		return err
	}
	var pending []Item // episodes whose title prints no number
	for n := 1; ; n++ {
		if err := Check(p, KindList); err != nil {
			return err
		}
		items, pages, err := ParseList(p)
		if err != nil {
			return err
		}
		r.pages = pages
		for _, it := range items {
			if r.cfg.Limit > 0 && r.res.Visited >= r.cfg.Limit {
				r.say("stopped at the limit of %d pages visited; the next run goes on from here", r.cfg.Limit)
				return nil
			}
			if r.cfg.Verify > 0 && r.res.Verified >= r.cfg.Verify {
				return nil
			}
			r.seen = append(r.seen, it)
			switch {
			case it.Rephil == -1:
				pending = append(pending, it)
			case it.Rephil > 0:
				if r.cfg.Verify == 0 {
					err = r.episode(ctx, it, it.Rephil)
				}
			default:
				err = r.sermon(ctx, it)
			}
			if err != nil {
				return err
			}
		}
		if n >= pages {
			break
		}
		if p, err = r.pb.ListPage(ctx, n+1); err != nil {
			return err
		}
	}
	if r.cfg.Verify > 0 {
		return nil
	}
	if r.cfg.EpisodePages {
		if err := r.pendingEpisodes(ctx, pending); err != nil {
			return err
		}
	}
	return r.series(ctx)
}

func (r *browseRun) pendingEpisodes(ctx context.Context, pending []Item) error {
	// A title with no number is numbered by its place among all the episodes,
	// the oldest being 1: known only once the whole list has been read.
	if len(pending) > 0 {
		all := 0
		for _, it := range r.seen {
			if it.Rephil != 0 {
				all++
			}
		}
		pos := map[string]int{}
		k := all
		for _, it := range r.seen {
			if it.Rephil != 0 {
				pos[it.Code] = k
				k--
			}
		}
		for _, it := range pending {
			if r.cfg.Limit > 0 && r.res.Visited >= r.cfg.Limit {
				break
			}
			if err := r.episode(ctx, it, pos[it.Code]); err != nil {
				return err
			}
		}
	}
	return nil
}

// ---- sermons

func (r *browseRun) sermon(ctx context.Context, it Item) error {
	addr := it.Address()
	hasRecord := r.byTitle[foldTitle(it.Title)]
	if r.cfg.Verify > 0 {
		if !hasRecord {
			return nil
		}
	} else if r.links[addr] || (r.altCodes[it.Code] && hasRecord) {
		return nil
	}
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
	var rec Record
	var how string
	var merr error
	if perr == nil && (s.HasNotes || s.Reference != "") {
		rec, how, merr = MatchNote(note, r.cfg.Rows)
	} else if perr == nil {
		merr = errors.New("a sermon with no notes and no passage has no record")
	} else {
		merr = perr
	}
	if r.cfg.Verify > 0 {
		return r.verify(it, s, np, rec, merr)
	}
	// A sermon with no notes is not recorded: only its link is kept.
	recordable := s.HasNotes && merr == nil
	if s.HasNotes && merr != nil {
		r.say("%s (%s): not recorded: %v; its link is not kept either, so the next run reads it again", it.Title, s.Date, merr)
		return nil
	}
	if recordable {
		if how != "" {
			r.say("%s: %s", it.Title, how)
		}
		if err := r.record(rec, np); err != nil {
			return err
		}
	}
	if err := r.addLink(addr); err != nil {
		return err
	}
	if !s.HasNotes {
		r.say("%s (%s): no notes; link kept", it.Title, s.Date)
	}
	return r.writeFeed()
}

// note is the Note the page gives, as the mail's alert would: the notes
// page's title and passage, else the sermon page's.
func (r *browseRun) note(s Sermon, np NotesPage) (Note, error) {
	title, ref := s.Title, s.Reference
	if np.Title != "" {
		title = np.Title
		if np.Reference != "" {
			ref = np.Reference
		}
	}
	date := s.Date
	if date == "" {
		date = np.Date
	}
	n := Note{Title: title, Reference: ref, Date: date, Speaker: s.Speaker}
	if ref != "" {
		places, err := ReferencesToOSIS(ref)
		if err != nil {
			return n, err
		}
		n.Ref, n.Also = places[0], strings.Join(places[1:], AlsoSep)
	}
	return n, nil
}

// record writes the notes text, then the record, as fetch does: the file
// first, then the records.
func (r *browseRun) record(rec Record, np NotesPage) error {
	for _, have := range r.records {
		if have.ID != rec.ID {
			continue
		}
		if have == rec || (have.Title == rec.Title && have.Date == rec.Date && have.Ref == rec.Ref && have.Also == rec.Also) {
			return r.keepText(rec.ID+".txt", np.Text)
		}
		r.say("%s (%s): disagrees with the record kept for row %s (kept '%s', %s, %s); not replaced", rec.Title, rec.Date, rec.ID, have.Title, have.Ref, have.Date)
		return nil
	}
	if err := r.keepText(rec.ID+".txt", np.Text); err != nil {
		return err
	}
	r.records = append(r.records, rec)
	sort.Slice(r.records, func(i, j int) bool {
		if r.records[i].Date != r.records[j].Date {
			return r.records[i].Date < r.records[j].Date
		}
		return r.records[i].ID < r.records[j].ID
	})
	if err := writeRecords(r.cfg.Out, r.records); err != nil {
		return err
	}
	r.byTitle[foldTitle(rec.Title)] = true
	r.res.Recorded++
	r.say("%s (%s): recorded as row %s", rec.Title, rec.Date, rec.ID)
	return nil
}

// keepText keeps text as name in NotesDir (0700, file 0600): never replaced
// by other words, and refused inside a git work tree as the PDFs are.
func (r *browseRun) keepText(name, text string) error {
	if text == "" {
		return nil
	}
	store := &NotesStore{Dir: r.cfg.NotesDir}
	if err := store.checkDir(); err != nil {
		return err
	}
	path := filepath.Join(r.cfg.NotesDir, name)
	if have, err := os.ReadFile(path); err == nil {
		if string(have) != text {
			r.say("%s already holds other words; not replaced", path)
		}
		return nil
	}
	if err := os.MkdirAll(r.cfg.NotesDir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(r.cfg.NotesDir, ".notes-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	_, err = tmp.WriteString(text)
	if err == nil {
		err = tmp.Chmod(0o600)
	}
	if err == nil {
		err = tmp.Sync()
	}
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Rename(tmp.Name(), path)
	}
	return err
}

func (r *browseRun) readLinks() error {
	b, err := os.ReadFile(r.cfg.LinksOut)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, l := range strings.Split(string(b), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			r.links[l] = true
		}
	}
	return nil
}

// addLink keeps the sermon's full address, one a line, sorted, in LinksOut.
func (r *browseRun) addLink(addr string) error {
	if r.links[addr] {
		return nil
	}
	r.links[addr] = true
	var all []string
	for l := range r.links {
		all = append(all, l)
	}
	sort.Strings(all)
	if err := writeFileAtomic(r.cfg.LinksOut, []byte(strings.Join(all, "\n")+"\n"), 0o644); err != nil {
		return err
	}
	r.res.Linked++
	return nil
}

// ---- verify

// verify says how the record the browser would make compares with the one
// the mail made for the same Sunday, and writes nothing.
func (r *browseRun) verify(it Item, s Sermon, np NotesPage, rec Record, merr error) error {
	r.res.Verified++
	if merr != nil {
		r.res.Differ++
		r.say("%s (%s): not matched: %v", it.Title, s.Date, merr)
		return nil
	}
	var kept *Record
	for i := range r.records {
		if r.records[i].ID == rec.ID {
			kept = &r.records[i]
		}
	}
	if kept == nil {
		r.res.Differ++
		r.say("row %s %s (%s): no record made by the mail for this row", rec.ID, it.Title, s.Date)
		return nil
	}
	var diffs []string
	cmp := func(name, a, b string) {
		if a != b {
			diffs = append(diffs, fmt.Sprintf("%s: browser '%s', mail '%s'", name, a, b))
		}
	}
	cmp("title", rec.Title, kept.Title)
	cmp("date", rec.Date, kept.Date)
	cmp("ref", rec.Ref, kept.Ref)
	cmp("also", rec.Also, kept.Also)
	cmp("link code", s.Code, it.Code)
	notes := "not compared (no PDF text)"
	if r.cfg.PDFText != nil && np.Text != "" {
		if pdf, err := r.cfg.PDFText(rec.ID); err != nil {
			notes = "not compared: " + err.Error()
		} else if same, detail := sameWords(pdf, np.Text); same {
			notes = "same words"
		} else {
			notes = "differs"
			diffs = append(diffs, "notes text: "+detail)
		}
	}
	if len(diffs) == 0 {
		r.say("row %s %s (%s): same as the mail's record (title, date, row id, ref); link code %s; notes text: %s", rec.ID, it.Title, s.Date, it.Code, notes)
		return nil
	}
	r.res.Differ++
	r.say("row %s %s (%s): differs from the mail's record: %s", rec.ID, it.Title, s.Date, strings.Join(diffs, "; "))
	return nil
}

// uiWords are what the notes page prints that the PDF does not.
var uiWords = map[string]bool{"reveal": true, "answer": true, "clear": true, "email": true, "created": true, "by": true, "subsplash": true, "inc": true}

func words(s string) []string {
	var out []string
	var cur strings.Builder
	flush := func() {
		if cur.Len() > 0 {
			out = append(out, strings.ToLower(cur.String()))
			cur.Reset()
		}
	}
	for _, c := range s {
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' {
			cur.WriteRune(c)
		} else {
			flush()
		}
	}
	flush()
	return out
}

// sameWords is whether the PDF's text and the page's hold the same words in
// the same order, leaving out the blanks, the page's own buttons and its
// footer, which the PDF does not print.
func sameWords(pdf, page string) (bool, string) {
	a, b := words(pdf), words(page)
	clean := func(ws []string) []string {
		var out []string
		for _, w := range ws {
			if !uiWords[w] && w != "2009" && w != "2026" {
				out = append(out, w)
			}
		}
		return out
	}
	// A PDF's text runs some words together or splits them ("2 Peter" and
	// "petertruth"), so the words are compared with the spaces left out.
	x, y := strings.Join(clean(a), ""), strings.Join(clean(b), "")
	if x == y {
		return true, ""
	}
	i := 0
	for i < len(x) && i < len(y) && x[i] == y[i] {
		i++
	}
	snip := func(s string) string {
		end := i + 20
		if end > len(s) {
			end = len(s)
		}
		if i > len(s) {
			return ""
		}
		return s[i:end]
	}
	return false, fmt.Sprintf("the PDF has %d letters and digits and the page %d; they part at %d (PDF '%s', page '%s')", len(x), len(y), i, snip(x), snip(y))
}

// ---- Morning RePhil

func readRephil(path string) ([]RephilRow, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var rows []RephilRow
	for n, l := range strings.Split(string(b), "\n") {
		if strings.TrimSpace(l) == "" {
			continue
		}
		var row RephilRow
		if err := json.Unmarshal([]byte(l), &row); err != nil {
			return nil, fmt.Errorf("%s line %d: %w", path, n+1, err)
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func writeRephil(path string, rows []RephilRow) error {
	// Oldest first; an episode keeps its place among those of its own day.
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].Date < rows[j].Date })
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	for _, row := range rows {
		if err := enc.Encode(row); err != nil {
			return err
		}
	}
	return writeFileAtomic(path, buf.Bytes(), 0o644)
}

// episode puts one episode on file under number: a row already there for
// that number gets its share link from the list and no page is loaded; any
// other is read from its page. A row with a share link is done.
func (r *browseRun) episode(ctx context.Context, it Item, number int) error {
	share := it.Short()
	for _, row := range r.rephil {
		if row.Share == share {
			return nil
		}
	}
	if it.Rephil > 0 {
		for i := range r.rephil {
			if r.rephil[i].Number == number && r.rephil[i].Share == "" {
				r.rephil[i].Share = share
				r.res.Episodes++
				r.say("%s: share link added to the episode on file", it.Title)
				if err := writeRephil(r.cfg.Rephil, r.rephil); err != nil {
					return err
				}
				return r.writeFeed()
			}
		}
	}
	if !r.cfg.EpisodePages {
		return nil // the series page writes it
	}
	p, err := r.pb.Open(ctx, it.Address())
	if err != nil {
		return err
	}
	if err := Check(p, KindEpisode); err != nil {
		return err
	}
	ep, err := ParseEpisode(p)
	if err != nil {
		return err
	}
	r.res.Visited++
	// The channel's list (YouTube) may hold the same episode under a title
	// with no number, numbered by its place there: the row of that day with no
	// share link is this episode.
	for i := range r.rephil {
		have := r.rephil[i]
		if have.Share == "" && have.Date == ep.Date && (have.Number == number || RephilNumber(have.Title) == -1) {
			r.rephil[i].Share = share
			r.res.Episodes++
			r.say("%s (%s): the episode on file as '%s' (number %d); share link added", ep.Title, ep.Date, have.Title, have.Number)
			if err := writeRephil(r.cfg.Rephil, r.rephil); err != nil {
				return err
			}
			return r.writeFeed()
		}
	}
	row := RephilRow{Number: number, Title: ep.Title, Date: ep.Date, Share: share}
	if it.Rephil == -1 {
		r.say("%s: no number in its title; numbered %d by its place among all the episodes (1 is the oldest)", it.Title, number)
	}
	if ep.Text != "" {
		row.NotesFile = fmt.Sprintf("rephil-%d.txt", number)
		if it.Rephil == -1 {
			row.NotesFile = fmt.Sprintf("rephil-%s.txt", it.Code)
		}
		if err := r.keepText(row.NotesFile, ep.Text+"\n"); err != nil {
			return err
		}
	}
	r.rephil = append(r.rephil, row)
	if err := writeRephil(r.cfg.Rephil, r.rephil); err != nil {
		return err
	}
	r.res.Episodes++
	r.say("%s (%s): episode written", ep.Title, ep.Date)
	return r.writeFeed()
}

// ---- the Jobs page

// writeFeed writes the Jobs page's feed: each item the list has shown so far,
// newest first, with a tick once the tool has read it (a sermon's address kept,
// an episode on file with its share link), and the oldest ticked one as the
// place. It is written whole to a temporary file and renamed.
func (r *browseRun) writeFeed() error {
	if r.cfg.Feed == "" || r.cfg.Verify > 0 {
		return nil
	}
	shares := map[string]bool{}
	for _, row := range r.rephil {
		if row.Share != "" {
			shares[row.Share] = true
		}
	}
	f := feedFile{ID: "subsplash-browse", Title: "Subsplash through the browser", Updated: r.cfg.Now().UTC().Format(time.RFC3339), Checks: []string{"read"}, Items: []feedItem{}}
	here := -1
	for i, it := range r.seen {
		done := r.links[it.Address()] || shares[it.Short()]
		f.Items = append(f.Items, feedItem{Label: it.Title, Link: it.Address(), Done: map[string]bool{"read": done}})
		if done {
			here = i
		}
	}
	if here >= 0 {
		f.Here = &here
	}
	f.Note = fmt.Sprintf("%d of %d list pages walked, %d recorded, %d sermon links, %d episodes", r.pageNo(), r.pages, r.res.Recorded, r.res.Linked, r.res.Episodes)
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(f); err != nil {
		return err
	}
	return writeFileAtomic(r.cfg.Feed, buf.Bytes(), 0o644)
}

func (r *browseRun) pageNo() int {
	n := (len(r.seen) + 19) / 20
	if n < 1 {
		n = 1
	}
	return n
}

// writeFileAtomic writes b to a temporary file beside path, syncs it and
// renames it into place.
func writeFileAtomic(path string, b []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".browse-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	_, err = tmp.Write(b)
	if err == nil {
		err = tmp.Chmod(mode)
	}
	if err == nil {
		err = tmp.Sync()
	}
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Rename(tmp.Name(), path)
	}
	if err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}

// DefaultFeed is where the Jobs page's feeds are kept on the Laptop: the same
// directory the hourly script's feed is in.
func DefaultFeed() string {
	dir := os.Getenv("POSTERN_DATA")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil || home == "" {
			return ""
		}
		dir = filepath.Join(home, ".local", "state", "postern")
	}
	return filepath.Join(dir, "jobs", "subsplash-browse.json")
}

// pdfText is the text of a notes PDF, read with pdftotext.
func pdfText(path string) (string, error) {
	out, err := exec.Command("pdftotext", path, "-").Output()
	if err != nil {
		return "", fmt.Errorf("pdftotext %s: %w", path, err)
	}
	return string(out), nil
}

// ---- the Morning RePhil series page

// series reads the series page, pressing View More until it shows every
// episode, and writes each episode not yet on file: number (the title's,
// else its place among all the episodes, the oldest being 1), title, date and
// share link. Rows are written after each load. An episode already on file
// with a share link is left as it is; an episode on file from YouTube with no
// share link (its number, or an unnumbered title of the same day) is given
// its share link only.
func (r *browseRun) series(ctx context.Context) error {
	if r.cfg.Series == "" {
		return nil
	}
	if r.cfg.Limit > 0 && r.res.Visited >= r.cfg.Limit {
		return nil
	}
	p, err := r.pb.Open(ctx, r.cfg.Series)
	if err != nil {
		return err
	}
	for {
		if err := Check(p, KindSeries); err != nil {
			return err
		}
		items, err := ParseSeries(p)
		if err != nil {
			return err
		}
		if err := r.putEpisodes(items, p.Total); err != nil {
			return err
		}
		if !p.More || (p.Total > 0 && len(items) >= p.Total) || r.shared() >= p.Total && p.Total > 0 {
			return nil
		}
		prev := len(items)
		if p, err = r.pb.More(ctx); err != nil {
			return err
		}
		if err := Check(p, KindSeries); err != nil {
			return err
		}
		if now, err := ParseSeries(p); err == nil && len(now) <= prev {
			return Stop{"unexpected page: View More added no episode at " + p.URL}
		}
	}
}

// shared is how many episodes on file hold a share link.
func (r *browseRun) shared() int {
	n := 0
	for _, row := range r.rephil {
		if row.Share != "" {
			n++
		}
	}
	return n
}

func (r *browseRun) putEpisodes(items []SeriesItem, total int) error {
	changed := false
	have := map[string]bool{}
	for _, row := range r.rephil {
		have[row.Share] = true
	}
	for idx, it := range items {
		share := shortBase + it.Code
		if have[share] {
			continue
		}
		number := it.Number
		if number == 0 {
			// Its place among all the episodes: known when the series has
			// shown them all (total), counted from the oldest.
			if total == 0 || idx >= total {
				continue
			}
			number = total - idx
			r.say("%s (%s): no number in its title; numbered %d by its place among all the episodes (1 is the oldest)", it.Title, it.Date, number)
		}
		filled := false
		for i := range r.rephil {
			row := r.rephil[i]
			if row.Share == "" && (row.Number == number && it.Number > 0 || row.Date == it.Date && (row.Number == number || RephilNumber(row.Title) == -1)) {
				r.rephil[i].Share = share
				filled = true
				r.say("%s (%s): share link added to the episode on file", it.Title, it.Date)
				break
			}
		}
		if !filled {
			r.rephil = append(r.rephil, RephilRow{Number: number, Title: it.Title, Date: it.Date, Share: share})
			r.say("%s (%s): episode written", it.Title, it.Date)
		}
		have[share] = true
		r.res.Episodes++
		changed = true
	}
	if !changed {
		return nil
	}
	if err := writeRephil(r.cfg.Rephil, r.rephil); err != nil {
		return err
	}
	return r.writeFeed()
}
