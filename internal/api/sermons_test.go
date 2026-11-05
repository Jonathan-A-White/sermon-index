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
