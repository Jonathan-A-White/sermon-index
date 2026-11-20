package main

import (
	"os"
	"path/filepath"
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
		{ID: "", Date: "2031-03-16", Title: "A Shared Sunday", Ref: "Luke.1.1"},
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

// A row holding the same code under another slug spelling is already linked:
// Subsplash's slug for an apostrophe is a hyphen or nothing, so the code is
// the identity, not the whole address. A different code is still a conflict.
func TestLinkSameCodeOtherSlugOrOtherCode(t *testing.T) {
	rs := rows()
	rs[5].AltURL = str(addr("bbb2222", "god-s-quiet-work").URL)
	res := Link(rs, recs(), []Address{addr("bbb2222", "gods-quiet-work")})
	if len(res.Set) != 0 || res.Already != 1 || len(res.Report) != 0 {
		t.Errorf("set = %v, already = %d, report = %v", res.Set, res.Already, res.Report)
	}
	rs[5].AltURL = str(addr("zzz9999", "god-s-quiet-work").URL)
	res = Link(rs, recs(), []Address{addr("bbb2222", "gods-quiet-work")})
	if res.Already != 0 || len(res.Report) != 1 || !strings.Contains(res.Report[0], "conflict") {
		t.Errorf("a different code: already = %d, report = %v", res.Already, res.Report)
	}
}

// Two different links claiming one row, neither with a date: neither is
// written, and the row is named once.
func TestLinkTwoLinksOneRow(t *testing.T) {
	res := Link(rows(), nil, []Address{addr("aaa1111", "the-garden-gate"), addr("hhh8888", "the-garden-gate")})
	if len(res.Set) != 0 || len(res.Report) != 1 {
		t.Fatalf("set = %v, report = %v", res.Set, res.Report)
	}
	if line := res.Report[0]; !strings.Contains(line, "conflict") || !strings.Contains(line, "aaa1111") || !strings.Contains(line, "hhh8888") {
		t.Errorf("line = %q", line)
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

// A record's date is the Sunday, but a row may be dated the Saturday or the
// Monday; the record's id names the row, and that row is the one linked.
func TestLinkByRecordIDWhenRowIsOffTheSunday(t *testing.T) {
	mk := func(id, title, date string) sources.Row {
		return sources.Row{ID: id, Title: title, URL: "https://www.youtube.com/watch?v=" + id, Date: str(date)}
	}
	rs := []sources.Row{
		mk("mon", "Brookfield First - Online Service | May 5, 2031", "2031-05-05"),
		mk("sat", "Brookfield First - Online Service | May 10, 2031", "2031-05-10"),
		mk("sun", "Brookfield First - Online Service | May 11, 2031", "2031-05-11"),
	}
	rc := []Record{
		{ID: "mon", Date: "2031-05-04", Title: "Monday Child", Ref: "Luke.1.1"},
		{ID: "sat", Date: "2031-05-11", Title: "Saturday Child", Ref: "Luke.1.2"},
	}
	res := Link(rs, rc, []Address{addr("mmm1111", "monday-child"), addr("sss2222", "saturday-child")})
	if res.Set["mon"] != addr("mmm1111", "monday-child").URL || res.Set["sat"] != addr("sss2222", "saturday-child").URL || len(res.Set) != 2 {
		t.Errorf("set = %v, report = %v", res.Set, res.Report)
	}
}

// A record with no id, or an id no row has, falls back to its exact date.
func TestLinkByRecordDateWhenIDIsMissingOrUnknown(t *testing.T) {
	rs := rows()
	rc := []Record{
		{ID: "", Date: "2031-03-09", Title: "No Id Here", Ref: "Luke.1.1"},
		{ID: "nobody", Date: "2031-03-23", Title: "Unknown Id", Ref: "Luke.1.2"},
	}
	res := Link(rs, rc, []Address{addr("ggg1111", "no-id-here"), addr("hhh2222", "unknown-id")})
	if res.Set["v2"] != addr("ggg1111", "no-id-here").URL || res.Set["v5"] != addr("hhh2222", "unknown-id").URL {
		t.Errorf("set = %v, report = %v", res.Set, res.Report)
	}
}

// dated is an address with the date Subsplash gives its sermon.
func dated(code, slug, date string) Address {
	a := addr(code, slug)
	a.Date = date
	return a
}

func rowOn(id, title, date string, alt *string) sources.Row {
	return sources.Row{ID: id, Title: title, URL: "https://www.youtube.com/watch?v=" + id, Date: str(date), AltURL: alt}
}

// Two links claim one row with different addresses: the one whose date is the
// row's date is linked (the other is named, not written).
func TestLinkConflictResolvedByDate(t *testing.T) {
	rs := []sources.Row{rowOn("ZTuy", "Walking in the Fear of the Lord", "2026-01-25", nil)}
	res := Link(rs, nil, []Address{
		dated("phf3pj7", "walking-in-the-fear-of-the-lord", "2025-06-01"),
		dated("z426dy7", "walking-in-the-fear-of-the-lord", "2026-01-25"),
	})
	if res.Set["ZTuy"] != addr("z426dy7", "walking-in-the-fear-of-the-lord").URL || len(res.Set) != 1 {
		t.Errorf("set = %v, report = %v", res.Set, res.Report)
	}
	for _, l := range res.Report {
		if strings.Contains(l, "none written") {
			t.Errorf("report line %q says none written", l)
		}
	}
}

// The notes record's date stands in for a link with no date of its own.
func TestLinkConflictResolvedByNotesRecordDate(t *testing.T) {
	rs := []sources.Row{rowOn("r1", "The Narrow Door", "2031-03-09", nil)}
	rc := []Record{{ID: "r1", Date: "2031-03-09", Title: "The Narrow Door"}}
	res := Link(rs, rc, []Address{
		dated("old1111", "the-narrow-door", "2029-03-10"),
		addr("new2222", "the-narrow-door"), // no date: the one record's date, 2031-03-09
	})
	if res.Set["r1"] != addr("new2222", "the-narrow-door").URL {
		t.Errorf("set = %v, report = %v", res.Set, res.Report)
	}
}

// No claim's date is the row's: none is written, and the row is named once.
func TestLinkConflictNoDateMatchesNamedOnce(t *testing.T) {
	rs := []sources.Row{rowOn("aIVm", "Brookfield First Sunday Service (Live)", "2022-12-30", nil)}
	slug := "brookfield-first-sunday-service-live"
	res := Link(rs, nil, []Address{
		dated("pzxx57g", slug, "2026-05-24"),
		dated("y9782m5", slug, "2025-11-30"),
		dated("6qkrpmp", slug, "2025-07-27"),
	})
	if len(res.Set) != 0 || len(res.Report) != 1 {
		t.Fatalf("set = %v, report = %v", res.Set, res.Report)
	}
	for _, w := range []string{"aIVm", "pzxx57g", "y9782m5", "6qkrpmp", "none written"} {
		if !strings.Contains(res.Report[0], w) {
			t.Errorf("report %q lacks %q", res.Report[0], w)
		}
	}
}

// Two sermons on the row's date (one sermon in two series): none is guessed,
// unless the row already holds one of them, which then stays and the other is
// named.
func TestLinkConflictSeveralDatesMatch(t *testing.T) {
	slug := "walking-in-the-fear-of-the-lord"
	two := []Address{dated("phf3pj7", slug, "2026-01-25"), dated("z426dy7", slug, "2026-01-25")}

	res := Link([]sources.Row{rowOn("ZTuy", "Walking in the Fear of the Lord", "2026-01-25", nil)}, nil, two)
	if len(res.Set) != 0 || len(res.Report) != 1 || !strings.Contains(res.Report[0], "phf3pj7") || !strings.Contains(res.Report[0], "z426dy7") {
		t.Errorf("unheld: set = %v, report = %v", res.Set, res.Report)
	}

	held := str(addr("z426dy7", slug).URL)
	res = Link([]sources.Row{rowOn("ZTuy", "Walking in the Fear of the Lord", "2026-01-25", held)}, nil, two)
	if len(res.Set) != 0 || res.Already != 1 || len(res.Report) != 1 || !strings.Contains(res.Report[0], "phf3pj7") {
		t.Errorf("held: set = %v, already = %d, report = %v", res.Set, res.Already, res.Report)
	}
}

// A link whose title no row has is found by its date: a generic row of that
// date, or the one row of that date whose title is close.
func TestLinkNoTitleMatchFoundByDate(t *testing.T) {
	rs := []sources.Row{
		rowOn("g1", "Brookfield First Sunday Service (Live)", "2022-03-27", nil),
		rowOn("o1", "Another Sermon Entirely", "2022-03-06", nil),
		rowOn("f1", "Failure Isn't Final Hope", "2022-03-13", nil),
		rowOn("f2", "Lessons Learned from Failure", "2022-04-03", nil),
		rowOn("f3", "Dividing Line", "2022-04-03", nil),
	}
	res := Link(rs, nil, []Address{
		dated("2g4g8b4", "lessons-learned-in-failure", "2022-03-27"),   // generic row, by date alone
		dated("yxww5pq", "failure-isn-t-final", "2022-03-13"),          // close title
		dated("lll1111", "lessons-learned-in-failure-2", "2022-04-03"), // close title among two rows
	})
	if res.Set["g1"] != addr("2g4g8b4", "lessons-learned-in-failure").URL {
		t.Errorf("generic: set = %v, report = %v", res.Set, res.Report)
	}
	if res.Set["f1"] != addr("yxww5pq", "failure-isn-t-final").URL {
		t.Errorf("close: set = %v, report = %v", res.Set, res.Report)
	}
	if res.Set["f2"] != addr("lll1111", "lessons-learned-in-failure-2").URL {
		t.Errorf("close among two: set = %v, report = %v", res.Set, res.Report)
	}
}

// With no date, or no row that date that fits, the old report stands.
func TestLinkNoTitleMatchNoDateStillReported(t *testing.T) {
	rs := []sources.Row{rowOn("o1", "Another Sermon Entirely", "2022-03-06", nil), rowOn("o2", "A Second One", "2022-03-06", nil)}
	res := Link(rs, nil, []Address{
		addr("aaa1111", "lessons-learned-in-failure"),
		dated("bbb2222", "lessons-learned-in-failure", "2022-03-06"), // two rows, neither close nor generic
	})
	if len(res.Set) != 0 || len(res.Report) != 2 {
		t.Fatalf("set = %v, report = %v", res.Set, res.Report)
	}
	for _, l := range res.Report {
		if !strings.Contains(l, "no row has this title and no note has it") {
			t.Errorf("line = %q", l)
		}
	}
}

func TestReadInventoryDates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "inv.jsonl")
	os.WriteFile(path, []byte(`{"code":"phf3pj7","date":"2026-01-25","title":"x"}
not json
{"code":"nodate"}
{"code":"z426dy7","date":"2026-01-25"}
`), 0o644)
	got, err := ReadInventoryDates(path)
	if err != nil || len(got) != 2 || got["phf3pj7"] != "2026-01-25" || got["z426dy7"] != "2026-01-25" {
		t.Errorf("got = %v, err = %v", got, err)
	}
	if got, err := ReadInventoryDates(path + ".missing"); err != nil || len(got) != 0 {
		t.Errorf("missing: got = %v, err = %v", got, err)
	}
}

// A row that holds the link's code under another slug (Subsplash spells an
// apostrophe as "-s-" or "s") already holds that link: no conflict.
func TestLinkSameCodeOtherSlugIsAlready(t *testing.T) {
	rs := []sources.Row{rowOn("BIUj", "God's Strange Plan", "2025-01-05",
		str(addressPrefix+"zqgpvn7-gods-strange-plan"))}
	res := Link(rs, nil, []Address{addr("zqgpvn7", "god-s-strange-plan")})
	if len(res.Set) != 0 || res.Already != 1 || len(res.Report) != 0 {
		t.Errorf("set = %v, already = %d, report = %v", res.Set, res.Already, res.Report)
	}
}
