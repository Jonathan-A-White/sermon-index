package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Jonathan-A-White/sermon-index/internal/sources"
)

const feedAlt = "https://subsplash.com/u/brookfieldfirst/media/d/"

func feedRows() []sources.Row {
	a1, a2, a3 := feedAlt+"aaa1111-the-garden-gate", feedAlt+"bbb2222-the-narrow-door", feedAlt+"ccc3333-gods-quiet-work"
	return []sources.Row{
		// Deliberately not in date order: the feed sorts.
		{ID: "v2", Title: "Brookfield First - Online Service | March 9, 2031", Ref: sp("Luke.13.22-Luke.13.30"), URL: "y2", Date: sp("2031-03-09"), AltURL: &a2},
		{ID: "v1", Title: "The Garden Gate", URL: "y1", Date: sp("2031-03-02"), AltURL: &a1},
		{ID: "v5", Title: "God's Quiet Work", Ref: sp("Mark.1.1-Mark.1.8"), URL: "y5", Date: sp("2031-04-06"), AltURL: &a3},
		{ID: "v6", Title: "Brookfield First - Online Service | April 13, 2031", Ref: sp("Mark.2.1"), URL: "y6", Date: sp("2031-04-13")},
		{ID: "v3", Title: "Brookfield First - Online Service | March 16, 2031", Ref: sp("Mark.3.1"), URL: "y3", Date: sp("2031-03-16")},
		{ID: "v0", Title: "Undated", URL: "y0"},
	}
}

func feedRecords() []Record {
	return []Record{
		{ID: "v2", Date: "2031-03-09", Title: "The Narrow Door", Ref: "Luke.13.22-Luke.13.30"},
		{ID: "v5", Date: "2031-04-06", Title: "God's Quiet Work"},
		// No id: found by its exact date (v3 alone is dated March 16).
		{Date: "2031-03-16", Title: "A Dated Record"},
		// An id no row has and a date no row has: matches nothing.
		{ID: "zz", Date: "2031-12-25", Title: "Nobody's"},
	}
}

func feedServed() map[string]servedRow {
	return map[string]servedRow{
		"v2": {ID: "v2", AltURL: sp(feedAlt + "bbb2222-the-narrow-door")},
		"v5": {ID: "v5"}, // served without its link: not loaded yet
		"v6": {ID: "v6"},
		"v3": {ID: "v3"},
	}
}

func TestBuildFeedOrdersChecksAndPlacesHere(t *testing.T) {
	now := time.Date(2031, 5, 1, 12, 0, 0, 0, time.UTC)
	f := BuildFeed(feedRows(), feedRecords(), nil, feedServed(), now)
	if f.ID != "subsplash-links" || f.Title != "Subsplash links" || f.Updated != "2031-05-01T12:00:00Z" {
		t.Errorf("header: %+v", f)
	}
	if strings.Join(f.Checks, ",") != "mail,link,live" {
		t.Errorf("checks %v", f.Checks)
	}
	type want struct {
		label, date       string
		link              string
		mail, link_, live bool
	}
	wants := []want{
		{"Online Service 2031-04-13", "2031-04-13", "", false, false, false},
		{"God's Quiet Work", "2031-04-06", feedAlt + "ccc3333-gods-quiet-work", true, true, false},
		{"A Dated Record", "2031-03-16", "", true, false, false},
		{"The Narrow Door", "2031-03-09", feedAlt + "bbb2222-the-narrow-door", true, true, true},
		{"The Garden Gate", "2031-03-02", feedAlt + "aaa1111-the-garden-gate", false, true, false},
		{"Undated", "", "", false, false, false},
	}
	if len(f.Items) != len(wants) {
		t.Fatalf("%d items, want %d: %+v", len(f.Items), len(wants), f.Items)
	}
	for i, w := range wants {
		it := f.Items[i]
		if it.Label != w.label || it.Date != w.date || it.Link != w.link ||
			it.Done["mail"] != w.mail || it.Done["link"] != w.link_ || it.Done["live"] != w.live || len(it.Done) != 3 {
			t.Errorf("item %d = %+v, want %+v", i, it, w)
		}
	}
	// He works newest to oldest: his place is the oldest row with mail and link.
	if f.Here == nil || *f.Here != 3 {
		t.Errorf("here = %v, want 3 (The Narrow Door)", f.Here)
	}
}

func TestBuildFeedHereAbsentWhenNoRowHasMailAndLink(t *testing.T) {
	f := BuildFeed(feedRows()[3:5], nil, nil, nil, time.Now())
	if f.Here != nil {
		t.Errorf("here = %d, want none", *f.Here)
	}
	b, _ := json.Marshal(f)
	if strings.Contains(string(b), `"here"`) {
		t.Errorf("here written: %s", b)
	}
}

func TestBuildFeedLiveWhenServiceHoldsEverything(t *testing.T) {
	rows := feedRows()
	// v0 has no passage, so the service cannot serve it; the others match.
	a1 := feedAlt + "aaa1111-the-garden-gate"
	rows[5].AltURL = &a1
	rows[1].Ref = sp("Gen.2.8")
	served := map[string]servedRow{}
	for _, r := range rows {
		if r.Ref != nil {
			served[r.ID] = servedRow{ID: r.ID, Title: r.Title, URL: r.URL, Date: r.Date, AltURL: r.AltURL, Ref: *r.Ref, Role: "text"}
		}
	}
	f := BuildFeed(rows, nil, nil, served, time.Now())
	for _, it := range f.Items {
		if it.Done["link"] != it.Done["live"] {
			t.Errorf("%s: link %v live %v, want equal when the service holds rows.jsonl", it.Label, it.Done["link"], it.Done["live"])
		}
	}
}

// Acceptance 3: titles and dates only.
func TestFeedHoldsNoNotesTextAndNoAddress(t *testing.T) {
	rows := feedRows()
	rows[0].Notes = sp("SECRET-NOTES-TEXT someone@example.org")
	rows[0].Preacher = sp("A. Preacher")
	recs := feedRecords()
	f := BuildFeed(rows, recs, nil, feedServed(), time.Now())
	b, _ := json.Marshal(f)
	for _, bad := range []string{"SECRET", "@", "A. Preacher", "youtube", "Luke.13", "y2"} {
		if strings.Contains(string(b), bad) {
			t.Errorf("the feed holds %q: %s", bad, b)
		}
	}
	var m map[string]any
	json.Unmarshal(b, &m)
	for k := range m {
		switch k {
		case "id", "title", "updated", "checks", "items", "here":
		default:
			t.Errorf("unexpected key %q", k)
		}
	}
	for _, it := range m["items"].([]any) {
		for k := range it.(map[string]any) {
			switch k {
			case "label", "date", "link", "done":
			default:
				t.Errorf("unexpected item key %q", k)
			}
		}
	}
}

func fakeService(t *testing.T, served []map[string]any) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"total": len(served), "rows": served})
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestFeedFlagWritesFileAtomically(t *testing.T) {
	dir := t.TempDir()
	rows := filepath.Join(dir, "rows.jsonl")
	notes := filepath.Join(dir, "notes.jsonl")
	var rl []string
	for _, r := range feedRows() {
		b, _ := json.Marshal(r)
		rl = append(rl, string(b))
	}
	os.WriteFile(rows, []byte(strings.Join(rl, "\n")+"\n"), 0o644)
	var nl []string
	for _, r := range feedRecords() {
		b, _ := json.Marshal(r)
		nl = append(nl, string(b))
	}
	os.WriteFile(notes, []byte(strings.Join(nl, "\n")+"\n"), 0o644)
	srv := fakeService(t, []map[string]any{{"id": "v2", "alt_url": feedAlt + "bbb2222-the-narrow-door", "role": "text", "ref": "Luke.13.22-Luke.13.30"}})
	out := filepath.Join(dir, "jobs", "subsplash-links.json")
	var o, e bytes.Buffer
	now := time.Date(2031, 5, 1, 12, 0, 0, 0, time.UTC)
	code := cli([]string{"--feed", out, "--service", srv.URL, "--rows", rows, "--notes", notes}, &o, &e, deps{now: func() time.Time { return now }})
	if code != 0 {
		t.Fatalf("exit %d: %s%s", code, o.String(), e.String())
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	var f Feed
	if err := json.Unmarshal(b, &f); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, b)
	}
	if len(f.Items) != 6 || f.Items[3].Label != "The Narrow Door" || !f.Items[3].Done["live"] || f.Items[1].Done["live"] {
		t.Errorf("feed: %s", b)
	}
	// No temporary file is left beside it.
	ents, _ := os.ReadDir(filepath.Dir(out))
	if len(ents) != 1 {
		t.Errorf("jobs directory holds %d files, want 1", len(ents))
	}
	// A service that cannot be asked leaves the old file as it was and fails.
	srv.Close()
	code = cli([]string{"--feed", out, "--service", srv.URL, "--rows", rows, "--notes", notes}, &o, &e, deps{now: func() time.Time { return now.Add(time.Hour) }})
	if code != 1 {
		t.Errorf("exit %d with the service down, want 1", code)
	}
	if again, _ := os.ReadFile(out); !bytes.Equal(again, b) {
		t.Error("the old feed was changed by a failed run")
	}
	// --feed needs --service.
	if code := cli([]string{"--feed", out, "--rows", rows}, &o, &e, deps{}); code != 2 {
		t.Errorf("--feed without --service: exit %d, want 2", code)
	}
}

// A row whose sermon has no notes counts its mail as done and says so.
func TestBuildFeedMarksANoNotesRow(t *testing.T) {
	a := feedAlt + "ddd4444-the-running-man"
	rows := []sources.Row{
		{ID: "e1", Title: "The Running Man", Ref: sp("Luke.24.33-Luke.24.46"), URL: "y", Date: sp("2031-04-06"), AltURL: &a},
		{ID: "e2", Title: "Brookfield First - Online Service | April 13, 2031", Ref: sp("Mark.1.1"), URL: "y", Date: sp("2031-04-13")},
	}
	f := BuildFeed(rows, nil, map[string]bool{"e1": true}, nil, time.Now())
	if len(f.Items) != 2 {
		t.Fatalf("%+v", f)
	}
	if it := f.Items[1]; it.Label != "The Running Man (no notes)" || !it.Done["mail"] || !it.Done["link"] {
		t.Errorf("the no-notes row: %+v", it)
	}
	if it := f.Items[0]; strings.Contains(it.Label, "no notes") || it.Done["mail"] {
		t.Errorf("a row not in the list: %+v", it)
	}
	if f.Here == nil || *f.Here != 1 {
		t.Errorf("here %v, want the no-notes row (index 1)", f.Here)
	}
}

func TestFeedFlagReadsTheNoNotesFile(t *testing.T) {
	dir := t.TempDir()
	rows := filepath.Join(dir, "rows.jsonl")
	a := feedAlt + "ddd4444-the-running-man"
	b, _ := json.Marshal(sources.Row{ID: "e1", Title: "The Running Man", URL: "y", Date: sp("2031-04-06"), AltURL: &a})
	os.WriteFile(rows, append(b, '\n'), 0o644)
	nn := filepath.Join(dir, "no-notes.tsv")
	os.WriteFile(nn, []byte("date\trow\ttxid\n2031-04-05\te1\tdirect:x\n"), 0o644)
	srv := fakeService(t, nil)
	out := filepath.Join(dir, "jobs", "f.json")
	var o, e bytes.Buffer
	code := cli([]string{"--feed", out, "--service", srv.URL, "--rows", rows, "--notes", filepath.Join(dir, "none.jsonl"), "--no-notes-file", nn}, &o, &e, deps{})
	if code != 0 {
		t.Fatalf("exit %d: %s%s", code, o.String(), e.String())
	}
	got, _ := os.ReadFile(out)
	if !strings.Contains(string(got), "The Running Man (no notes)") || !strings.Contains(string(got), `"mail": true`) {
		t.Errorf("feed: %s", got)
	}
}
