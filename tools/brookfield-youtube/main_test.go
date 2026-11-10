package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Jonathan-A-White/sermon-index/internal/sources"
)

// testKey is a made-up key; nothing real is ever in a test.
const testKey = "TESTKEY-not-a-real-key-0000000000"

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// fakeAPI serves the recorded replies: channels.list, and playlistItems.list
// in two pages. It records each request.
func fakeAPI(t *testing.T, reqs *[]*http.Request) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*reqs = append(*reqs, r.Clone(context.Background()))
		if r.Header.Get("X-Goog-Api-Key") != testKey {
			http.Error(w, `{"error":{"code":403,"message":"The request is missing a valid API key."}}`, 403)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/youtube/v3/channels":
			w.Write(fixture(t, "channel.json"))
		case "/youtube/v3/playlistItems":
			if r.URL.Query().Get("pageToken") == "PAGE2TOKEN" {
				w.Write(fixture(t, "uploads-page2.json"))
			} else {
				w.Write(fixture(t, "uploads-page1.json"))
			}
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestClientWalksTheUploadsPlaylist(t *testing.T) {
	var reqs []*http.Request
	srv := fakeAPI(t, &reqs)
	c := &Client{Base: srv.URL, Key: testKey, HTTP: srv.Client()}

	pl, err := c.UploadsPlaylist(context.Background(), "@BrookfieldFirst")
	if err != nil {
		t.Fatal(err)
	}
	if pl != "UUYJE6WQ5ZJ9vkbWKjrxZGSg" {
		t.Errorf("uploads playlist %q", pl)
	}
	items, err := c.Uploads(context.Background(), pl)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 10 {
		t.Fatalf("%d items, want 10 (5 and 5)", len(items))
	}
	if len(reqs) != 3 {
		t.Fatalf("%d requests, want 3 (channels.list, then two pages)", len(reqs))
	}
	q := reqs[0].URL.Query()
	if q.Get("forHandle") != "@BrookfieldFirst" || q.Get("part") != "contentDetails" {
		t.Errorf("channels.list query %v", q)
	}
	q = reqs[1].URL.Query()
	if q.Get("part") != "snippet,contentDetails" || q.Get("maxResults") != "50" || q.Get("playlistId") != pl || q.Get("pageToken") != "" {
		t.Errorf("first page query %v", q)
	}
	if reqs[2].URL.Query().Get("pageToken") != "PAGE2TOKEN" {
		t.Errorf("second page query %v", reqs[2].URL.Query())
	}
	for _, r := range reqs {
		if strings.Contains(r.URL.String(), testKey) || r.URL.Query().Get("key") != "" {
			t.Errorf("the key is in the address: %s", r.URL)
		}
	}
	if c.Units != 3 {
		t.Errorf("quota %d units, want 3 (1 each)", c.Units)
	}
}

func TestClientErrorsNeverCarryTheKey(t *testing.T) {
	// A server that echoes the key in its error, and one that is gone.
	echo := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":{"code":400,"message":"API key not valid: `+testKey+`"}}`, 400)
	}))
	defer echo.Close()
	c := &Client{Base: echo.URL, Key: testKey, HTTP: echo.Client()}
	_, err := c.UploadsPlaylist(context.Background(), "@x")
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
	c = &Client{Base: base + "/?k=" + testKey, Key: testKey, HTTP: http.DefaultClient}
	_, err = c.UploadsPlaylist(context.Background(), "@x")
	if err == nil || strings.Contains(err.Error(), testKey) {
		t.Errorf("connection error: %v", err)
	}
}

func TestClientHandleNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"kind":"youtube#channelListResponse","pageInfo":{"totalResults":0,"resultsPerPage":5}}`))
	}))
	defer srv.Close()
	c := &Client{Base: srv.URL, Key: testKey, HTTP: srv.Client()}
	if _, err := c.UploadsPlaylist(context.Background(), "@Nobody"); err == nil || !strings.Contains(err.Error(), "@Nobody") {
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

func fixtureItems(t *testing.T) []Item {
	t.Helper()
	var reqs []*http.Request
	srv := fakeAPI(t, &reqs)
	c := &Client{Base: srv.URL, Key: testKey, HTTP: srv.Client()}
	items, err := c.Uploads(context.Background(), "UUYJE6WQ5ZJ9vkbWKjrxZGSg")
	if err != nil {
		t.Fatal(err)
	}
	return items
}

func str(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func rowByID(rows []sources.Row, id string) *sources.Row {
	for i := range rows {
		if rows[i].ID == id {
			return &rows[i]
		}
	}
	return nil
}

func TestBuildSortsTheVideos(t *testing.T) {
	res := Build(fixtureItems(t))

	if res.Seen != 10 {
		t.Errorf("seen %d", res.Seen)
	}
	if len(res.Skipped) != 1 || res.Skipped[0].ID != "privVid0001" {
		t.Errorf("skipped %+v", res.Skipped)
	}
	if len(res.Rows) != 6 || len(res.Rephil) != 3 {
		t.Fatalf("%d sermons, %d RePhil", len(res.Rows), len(res.Rephil))
	}

	// Sermon rows: the video's title as printed, the watch link, the date.
	r := rowByID(res.Rows, "genSodom001")
	if r == nil {
		t.Fatal("no row for the sermon whose description names a passage")
	}
	if r.Title != "Brookfield First - Online Service | August 23, 2020" {
		t.Errorf("title %q", r.Title)
	}
	if r.URL != "https://www.youtube.com/watch?v=genSodom001" {
		t.Errorf("url %q", r.URL)
	}
	if str(r.Ref) != "Gen.18.16-Gen.18.33" || str(r.Preacher) != "Phil Morgan" {
		t.Errorf("ref %q preacher %q", str(r.Ref), str(r.Preacher))
	}
	if str(r.Date) != "2020-08-23" {
		t.Errorf("date %q: the title's own date is the preaching day", str(r.Date))
	}
	if str(r.Kind) != "link" {
		t.Errorf("kind %q", str(r.Kind))
	}
	if strings.Contains(str(r.Notes), "Sodom") || strings.Contains(r.Title, "Sodom") {
		t.Errorf("a sermon title out of the description was kept: %q", str(r.Notes))
	}

	if j := rowByID(res.Rows, "johnWiley001"); j == nil || str(j.Preacher) != "Jim Wiley" || str(j.Ref) != "John.5.1-John.5.9" {
		t.Errorf("john row %+v", j)
	}
	// A passage in the title.
	if x := rowByID(res.Rows, "romans5Ttl1"); x == nil || str(x.Ref) != "Rom.5.1-Rom.5.11" || x.Preacher != nil || str(x.Date) != "2026-09-15" {
		t.Errorf("romans row %+v", x)
	}
	// A sermon with no passage and no preacher is kept all the same.
	n := rowByID(res.Rows, "onlSvc0001A")
	if n == nil || n.Ref != nil || n.Preacher != nil || str(n.Date) != "2026-11-01" {
		t.Errorf("plain service row %+v", n)
	}
	// A text over more than 3 chapters is a series with its first chapter.
	e := rowByID(res.Rows, "ephSeries001")
	if e == nil || e.Series == nil || !*e.Series || str(e.Ref) != "Eph.1" {
		t.Errorf("series row %+v", e)
	}
	// A passage the book table refuses: row kept, no ref, and listed.
	b := rowByID(res.Rows, "badRef000001")
	if b == nil || b.Ref != nil {
		t.Errorf("bad ref row %+v", b)
	}
	if len(res.BadRefs) != 1 || res.BadRefs[0].ID != "badRef000001" {
		t.Errorf("bad refs %+v", res.BadRefs)
	}
	// No row of the rows is a Morning RePhil.
	for _, r := range res.Rows {
		if strings.Contains(strings.ToLower(r.Title), "morning rephil") {
			t.Errorf("RePhil in the sermons: %q", r.Title)
		}
	}
	// Rows are in publish order, oldest first.
	for i := 1; i < len(res.Rows); i++ {
		if res.Published[res.Rows[i-1].ID] > res.Published[res.Rows[i].ID] {
			t.Errorf("rows out of publish order at %d", i)
		}
	}

	if res.WithPreacher != 2 || res.WithPassage != 4 || res.SeriesRows != 1 {
		t.Errorf("preacher %d passage %d series %d", res.WithPreacher, res.WithPassage, res.SeriesRows)
	}
	if res.First != "2020-07-12" || res.Last != "2026-11-09" {
		t.Errorf("date range %s to %s", res.First, res.Last)
	}
}

func TestBuildNumbersTheMorningRePhil(t *testing.T) {
	res := Build(fixtureItems(t))
	want := []RephilRow{
		{Number: 12, Title: "MORNING REPHIL #12", Date: "2024-05-02", URL: "https://www.youtube.com/watch?v=repHilMid012"},
		{Number: 477, Title: "Morning RePhil #477", Date: "2026-11-07", URL: "https://www.youtube.com/watch?v=repHil477A1"},
		// No number in the title: its place in publish order (third).
		{Number: 3, Title: "Morning RePhil", Date: "2026-11-09", URL: "https://www.youtube.com/watch?v=repHilNoNo1"},
	}
	if len(res.Rephil) != len(want) {
		t.Fatalf("%d RePhil", len(res.Rephil))
	}
	for i := range want {
		if res.Rephil[i] != want[i] {
			t.Errorf("RePhil %d: %+v, want %+v", i, res.Rephil[i], want[i])
		}
	}
	if len(res.Unnumbered) != 1 || res.Unnumbered[0] != "repHilNoNo1" {
		t.Errorf("unnumbered %v", res.Unnumbered)
	}
}

func TestTextPassages(t *testing.T) {
	for _, c := range []struct {
		in   string
		want []string
	}{
		{"Romans 5:1-11", []string{"Rom.5.1-Rom.5.11"}},
		{"from Mark 6:35–44.", []string{"Mark.6.35-Mark.6.44"}},
		{"Psalm 23", []string{"Ps.23"}},
		{"Psalms 23:1", []string{"Ps.23.1"}},
		{"1 John 4:7-12 and John 3:16", []string{"1John.4.7-1John.4.12", "John.3.16"}},
		{"Song of Solomon 2:1", []string{"Song.2.1"}},
		{"Genesis 1:1-2:3", []string{"Gen.1.1-Gen.2.3"}},
		{"Ephesians 1-6", []string{"Eph.1-Eph.6"}},
		{"2 Peter 2:4-10a", []string{"2Pet.2.4-2Pet.2.10"}},
		{"1 Peter 4:12b-19", []string{"1Pet.4.12-1Pet.4.19"}},
		{"2 Peter 2:10a", []string{"2Pet.2.10"}},
		{"John 3:16b-4:3a", []string{"John.3.16-John.4.3"}},
		{"Online Service | August 23, 2020", nil},
		{"Brookfield, CT 06804 and text 73256", nil},
		{"Acts of kindness", nil},
	} {
		got, bad := textPassages(c.in)
		if len(bad) != 0 {
			t.Errorf("%q: unreadable %v", c.in, bad)
		}
		if strings.Join(got, ",") != strings.Join(c.want, ",") {
			t.Errorf("%q: %v, want %v", c.in, got, c.want)
		}
	}
	if _, bad := textPassages("Romans 99:1"); len(bad) != 1 {
		t.Errorf("Romans 99:1 should be unreadable, got %v", bad)
	}
	// A letter on a chapter is not a half verse: it is listed, never read short.
	if got, bad := textPassages("Isaiah 40-41a"); len(bad) != 1 || len(got) != 0 {
		t.Errorf("Isaiah 40-41a: read %v, unreadable %v; want it unreadable", got, bad)
	}
}

func TestTextPreacher(t *testing.T) {
	for in, want := range map[string]string{
		`Sermon Title: "X" (Pastor Phil Morgan) from Mark 6:35-44`: "Phil Morgan",
		`(Pastor Jim Wiley)`:                           "Jim Wiley",
		`Sermon (Rev. Ann Lee) from Luke 1`:            "Ann Lee",
		`Pastor Phil Morgan's office is open`:          "",
		`Give online (one-time or recurring)`:          "",
		`Brookfield First is a gospel-centered church`: "",
	} {
		if got := textPreacher(in); got != want {
			t.Errorf("%q: %q, want %q", in, got, want)
		}
	}
}

func runFixture(t *testing.T, out, rephil, report string) (string, error) {
	t.Helper()
	return runFixtureWith(t, out, rephil, report, "")
}

func runFixtureWith(t *testing.T, out, rephil, report, list string) (string, error) {
	t.Helper()
	var reqs []*http.Request
	srv := fakeAPI(t, &reqs)
	c := &Client{Base: srv.URL, Key: testKey, HTTP: srv.Client()}
	var stdout bytes.Buffer
	err := run(context.Background(), c, "@BrookfieldFirst", out, rephil, report, list, &stdout)
	return stdout.String(), err
}

func TestRunWritesTheSameBytesTwice(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "sources", "brookfield-first")
	rephil := filepath.Join(dir, "data", "morning-rephil.jsonl")
	report := filepath.Join(out, "REPORT.md")

	var first = map[string][]byte{}
	for i := 0; i < 2; i++ {
		stdout, err := runFixture(t, out, rephil, report)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(stdout, testKey) {
			t.Error("the key was printed")
		}
		for _, p := range []string{filepath.Join(out, "rows.jsonl"), filepath.Join(out, "source.toml"), report, rephil} {
			b, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			if i == 0 {
				first[p] = b
			} else if !bytes.Equal(first[p], b) {
				t.Errorf("%s: the second run wrote other bytes", p)
			}
			if bytes.Contains(b, []byte(testKey)) || bytes.Contains(b, []byte("AI"+"za")) {
				t.Errorf("%s holds a key", p)
			}
		}
	}

	// rows.jsonl: valid rows, every url a watch link, no HTML escaping.
	rows := first[filepath.Join(out, "rows.jsonl")]
	if n := bytes.Count(rows, []byte("\n")); n != 6 {
		t.Errorf("%d rows", n)
	}
	for _, line := range strings.Split(strings.TrimSpace(string(rows)), "\n") {
		var r sources.Row
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(r.URL, "https://www.youtube.com/watch?v=") {
			t.Errorf("url %q", r.URL)
		}
	}

	// morning-rephil.jsonl: number, title, date, url.
	lines := strings.Split(strings.TrimSpace(string(first[rephil])), "\n")
	if len(lines) != 3 {
		t.Fatalf("%d RePhil lines", len(lines))
	}
	if lines[1] != `{"number":477,"title":"Morning RePhil #477","date":"2026-11-07","url":"https://www.youtube.com/watch?v=repHil477A1"}` {
		t.Errorf("RePhil line %s", lines[1])
	}

	// The source passes the rig's check.
	faults, warnings := sources.CheckReport(out)
	if len(faults) != 0 || len(warnings) != 0 {
		t.Errorf("check: faults %v warnings %v", faults, warnings)
	}
	toml := string(first[filepath.Join(out, "source.toml")])
	for _, w := range []string{`id = "brookfield-first"`, "on_by_default = false", `credit = "Brookfield First (Brookfield Assembly of God), Brookfield CT"`, "licence_note"} {
		if !strings.Contains(toml, w) {
			t.Errorf("source.toml lacks %q", w)
		}
	}

	// REPORT.md: the counts, the date range, what was not sorted, the quota.
	rep := string(first[report])
	for _, w := range []string{
		"Videos seen: 10", "Sermons: 6", "with a preacher: 2", "with a passage: 4", "Morning RePhil episodes: 3",
		"2020-07-12", "2026-11-09", "Quota used: 3 units", "privVid0001", "badRef000001", "repHilNoNo1",
	} {
		if !strings.Contains(rep, w) {
			t.Errorf("REPORT.md lacks %q\n%s", w, rep)
		}
	}
}

func TestRunStopsOnAnAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":{"code":403,"message":"quotaExceeded"}}`, 403)
	}))
	defer srv.Close()
	c := &Client{Base: srv.URL, Key: testKey, HTTP: srv.Client()}
	dir := t.TempDir()
	err := run(context.Background(), c, "@BrookfieldFirst", dir, filepath.Join(dir, "r.jsonl"), "", "", &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "403") {
		t.Fatalf("error %v", err)
	}
	if _, serr := os.Stat(filepath.Join(dir, "rows.jsonl")); !errors.Is(serr, os.ErrNotExist) {
		t.Error("rows.jsonl written after a failed run")
	}
}

// readRowsFile reads a rows.jsonl as rows, for a test.
func readRowsFile(t *testing.T, path string) []sources.Row {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var rows []sources.Row
	for _, line := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		var r sources.Row
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			t.Fatal(err)
		}
		rows = append(rows, r)
	}
	return rows
}

// A regeneration keeps the alt_url that tools/subsplash-links wrote to a row,
// matched on the row's id; a row that is gone loses it and a new row has none;
// url (YouTube) is rewritten from the API; and a second run changes nothing.
func TestRunKeepsTheAltURLsOfTheRowsItRewrites(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "sources", "brookfield-first")
	rephil := filepath.Join(dir, "data", "morning-rephil.jsonl")
	report := filepath.Join(out, "REPORT.md")

	// The rows as the API makes them, to take two ids from.
	if _, err := runFixture(t, out, rephil, report); err != nil {
		t.Fatal(err)
	}
	fresh := readRowsFile(t, filepath.Join(out, "rows.jsonl"))
	if len(fresh) < 3 {
		t.Fatalf("the fixture makes %d rows, want at least 3", len(fresh))
	}
	kept := fresh[0]
	if kept.AltURL != nil {
		t.Fatal("a fresh row has an alt_url")
	}

	// The rows file as it stands after subsplash-links: the first row has an
	// alt_url and an old url, a row the channel no longer lists has an
	// alt_url, and the second row (new since) is not there at all.
	const keptAlt = "https://subsplash.com/u/brookfieldfirst/media/d/abc1234-the-garden-gate"
	const goneAlt = "https://subsplash.com/u/brookfieldfirst/media/d/zzz9999-a-removed-sermon"
	old := kept
	old.AltURL = ptr(keptAlt)
	old.URL = "https://www.youtube.com/watch?v=oldaddress"
	gone := sources.Row{ID: "removedVid1", Title: "A Removed Sermon", URL: watchURL + "removedVid1", AltURL: ptr(goneAlt)}
	var seed bytes.Buffer
	for _, r := range []sources.Row{old, gone} {
		if err := writeLine(&seed, r); err != nil {
			t.Fatal(err)
		}
	}
	rowsPath := filepath.Join(out, "rows.jsonl")
	if err := os.WriteFile(rowsPath, seed.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}

	var first []byte
	for i := 0; i < 2; i++ {
		if _, err := runFixture(t, out, rephil, report); err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(rowsPath)
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			first = b
		} else if !bytes.Equal(first, b) {
			t.Error("the second regeneration wrote other bytes")
		}
	}

	rows := readRowsFile(t, rowsPath)
	if len(rows) != len(fresh) {
		t.Fatalf("%d rows after the regeneration, want %d", len(rows), len(fresh))
	}
	for _, r := range rows {
		switch r.ID {
		case kept.ID:
			if r.AltURL == nil || *r.AltURL != keptAlt {
				t.Errorf("row %s lost its alt_url: %v", r.ID, r.AltURL)
			}
			if r.URL != kept.URL {
				t.Errorf("row %s: url is %q, want the API's %q", r.ID, r.URL, kept.URL)
			}
		default:
			if r.AltURL != nil {
				t.Errorf("row %s has an alt_url %q, want none", r.ID, *r.AltURL)
			}
		}
		if r.ID == "removedVid1" {
			t.Error("the removed row is back")
		}
	}
	if strings.Contains(string(first), goneAlt) {
		t.Error("the alt_url of a row that no longer exists was kept")
	}
}

// A rows file that cannot be read stops the run and is left as it is: going on
// would drop the alt_url of every row in it.
func TestRunStopsOnAnUnreadableRowsFile(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "sources", "brookfield-first")
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}
	bad := []byte("{\"id\": \"a\", \"alt_url\": \n")
	rowsPath := filepath.Join(out, "rows.jsonl")
	if err := os.WriteFile(rowsPath, bad, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := runFixture(t, out, filepath.Join(dir, "r.jsonl"), filepath.Join(out, "REPORT.md")); err == nil {
		t.Fatal("a run over an unreadable rows file went on")
	}
	if b, _ := os.ReadFile(rowsPath); !bytes.Equal(b, bad) {
		t.Error("the unreadable rows file was rewritten")
	}
}
