package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Jonathan-A-White/sermon-index/internal/sources"
)

type sermonsJSON struct {
	Ref   string `json:"ref"`
	Total int    `json:"total"`
	Rows  []struct {
		Source string  `json:"source"`
		ID     string  `json:"id"`
		Title  string  `json:"title"`
		AltURL *string `json:"alt_url"`
		Role   string  `json:"role"`
		Width  int     `json:"width"`
		Ref    string  `json:"ref"`
		Kind   string  `json:"kind"`
		Date   *string `json:"date"`
		URL    string  `json:"url"`
		Preach *string `json:"preacher"`
	} `json:"rows"`
}

func getSermons(t *testing.T, ts *testServer, target string) (int, sermonsJSON, *http.Response) {
	t.Helper()
	res := ts.do("GET", target)
	var s sermonsJSON
	if res.StatusCode == 200 {
		if err := json.NewDecoder(res.Body).Decode(&s); err != nil {
			t.Fatal(err)
		}
	}
	return res.StatusCode, s, res
}

func TestSermonsByRef(t *testing.T) {
	ts := newTestServer(t, loadedDB(t))
	code, s, _ := getSermons(t, ts, "/v1/sermons?ref=rom.5.8")
	if code != 200 || s.Ref != "Rom.5.8" || s.Total != 1 || len(s.Rows) != 1 || s.Rows[0].Title != "Love's Commendation" ||
		s.Rows[0].Role != "text" || s.Rows[0].Width != 1 || s.Rows[0].Source != "spurgeon-20" || s.Rows[0].ID != "104" {
		t.Errorf("rom.5.8: %d %+v", code, s)
	}

	ids := func(s sermonsJSON) []string {
		var out []string
		for _, r := range s.Rows {
			out = append(out, r.ID)
		}
		return out
	}
	_, s, _ = getSermons(t, ts, "/v1/sermons?ref=Rom.5")
	if want := []string{"37", "104", "330", "1191", "2544", "3392"}; !reflect.DeepEqual(ids(s), want) || s.Total != 6 || s.Ref != "Rom.5" {
		t.Errorf("Rom.5: %v total %d", ids(s), s.Total)
	}
	_, s, _ = getSermons(t, ts, "/v1/sermons?ref=Rom.5&limit=2&offset=3")
	if want := []string{"1191", "2544"}; !reflect.DeepEqual(ids(s), want) || s.Total != 6 {
		t.Errorf("Rom.5 limit 2 offset 3: %v total %d", ids(s), s.Total)
	}
	_, s, _ = getSermons(t, ts, "/v1/sermons?ref=Rom.5&offset=6")
	if s.Rows == nil || len(s.Rows) != 0 || s.Total != 6 {
		t.Errorf("past the end: %+v", s)
	}
	_, s, _ = getSermons(t, ts, "/v1/sermons?ref=Rom.5&source=nosuch,spurgeon-20")
	if len(s.Rows) != 6 {
		t.Errorf("source=nosuch,spurgeon-20: %v", ids(s))
	}
	_, s, _ = getSermons(t, ts, "/v1/sermons?ref=Rom.5&source=themes-demo")
	if len(s.Rows) != 0 || s.Total != 0 {
		t.Errorf("source=themes-demo: %v", ids(s))
	}
	// A range in Lampas's codes, and a whole book.
	_, s, _ = getSermons(t, ts, "/v1/sermons?ref=rom.3-rom.5")
	if s.Ref != "Rom.3-Rom.5" || s.Total != 7 {
		t.Errorf("rom.3-rom.5: %s %d", s.Ref, s.Total)
	}
	_, s, _ = getSermons(t, ts, "/v1/sermons?ref=Jude")
	if s.Ref != "Jude" || s.Total != 0 || s.Rows == nil {
		t.Errorf("Jude: %+v", s)
	}
	// The default limit is 50 and the most is 200.
	if code, _, _ := getSermons(t, ts, "/v1/sermons?ref=Gen-Rev&limit=200"); code != 200 {
		t.Errorf("limit=200: %d", code)
	}
	_, s, _ = getSermons(t, ts, "/v1/sermons?ref=Gen-Rev")
	if s.Total != 19 || len(s.Rows) != 19 {
		t.Errorf("Gen-Rev: total %d, %d rows", s.Total, len(s.Rows))
	}
}

func TestSermonsDefaultLimitIs50AndTheMost200(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "many")
	os.Mkdir(dir, 0o755)
	toml, err := os.ReadFile(filepath.Join(spurgeon20, "source.toml"))
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(dir, "source.toml"), []byte(strings.Replace(string(toml), `id = "spurgeon-20"`, `id = "many"`, 1)), 0o644)
	var rows strings.Builder
	for i := 1; i <= 210; i++ {
		fmt.Fprintf(&rows, `{"id": "%d", "title": "Sermon %d", "ref": "Rom.5.8", "url": "https://example.org/%d"}`+"\n", i, i, i)
	}
	os.WriteFile(filepath.Join(dir, "rows.jsonl"), []byte(rows.String()), 0o644)
	db := loadedDB(t)
	if _, err := sources.Load(context.Background(), db, dir); err != nil {
		t.Fatal(err)
	}
	ts := newTestServer(t, db)
	_, s, _ := getSermons(t, ts, "/v1/sermons?ref=Rom.5.8&source=many")
	if s.Total != 210 || len(s.Rows) != 50 || s.Rows[0].ID != "1" || s.Rows[49].ID != "50" {
		t.Errorf("default: total %d, %d rows", s.Total, len(s.Rows))
	}
	_, s, _ = getSermons(t, ts, "/v1/sermons?ref=Rom.5.8&source=many&limit=200&offset=5")
	if s.Total != 210 || len(s.Rows) != 200 || s.Rows[0].ID != "6" {
		t.Errorf("limit=200 offset=5: total %d, %d rows", s.Total, len(s.Rows))
	}
}

func TestSermonsRefusesBadQueries(t *testing.T) {
	ts := newTestServer(t, loadedDB(t))
	cases := []struct {
		target, code, detail string
	}{
		{"/v1/sermons", "bad_ref", "ref is required: an OSIS reference such as Rom.5.8, Rom.5 or Rom.5.1-Rom.5.11"},
		{"/v1/sermons?ref=", "bad_ref", "ref is required: an OSIS reference such as Rom.5.8, Rom.5 or Rom.5.1-Rom.5.11"},
		{"/v1/sermons?ref=Rom.5.99", "bad_ref", "ref 'Rom.5.99': verse past the chapter"},
		{"/v1/sermons?ref=Tob.1", "bad_ref", "ref 'Tob.1': unknown book 'Tob'"},
		{"/v1/sermons?ref=Rom.5.1-11", "bad_ref", "ref 'Rom.5.1-11': the end of a range is a whole reference (Rom.5.1-Rom.5.11)"},
		{"/v1/sermons?ref=Rom.5&limit=0", "bad_limit", "limit must be a whole number from 1 to 200"},
		{"/v1/sermons?ref=Rom.5&limit=201", "bad_limit", "limit must be a whole number from 1 to 200"},
		{"/v1/sermons?ref=Rom.5&limit=ten", "bad_limit", "limit must be a whole number from 1 to 200"},
		{"/v1/sermons?ref=Rom.5&offset=-1", "bad_offset", "offset must be a whole number, 0 or more"},
		{"/v1/sermons?ref=Rom.5&offset=x", "bad_offset", "offset must be a whole number, 0 or more"},
		{"/v1/themes", "bad_ref", "ref is required: an OSIS reference such as Rom.5.8, Rom.5 or Rom.5.1-Rom.5.11"},
		{"/v1/themes?ref=Ps.151", "bad_ref", "ref 'Ps.151': chapter past the book"},
	}
	for _, c := range cases {
		res := ts.do("GET", c.target)
		if res.StatusCode != 400 {
			t.Errorf("%s: %d", c.target, res.StatusCode)
		}
		code, detail := errorBody(t, res)
		if code != c.code || detail != c.detail {
			t.Errorf("%s: {%q, %q}, want {%q, %q}", c.target, code, detail, c.code, c.detail)
		}
	}
}

// The tiers over the fixtures: Romans 5:8's passage row is 104; its theme
// row is 1191 (Rom.5.6 meets the demo Atonement's Rom.5.6-Rom.5.11), and 104
// is not repeated in it though Love and Atonement both link it.
func TestSermonsTiers(t *testing.T) {
	ts := newTestServer(t, loadedDB(t))
	both := getTiers(t, ts, "/v1/sermons?ref=rom.5.8&tier=theme")
	if both.Ref != "Rom.5.8" || both.Passage == nil || both.Passage.Total != 1 || !both.Passage.has("spurgeon-20", "104") {
		t.Errorf("passage tier: %+v", both.Passage)
	}
	if both.Theme == nil || both.Theme.Total != 1 || len(both.Theme.Rows) != 1 || !both.Theme.has("spurgeon-20", "1191") {
		t.Fatalf("theme tier: %+v", both.Theme)
	}
	if r := both.Theme.Rows[0]; r.Ref != "Rom.5.6" || r.Role != "text" || r.Theme.Source != "themes-demo" || r.Theme.Topic != "atonement" ||
		r.Theme.Label != "Atonement" || r.Theme.Part != nil || r.Theme.PartLabel != nil {
		t.Errorf("theme row: %+v %+v", r, r.Theme)
	}

	only := getTiers(t, ts, "/v1/sermons?ref=rom.5.8&tier=theme-only")
	if only.Passage != nil || only.Theme == nil || only.Theme.Total != 1 {
		t.Errorf("theme-only: %+v", only)
	}
	// The comma list names the same two.
	if c := getTiers(t, ts, "/v1/sermons?ref=rom.5.8&tier=passage,theme"); c.Passage == nil || c.Theme == nil {
		t.Errorf("tier=passage,theme: %+v", c)
	}
	// theme= restricts to a topic; another topic's sermons are not listed.
	if r := getTiers(t, ts, "/v1/sermons?ref=rom.5.8&tier=theme&theme=themes-demo:love"); r.Theme.Total != 0 || r.Theme.Rows == nil {
		t.Errorf("theme=themes-demo:love: %+v", r.Theme)
	}
	if r := getTiers(t, ts, "/v1/sermons?ref=rom.5.8&tier=theme&theme=themes-demo:atonement"); r.Theme.Total != 1 {
		t.Errorf("theme=themes-demo:atonement: %+v", r.Theme)
	}
	// limit and offset apply to each tier.
	if r := getTiers(t, ts, "/v1/sermons?ref=Rom.5&tier=theme&limit=2&offset=5"); len(r.Passage.Rows) != 1 || r.Passage.Total != 6 || r.Theme.Rows == nil {
		t.Errorf("Rom.5 limit 2 offset 5: %+v %+v", r.Passage, r.Theme)
	}
	// With no tier the answer is the flat one, as before.
	res := ts.do("GET", "/v1/sermons?ref=rom.5.8")
	var flat map[string]json.RawMessage
	if err := json.Unmarshal([]byte(body(t, res)), &flat); err != nil || flat["rows"] == nil || flat["passage"] != nil || flat["theme"] != nil {
		t.Errorf("no tier: %v %v", flat, err)
	}
	if res := ts.do("GET", "/v1/sermons?ref=rom.5.8&tier=passage"); body(t, res) != body(t, ts.do("GET", "/v1/sermons?ref=rom.5.8")) {
		t.Error("tier=passage is not the flat answer")
	}
}

func TestSermonsRefusesBadTierAndTheme(t *testing.T) {
	ts := newTestServer(t, loadedDB(t))
	for _, c := range []struct{ target, code string }{
		{"/v1/sermons?ref=Rom.5&tier=story", "bad_tier"},
		{"/v1/sermons?ref=Rom.5&tier=", "bad_tier"},
		{"/v1/sermons?ref=Rom.5&tier=theme&theme=love", "bad_theme"},
		{"/v1/sermons?ref=Rom.5&tier=theme&theme=:love", "bad_theme"},
	} {
		res := ts.do("GET", c.target)
		if res.StatusCode != 400 {
			t.Errorf("%s: %d", c.target, res.StatusCode)
			continue
		}
		if code, detail := errorBody(t, res); code != c.code || detail == "" {
			t.Errorf("%s: {%q, %q}", c.target, code, detail)
		}
	}
}

// A `series` row (a catalogue's whole-book teaching, here Romans in 16
// chapters) is kept, and found by its title, but a passage never answers with
// it: Romans 5:8 gets the one verse's sermon alone.
func TestSermonsExcludeSeriesRows(t *testing.T) {
	db := loadedDB(t)
	dir := filepath.Join(t.TempDir(), "series-demo")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	toml := `id = "series-demo"
name = "Series demo"
kind = "sermons"
licence = "Public domain."
credit = "Someone"
homepage = "https://example.org"
terms_url = "https://example.org/terms"
added = 2026-11-05
on_by_default = false
`
	rows := `{"id": "v8", "title": "Commended Love", "ref": "Rom.5.8", "url": "https://example.org/v8"}
{"id": "all", "title": "Romans Whole Book Overview", "ref": "Rom.1.1-Rom.16.24", "url": "https://example.org/all", "series": true}
`
	for name, body := range map[string]string{"source.toml": toml, "rows.jsonl": rows} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := sources.Load(context.Background(), db, dir); err != nil {
		t.Fatal(err)
	}
	ts := newTestServer(t, db)
	for _, target := range []string{"/v1/sermons?ref=rom.5.8", "/v1/sermons?ref=Rom.5", "/v1/sermons?ref=Rom", "/v1/sermons?ref=Rom.5.8&tier=theme"} {
		code, s, _ := getSermons(t, ts, target)
		if code != 200 {
			t.Fatalf("%s: %d", target, code)
		}
		for _, r := range s.Rows {
			if r.Source == "series-demo" && r.ID == "all" {
				t.Errorf("%s answered the series row: %+v", target, r)
			}
		}
	}
	code, s, _ := getSermons(t, ts, "/v1/sermons?ref=rom.5.8&source=series-demo")
	if code != 200 || s.Total != 1 || s.Rows[0].ID != "v8" {
		t.Errorf("series-demo at rom.5.8: %d %+v", code, s)
	}
	// ... but the row is in the database and search finds it by its title.
	res := ts.do("GET", "/v1/search?q=Romans+Overview")
	b := body(t, res)
	if !strings.Contains(b, `"id":"all"`) && !strings.Contains(b, `"id": "all"`) {
		t.Errorf("search does not find the series row: %s", b)
	}
}

// rawTheme reads target's theme tier as raw JSON objects, so a test sees a
// key left out (no "direct" on a row reached through its text) and a null.
func rawTheme(t *testing.T, ts *testServer, target string) (int, []map[string]any) {
	t.Helper()
	res := ts.do("GET", target)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET %s: %s: %s", target, res.Status, body(t, res))
	}
	var out struct {
		Theme struct {
			Total int              `json:"total"`
			Rows  []map[string]any `json:"rows"`
		} `json:"theme"`
	}
	if err := json.Unmarshal([]byte(body(t, res)), &out); err != nil {
		t.Fatal(err)
	}
	return out.Theme.Total, out.Theme.Rows
}

// Direct links (sermon-topics.jsonl): the fixture links 12, The Peculiar
// Sleep of the Beloved (Psalm 127:2), to the demo topic Grace, which links
// Romans 5:15-21 and Ephesians 2:8. At Ephesians 2:8 its row comes first,
// marked "direct": true and naming Grace; 37, 330 and 2544, whose texts
// meet Grace in Romans 5, follow as before, with no "direct" key.
func TestSermonsDirectLinks(t *testing.T) {
	ts := newTestServer(t, loadedDB(t))
	total, rows := rawTheme(t, ts, "/v1/sermons?ref=eph.2.8&tier=theme-only")
	var ids []string
	for _, r := range rows {
		ids = append(ids, r["id"].(string))
	}
	if want := []string{"12", "37", "330", "2544"}; total != 4 || !reflect.DeepEqual(ids, want) {
		t.Fatalf("eph.2.8: total %d, rows %v, want %v", total, ids, want)
	}
	d := rows[0]
	theme, _ := d["theme"].(map[string]any)
	if d["direct"] != true || d["source"] != "spurgeon-20" || d["ref"] != "Ps.127.2" || d["role"] != "text" || d["width"] != 1.0 ||
		theme["source"] != "themes-demo" || theme["topic"] != "grace" || theme["label"] != "Grace" || theme["part"] != nil || theme["part_label"] != nil {
		t.Errorf("the direct row: %v", d)
	}
	for _, r := range rows[1:] {
		if _, has := r["direct"]; has {
			t.Errorf("a row reached through its text has a direct key: %v", r)
		}
	}

	// direct=only: the direct rows alone, and total counts them only.
	if total, rows := rawTheme(t, ts, "/v1/sermons?ref=eph.2.8&tier=theme&direct=only"); total != 1 || len(rows) != 1 || rows[0]["id"] != "12" {
		t.Errorf("direct=only: %d %v", total, rows)
	}
	// theme= names the direct link's topic, or another.
	if _, rows := rawTheme(t, ts, "/v1/sermons?ref=Rom.5&tier=theme-only&theme=themes-demo:grace"); len(rows) == 0 || rows[0]["id"] != "12" || rows[0]["direct"] != true {
		t.Errorf("theme=themes-demo:grace at Rom.5: %v", rows)
	}
	if _, rows := rawTheme(t, ts, "/v1/sermons?ref=Rom.5&tier=theme-only&theme=themes-demo:atonement"); len(rows) != 0 {
		for _, r := range rows {
			if r["id"] == "12" {
				t.Errorf("theme=themes-demo:atonement lists the sermon linked to grace: %v", r)
			}
		}
	}
	// source= leaves a direct row of a source not named out.
	if total, _ := rawTheme(t, ts, "/v1/sermons?ref=eph.2.8&tier=theme-only&direct=only&source=nosuch"); total != 0 {
		t.Errorf("source=nosuch: %d", total)
	}

	// A bad direct is 400 bad_direct; tier=passage ignores a good one.
	for _, target := range []string{"/v1/sermons?ref=eph.2.8&tier=theme&direct=yes", "/v1/sermons?ref=eph.2.8&tier=theme&direct=", "/v1/sermons?ref=eph.2.8&direct=all"} {
		res := ts.do("GET", target)
		if res.StatusCode != 400 {
			t.Errorf("%s: %d", target, res.StatusCode)
			continue
		}
		if code, detail := errorBody(t, res); code != "bad_direct" || detail != "direct must be only (the theme tier's directly linked sermons alone), or left out" {
			t.Errorf("%s: {%q, %q}", target, code, detail)
		}
	}
	if a, b := body(t, ts.do("GET", "/v1/sermons?ref=rom.5.8&tier=passage&direct=only")), body(t, ts.do("GET", "/v1/sermons?ref=rom.5.8")); a != b {
		t.Errorf("tier=passage&direct=only is not the flat answer:\n%s\n%s", a, b)
	}
}

// pagesOf reads every page of the tier named (passage or theme) of target,
// limit rows at a time from offset, and returns the "source:id" of each row.
func pagesOf(t *testing.T, ts *testServer, target, tier string, limit int) []string {
	t.Helper()
	var ids []string
	for offset := 0; ; {
		ts.clock.add(time.Minute) // the rate limit is not under test
		got := getTiers(t, ts, fmt.Sprintf("%s&limit=%d&offset=%d", target, limit, offset))
		page := got.Theme
		if tier == "passage" {
			page = got.Passage
		}
		if page == nil {
			return ids
		}
		for _, r := range page.Rows {
			ids = append(ids, r.Source+":"+r.ID)
		}
		offset += len(page.Rows)
		if len(page.Rows) == 0 || offset >= page.Total {
			return ids
		}
	}
}

// Wherever both tiers are listed, no sermon is in both: over the whole Bible
// with a named theme too (the theme-only ask keeps the passage rows, as the
// count of a theme's sermons; the tier=theme ask does not).
func TestTiersAreDifferentSermons(t *testing.T) {
	ts := newTestServer(t, loadedDB(t))
	refs := []string{"eph.2.8", "Rom.5", "Gen.1.1-Rev.22.21"}
	themes := []string{"", "&theme=themes-demo:grace", "&theme=themes-demo:atonement"}
	for _, ref := range refs {
		for _, theme := range themes {
			for _, tier := range []string{"theme", "passage,theme", "theme-only"} {
				for _, direct := range []string{"", "&direct=only"} {
					for _, limit := range []int{200, 2, 1} {
						target := "/v1/sermons?ref=" + ref + "&tier=" + tier + theme + direct
						name := fmt.Sprintf("%s limit=%d", target, limit)
						themeIDs := pagesOf(t, ts, target, "theme", limit)
						seen := map[string]bool{}
						for _, id := range themeIDs {
							if seen[id] {
								t.Errorf("%s: %s is listed twice in the theme tier", name, id)
							}
							seen[id] = true
						}
						if tier == "theme-only" {
							continue
						}
						for _, id := range pagesOf(t, ts, target, "passage", limit) {
							if seen[id] {
								t.Errorf("%s: %s is in both tiers", name, id)
							}
						}
					}
				}
			}
		}
	}
	// theme-only over the whole Bible with a named theme still answers the
	// topic's sermons; beside the passage tier the theme tier is empty.
	whole := "/v1/sermons?ref=Gen.1.1-Rev.22.21&theme=themes-demo:grace"
	if got := pagesOf(t, ts, whole+"&tier=theme-only", "theme", 200); len(got) == 0 {
		t.Error("theme-only over the whole Bible with a named theme answers nothing")
	}
	if got := pagesOf(t, ts, whole+"&tier=theme", "theme", 200); len(got) != 0 {
		t.Errorf("tier=theme over the whole Bible lists the passage rows again: %v", got)
	}
}
