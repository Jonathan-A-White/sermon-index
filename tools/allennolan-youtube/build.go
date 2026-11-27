package main

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
	_ "time/tzdata" // the pastor's zone, on a host with no zoneinfo

	"github.com/Jonathan-A-White/sermon-index/internal/passage"
	"github.com/Jonathan-A-White/sermon-index/internal/sources"
)

// Where a row's passage came from, best first (rank below).
const (
	fromTitle        = "title"
	fromTitleChapter = "title chapter"
	fromPlaylist     = "playlist"
	fromDescription  = "description"
)

func rank(from string) int {
	switch from {
	case fromTitle:
		return 4
	case fromTitleChapter:
		return 3
	case fromPlaylist:
		return 2
	case fromDescription:
		return 1
	}
	return 0
}

// A video in a book series with no more chapters than this is a passage; a
// text over more is a whole book or a summary of one, and is no passage.
const maxChapters = 3

// Series is a playlist named by a Bible book (or a chapter of one): one of the
// book series of the channel.
type Series struct {
	Playlist
	Book      string // the OSIS name of the book the playlist is named by ("Rev")
	Scope     string // the OSIS reference of the chapter or chapters the title names ("Rev.5"), or ""
	ByChapter bool   // the title says "Chapter": a chapter number in a video's title is a chapter
}

// SeriesItems is a book series with the videos the API listed in it.
type SeriesItems struct {
	Series
	Items []Item
}

// Listing is what the channel's playlists were sorted into.
type Listing struct {
	Channel   Channel
	Playlists []Playlist // every playlist of the channel
	LeftOut   []Playlist // playlists whose title names a book but that are no book series
}

// bookWordRE finds a book's full name in a playlist title (a name as the book
// table prints it, "Psalm(s)" and "Song of Songs"), longest first. Only full
// names: an abbreviation in free text ("Is", "Am") is a word.
var bookWordRE = func() *regexp.Regexp {
	names := []string{"Psalms", "Psalm", "Song of Songs"}
	for _, b := range passage.Books() {
		names = append(names, b.Name)
	}
	sort.SliceStable(names, func(i, j int) bool { return len(names[i]) > len(names[j]) })
	for i, n := range names {
		names[i] = strings.ReplaceAll(regexp.QuoteMeta(n), `\ `, `\s+`)
	}
	return regexp.MustCompile(`(?i)\b(` + strings.Join(names, "|") + `)\b`)
}()

var (
	// A title that names a book is a book series only if it also says so: "Book
	// of", "Gospel of", "Bible Study", "Verse by Verse" or a chapter. "Genesis
	// Mysteries Explained", "Daniel and Revelation" and "Genesis: The Story of
	// Joseph" name a book and are topical.
	seriesMarkerRE = regexp.MustCompile(`(?i)\b(book of|gospel of|bible study|verse by verse|chapters?)\b`)
	chapterWordRE  = regexp.MustCompile(`(?i)\bchapters?\b`)
	// The chapter or chapters straight after the book: "Isaiah 53", "Revelation
	// Chapter 14", "Revelation Chapters 15 and 16".
	scopeRE = regexp.MustCompile(`(?i)^\s*(?:chapters?\s+)?(\d{1,3})(?:\s*(?:and|&|-|–|—|to)\s*(\d{1,3}))?\b`)
	// A timestamp straight before a passage ("20:18 Amos 1:1-2"): a line of a
	// description's table of contents, not the video's text.
	timestampBeforeRE = regexp.MustCompile(`\d{1,2}:\d{2}(?::\d{2})?\W{0,4}$`)
	privateRE         = regexp.MustCompile(`(?i)^(private|deleted) video$`)
	number            = passage.NumberPattern
	explicitRE        = regexp.MustCompile(`(?i)\b(` + passage.BookNamePattern() + `)\.?\s+(?:chapters?\s+)?([1-9]\d{0,2})(?::(` + number + `))?(?:\s*(?:[-–—]|&|and)\s*(` + number + `)(?::(` + number + `))?)?\b`)
	bareVerseRE       = regexp.MustCompile(`\(\s*([1-9]\d{0,2}):(` + number + `)(?:\s*[-–—]\s*(` + number + `)(?::(` + number + `))?)?\s*\)`)
)

// ClassifyPlaylist says whether a playlist is a book series: its title names a
// Bible book and says it is a study, a verse-by-verse, a chapter or the book
// itself. Topical playlists, one-offs, years, podcasts and shorts are not.
func ClassifyPlaylist(p Playlist) (Series, bool) {
	loc := bookWordRE.FindStringSubmatchIndex(p.Title)
	if loc == nil {
		return Series{}, false
	}
	book, ok := passage.FullBookOSIS(p.Title[loc[2]:loc[3]])
	if !ok {
		return Series{}, false
	}
	rest := p.Title[loc[1]:]
	m := scopeRE.FindStringSubmatch(rest)
	if !seriesMarkerRE.MatchString(p.Title) && m == nil {
		return Series{}, false
	}
	s := Series{Playlist: p, Book: book, ByChapter: chapterWordRE.MatchString(p.Title)}
	if m != nil {
		if ref, err := placeRef(book, m[1], "", m[2], ""); err == nil {
			if r, err := passage.Parse(ref); err == nil && r.Chapters() <= maxChapters {
				s.Scope = ref
			}
		}
	}
	return s, true
}

// placeRef writes a captured place (book, chapter, verse, end, end verse) as
// an OSIS reference the book table accepts, in its canonical form.
func placeRef(book, c1, v1, x, y string) (string, error) {
	ref, err := passage.RangeOSIS(book, c1, v1, x, y)
	if err != nil {
		return "", err
	}
	r, err := passage.Parse(ref)
	if err != nil {
		return "", err
	}
	return r.OSIS, nil
}

// place is a passage a text names.
type place struct {
	at       int
	book     string
	ref      string
	hasVerse bool
}

// places finds the passages a text names, in the order they occur: "Genesis
// 12:1-3", "Rev. 6:9-11", "Revelation 17 & 18", and, with the series' book,
// a bare "(6:12-17)". A place the book table refuses, or that spans more than
// maxChapters chapters, is left out.
func places(text, seriesBook string, description bool) []place {
	var out []place
	add := func(at int, book, c1, v1, x, y string) {
		if description {
			line := text[:at]
			if i := strings.LastIndexByte(line, '\n'); i >= 0 {
				line = line[i+1:]
			}
			if timestampBeforeRE.MatchString(line) {
				return
			}
		}
		ref, err := placeRef(book, c1, v1, x, y)
		if err != nil {
			return
		}
		if r, err := passage.Parse(ref); err != nil || r.Chapters() > maxChapters {
			return
		}
		out = append(out, place{at: at, book: book, ref: ref, hasVerse: v1 != "" || y != ""})
	}
	for _, m := range explicitRE.FindAllStringSubmatchIndex(text, -1) {
		g := func(i int) string {
			if m[2*i] < 0 {
				return ""
			}
			return text[m[2*i]:m[2*i+1]]
		}
		book, ok := passage.BookOSIS(g(1))
		if !ok {
			continue
		}
		add(m[0], book, g(2), g(3), g(4), g(5))
	}
	if seriesBook != "" {
		for _, m := range bareVerseRE.FindAllStringSubmatchIndex(text, -1) {
			g := func(i int) string {
				if m[2*i] < 0 {
					return ""
				}
				return text[m[2*i]:m[2*i+1]]
			}
			add(m[0], seriesBook, g(1), g(2), g(3), g(4))
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].at < out[j].at })
	return out
}

// VideoRef is the passage of a video in a book series, and where it came from.
// In order of trust:
//
//  1. verses in the title ("Genesis 12:1-3", "(6:12-17)" with the series' book);
//  2. a chapter in the title ("Revelation 4"), only in a series that says
//     "Chapter" in its title: in "Genesis Bible Study 5" or "Amos 6" the
//     number is the part of the study, and no chapter;
//  3. the chapter or chapters the playlist's own title names ("Isaiah 53"),
//     for a video whose title names the book too ("Slain Lamb of Revelation"):
//     a title that names no book may be any video the pastor filed there;
//  4. the one passage of the series' book, a range of verses and not one
//     verse, that the description names outside a line of timestamps (a
//     description that names two names none).
//
// Nothing is guessed from a part number, and a whole book or a summary of
// several chapters is no passage. An empty ref has no passage.
func VideoRef(it Item, s Series) (ref, from string) {
	title := places(it.Title, s.Book, false)
	for _, p := range title {
		if p.hasVerse {
			return p.ref, fromTitle
		}
	}
	if s.ByChapter {
		for _, p := range title {
			if p.book == s.Book {
				return p.ref, fromTitleChapter
			}
		}
	}
	if s.Scope != "" && namesBook(it.Title, s.Book) {
		return s.Scope, fromPlaylist
	}
	var one string
	n := 0
	for _, p := range places(it.Description, "", true) {
		if p.hasVerse && p.book == s.Book && p.ref != one {
			one = p.ref
			n++
		}
	}
	// A single verse in a description is an illustration, a range the text.
	if n == 1 && strings.Contains(one, "-") {
		return one, fromDescription
	}
	return "", ""
}

// pastorZone is where the pastor is (Tahlequah, Oklahoma): the day a video went
// up is the day on his clock, not in UTC.
var pastorZone = mustZone("America/Chicago")

func mustZone(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

// datePart is the day of an RFC 3339 instant on the pastor's clock.
func datePart(rfc3339 string) string {
	if t, err := time.Parse(time.RFC3339, rfc3339); err == nil {
		return t.In(pastorZone).Format("2006-01-02")
	}
	return ""
}

// Skip is a video of a book series that is no row.
type Skip struct{ ID, Title, Why string }

const (
	whyNoPassage = "no passage: its title and description name none the tool can read"
	whyPrivate   = "private or deleted: the API gives no title"
	whyNoID      = "no video id"
)

// Result is what the channel's book playlists became.
type Result struct {
	Channel   Channel
	Units     int // quota units spent
	Playlists int // every playlist of the channel
	Series    []Series
	LeftOut   []Playlist

	Entries int // playlist entries read; a video in two playlists is two
	Videos  int // distinct videos
	Rows    []sources.Row
	Skipped []Skip
	// By counts the rows by where the passage came from.
	By          map[string]int
	First, Last string // the dates of the oldest and the newest row
	Stopped     bool   // the stop rule applied: no rows were written
}

// Private is the number of skipped videos that are private or deleted.
func (r *Result) Private() int {
	n := 0
	for _, s := range r.Skipped {
		if s.Why == whyPrivate || s.Why == whyNoID {
			n++
		}
	}
	return n
}

// Share is the share of the videos the API lists by name that have a passage.
func (r *Result) Share() float64 {
	listable := r.Videos - r.Private()
	if listable <= 0 {
		return 0
	}
	return float64(len(r.Rows)) / float64(listable)
}

// Build makes the rows: one a distinct video of the book playlists, with the
// best passage any of its playlists gives it.
func Build(l *Listing, series []SeriesItems) *Result {
	res := &Result{Channel: l.Channel, Playlists: len(l.Playlists), LeftOut: l.LeftOut, By: map[string]int{}}
	type video struct {
		item   Item
		series Series
		ref    string
		from   string
	}
	var order []string
	vids := map[string]*video{}
	noID := 0
	for _, s := range series {
		res.Series = append(res.Series, s.Series)
		for _, it := range s.Items {
			res.Entries++
			if it.VideoID == "" {
				noID++
				res.Videos++
				res.Skipped = append(res.Skipped, Skip{"(none)", it.Title, whyNoID})
				continue
			}
			v, seen := vids[it.VideoID]
			if !seen {
				v = &video{item: it, series: s.Series}
				vids[it.VideoID] = v
				order = append(order, it.VideoID)
				res.Videos++
			}
			if privateRE.MatchString(strings.TrimSpace(it.Title)) {
				continue
			}
			ref, from := VideoRef(it, s.Series)
			if rank(from) > rank(v.from) {
				v.ref, v.from, v.series = ref, from, s.Series
			}
		}
	}
	type dated struct {
		at  string
		row sources.Row
	}
	var rows []dated
	for _, id := range order {
		v := vids[id]
		if privateRE.MatchString(strings.TrimSpace(v.item.Title)) {
			res.Skipped = append(res.Skipped, Skip{id, v.item.Title, whyPrivate})
			continue
		}
		if v.ref == "" {
			res.Skipped = append(res.Skipped, Skip{id, v.item.Title, whyNoPassage})
			continue
		}
		row := sources.Row{
			ID: id, Title: v.item.Title, URL: watchURL + id,
			Kind: ptr("link"), Preacher: ptr(preacher), Ref: ptr(v.ref),
		}
		if d := datePart(v.item.Published); d != "" {
			row.Date = ptr(d)
		}
		row.Notes = ptr(note(v.from, v.series))
		res.By[v.from]++
		rows = append(rows, dated{v.item.Published, row})
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].at != rows[j].at {
			return rows[i].at < rows[j].at
		}
		return rows[i].row.ID < rows[j].row.ID
	})
	for _, r := range rows {
		res.Rows = append(res.Rows, r.row)
		if r.row.Date != nil {
			d := *r.row.Date
			if res.First == "" || d < res.First {
				res.First = d
			}
			if d > res.Last {
				res.Last = d
			}
		}
	}
	return res
}

// note says where a row's passage came from; it holds nothing of a description.
func note(from string, s Series) string {
	switch from {
	case fromTitle:
		return fmt.Sprintf("Passage named in the video's title (playlist %q).", s.Title)
	case fromTitleChapter:
		return fmt.Sprintf("Chapter named in the video's title, in a playlist of chapters (%q).", s.Title)
	case fromPlaylist:
		return fmt.Sprintf("The video's title names no passage; ref is the chapter its playlist %q is named by.", s.Title)
	default:
		return fmt.Sprintf("The one passage of the book that the video's description names, in the playlist %q.", s.Title)
	}
}

func ptr[T any](v T) *T { return &v }

// namesBook says whether a title names the book (by its full name).
func namesBook(title, book string) bool {
	for _, m := range bookWordRE.FindAllString(title, -1) {
		if b, ok := passage.FullBookOSIS(m); ok && b == book {
			return true
		}
	}
	return false
}
