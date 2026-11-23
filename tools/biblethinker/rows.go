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

// Series is one of Mike Winger's book-by-book series on the site: the slug of
// its archive page, the OSIS name of the book it goes through and the name
// the site prints for it.
type Series struct {
	Slug string
	Book string
	Name string
}

// BookSeries are the book series through the Bible the site has. The other
// categories of the site (Q&A sessions, livestreams, topics, debates,
// interviews) are not book series and are not read.
var BookSeries = []Series{
	{Slug: "1-peter", Book: "1Pet", Name: "1 Peter"},
	{Slug: "hebrews", Book: "Heb", Name: "Hebrews"},
	{Slug: "mark", Book: "Mark", Name: "Mark"},
	{Slug: "romans", Book: "Rom", Name: "Romans"},
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
	Rows     []Row    // in date order
	Skips    []Skip   // Seen - len(Rows) of them
	Disagree []string // a post whose title and slug name different passages (the title is used)
	NoDate   int      // rows whose post the REST API gave no date for
}

// BuildSeries makes a row of every episode of the series that has a passage
// and a YouTube id of its own, and names each one that has not. dates maps a
// post id to its date, "2015-09-13".
func BuildSeries(s Series, eps []Episode, dates map[int]string) SeriesResult {
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
		var reasons []string
		ref, cited, err := RefFromTitle(s.Book, e.Title)
		fromSlug := false
		slugRefs := RefsFromSlug(s.Book, e.Slug)
		switch {
		case err != nil:
			reasons = append(reasons, "the passage in the title is not in "+s.Name+": "+err.Error())
		case ref == "" && len(slugRefs) == 1:
			ref, fromSlug = slugRefs[0], true
		case ref == "" && len(slugRefs) > 1:
			reasons = append(reasons, "no passage in the title, and the digits ending the post's address could be any of "+strings.Join(slugRefs, ", "))
		case ref == "":
			reasons = append(reasons, "no passage in the title or the post's address")
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
