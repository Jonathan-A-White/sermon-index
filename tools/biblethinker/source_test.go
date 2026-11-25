package main

import (
	"bufio"
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"testing"
)

// The committed source: sources/biblethinker/rows.jsonl, as the tool wrote it.

func committedRows(t *testing.T) []Row {
	t.Helper()
	f, err := os.Open("../../sources/biblethinker/rows.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var rows []Row
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		var r Row
		if err := json.Unmarshal(sc.Bytes(), &r); err != nil {
			t.Fatalf("%q: %v", sc.Text(), err)
		}
		rows = append(rows, r)
	}
	return rows
}

func TestCommittedRowForHopeInGrief(t *testing.T) {
	const slug = "hope-in-grief-born-again-jesus-resurrection-1-peter-13-7"
	for _, r := range committedRows(t) {
		if r.ID != slug {
			continue
		}
		if r.Ref == nil || *r.Ref != "1Pet.1.3-1Pet.1.7" {
			t.Errorf("ref = %v", r.Ref)
		}
		if r.URL != "https://biblethinker.org/"+slug+"/" {
			t.Errorf("url = %q", r.URL)
		}
		if r.AltURL == nil || *r.AltURL != "https://www.youtube.com/watch?v=n-0M1f1iJ5Y" {
			t.Errorf("alt_url = %v", r.AltURL)
		}
		if r.Preacher == nil || *r.Preacher != "Mike Winger" {
			t.Errorf("preacher = %v", r.Preacher)
		}
		return
	}
	t.Fatalf("no row %s", slug)
}

var watchURL = regexp.MustCompile(`^https://www\.youtube\.com/watch\?v=[A-Za-z0-9_-]{11}$`)

func TestCommittedRowsAreAllOfTheCategoriesRead(t *testing.T) {
	rows := committedRows(t)
	if len(rows) < 150 {
		t.Fatalf("only %d rows", len(rows))
	}
	seen := map[string]bool{}
	cats := map[string]int{}
	for _, r := range rows {
		if seen[r.ID] {
			t.Errorf("duplicate id %s", r.ID)
		}
		seen[r.ID] = true
		if !strings.HasPrefix(r.URL, "https://biblethinker.org/") || r.URL != "https://biblethinker.org/"+r.ID+"/" {
			t.Errorf("%s: url %q is not the post's address", r.ID, r.URL)
		}
		// videoOf() in Lampas plays a watch address with an 11-character id.
		if r.AltURL == nil || !watchURL.MatchString(*r.AltURL) {
			t.Errorf("%s: alt_url %v is not a YouTube watch address", r.ID, r.AltURL)
		}
		if r.Ref == nil {
			t.Errorf("%s: a row with no passage", r.ID)
		}
		if r.Preacher == nil || *r.Preacher != "Mike Winger" {
			t.Errorf("%s: preacher %v", r.ID, r.Preacher)
		}
		if r.Date == nil || r.DurationS == nil {
			t.Errorf("%s: date %v, duration %v", r.ID, r.Date, r.DurationS)
		}
		found := false
		for _, c := range Categories {
			if r.Notes != nil && strings.HasPrefix(*r.Notes, "Series: "+c.Name+" ("+c.URL()+").") {
				found = true
				cats[c.Slug]++
			}
		}
		if !found {
			t.Errorf("%s: notes %v name none of the categories read", r.ID, r.Notes)
		}
	}
	// the Old Testament Law's two posts name no passage anywhere (REPORT.md)
	for _, c := range Categories {
		if cats[c.Slug] == 0 && c.Slug != "understanding-the-old-testament-law" {
			t.Errorf("no row of the category %s", c.Slug)
		}
	}
}

func TestCommittedRowFoundInThePostsText(t *testing.T) {
	const slug = "melchizedek-and-jesus-how-to-find-jesus-in-the-old-testament-pt-6"
	for _, r := range committedRows(t) {
		if r.ID != slug {
			continue
		}
		if r.Ref == nil || *r.Ref != "Gen.14" {
			t.Errorf("ref = %v", r.Ref)
		}
		if r.Notes == nil || !strings.Contains(*r.Notes, "the post's own text names") {
			t.Errorf("notes = %v", r.Notes)
		}
		return
	}
	t.Fatalf("no row %s", slug)
}

func TestCommittedSourceTomlQuotesBothWordsAndNamesTheCategories(t *testing.T) {
	b, err := os.ReadFile("../../sources/biblethinker/source.toml")
	if err != nil {
		t.Fatal(err)
	}
	s := strings.ReplaceAll(string(b), "\n# ", " ") // a quote runs over comment lines
	for _, want := range []string{
		"Do Bible thinker should just be sermons in his book but book series though the Bible",
		"Mike Winger have other sermons that are on Bible thinker but are not a part of his Through the Bible series but they're genuine Sons",
		"direct:0f79878bd2671f15431dd39dc1ba151415e0240fd3f56d0d6adb277f6133347d",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("source.toml lacks %q", want)
		}
	}
	for _, c := range append(append([]Series{}, Categories...), notReadSeries()...) {
		if !strings.Contains(s, c.Slug) {
			t.Errorf("source.toml does not name the category %s", c.Slug)
		}
	}
	if strings.Contains(s, "book-by-book series") {
		t.Error("source.toml still says book-by-book series")
	}
}

func notReadSeries() []Series {
	var out []Series
	for _, slug := range NotReadCategories {
		out = append(out, Series{Slug: slug})
	}
	return out
}
