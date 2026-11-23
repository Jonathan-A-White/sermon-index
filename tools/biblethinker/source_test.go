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

func TestCommittedRowsAreAllBookSeriesEpisodes(t *testing.T) {
	rows := committedRows(t)
	if len(rows) < 100 {
		t.Fatalf("only %d rows", len(rows))
	}
	seen := map[string]bool{}
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
		if r.Notes == nil || !regexp.MustCompile(`^Series: (1 Peter|Hebrews|Mark|Romans) \(`).MatchString(*r.Notes) {
			t.Errorf("%s: notes %v name no book series", r.ID, r.Notes)
		}
	}
}
