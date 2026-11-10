package main

import (
	"strings"
	"testing"

	"github.com/Jonathan-A-White/sermon-index/internal/sources"
)

func str(s string) *string { return &s }

// Invented rows: a few Subsplash-style titles, the rest YouTube's.
func rows() []sources.Row {
	mk := func(id, title, date string) sources.Row {
		return sources.Row{ID: id, Title: title, URL: "https://www.youtube.com/watch?v=" + id, Date: str(date)}
	}
	return []sources.Row{
		mk("v1", "The Garden Gate", "2031-03-02"),
		mk("v2", "Brookfield First - Online Service | March 9, 2031", "2031-03-09"),
		mk("v3", "Brookfield First - Online Service | March 16, 2031", "2031-03-16"),
		mk("v4", "Brookfield First - Second Service | March 16, 2031", "2031-03-16"),
		mk("v5", "Brookfield First - Online Service | March 23, 2031", "2031-03-23"),
		mk("v6", "God's Quiet Work", "2031-04-06"),
		mk("v7", "Twice Told", "2031-04-13"),
		mk("v8", "Twice Told!", "2031-04-20"),
	}
}

func recs() []Record {
	return []Record{
		{ID: "v2", Date: "2031-03-09", Title: "The Narrow Door", Ref: "Luke.13.22-Luke.13.30"},
		{ID: "v3", Date: "2031-03-16", Title: "A Shared Sunday", Ref: "Luke.1.1"},
		{ID: "x", Date: "2031-03-30", Title: "The Lost Sheep", Ref: "Luke.15.1-Luke.15.7"},
	}
}

func addr(code, slug string) Address {
	return Address{Code: code, Slug: slug, URL: "https://subsplash.com/u/brookfieldfirst/media/d/" + code + "-" + slug}
}

func TestSlug(t *testing.T) {
	cases := map[string]string{
		"The Day of the Lord":       "the-day-of-the-lord",
		"  God's Quiet Work ":       "gods-quiet-work",
		"God’s Quiet Work":          "gods-quiet-work",
		"Love — and Law: Part 2":    "love-and-law-part-2",
		"Brookfield First | Day 1!": "brookfield-first-day-1",
		"Café Talk":                 "caf-talk",
	}
	for in, want := range cases {
		if got := Slugs(in)[0]; got != want {
			t.Errorf("Slugs(%q)[0] = %q, want %q", in, got, want)
		}
	}
}

// Acceptance 4: route one, the slug equals the slug of a row's title.
func TestLinkByRowTitle(t *testing.T) {
	res := Link(rows(), recs(), []Address{addr("aaa1111", "the-garden-gate"), addr("bbb2222", "gods-quiet-work")})
	if res.Set["v1"] != addr("aaa1111", "the-garden-gate").URL || res.Set["v6"] != addr("bbb2222", "gods-quiet-work").URL {
		t.Errorf("set = %v", res.Set)
	}
	if len(res.Report) != 0 {
		t.Errorf("report = %v", res.Report)
	}
}

// The slug Subsplash makes of an apostrophe is either nothing or a hyphen.
func TestLinkApostropheBothWays(t *testing.T) {
	res := Link(rows(), nil, []Address{addr("bbb2222", "god-s-quiet-work")})
	if res.Set["v6"] == "" {
		t.Errorf("set = %v, report = %v", res.Set, res.Report)
	}
}

// Acceptance 4: route two, the slug equals the slug of a notes title, and
// that record's date finds the row.
func TestLinkByNotesTitleThenDate(t *testing.T) {
	res := Link(rows(), recs(), []Address{addr("ccc3333", "the-narrow-door")})
	if res.Set["v2"] != addr("ccc3333", "the-narrow-door").URL {
		t.Errorf("set = %v, report = %v", res.Set, res.Report)
	}
}

func TestLinkNotesDateWithTwoRowsOrNone(t *testing.T) {
	res := Link(rows(), recs(), []Address{
		addr("ddd4444", "a-shared-sunday"), // 2031-03-16: v3 and v4
		addr("eee5555", "the-lost-sheep"),  // 2031-03-30: no row
	})
	if len(res.Set) != 0 {
		t.Errorf("set = %v: nothing should be written", res.Set)
	}
	if len(res.Report) != 2 {
		t.Fatalf("report = %v", res.Report)
	}
	if !strings.Contains(res.Report[0], "ddd4444") || !strings.Contains(res.Report[0], "a-shared-sunday") || !strings.Contains(res.Report[0], "2031-03-16") {
		t.Errorf("report[0] = %q", res.Report[0])
	}
	if !strings.Contains(res.Report[1], "the-lost-sheep") || !strings.Contains(res.Report[1], "2031-03-30") {
		t.Errorf("report[1] = %q", res.Report[1])
	}
}

func TestLinkTwoRowsWithOneTitle(t *testing.T) {
	res := Link(rows(), nil, []Address{addr("fff6666", "twice-told")})
	if len(res.Set) != 0 || len(res.Report) != 1 || !strings.Contains(res.Report[0], "v7") || !strings.Contains(res.Report[0], "v8") {
		t.Errorf("set = %v, report = %v", res.Set, res.Report)
	}
}

// Acceptance 6: an unmatched link is reported with its slug.
func TestLinkUnmatchedReportsSlug(t *testing.T) {
	res := Link(rows(), recs(), []Address{addr("ggg7777", "nothing-like-it")})
	if len(res.Set) != 0 || len(res.Report) != 1 {
		t.Fatalf("set = %v, report = %v", res.Set, res.Report)
	}
	if !strings.Contains(res.Report[0], "ggg7777") || !strings.Contains(res.Report[0], "nothing-like-it") {
		t.Errorf("report = %q", res.Report[0])
	}
}

// Acceptance 5: a different alt_url is a conflict; the same one is already done.
func TestLinkConflictAndAlready(t *testing.T) {
	rs := rows()
	rs[0].AltURL = str("https://subsplash.com/u/brookfieldfirst/media/d/old0000-the-garden-gate")
	rs[5].AltURL = str(addr("bbb2222", "gods-quiet-work").URL)
	res := Link(rs, recs(), []Address{addr("aaa1111", "the-garden-gate"), addr("bbb2222", "gods-quiet-work")})
	if len(res.Set) != 0 {
		t.Errorf("set = %v", res.Set)
	}
	if res.Already != 1 {
		t.Errorf("already = %d", res.Already)
	}
	if len(res.Report) != 1 || !strings.Contains(res.Report[0], "conflict") || !strings.Contains(res.Report[0], "old0000") {
		t.Errorf("report = %v", res.Report)
	}
}

// Two different links claiming one row: neither is written.
func TestLinkTwoLinksOneRow(t *testing.T) {
	res := Link(rows(), nil, []Address{addr("aaa1111", "the-garden-gate"), addr("hhh8888", "the-garden-gate")})
	if len(res.Set) != 0 || len(res.Report) != 2 {
		t.Errorf("set = %v, report = %v", res.Set, res.Report)
	}
	for _, line := range res.Report {
		if !strings.Contains(line, "conflict") {
			t.Errorf("line = %q", line)
		}
	}
}

// A repeat of one address (a PDF and the link file) counts once.
func TestLinkSameAddressTwice(t *testing.T) {
	a := addr("aaa1111", "the-garden-gate")
	res := Link(rows(), nil, []Address{a, a})
	if len(res.Set) != 1 || len(res.Report) != 0 {
		t.Errorf("set = %v, report = %v", res.Set, res.Report)
	}
}
