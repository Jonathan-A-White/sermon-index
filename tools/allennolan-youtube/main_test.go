package main

import (
	"bytes"
	"context"
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

// testKey is a made-up key; nothing real is ever in a test.
const testKey = "TESTKEY-not-a-real-key-0000000000"

const testChannel = "UCSpQNTtLDeQ5Z571g9j0LRg"

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// fakeAPI serves the recorded answers: channels.list, playlists.list in two
// pages and playlistItems.list for each playlist of the fixtures. It records
// each request.
func fakeAPI(t *testing.T, reqs *[]*http.Request) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*reqs = append(*reqs, r.Clone(context.Background()))
		if r.Header.Get("X-Goog-Api-Key") != testKey {
			http.Error(w, `{"error":{"code":403,"message":"The request is missing a valid API key."}}`, 403)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		q := r.URL.Query()
		switch r.URL.Path {
		case "/youtube/v3/channels":
			w.Write(fixture(t, "channel.json"))
		case "/youtube/v3/playlists":
			if q.Get("channelId") != testChannel {
				http.NotFound(w, r)
				return
			}
			if q.Get("pageToken") == "PLPAGE2" {
				w.Write(fixture(t, "playlists-page2.json"))
			} else {
				w.Write(fixture(t, "playlists-page1.json"))
			}
		case "/youtube/v3/playlistItems":
			b, err := os.ReadFile(filepath.Join("testdata", "items-"+q.Get("playlistId")+".json"))
			if err != nil {
				http.Error(w, `{"error":{"code":404,"message":"playlistNotFound"}}`, 404)
				return
			}
			w.Write(b)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func newClient(srv *httptest.Server) *Client {
	return &Client{Base: srv.URL, Key: testKey, HTTP: srv.Client(), Gap: time.Nanosecond}
}

func TestClientReadsChannelPlaylistsAndItems(t *testing.T) {
	var reqs []*http.Request
	c := newClient(fakeAPI(t, &reqs))
	ctx := context.Background()

	ch, err := c.Channel(ctx, testChannel)
	if err != nil {
		t.Fatal(err)
	}
	if ch.ID != testChannel || ch.Title != "Pastor Allen Nolan" || ch.Handle != "@allennolan" {
		t.Errorf("channel %+v", ch)
	}
	pls, err := c.Playlists(ctx, testChannel)
	if err != nil {
		t.Fatal(err)
	}
	if len(pls) != 9 {
		t.Fatalf("%d playlists, want 9 (5 and 4)", len(pls))
	}
	if pls[0] != (Playlist{ID: "PLgen", Title: "Book of Genesis Bible Study", Count: 4}) {
		t.Errorf("first playlist %+v", pls[0])
	}
	items, err := c.PlaylistItems(ctx, "PLrev")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 7 || items[0].VideoID != "revAAAAAA01" || items[0].Published != "2020-01-15T14:00:00Z" {
		t.Errorf("items %+v", items)
	}
	if len(reqs) != 4 {
		t.Fatalf("%d requests, want 4 (channels.list, two playlists.list pages, one playlistItems.list)", len(reqs))
	}
	q := reqs[0].URL.Query()
	if q.Get("id") != testChannel || q.Get("part") != "snippet" {
		t.Errorf("channels.list query %v", q)
	}
	q = reqs[1].URL.Query()
	if q.Get("channelId") != testChannel || q.Get("part") != "snippet,contentDetails" || q.Get("maxResults") != "50" {
		t.Errorf("playlists.list query %v", q)
	}
	if reqs[2].URL.Query().Get("pageToken") != "PLPAGE2" {
		t.Errorf("second playlists page query %v", reqs[2].URL.Query())
	}
	q = reqs[3].URL.Query()
	if q.Get("playlistId") != "PLrev" || q.Get("part") != "snippet,contentDetails" {
		t.Errorf("playlistItems.list query %v", q)
	}
	for _, r := range reqs {
		if strings.Contains(r.URL.String(), testKey) || r.URL.Query().Get("key") != "" {
			t.Errorf("the key is in the address: %s", r.URL)
		}
		if r.Method != http.MethodGet {
			t.Errorf("%s %s: the tool only reads", r.Method, r.URL.Path)
		}
	}
	if c.Units != 4 {
		t.Errorf("quota %d units, want 4 (1 each)", c.Units)
	}
}

// Only the three public-metadata methods are called, never a page of
// allennolan.com or a Subsplash address.
func TestClientCallsOnlyTheAPI(t *testing.T) {
	var reqs []*http.Request
	c := newClient(fakeAPI(t, &reqs))
	if _, _, err := discover(context.Background(), c, testChannel); err != nil {
		t.Fatal(err)
	}
	for _, r := range reqs {
		switch r.URL.Path {
		case "/youtube/v3/channels", "/youtube/v3/playlists", "/youtube/v3/playlistItems":
		default:
			t.Errorf("a call to %s", r.URL.Path)
		}
	}
	if strings.Contains(apiBase, "allennolan.com") || !strings.HasPrefix(apiBase, "https://www.googleapis.com") {
		t.Errorf("the API's origin is %s", apiBase)
	}
}

func TestClientErrorsNeverCarryTheKey(t *testing.T) {
	echo := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":{"code":400,"message":"API key not valid: `+testKey+`"}}`, 400)
	}))
	defer echo.Close()
	c := &Client{Base: echo.URL, Key: testKey, HTTP: echo.Client(), Gap: time.Nanosecond}
	_, err := c.Playlists(context.Background(), testChannel)
	if err == nil {
		t.Fatal("a 400 was taken")
	}
	if strings.Contains(err.Error(), testKey) {
		t.Errorf("error holds the key: %v", err)
	}
	if !strings.Contains(err.Error(), "400") {
		t.Errorf("error does not say what happened: %v", err)
	}

	dead := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	base := dead.URL
	dead.Close()
	c = &Client{Base: base + "/?k=" + testKey, Key: testKey, HTTP: http.DefaultClient, Gap: time.Nanosecond}
	if _, err = c.Playlists(context.Background(), testChannel); err == nil || strings.Contains(err.Error(), testKey) {
		t.Errorf("connection error: %v", err)
	}
}

func TestClientChannelNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"kind":"youtube#channelListResponse","pageInfo":{"totalResults":0,"resultsPerPage":5}}`))
	}))
	defer srv.Close()
	c := &Client{Base: srv.URL, Key: testKey, HTTP: srv.Client(), Gap: time.Nanosecond}
	if _, err := c.Channel(context.Background(), "UCnobody"); err == nil || !strings.Contains(err.Error(), "UCnobody") {
		t.Errorf("no channel: %v", err)
	}
}

func TestKeyFromEnv(t *testing.T) {
	if _, err := keyFromEnv(func(string) string { return "" }); err == nil {
		t.Fatal("an empty key was taken")
	} else if !strings.Contains(err.Error(), "YOUTUBE_API_KEY") {
		t.Errorf("the message does not name the variable: %v", err)
	}
	k, err := keyFromEnv(func(n string) string {
		if n == "YOUTUBE_API_KEY" {
			return "  " + testKey + "\n"
		}
		return ""
	})
	if err != nil || k != testKey {
		t.Errorf("key %q, %v", k, err)
	}
}

func TestClassifyPlaylist(t *testing.T) {
	tests := []struct {
		title     string
		ok        bool
		book      string
		scope     string
		byChapter bool
	}{
		{"Book of Genesis Bible Study", true, "Gen", "", false},
		{"The Book of Revelation Bible Study Verse by Verse", true, "Rev", "", false},
		{"Book of Amos Bible Study Verse by Verse", true, "Amos", "", false},
		{"Abraham: The Book of Genesis | Pastor Allen Nolan Sermons", true, "Gen", "", false},
		{"The Book of Leviticus | Pastor Allen Nolan", true, "Lev", "", false},
		{"Book of Proverbs | Pastor Allen Nolan Sermons", true, "Prov", "", false},
		{"The Word: Gospel of John Series with Pastor Allen Nolan", true, "John", "", false},
		{"Revelation Bible Study by Chapters", true, "Rev", "", true},
		{"Revelation Chapter 5 | Allen Nolan", true, "Rev", "Rev.5", true},
		{"Revelation Chapters 15 and 16 Explained | Allen Nolan", true, "Rev", "Rev.15-Rev.16", true},
		{"Isaiah 53 Verse by Verse Bible Study | Allen Nolan", true, "Isa", "Isa.53", false},
		{"The Meaning of Isaiah 53 Verse by Verse", true, "Isa", "Isa.53", false},
		{"1 Corinthians 13 | Pastor Allen Nolan", true, "1Cor", "1Cor.13", false},
		// A book's name in a topical playlist's title makes no book series.
		{"Genesis Mysteries Explained: Nephilim, Enoch, the Firmament & the Flood", false, "", "", false},
		{"Daniel and Revelation | Allen Nolan", false, "", "", false},
		{"Genesis: The Story of Joseph | Pastor Allen Nolan", false, "", "", false},
		{"The Spiritual Gifts (Manifestation Gifts of 1 Corinthians)", false, "", "", false},
		{"The Miraculous Sign: Isaiah's Amazing Prophecy | Allen Nolan", false, "", "", false},
		// No book, whatever else it says.
		{"The Grace of God Bible Study", false, "", "", false},
		{"The New Testament Bible Study Podcast with Pastor Allen Nolan", false, "", "", false},
		{"Why God Isn’t Answering Your Prayers", false, "", "", false},
		{"Youtube Shorts", false, "", "", false},
		{"Work | Allen Nolan", false, "", "", false},
	}
	for _, tc := range tests {
		s, ok := ClassifyPlaylist(Playlist{ID: "PLx", Title: tc.title, Count: 3})
		if ok != tc.ok {
			t.Errorf("%q: series %v, want %v", tc.title, ok, tc.ok)
			continue
		}
		if !ok {
			continue
		}
		if s.Book != tc.book || s.Scope != tc.scope || s.ByChapter != tc.byChapter {
			t.Errorf("%q: book %q scope %q byChapter %v, want %q %q %v", tc.title, s.Book, s.Scope, s.ByChapter, tc.book, tc.scope, tc.byChapter)
		}
	}
}

func TestVideoRef(t *testing.T) {
	gen := Series{Book: "Gen"}
	rev := Series{Book: "Rev"}
	revChapters := Series{Book: "Rev", ByChapter: true}
	isa53 := Series{Book: "Isa", Scope: "Isa.53"}
	tests := []struct {
		name  string
		s     Series
		title string
		desc  string
		ref   string
		from  string
	}{
		{"verses in the title", gen, "Abraham's Divine Calling (Genesis 12:1-3) | Part 32", "", "Gen.12.1-Gen.12.3", fromTitle},
		{"a single verse", rev, "The Verse: Revelation 1:8", "", "Rev.1.8", fromTitle},
		{"an abbreviation with a point", rev, "Will People Be SAVED After Rapture? Revelation Explained 26 (Rev. 6:9-11)", "", "Rev.6.9-Rev.6.11", fromTitle},
		{"verses with no book take the playlist's", rev, "Revelation Explained 27: Prerequisite for Jesus' Return (6:12-17)", "", "Rev.6.12-Rev.6.17", fromTitle},
		{"across chapters", rev, "Revelation 21:23-22:5 The Tree of Life", "", "Rev.21.23-Rev.22.5", fromTitle},
		{"another book than the playlist's", rev, "Dreams in Revelation (Daniel 8:1-2, 16-26)", "", "Dan.8.1-Dan.8.2", fromTitle},
		// "Genesis 5" in a numbered study is part 5, not chapter 5.
		{"a part number is no chapter", gen, "Everything You Need to Know About the First Days of Creation: Genesis 5 with Pastor Allen Nolan", "", "", ""},
		{"a part number of Amos", Series{Book: "Amos"}, "Book of Amos Bible Study Syrians TORTURING Their Prisoners of War: Amos 6", "", "", ""},
		{"a study number is no chapter", gen, "Young Earth or Old Earth? Here's What the Bible Says | Genesis Study 2", "", "", ""},
		{"a chapter range of a whole book is no passage", rev, "Summary of Revelation 1-14", "", "", ""},
		{"a chapter in a by-chapter playlist", revChapters, "In-Depth Look at Revelation 4: The Heavenly Throne", "", "Rev.4", fromTitleChapter},
		{"the word Chapter", revChapters, "Revelation Chapter 1: Jesus and the Coming Apocalypse", "", "Rev.1", fromTitleChapter},
		{"two chapters with an ampersand", revChapters, "The Truth About Revelation 17 & 18: Babylon and the Beast", "", "Rev.17-Rev.18", fromTitleChapter},
		{"two chapters with and", revChapters, "Bowls of Wrath (Rev. 16 and 17)", "", "Rev.16-Rev.17", fromTitleChapter},
		{"a part is no chapter in a by-chapter playlist", revChapters, "Leaving Your First Love: Revelation Part 2", "", "", ""},
		{"a chapter of another book is not the series' text", revChapters, "Daniel 9 and the Seventy Weeks", "", "", ""},
		{"the playlist's own chapter", isa53, "Jesus' Suffering and Great Glory: Isaiah 53 Verse by Verse 2", "", "Isa.53", fromPlaylist},
		{"a title that names the book, in a chapter's playlist", Series{Book: "Rev", Scope: "Rev.5"}, "Slain Lamb with Seven Horns and Seven Eyes in Revelation", "", "Rev.5", fromPlaylist},
		{"a title that names no book may be any video filed there", Series{Book: "Rev", Scope: "Rev.5"}, "5 Things Jesus Predicted For The End Times", "", "", ""},
		{"the title's verses beat the playlist's chapter", Series{Book: "Rev", Scope: "Rev.5"}, "The Scroll (Revelation 5:1-5)", "", "Rev.5.1-Rev.5.5", fromTitle},
		{"one passage of the book in the description", rev, "Who Are the Two Witnesses In Revelation?", "The two witnesses of Revelation 11:3-12 are explained. Also see Daniel 9:1 and John 12.", "Rev.11.3-Rev.11.12", fromDescription},
		{"a single verse in the description is an illustration", gen, "The First Days of Creation", "Is Genesis 1:16 contradictory?", "", ""},
		{"a timestamp is no chapter", rev, "Leaving Your First Love", "The Basic Outline of Revelation 02:21 Pattern of the 7 Letters", "", ""},
		{"a line of a table of contents is no text", rev, "Outline of Revelation", "0:00 Intro\n20:18 Revelation 2:1-7\n21:05 Close", "", ""},
		{"two passages of the book in the description are no one text", gen, "In-Depth Genesis Bible Study", "See Genesis 12:3 and Genesis 19:3-6 and Genesis 15:6.", "", ""},
		{"a chapter in the description is no verse", gen, "Why the Nephilim Are More Than Just Giants", "who the Sons of God are in Genesis 6?", "", ""},
		{"nothing named", gen, "Does the Bible Contradict Science? | Book of Genesis Bible Study 3", "Click here for the FREE study notes", "", ""},
		{"a verse the book table refuses", rev, "Revelation 99:1 and nothing else", "", "", ""},
	}
	for _, tc := range tests {
		ref, from := VideoRef(Item{Title: tc.title, Description: tc.desc}, tc.s)
		if ref != tc.ref || from != tc.from {
			t.Errorf("%s: %q gives %q (%s), want %q (%s)", tc.name, tc.title, ref, from, tc.ref, tc.from)
		}
	}
}

func TestDateIsTheDayOnThePastorsClock(t *testing.T) {
	// 02:30 UTC is the evening before in Oklahoma.
	if got := datePart("2025-04-30T02:30:00Z"); got != "2025-04-29" {
		t.Errorf("date %q, want 2025-04-29", got)
	}
	if got := datePart("2020-01-15T14:00:00Z"); got != "2020-01-15" {
		t.Errorf("date %q", got)
	}
	if got := datePart("not a time"); got != "" {
		t.Errorf("date %q", got)
	}
}

// fixtureResult runs discover and Build over the recorded answers.
func fixtureResult(t *testing.T) *Result {
	t.Helper()
	var reqs []*http.Request
	c := newClient(fakeAPI(t, &reqs))
	ch, series, err := discover(context.Background(), c, testChannel)
	if err != nil {
		t.Fatal(err)
	}
	res := Build(ch, series)
	res.Units = c.Units
	return res
}

func TestBuildMakesOneRowAVideoOfBookPlaylistsOnly(t *testing.T) {
	res := fixtureResult(t)

	var ids []string
	for _, s := range res.Series {
		ids = append(ids, s.ID)
	}
	if got, want := strings.Join(ids, ","), "PLgen,PLrev,PLrev5,PLchap,PLisa,PLcor"; got != want {
		t.Errorf("book playlists %s, want %s", got, want)
	}
	var left []string
	for _, p := range res.LeftOut {
		left = append(left, p.ID)
	}
	if got, want := strings.Join(left, ","), "PLmyst"; got != want {
		t.Errorf("playlists that name a book and were left out: %s, want %s", got, want)
	}

	want := map[string]struct{ ref, date string }{
		"genAAAAAA03": {"Gen.12.1-Gen.12.3", "2020-07-29"},
		"revAAAAAA01": {"Rev.6.12-Rev.6.17", "2020-01-15"},
		"revAAAAAA02": {"Rev.6.7-Rev.6.8", "2020-01-08"},
		"revAAAAAA03": {"Rev.5", "2020-01-17"},
		"revAAAAAA05": {"Rev.11.3-Rev.11.12", "2020-02-15"},
		"revAAAAAA06": {"Dan.8.1-Dan.8.2", "2019-12-21"},
		"revAAAAAA07": {"Rev.12.7-Rev.12.17", "2020-09-16"},
		"rev5AAAAAA1": {"Rev.5", "2019-12-14"},
		"chapAAAAAA1": {"Rev.1", "2024-10-23"},
		"chapAAAAAA2": {"Rev.17-Rev.18", "2025-05-07"},
		"chapAAAAAA3": {"Rev.16-Rev.17", "2025-04-29"},
		"isaAAAAAAA1": {"Isa.53", "2022-04-25"},
		"isaAAAAAAA2": {"Isa.53", "2022-05-02"},
		"corAAAAAAA1": {"1Cor.13", "2025-07-18"},
	}
	if len(res.Rows) != len(want) {
		t.Errorf("%d rows, want %d", len(res.Rows), len(want))
	}
	for _, r := range res.Rows {
		w, ok := want[r.ID]
		if !ok {
			t.Errorf("unexpected row %s %q", r.ID, r.Title)
			continue
		}
		if r.Ref == nil || *r.Ref != w.ref {
			t.Errorf("%s: ref %v, want %s", r.ID, r.Ref, w.ref)
		}
		if r.Date == nil || *r.Date != w.date {
			t.Errorf("%s: date %v, want %s", r.ID, r.Date, w.date)
		}
		if r.URL != "https://www.youtube.com/watch?v="+r.ID {
			t.Errorf("%s: url %s", r.ID, r.URL)
		}
		if r.Preacher == nil || *r.Preacher != "Allen Nolan" {
			t.Errorf("%s: preacher %v", r.ID, r.Preacher)
		}
		if r.Kind == nil || *r.Kind != "link" {
			t.Errorf("%s: kind %v", r.ID, r.Kind)
		}
		if r.Notes == nil || *r.Notes == "" {
			t.Errorf("%s: no note on where the passage came from", r.ID)
		}
		delete(want, r.ID)
	}
	for id := range want {
		t.Errorf("no row for %s", id)
	}

	// Every video of a book playlist is a row or counted skipped.
	if res.Videos != 18 || res.Entries != 19 {
		t.Errorf("%d videos in %d entries, want 18 in 19 (one video is in two playlists)", res.Videos, res.Entries)
	}
	if res.Videos != len(res.Rows)+len(res.Skipped) {
		t.Errorf("videos %d, rows %d, skipped %d: some video is neither", res.Videos, len(res.Rows), len(res.Skipped))
	}
	skipped := map[string]string{}
	for _, s := range res.Skipped {
		skipped[s.ID] = s.Why
	}
	for _, id := range []string{"genAAAAAA01", "genAAAAAA02", "revAAAAAA04"} {
		if !strings.Contains(skipped[id], "no passage") {
			t.Errorf("%s skipped for %q, want no passage", id, skipped[id])
		}
	}
	if !strings.Contains(skipped["genAAAAAA04"], "private") {
		t.Errorf("the private video skipped for %q", skipped["genAAAAAA04"])
	}
	if len(res.Skipped) != 4 {
		t.Errorf("%d skipped, want 4: %v", len(res.Skipped), res.Skipped)
	}
	if res.Videos-res.Private() != 17 {
		t.Errorf("listable videos %d, want 17", res.Videos-res.Private())
	}
	if s := res.Share(); s < 0.8 || s > 0.9 {
		t.Errorf("share with a passage %.2f, want 14 of 17", s)
	}
	if res.First != "2019-12-14" || res.Last != "2025-07-18" {
		t.Errorf("dates %s to %s", res.First, res.Last)
	}
}

func TestBuildKeepsTheBestPassageOfAVideoInTwoPlaylists(t *testing.T) {
	// revAAAAAA03 is in PLrev (no passage in its title) and PLrev5 (the
	// chapter); it is one row, with the chapter.
	res := fixtureResult(t)
	n := 0
	for _, r := range res.Rows {
		if r.ID == "revAAAAAA03" {
			n++
			if *r.Ref != "Rev.5" {
				t.Errorf("ref %s", *r.Ref)
			}
		}
	}
	if n != 1 {
		t.Errorf("%d rows for the video in two playlists, want 1", n)
	}
}

func TestRowsPassTheSourceChecker(t *testing.T) {
	res := fixtureResult(t)
	out := filepath.Join(t.TempDir(), "allennolan")
	var buf bytes.Buffer
	if err := write(res, out, filepath.Join(out, "REPORT.md"), &buf); err != nil {
		t.Fatal(err)
	}
	faults, warnings := sources.CheckReport(out)
	if len(faults) != 0 {
		t.Errorf("the checker faults: %v", faults)
	}
	if len(warnings) != 0 {
		t.Errorf("the checker warns: %v", warnings)
	}
	rows, err := os.ReadFile(filepath.Join(out, "rows.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(rows), "\n"); n != 14 {
		t.Errorf("%d lines in rows.jsonl, want 14", n)
	}
	if strings.Contains(strings.ToLower(string(rows)), "subsplash") {
		t.Error("a row names Subsplash")
	}
	for _, line := range strings.Split(strings.TrimSpace(string(rows)), "\n") {
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatal(err)
		}
		for _, f := range []string{"id", "title", "ref", "url", "preacher", "date"} {
			if m[f] == nil || m[f] == "" {
				t.Errorf("row %v has no %s", m["id"], f)
			}
		}
	}
	// Nothing of a description reaches a file; the key is in none.
	for _, name := range []string{"rows.jsonl", "source.toml", "REPORT.md"} {
		b, err := os.ReadFile(filepath.Join(out, name))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(b), "fixture description") || strings.Contains(string(b), testKey) {
			t.Errorf("%s holds a description or the key", name)
		}
	}
}

func TestWriteIsStable(t *testing.T) {
	res := fixtureResult(t)
	read := func() map[string]string {
		out := t.TempDir()
		if err := write(res, out, filepath.Join(out, "REPORT.md"), &bytes.Buffer{}); err != nil {
			t.Fatal(err)
		}
		m := map[string]string{}
		for _, n := range []string{"rows.jsonl", "source.toml", "REPORT.md"} {
			b, _ := os.ReadFile(filepath.Join(out, n))
			m[n] = string(b)
		}
		return m
	}
	a, b := read(), read()
	for n := range a {
		if a[n] != b[n] {
			t.Errorf("%s differs between two writes", n)
		}
	}
	// Oldest first.
	var dates []string
	for _, l := range strings.Split(strings.TrimSpace(a["rows.jsonl"]), "\n") {
		var r sources.Row
		json.Unmarshal([]byte(l), &r)
		dates = append(dates, *r.Date)
	}
	for i := 1; i < len(dates); i++ {
		if dates[i] < dates[i-1] {
			t.Errorf("rows are not oldest first: %v", dates)
			break
		}
	}
}

func TestReportNamesEachPlaylistAndCountsEveryVideo(t *testing.T) {
	res := fixtureResult(t)
	res.Units = 9
	md := res.Markdown()
	for _, want := range []string{
		"PLgen", "Book of Genesis Bible Study", "PLrev5", `Revelation Chapter 5 \| Allen Nolan`, "PLisa",
		"Videos found: 18",
		"Rows written: 14",
		"Skipped: 4",
		"no passage",
		"private",
		"III.E",
		"30 calendar days",
		"must either delete or refresh the stored data",
		"No Subsplash",
		"allennolan.com",
		"Quota used: 9 units",
		// The playlist that names a book but is no book series is named.
		"PLmyst",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("REPORT.md lacks %q", want)
		}
	}
	// The topical playlists are not listed as series.
	for _, not := range []string{"PLpray", "PLshorts"} {
		if strings.Contains(md, not) {
			t.Errorf("REPORT.md names %s, which is no book playlist and names no book", not)
		}
	}
	// Each series line has its id, title and video count.
	if !strings.Contains(md, "`PLrev` | The Book of Revelation Bible Study Verse by Verse | 7") {
		t.Errorf("the series table has no row for PLrev:\n%s", md)
	}
}

func TestSourceTOMLQuotesThePolicyAndPassesTheChecker(t *testing.T) {
	res := fixtureResult(t)
	toml := res.SourceTOML()
	for _, want := range []string{
		`id = "allennolan"`,
		`kind = "sermons"`,
		"not longer than 30 calendar days",
		"WATCH THE 30 DAYS",
		`terms_url = "https://developers.google.com/youtube/terms/developer-policies"`,
		"on_by_default = false",
	} {
		if !strings.Contains(toml, want) {
			t.Errorf("source.toml lacks %q", want)
		}
	}
	if strings.Contains(toml, testKey) {
		t.Error("source.toml holds the key")
	}
}

// A refresh whose share of passages is under 30% writes the report and no
// rows, and says so as an error.
func TestStopRuleWritesTheReportAndNoRows(t *testing.T) {
	var reqs []*http.Request
	c := newClient(fakeAPI(t, &reqs))
	ch, series, err := discover(context.Background(), c, testChannel)
	if err != nil {
		t.Fatal(err)
	}
	// Keep only Genesis's videos that name no passage.
	var few []SeriesItems
	for _, s := range series {
		if s.ID != "PLgen" {
			continue
		}
		var items []Item
		for _, it := range s.Items {
			if it.VideoID != "genAAAAAA03" {
				items = append(items, it)
			}
		}
		s.Items = items
		few = append(few, s)
	}
	res := Build(ch, few)
	if res.Share() >= minShare {
		t.Fatalf("share %.2f; the fixture should be under %.2f", res.Share(), minShare)
	}
	out := t.TempDir()
	report := filepath.Join(out, "REPORT.md")
	err = write(res, out, report, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "30%") {
		t.Fatalf("error %v, want the 30%% rule", err)
	}
	if _, err := os.Stat(report); err != nil {
		t.Errorf("no REPORT.md: %v", err)
	}
	for _, n := range []string{"rows.jsonl", "source.toml"} {
		if _, err := os.Stat(filepath.Join(out, n)); err == nil {
			t.Errorf("%s was written under the stop rule", n)
		}
	}
	b, _ := os.ReadFile(report)
	if !strings.Contains(string(b), "Stop rule") {
		t.Errorf("the report does not say the stop rule applied:\n%s", b)
	}
}

func TestRunReadsTheChannelAndWritesTheSource(t *testing.T) {
	var reqs []*http.Request
	c := newClient(fakeAPI(t, &reqs))
	out := t.TempDir()
	var stdout bytes.Buffer
	if err := run(context.Background(), c, testChannel, out, "", &stdout); err != nil {
		t.Fatal(err)
	}
	// channels.list, 2 playlists.list pages, one playlistItems.list for each of
	// the 6 book playlists.
	if len(reqs) != 9 || c.Units != 9 {
		t.Errorf("%d requests, %d units, want 9 and 9", len(reqs), c.Units)
	}
	for _, r := range reqs {
		if r.URL.Path == "/youtube/v3/playlistItems" {
			switch r.URL.Query().Get("playlistId") {
			case "PLpray", "PLmyst", "PLshorts":
				t.Errorf("the items of %s, which is no book series, were read", r.URL.Query().Get("playlistId"))
			}
		}
	}
	if !strings.Contains(stdout.String(), "rows 14") {
		t.Errorf("summary %q", stdout.String())
	}
	md, err := os.ReadFile(filepath.Join(out, "REPORT.md"))
	if err != nil || !strings.Contains(string(md), "Quota used: 9 units") {
		t.Errorf("report: %v\n%s", err, md)
	}
	if strings.Contains(stdout.String(), testKey) {
		t.Error("the summary holds the key")
	}
}
