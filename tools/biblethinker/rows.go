package main

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/Jonathan-A-White/sermon-index/internal/passage"
)

const (
	preacher = "Mike Winger"
	siteRoot = "https://biblethinker.org"
)

// Series is one of the site's categories the tool reads: the slug of its
// archive page, the name the site prints for it and, for a book-by-book
// series, the OSIS name of the book it goes through. A topical category has no
// book ("") and a post's passage may be in any book.
type Series struct {
	Slug string
	Book string
	Name string
}

// Topical reports whether the category is not a series through one book.
func (s Series) Topical() bool { return s.Book == "" }

// Categories are the categories the tool reads, in the order it reads them (a
// post in two is the row of the first): the four book series, then the
// Governor's 2026-11-24 widening to every post that names a passage.
var Categories = []Series{
	{Slug: "1-peter", Book: "1Pet", Name: "1 Peter"},
	{Slug: "hebrews", Book: "Heb", Name: "Hebrews"},
	{Slug: "mark", Book: "Mark", Name: "Mark"},
	{Slug: "romans", Book: "Rom", Name: "Romans"},
	{Slug: "how-to-find-jesus-in-the-old-testament", Name: "How to Find Jesus in the Old Testament"},
	{Slug: "wisdom-in-the-word", Name: "Wisdom in the Word"},
	{Slug: "understanding-the-old-testament-law", Name: "Understanding the Old Testament Law"},
	{Slug: "past-livestreams-and-misc-topical-teachings", Name: "Past Livestreams & Misc. Topical Teachings"},
}

// NotReadCategories are the categories whose posts are left out (Q&A, debates,
// interviews, reactions and reviews, worship songs), by slug.
var NotReadCategories = []string{
	"past-qa-sessions",
	"debates",
	"interviews",
	"thinking-biblically-reviews",
	"worship-songs",
	"christians-react-to-20-atheist-arguments-against-god",
}

// URL is the address of the series' archive page.
func (s Series) URL() string { return siteRoot + "/" + s.Slug + "/" }

// Row is a line of rows.jsonl (docs/sources.md), in the order the files print.
type Row struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Ref       *string  `json:"ref,omitempty"`
	URL       string   `json:"url"`
	AltURL    *string  `json:"alt_url,omitempty"`
	Kind      *string  `json:"kind,omitempty"`
	Preacher  *string  `json:"preacher,omitempty"`
	Date      *string  `json:"date,omitempty"`
	DurationS *int     `json:"duration_s,omitempty"`
	Cited     []string `json:"cited,omitempty"`
	Notes     *string  `json:"notes,omitempty"`
	Series    *bool    `json:"series,omitempty"`
}

// Skip is an episode of a series archive that is not a row, and why.
type Skip struct {
	Slug   string
	Title  string
	URL    string
	Reason string
}

// SeriesResult is what one series came to.
type SeriesResult struct {
	Series   Series
	Seen     int      // articles on the series' archive pages
	Dups     int      // of them, posts an earlier category already has (its row is that category's)
	Rows     []Row    // in date order
	Skips    []Skip   // Seen - Dups - len(Rows) of them
	Disagree []string // a post whose title and slug name different passages (the title is used)
	NoDate   int      // rows whose post the REST API gave no date for
}

// SplitSeen cuts the posts of a category into those no earlier category has
// (fresh) and those one has (dups), by address, and marks the fresh as seen.
func SplitSeen(eps []Episode, seen map[string]bool) (fresh, dups []Episode) {
	for _, e := range eps {
		if seen[e.URL] {
			dups = append(dups, e)
			continue
		}
		seen[e.URL] = true
		fresh = append(fresh, e)
	}
	return fresh, dups
}

// BuildSeries makes a row of every episode of the series that has a passage
// and a YouTube id of its own, and names each one that has not. dates maps a
// post id to its date, "2015-09-13". The passage is read from the title, else
// from the digits ending the post's address and, for a post that names none,
// from the post's own text, which bodyOf returns (bodyOf may be nil); bodyOf
// is asked only for a post the title and the address leave without a passage.
func BuildSeries(s Series, eps []Episode, dates map[int]string, bodyOf func(Episode) string) SeriesResult {
	res := SeriesResult{Series: s, Seen: len(eps)}
	type dated struct {
		row  Row
		post int
	}
	var rows []dated
	for _, e := range eps {
		skip := func(reason string) {
			res.Skips = append(res.Skips, Skip{Slug: e.Slug, Title: e.Title, URL: e.URL, Reason: reason})
		}
		if !hasCategory(e, s.Slug) {
			skip(fmt.Sprintf("not in the %s series (its categories: %s)", s.Slug, strings.Join(e.Categories, ", ")))
			continue
		}
		if no := notReadCategory(e); no != "" {
			skip(fmt.Sprintf("left out: also filed under the category %s, which is not read", no))
			continue
		}
		var reasons []string
		ref, cited, slugRefs, err := passageOfTitle(s, e)
		fromSlug, fromBody := false, false
		switch {
		case err != nil:
			reasons = append(reasons, "the passage in the title is not in "+s.Name+": "+err.Error())
		case ref == "" && len(slugRefs) == 1:
			ref, fromSlug = slugRefs[0], true
		case ref == "":
			if bodyOf != nil && e.YouTubeID != "" {
				ref = refFromBody(s, bodyOf(e))
			}
			switch {
			case ref != "":
				fromBody = true
			case len(slugRefs) > 1:
				reasons = append(reasons, "no passage in the title or the post's text (body), and the digits ending the post's address could be any of "+strings.Join(slugRefs, ", "))
			default:
				reasons = append(reasons, "no passage in the title, the post's address or the post's text (body)")
			}
		case len(slugRefs) > 0 && !contains(slugRefs, ref):
			res.Disagree = append(res.Disagree, fmt.Sprintf("%s: the title says %s, the address says %s; the title is used", e.Slug, ref, strings.Join(slugRefs, " or ")))
		}
		if e.YouTubeID == "" {
			reasons = append(reasons, "no YouTube id (the post embeds no video)")
		}
		if len(reasons) > 0 {
			skip(strings.Join(reasons, "; "))
			continue
		}
		note := fmt.Sprintf("Series: %s (%s). The YouTube id is the one embedded in the post.", s.Name, s.URL())
		if fromSlug {
			note += " The title prints no passage; it is read from the digits ending the post's address."
		}
		if fromBody {
			note += " The title and the post's address print no passage; it is the first one the post's own text names."
		}
		r := Row{
			ID:       e.Slug,
			Title:    e.Title,
			Ref:      &ref,
			URL:      e.URL,
			AltURL:   strp("https://www.youtube.com/watch?v=" + e.YouTubeID),
			Kind:     strp("link"),
			Preacher: strp(preacher),
			Cited:    cited,
			Notes:    &note,
		}
		if d, ok := dates[e.PostID]; ok && len(d) >= 10 {
			r.Date = strp(d[:10])
		} else {
			res.NoDate++
		}
		if e.DurationS > 0 {
			d := e.DurationS
			r.DurationS = &d
		}
		if rg, err := passage.Parse(ref); err == nil && rg.Chapters() > 3 {
			t := true
			r.Series = &t
		}
		rows = append(rows, dated{r, e.PostID})
	}
	sort.SliceStable(rows, func(i, j int) bool {
		a, b := rows[i].row.Date, rows[j].row.Date
		switch {
		case a != nil && b != nil && *a != *b:
			return *a < *b
		case (a == nil) != (b == nil):
			return a != nil // a row with no date last
		}
		return rows[i].post < rows[j].post
	})
	for _, d := range rows {
		res.Rows = append(res.Rows, d.row)
	}
	return res
}

// passageOfTitle reads the passage the title prints and the passages the
// digits ending the address can mean: in the series' book for a book series,
// in any book for a topical category.
func passageOfTitle(s Series, e Episode) (ref string, cited, slugRefs []string, err error) {
	if !s.Topical() {
		ref, cited, err = RefFromTitle(s.Book, e.Title)
		return ref, cited, RefsFromSlug(s.Book, e.Slug), err
	}
	if refs := RefsInText(e.Title, true); len(refs) > 0 {
		ref = refs[0]
	}
	return ref, nil, RefsFromSlugAnyBook(e.Slug), nil
}

// refFromBody is the first passage the post's text names (in the series' book
// for a book series), or "".
func refFromBody(s Series, body string) string {
	for _, r := range RefsInText(body, true) {
		if s.Topical() || strings.HasPrefix(r, s.Book+".") {
			return r
		}
	}
	return ""
}

// notReadCategory is the first of the post's categories that is not read, or "".
func notReadCategory(e Episode) string {
	for _, c := range NotReadCategories {
		if contains(e.Categories, c) {
			return c
		}
	}
	return ""
}

func hasCategory(e Episode, slug string) bool { return contains(e.Categories, slug) }

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func strp(s string) *string { return &s }

// WriteRows writes one JSON object a line.
func WriteRows(w io.Writer, rows []Row) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	for _, r := range rows {
		if err := enc.Encode(r); err != nil {
			return err
		}
	}
	return nil
}
