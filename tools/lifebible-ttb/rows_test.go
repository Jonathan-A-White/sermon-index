package main

import (
	"strings"
	"testing"
)

func TestBuildRows(t *testing.T) {
	urls := []string{
		ttbPrefix,
		ttbPrefix + "/isaiah",
		ttbPrefix + "/isaiah/1",
		ttbPrefix + "/romans/ch-5-verse-1-verse-11",
		ttbPrefix + "/genesis/1",
		ttbPrefix + "/matthew/in-depth/7-1",
		ttbPrefix + "/john/gone",
		ttbPrefix + "/acts/1",
	}
	pages := map[string]PageResult{
		urls[2]: {Page: Page{Title: "Isaiah 1-4:1: God confronts the Kingdom of Judah", YouTube: "ynUYAm1-W6k"}},
		urls[3]: {Page: Page{Title: "Romans 5 (Part 1) :1-11 • Justified by faith"}},
		urls[4]: {Page: Page{Title: "Genesis: Through the Bible", YouTube: "aaaaaaaaaaa"}},
		urls[5]: {Page: Page{Title: "Matthew 7 (Part 1) :1-5 • Judging Others", YouTube: "5l_txKdvc3s"}},
		urls[6]: {Err: "404 Not Found"},
		urls[7]: {Page: Page{Title: "Lamentations: Crying Out in Our Brokenness", YouTube: "bbbbbbbbbbb"}},
	}
	rows, rep := BuildRows(urls, pages)
	if len(rows) != 3 {
		t.Fatalf("%d rows, want 3: %+v", len(rows), rows)
	}
	r := rows[0]
	if r.ID != "isaiah.1" || r.Title != "Isaiah 1-4:1: God confronts the Kingdom of Judah" ||
		r.Ref == nil || *r.Ref != "Isa.1.1-Isa.4.1" ||
		r.URL != "https://www.lifebibleministry.com/through-the-bible/isaiah/1" ||
		r.AltURL == nil || *r.AltURL != "https://www.youtube.com/watch?v=ynUYAm1-W6k" ||
		r.Preacher == nil || *r.Preacher != "Pastor Paul LeBoutillier" ||
		r.Kind == nil || *r.Kind != "link" || r.Series != nil || r.Date != nil {
		t.Errorf("isaiah row: %+v", r)
	}
	if rows[1].ID != "matthew.in-depth.7-1" {
		t.Errorf("second row id %q", rows[1].ID)
	}
	// A whole book of more than 3 chapters is a series with its first chapter as ref.
	s := rows[2]
	if s.Series == nil || !*s.Series || s.Ref == nil || *s.Ref != "Lam.1" {
		t.Errorf("series row: %+v", s)
	}

	if rep.Pages != 8 || rep.Rows != 3 || rep.Series != 1 {
		t.Errorf("report %+v", rep)
	}
	reasons := map[string]string{}
	for _, sk := range rep.Skipped {
		reasons[sk.URL] = sk.Reason
	}
	want := map[string]string{
		urls[0]: ReasonList,
		urls[1]: ReasonBook,
		urls[3]: ReasonNoVideo,
		urls[4]: ReasonNoPassage,
		urls[6]: ReasonFetch,
	}
	for u, w := range want {
		if reasons[u] != w {
			t.Errorf("%s skipped as %q, want %q", u, reasons[u], w)
		}
	}
	if len(rep.Skipped) != 5 || rep.Rows+len(rep.Skipped) != rep.Pages {
		t.Errorf("every page must be a row or skipped: %d rows, %d skipped, %d pages", rep.Rows, len(rep.Skipped), rep.Pages)
	}
	md := rep.Markdown()
	for _, s := range []string{"no terms of use", "User-Agent: *", "Disallow: /api/", urls[3]} {
		if !strings.Contains(md, s) {
			t.Errorf("REPORT.md lacks %q", s)
		}
	}
}

func TestBuildRowsRepeatedID(t *testing.T) {
	// two addresses that differ only by a trailing slash name one id: the second is skipped, named.
	urls := []string{ttbPrefix + "/isaiah/1", ttbPrefix + "/isaiah/1/"}
	p := PageResult{Page: Page{Title: "Isaiah 1: x", YouTube: "ynUYAm1-W6k"}}
	rows, rep := BuildRows(urls, map[string]PageResult{urls[0]: p, urls[1]: p})
	if len(rows) != 1 || len(rep.Skipped) != 1 || rep.Skipped[0].Reason != ReasonRepeat {
		t.Errorf("%d rows, skipped %+v", len(rows), rep.Skipped)
	}
}
