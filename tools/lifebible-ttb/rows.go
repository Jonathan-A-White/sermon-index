package main

import (
	"strings"

	"github.com/Jonathan-A-White/sermon-index/internal/passage"
	"github.com/Jonathan-A-White/sermon-index/internal/sources"
)

const preacher = "Pastor Paul LeBoutillier"

// Why a sitemap address is not a row.
const (
	ReasonList      = "the Through the Bible list page"
	ReasonBook      = "a book page (the list of a book's teachings)"
	ReasonFetch     = "the page could not be read"
	ReasonNoPassage = "no passage in the page title"
	ReasonNoVideo   = "no YouTube video on the page"
	ReasonRepeat    = "the same address (or row id) as an earlier one in the sitemap"
)

// PageResult is what was read of one address: the page, or why it was not.
type PageResult struct {
	Page Page
	Err  string
}

// Skip is a sitemap address that is not a row, with why and, when the page
// was read, its title.
type Skip struct {
	URL, Reason, Title, Err string
}

// Report is what a run found: the counts the tool prints and the lists
// REPORT.md carries.
type Report struct {
	Pages   int // through-the-bible addresses in the sitemap
	Rows    int
	Wide    int // rows whose ref spans more than 3 chapters but are not series
	Series  int // rows over more than 3 chapters
	Skipped []Skip
}

// BuildRows makes one row of each page address that has a passage in its
// title and a YouTube video, and names every other address in Skipped. urls
// is the sitemap's through-the-bible addresses; pages holds what was read of
// each page.
func BuildRows(urls []string, pages map[string]PageResult) ([]sources.Row, *Report) {
	rep := &Report{Pages: len(urls)}
	var rows []sources.Row
	seen := map[string]bool{}
	skip := func(u, reason, title, err string) {
		rep.Skipped = append(rep.Skipped, Skip{URL: u, Reason: reason, Title: title, Err: err})
	}
	for _, u := range urls {
		slug := slugOf(u)
		switch {
		case slug == "":
			skip(u, ReasonList, "", "")
			continue
		case !strings.Contains(slug, "/"):
			skip(u, ReasonBook, "", "")
			continue
		}
		res, ok := pages[u]
		if !ok || res.Err != "" {
			err := res.Err
			if !ok {
				err = "not read"
			}
			skip(u, ReasonFetch, "", err)
			continue
		}
		ref, shown, series, ok := ParseTitle(res.Page.Title)
		if !ok {
			skip(u, ReasonNoPassage, res.Page.Title, "")
			continue
		}
		if res.Page.YouTube == "" {
			skip(u, ReasonNoVideo, res.Page.Title, "")
			continue
		}
		id := strings.ReplaceAll(slug, "/", ".")
		if seen[id] {
			skip(u, ReasonRepeat, res.Page.Title, "")
			continue
		}
		seen[id] = true

		row := sources.Row{
			ID:       id,
			Title:    shown,
			Ref:      ptr(ref),
			URL:      u,
			AltURL:   ptr("https://www.youtube.com/watch?v=" + res.Page.YouTube),
			Kind:     ptr("link"),
			Preacher: ptr(preacher),
		}
		if series {
			// A whole book of more than 3 chapters (docs/adding-a-source.md): a
			// series, kept and found by its title, with its first chapter as ref.
			row.Series = ptr(true)
			row.Notes = ptr("Series: the title names the whole book, so the row matches no passage and ref is its first chapter.")
			rep.Series++
		}
		if !series {
			if rg, err := passage.Parse(ref); err == nil && rg.Chapters() > sources.WideChapters {
				rep.Wide++
			}
		}
		rows = append(rows, row)
	}
	rep.Rows = len(rows)
	return rows, rep
}

func ptr[T any](v T) *T { return &v }
