// Command brookfield-youtube writes the sermon source sources/brookfield-first
// and the Morning RePhil list data/morning-rephil.jsonl from the YouTube
// channel @BrookfieldFirst, by YouTube's official Data API (v3).
//
//	YOUTUBE_API_KEY="$(mw secrets get youtube_api_key)" go run ./tools/brookfield-youtube
//
// It resolves the handle with channels.list, walks the channel's uploads
// playlist with playlistItems.list (50 a page) and keeps, for each video, only
// facts: its title as YouTube prints it, its date, the preacher and the
// passage where the title or the description names one, and the watch link.
// Nothing is downloaded: no video, no audio, no caption, no page. The
// description is read for a preacher and a passage and is never written out.
//
// The key is read from the environment variable YOUTUBE_API_KEY alone. It is
// sent in a request header, never written to a file, a log or the report, and
// any error text that would hold it has it blanked.
//
// It rewrites rows.jsonl whole, but alt_url is not its field: the one that
// tools/subsplash-links wrote to a row is read from the file first and kept on
// the row with the same id. A row the channel no longer lists loses it; a new
// row has none. url (the watch link) is always the API's.
//
// --merge-only applies passages.tsv to the rows already in rows.jsonl with no
// call to the API and no key (MergeRowsFile), and brings the counts in
// source.toml and REPORT.md to the list's. scripts/subsplash-hourly.sh runs it
// after tools/subsplash-links --read-messages has added the Governor's
// messages about sermons with no notes to passages.tsv (the lines with a row
// id, which are matched by that id and not by date).
//
// A second run over the same channel writes the same bytes (the order is
// publish order, oldest first), so a diff shows only new videos.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Jonathan-A-White/sermon-index/internal/passage"
	"github.com/Jonathan-A-White/sermon-index/internal/sources"
)

const (
	keyVar        = "YOUTUBE_API_KEY"
	defaultHandle = "@BrookfieldFirst"
	sourceID      = "brookfield-first"
	watchURL      = "https://www.youtube.com/watch?v="
)

func main() {
	handle := flag.String("handle", defaultHandle, "the channel's handle")
	out := flag.String("out", "sources/"+sourceID, "the source directory to write source.toml, rows.jsonl and REPORT.md in")
	rephil := flag.String("rephil", "data/morning-rephil.jsonl", "the Morning RePhil list to write")
	report := flag.String("report", "", "write the report here (default: REPORT.md in --out)")
	list := flag.String("passages", "", "the church's list of its sermons, merged by date into the rows (default: passages.tsv in --out; \"none\" merges nothing)")
	mergeOnly := flag.Bool("merge-only", false, "apply passages.tsv to the rows already in rows.jsonl, with no call to the API and no key")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "usage: "+keyVar+"=<key> brookfield-youtube [--handle @BrookfieldFirst] [--out sources/brookfield-first] [--rephil data/morning-rephil.jsonl] [--report <file>] [--passages <file>|none]\n       brookfield-youtube --merge-only [--out sources/brookfield-first] [--passages <file>]")
		os.Exit(2)
	}
	if *mergeOnly {
		list := *list
		if list == "" || list == "none" {
			list = filepath.Join(*out, "passages.tsv")
		}
		rep := *report
		if rep == "" {
			rep = filepath.Join(*out, "REPORT.md")
		}
		if err := runMergeOnly(*out, list, rep, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, "brookfield-youtube:", err)
			os.Exit(1)
		}
		return
	}
	key, err := keyFromEnv(os.Getenv)
	if err != nil {
		fmt.Fprintln(os.Stderr, "brookfield-youtube:", err)
		os.Exit(1)
	}
	if *report == "" {
		*report = filepath.Join(*out, "REPORT.md")
	}
	switch *list {
	case "":
		// A refresh must not wipe the merged fields, so a list that is
		// missing from its place is an error and not a run without it.
		*list = filepath.Join(*out, "passages.tsv")
	case "none":
		*list = ""
	}
	if *list != "" {
		if _, err := os.Stat(*list); err != nil {
			fmt.Fprintln(os.Stderr, "brookfield-youtube:", err, "(--passages none runs without the church's list)")
			os.Exit(1)
		}
	}
	c := &Client{Key: key, HTTP: &http.Client{Timeout: time.Minute}}
	if err := run(context.Background(), c, *handle, *out, *rephil, *report, *list, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "brookfield-youtube:", err)
		os.Exit(1)
	}
}

// runMergeOnly applies the church's list to the rows in out/rows.jsonl without
// the API (see MergeRowsFile) and brings the counts in
// out/source.toml and the report to the list's. It prints what it did and any
// entry it could not merge.
func runMergeOnly(out, listPath, reportPath string, stdout io.Writer) error {
	rowsPath := filepath.Join(out, "rows.jsonl")
	m, changed, err := MergeRowsFile(rowsPath, listPath)
	if err != nil {
		return err
	}
	for _, p := range m.Problems {
		fmt.Fprintf(stdout, "merge: %s %q: not merged: %s\n", p.Entry.Date, p.Entry.Title, p.Why)
	}
	patch := func(path string, f func(string, *Merge) string) error {
		b, err := os.ReadFile(path)
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		if got := f(string(b), m); got != string(b) {
			return os.WriteFile(path, []byte(got), 0o644)
		}
		return nil
	}
	if err := patch(filepath.Join(out, "source.toml"), renewSourceTOML); err != nil {
		return err
	}
	if err := patch(reportPath, renewReport); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "merge: %d entries in the list, %d merged (%d from the Governor's messages), %d row(s) changed\n", m.Entries, m.Merged, m.FromMessages, changed)
	return nil
}

// keyFromEnv reads the API key from the environment, or says plainly that it
// is not there. The key itself is never part of a message.
func keyFromEnv(getenv func(string) string) (string, error) {
	k := strings.TrimSpace(getenv(keyVar))
	if k == "" {
		return "", errors.New(keyVar + " is empty: run it as " + keyVar + `="$(mw secrets get youtube_api_key)" go run ./tools/brookfield-youtube`)
	}
	return k, nil
}

// run writes the source. listPath is the church's list (passages.tsv) to merge
// by date into the YouTube rows; empty merges nothing.
func run(ctx context.Context, c *Client, handle, out, rephilPath, reportPath, listPath string, stdout io.Writer) error {
	playlist, err := c.UploadsPlaylist(ctx, handle)
	if err != nil {
		return err
	}
	items, err := c.Uploads(ctx, playlist)
	if err != nil {
		return err
	}
	res := Build(items)
	res.Units = c.Units
	res.Handle = handle
	if listPath != "" {
		l, err := ReadList(listPath)
		if err != nil {
			return err
		}
		res.Merge = MergeList(res.Rows, l)
		res.recount()
	}
	// alt_url is written by tools/subsplash-links, not by this tool: a
	// regeneration keeps each row's, matched on the row id.
	alts, err := ReadAltURLs(filepath.Join(out, "rows.jsonl"))
	if err != nil {
		return err
	}
	KeepAltURLs(res.Rows, alts)

	if err := os.MkdirAll(out, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(rephilPath), 0o755); err != nil {
		return err
	}
	var rows bytes.Buffer
	for _, r := range res.Rows {
		if err := writeLine(&rows, r); err != nil {
			return err
		}
	}
	var rp bytes.Buffer
	for _, r := range res.Rephil {
		if err := writeLine(&rp, r); err != nil {
			return err
		}
	}
	files := []struct {
		path string
		body []byte
	}{
		{filepath.Join(out, "rows.jsonl"), rows.Bytes()},
		{rephilPath, rp.Bytes()},
		{filepath.Join(out, "source.toml"), []byte(res.SourceTOML())},
	}
	if reportPath != "" {
		files = append(files, struct {
			path string
			body []byte
		}{reportPath, []byte(res.Markdown())})
	}
	for _, f := range files {
		if err := os.WriteFile(f.path, f.body, 0o644); err != nil {
			return err
		}
	}
	fmt.Fprint(stdout, res.Summary())
	return nil
}

// writeLine writes v as one JSON line, without HTML escaping.
func writeLine(buf *bytes.Buffer, v any) error {
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

// RephilRow is a line of data/morning-rephil.jsonl.
type RephilRow struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Date   string `json:"date"`
	URL    string `json:"url"`
}

// Skip is an entry of the playlist that is not a video that can be listed.
type Skip struct{ ID, Title, Why string }

// Bad is a passage a title or description names that the book table refuses.
type Bad struct{ ID, Text string }

// Result is what a channel's playlist became.
type Result struct {
	Handle string
	Units  int // quota units spent

	Seen    int
	Skipped []Skip
	Rows    []sources.Row
	Rephil  []RephilRow
	// Published maps a video id to the instant it was published (RFC 3339).
	Published map[string]string

	WithPreacher, WithPassage, SeriesRows int
	TitleDates                            int // rows dated by the date their title prints
	DescriptionTitles                     int // descriptions that print a sermon title (not kept)
	BadRefs                               []Bad
	Unnumbered                            []string // RePhil ids numbered by position
	RepeatedNumbers                       []int    // RePhil numbers that two videos carry
	First, Last                           string   // the dates of the oldest and the newest video

	// Merge is what the church's list did to Rows; nil when none was given.
	Merge *Merge
}

var (
	privateTitle = regexp.MustCompile(`(?i)^(private|deleted) video$`)
	rephilRE     = regexp.MustCompile(`(?i)morning\s+rephil`)
	rephilNumRE  = regexp.MustCompile(`(?i)morning\s+rephil[^\d]{0,5}(\d+)`)
	titleDateRE  = regexp.MustCompile(`\b(January|February|March|April|May|June|July|August|September|October|November|December)\s+(\d{1,2}),\s+(\d{4})\b`)
	descTitleRE  = regexp.MustCompile(`(?i)\bsermon title:`)
	preacherRE   = regexp.MustCompile(`\((?:Pastor|Pastors|Rev\.?|Reverend|Dr\.?|Elder|Bishop)\s+([A-Z][A-Za-z.'’ -]{1,40})\)`)
)

// Build sorts the playlist's entries into sermons and Morning RePhil
// episodes, oldest first.
func Build(items []Item) *Result {
	res := &Result{Seen: len(items), Published: map[string]string{}}
	seen := map[string]bool{}
	var kept []Item
	for _, it := range items {
		switch {
		case it.VideoID == "":
			res.Skipped = append(res.Skipped, Skip{"(none)", it.Title, "no video id"})
		case privateTitle.MatchString(strings.TrimSpace(it.Title)):
			res.Skipped = append(res.Skipped, Skip{it.VideoID, it.Title, "private or deleted: the API gives no title"})
		case seen[it.VideoID]:
			res.Skipped = append(res.Skipped, Skip{it.VideoID, it.Title, "listed twice in the playlist"})
		default:
			seen[it.VideoID] = true
			kept = append(kept, it)
		}
	}
	sort.SliceStable(kept, func(i, j int) bool {
		if kept[i].Published != kept[j].Published {
			return kept[i].Published < kept[j].Published
		}
		return kept[i].VideoID < kept[j].VideoID
	})

	var dates []string
	numbers := map[int]int{}
	for _, it := range kept {
		res.Published[it.VideoID] = it.Published
		pubDate := datePart(it.Published)
		if rephilRE.MatchString(it.Title) {
			n := len(res.Rephil) + 1
			if m := rephilNumRE.FindStringSubmatch(it.Title); m != nil {
				n, _ = strconv.Atoi(m[1])
			} else {
				res.Unnumbered = append(res.Unnumbered, it.VideoID)
			}
			numbers[n]++
			res.Rephil = append(res.Rephil, RephilRow{n, it.Title, pubDate, watchURL + it.VideoID})
			dates = append(dates, pubDate)
			continue
		}
		row := sources.Row{ID: it.VideoID, Title: it.Title, URL: watchURL + it.VideoID, Kind: ptr("link")}
		date := pubDate
		if d := titleDate(it.Title); d != "" {
			date = d
			res.TitleDates++
		}
		if date != "" {
			row.Date = ptr(date)
			dates = append(dates, date)
		}
		if descTitleRE.MatchString(it.Description) {
			res.DescriptionTitles++
		}
		var notes []string
		if p := textPreacher(it.Title); p != "" {
			row.Preacher = ptr(p)
			notes = append(notes, "Preacher named in the video's title.")
		} else if p := textPreacher(it.Description); p != "" {
			row.Preacher = ptr(p)
			notes = append(notes, "Preacher named in the video's description.")
		}
		where := "title"
		refs, bad := textPassages(it.Title)
		if len(refs) == 0 {
			where = "description"
			var bad2 []string
			refs, bad2 = textPassages(it.Description)
			bad = append(bad, bad2...)
		}
		for _, b := range bad {
			res.BadRefs = append(res.BadRefs, Bad{it.VideoID, b})
		}
		if len(refs) > 0 {
			setPassage(&row, refs, where, &notes)
			res.WithPassage++
			if row.Series != nil {
				res.SeriesRows++
			}
		}
		if row.Preacher != nil {
			res.WithPreacher++
		}
		if len(notes) > 0 {
			row.Notes = ptr(strings.Join(notes, " "))
		}
		res.Rows = append(res.Rows, row)
	}
	sort.Strings(dates)
	if len(dates) > 0 {
		res.First, res.Last = dates[0], dates[len(dates)-1]
	}
	for n, c := range numbers {
		if c > 1 {
			res.RepeatedNumbers = append(res.RepeatedNumbers, n)
		}
	}
	sort.Ints(res.RepeatedNumbers)
	return res
}

func ptr[T any](v T) *T { return &v }

// datePart is the day of an RFC 3339 instant, in UTC as YouTube gives it.
func datePart(rfc3339 string) string {
	if t, err := time.Parse(time.RFC3339, rfc3339); err == nil {
		return t.UTC().Format("2006-01-02")
	}
	return ""
}

// titleDate reads "August 23, 2020" in a title as 2020-08-23: the Sunday the
// service was held, which the video's publish instant (UTC, after midnight)
// can miss by a day.
func titleDate(title string) string {
	m := titleDateRE.FindStringSubmatch(title)
	if m == nil {
		return ""
	}
	t, err := time.Parse("January 2, 2006", fmt.Sprintf("%s %s, %s", m[1], m[2], m[3]))
	if err != nil {
		return ""
	}
	return t.Format("2006-01-02")
}

// setPassage gives the row its first passage as the text (a series, when it
// spans more than 3 chapters, takes its first chapter and no passage of its
// own) and the others as cited.
func setPassage(row *sources.Row, refs []string, where string, notes *[]string) {
	first, err := passage.Parse(refs[0])
	if err != nil {
		return
	}
	text := first.OSIS
	if first.Chapters() > 3 {
		row.Series = ptr(true)
		b := passage.Books()[first.Lo/1000000-1]
		text = fmt.Sprintf("%s.%d", b.OSIS, first.Lo/1000%1000)
		*notes = append(*notes, fmt.Sprintf("Series: the video's %s names %s; ref is its first chapter.", where, first.OSIS))
	} else {
		*notes = append(*notes, fmt.Sprintf("Passage named in the video's %s.", where))
	}
	row.Ref = ptr(text)
	for _, r := range refs[1:] {
		if r != refs[0] {
			row.Cited = append(row.Cited, r)
		}
	}
}

// textPreacher reads a preacher out of "(Pastor Phil Morgan)": the name,
// without the honorific. Anything else in the text is left unread.
func textPreacher(s string) string {
	if m := preacherRE.FindStringSubmatch(s); m != nil {
		return strings.TrimSpace(m[1])
	}
	return ""
}

// passageRE finds a passage in a text: a book as passage.BookOSIS reads it
// (full name, "Psalm", "Song of Songs" or a usual abbreviation, "2 Cor."), a
// chapter, and verses.
var passageRE = regexp.MustCompile(`(?i)\b(` + passage.BookNamePattern() + `)\.?\s+(\d{1,3})(?::(` + passage.NumberPattern + `))?(?:\s*[-–—]\s*(` + passage.NumberPattern + `)(?::(` + passage.NumberPattern + `))?)?\b`)

// textPassages finds the passages a text names ("Genesis 18:16-33", "Psalm
// 23", "Mark 6:35–44") as OSIS references, in the order they occur, and lists
// separately the ones the book table refuses ("Romans 99:1"). A number with
// no book before it is not read.
func textPassages(s string) (refs, bad []string) {
	for _, m := range passageRE.FindAllStringSubmatch(s, -1) {
		book, _ := passage.BookOSIS(m[1])
		ref, err := passage.RangeOSIS(book, m[2], m[3], m[4], m[5])
		if err == nil {
			var r passage.Range
			r, err = passage.Parse(ref)
			ref = r.OSIS
		}
		if err != nil {
			bad = append(bad, strings.TrimSpace(m[0]))
			continue
		}
		dup := false
		for _, have := range refs {
			dup = dup || have == ref
		}
		if !dup {
			refs = append(refs, ref)
		}
	}
	return refs, bad
}
