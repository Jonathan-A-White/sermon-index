package api

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/Jonathan-A-White/sermon-index/internal/migrate"
	"github.com/Jonathan-A-White/sermon-index/internal/sources"
	"github.com/Jonathan-A-White/sermon-index/internal/store"
)

const (
	spurgeonCCEL = "../../sources/spurgeon-ccel"
	themesTCR    = "../../sources/themes-tcr1934"
	themesTorrey = "../../sources/themes-torrey"
)

type themeRowJSON struct {
	Source    string   `json:"source"`
	Topic     string   `json:"topic"`
	Label     string   `json:"label"`
	Part      *int     `json:"part"`
	PartLabel *string  `json:"part_label"`
	Ref       string   `json:"ref"`
	See       []string `json:"see"`
}

// themesOf asks /v1/themes?ref= and returns each theme source's rows.
func themesOf(t *testing.T, ts *testServer, ref string) map[string][]themeRowJSON {
	t.Helper()
	res := ts.do("GET", "/v1/themes?ref="+ref)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET /v1/themes?ref=%s: %s: %s", ref, res.Status, body(t, res))
	}
	var out struct {
		Rows []themeRowJSON `json:"rows"`
	}
	if err := json.Unmarshal([]byte(body(t, res)), &out); err != nil {
		t.Fatal(err)
	}
	by := map[string][]themeRowJSON{}
	for _, r := range out.Rows {
		by[r.Source] = append(by[r.Source], r)
	}
	return by
}

// topicLabels is the labels of the distinct topics of the rows, sorted.
func topicLabels(rows []themeRowJSON) []string {
	seen := map[string]string{}
	for _, r := range rows {
		seen[r.Topic] = r.Label
	}
	var out []string
	for _, l := range seen {
		out = append(out, l)
	}
	sort.Strings(out)
	return out
}

// The two real theme sources, Thompson's 1934 topics and Torrey's, load
// into the embedded Postgres and answer Romans 5:8 and Romans 5 with the
// topics the probe found (Lampas docs/sources/sermon-index.md, section 7).
func TestThemeBackboneAnswersRomans5(t *testing.T) {
	ctx := context.Background()
	db := store.TestDB(t)
	if _, err := migrate.Up(ctx, db); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{themesTCR, themesTorrey} {
		if fs := sources.Check(dir); len(fs) != 0 {
			t.Fatalf("check %s: %v", dir, sources.Faults(fs))
		}
		if _, err := sources.Load(ctx, db, dir); err != nil {
			t.Fatalf("load %s: %v", dir, err)
		}
	}
	ts := newTestServer(t, db)

	// Romans 5:8.
	by := themesOf(t, ts, "rom.5.8")
	if got := topicLabels(by["themes-tcr1934"]); len(got) != 2 || got[0] != "Love-Hatred" || got[1] != "Sinner's Friend, Christ the" {
		t.Errorf("Thompson, Romans 5:8: %v", got)
	}
	var ofGod bool
	for _, r := range by["themes-tcr1934"] {
		if r.Label == "Love-Hatred" && r.Part != nil && *r.Part == 4 && r.PartLabel != nil && *r.PartLabel == "Of God" && r.Ref == "Rom.5.8" {
			ofGod = true
		}
	}
	if !ofGod {
		t.Errorf("no row 'Love-Hatred (4) Of God' at Rom.5.8: %+v", by["themes-tcr1934"])
	}
	want := []string{"Atonement, The", "Atonement, Under the Law", "Love of God, The", "Reconciliation With God", "Salvation"}
	got := topicLabels(by["themes-torrey"])
	if len(got) != len(want) {
		t.Fatalf("Torrey, Romans 5:8: %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Torrey, Romans 5:8: %v, want %v", got, want)
			break
		}
	}

	// Romans 5, the whole chapter.
	by = themesOf(t, ts, "rom.5")
	if n := len(topicLabels(by["themes-tcr1934"])); n != 23 {
		t.Errorf("Thompson, Romans 5: %d topics, want 23: %v", n, topicLabels(by["themes-tcr1934"]))
	}
	if n := len(topicLabels(by["themes-torrey"])); n != 33 {
		t.Errorf("Torrey, Romans 5: %d topics, want 33: %v", n, topicLabels(by["themes-torrey"]))
	}

	// The counts the tool printed: a topic, its parts and its links.
	for _, c := range []struct {
		table, source string
		want          int
	}{
		{"themes", "themes-tcr1934", 1728},
		{"themes", "themes-torrey", 628},
		{"theme_parts", "themes-tcr1934", 2229},
		{"theme_parts", "themes-torrey", 23695},
		{"theme_links", "themes-tcr1934", 27076},
		{"theme_links", "themes-torrey", 38849},
	} {
		col := "source_id"
		if c.table == "theme_parts" {
			col = "theme_source_id"
		}
		var n int
		if err := db.QueryRow(ctx, "select count(*) from "+c.table+" where "+col+" = $1", c.source).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != c.want {
			t.Errorf("%s of %s: %d, want %d", c.table, c.source, n, c.want)
		}
	}

	// Loading again changes nothing.
	for _, dir := range []string{themesTCR, themesTorrey} {
		res, err := sources.Load(ctx, db, dir)
		if err != nil {
			t.Fatal(err)
		}
		if res.Removed != 0 {
			t.Errorf("a second load of %s removed %d", dir, res.Removed)
		}
	}
}

type tierJSON struct {
	Total int `json:"total"`
	Rows  []struct {
		Source string `json:"source"`
		ID     string `json:"id"`
		Title  string `json:"title"`
		Ref    string `json:"ref"`
		Role   string `json:"role"`
		Theme  *struct {
			Source    string  `json:"source"`
			Topic     string  `json:"topic"`
			Label     string  `json:"label"`
			Part      *int    `json:"part"`
			PartLabel *string `json:"part_label"`
		} `json:"theme"`
	} `json:"rows"`
}

type twoTiers struct {
	Ref     string    `json:"ref"`
	Passage *tierJSON `json:"passage"`
	Theme   *tierJSON `json:"theme"`
}

func getTiers(t *testing.T, ts *testServer, target string) twoTiers {
	t.Helper()
	res := ts.do("GET", target)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET %s: %s: %s", target, res.Status, body(t, res))
	}
	var out twoTiers
	if err := json.Unmarshal([]byte(body(t, res)), &out); err != nil {
		t.Fatal(err)
	}
	return out
}

// allThemeRows asks for target a page (200 rows) at a time until the theme
// tier is read whole; the answer is the first page's with every row.
func allThemeRows(t *testing.T, ts *testServer, target string) twoTiers {
	t.Helper()
	got := getTiers(t, ts, target+"&limit=200")
	for got.Theme != nil && len(got.Theme.Rows) < got.Theme.Total {
		more := getTiers(t, ts, target+"&limit=200&offset="+strconv.Itoa(len(got.Theme.Rows)))
		if len(more.Theme.Rows) == 0 {
			t.Fatalf("%s: a page of %d rows of %d is empty", target, len(got.Theme.Rows), got.Theme.Total)
		}
		got.Theme.Rows = append(got.Theme.Rows, more.Theme.Rows...)
	}
	return got
}

func (tt *tierJSON) has(source, id string) bool {
	for _, r := range tt.Rows {
		if r.Source == source && r.ID == id {
			return true
		}
	}
	return false
}

// The theme tier over the real Spurgeon, Thompson and Torrey files
// (story A5): Romans 5:8's second tier has Justification by Grace (126)
// and Love (229), and Love's Commendation (104), which is a passage row,
// is not repeated in it. The counts are logged (and in the story's
// closing comment) against the probe's, Nave's ATONEMENT 153.
func TestThemeTierOverTheRealFiles(t *testing.T) {
	ctx := context.Background()
	db := store.TestDB(t)
	if _, err := migrate.Up(ctx, db); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{spurgeonCCEL, themesTCR, themesTorrey} {
		if _, err := sources.Load(ctx, db, dir); err != nil {
			t.Fatalf("load %s: %v", dir, err)
		}
	}
	ts := newTestServer(t, db)

	for _, target := range []string{"/v1/sermons?ref=rom.5.8&tier=theme", "/v1/sermons?ref=rom.5.8&tier=theme-only"} {
		got := allThemeRows(t, ts, target)
		if got.Ref != "Rom.5.8" || got.Theme == nil {
			t.Fatalf("%s: %+v", target, got)
		}
		// 229 Love (1 John 4:19) meets Thompson's Love-Hatred and Torrey's
		// The Love of God. 126 Justification by Grace (Romans 3:24) does
		// not come in at Romans 5:8 from these two sources: none of their
		// topics of that verse links Romans 3:24 (Torrey's Justification
		// Before God links Romans 5:9, not 5:8); it does at Romans 5 below.
		if !got.Theme.has("spurgeon-ccel", "229") {
			t.Errorf("%s: the theme tier lacks 229 Love (%d rows)", target, got.Theme.Total)
		}
		if got.Theme.has("spurgeon-ccel", "126") {
			t.Errorf("%s: 126 is in the theme tier of Romans 5:8, which the data does not give", target)
		}
		if got.Theme.has("spurgeon-ccel", "104") {
			t.Errorf("%s: 104 Love's Commendation is a passage row and is in the theme tier too", target)
		}
		seen := map[string]bool{}
		for _, r := range got.Theme.Rows {
			k := r.Source + ":" + r.ID
			if seen[k] {
				t.Errorf("%s: %s is listed twice", target, k)
			}
			seen[k] = true
			if r.Theme == nil || r.Theme.Topic == "" || r.Theme.Label == "" || r.Role != "text" {
				t.Errorf("%s: row %s has no theme or is not a preaching text: %+v", target, k, r)
			}
		}
		if strings.HasSuffix(target, "=theme") {
			if got.Passage == nil || !got.Passage.has("spurgeon-ccel", "104") {
				t.Errorf("%s: the passage tier lacks 104: %+v", target, got.Passage)
			}
			for _, r := range got.Passage.Rows {
				if got.Theme.has(r.Source, r.ID) {
					t.Errorf("%s: %s:%s is in both tiers", target, r.Source, r.ID)
				}
			}
		} else if got.Passage != nil {
			t.Errorf("%s: a theme-only answer has passage rows: %+v", target, got.Passage)
		}
		per := map[string]int{}
		for _, r := range got.Theme.Rows {
			per[r.Theme.Source]++
		}
		t.Logf("%s: theme tier %d sermons (rows on this page by theme source: %v)", target, got.Theme.Total, per)
	}

	// The chapter: 126 Justification by Grace is in it, by Torrey's
	// Justification Before God (which links Romans 3:24 and Romans 5:9).
	chapter := allThemeRows(t, ts, "/v1/sermons?ref=rom.5&tier=theme-only")
	t.Logf("Romans 5 theme tier: %d sermons", chapter.Theme.Total)
	if !chapter.Theme.has("spurgeon-ccel", "126") || !chapter.Theme.has("spurgeon-ccel", "229") {
		t.Errorf("Romans 5's theme tier lacks 126 or 229 (%d rows)", chapter.Theme.Total)
	}
	for _, r := range chapter.Theme.Rows {
		if r.Source == "spurgeon-ccel" && r.ID == "126" {
			t.Logf("126 comes in by %s:%s %q", r.Theme.Source, r.Theme.Topic, r.Theme.Label)
		}
	}

	// The topic named on a row is one that links the ref and has at most
	// THEME_MAX_VERSES verses; ?theme= restricts to it.
	all := allThemeRows(t, ts, "/v1/sermons?ref=rom.5.8&tier=theme-only")
	for _, r := range all.Theme.Rows {
		var verses *int
		var linked bool
		err := db.QueryRow(ctx, `SELECT t.verses, EXISTS (SELECT 1 FROM theme_links l WHERE l.source_id = t.source_id AND l.theme_id = t.id
				AND l.span && '[45005008,45005009)'::int4range) FROM themes t WHERE t.source_id = $1 AND t.id = $2`, r.Theme.Source, r.Theme.Topic).Scan(&verses, &linked)
		if err != nil || verses == nil || *verses > store.ThemeMaxVerses || !linked {
			t.Errorf("%s:%s via %s:%s: verses %v, links the ref %v, err %v", r.Source, r.ID, r.Theme.Source, r.Theme.Topic, verses, linked, err)
		}
	}
	first := all.Theme.Rows[0].Theme
	one := allThemeRows(t, ts, "/v1/sermons?ref=rom.5.8&tier=theme-only&theme="+first.Source+":"+first.Topic)
	if one.Theme.Total == 0 || one.Theme.Total > all.Theme.Total {
		t.Errorf("theme=%s:%s: %d of %d", first.Source, first.Topic, one.Theme.Total, all.Theme.Total)
	}
	for _, r := range one.Theme.Rows {
		if r.Theme.Source != first.Source || r.Theme.Topic != first.Topic {
			t.Errorf("theme=%s:%s: row names %s:%s", first.Source, first.Topic, r.Theme.Source, r.Theme.Topic)
		}
	}

	// /v1/themes?ref= counts the sermons on each topic by preaching text.
	res := ts.do("GET", "/v1/themes?ref=rom.5.8")
	var th struct {
		Rows []struct {
			Source  string `json:"source"`
			Topic   string `json:"topic"`
			Label   string `json:"label"`
			Sermons int    `json:"sermons"`
			Verses  int    `json:"verses"`
		} `json:"rows"`
	}
	if err := json.Unmarshal([]byte(body(t, res)), &th); err != nil {
		t.Fatal(err)
	}
	var any bool
	for _, r := range th.Rows {
		t.Logf("themes on Romans 5:8: %s %q: %d sermons, %d verses", r.Source, r.Label, r.Sermons, r.Verses)
		if r.Sermons > 0 {
			any = true
		}
	}
	if !any {
		t.Error("no topic of Romans 5:8 has a sermon count")
	}

	// Each theme source on its own: the sermons its topics of Romans 5:8
	// put in the second tier (the topics' ids by ?theme=).
	for _, src := range []string{"themes-tcr1934", "themes-torrey"} {
		var ids []string
		seen := map[string]bool{}
		for _, r := range th.Rows {
			if id := r.Source + ":" + r.Topic; r.Source == src && !seen[id] {
				seen[id] = true
				ids = append(ids, id)
			}
		}
		own := allThemeRows(t, ts, "/v1/sermons?ref=rom.5.8&tier=theme-only&theme="+strings.Join(ids, ","))
		t.Logf("%s alone: %d topics of Romans 5:8, %d sermons in the theme tier", src, len(ids), own.Theme.Total)
		if own.Theme.Total == 0 || own.Theme.Total > all.Theme.Total {
			t.Errorf("%s alone: %d of %d", src, own.Theme.Total, all.Theme.Total)
		}
	}
}

// /v1/themes's rows say how many sermons are on the topic and how many
// verses it covers (the demo's Atonement: Rom.5.6-11 and Rom.3.25, 7 verses;
// the fixture's sermons 104 and 1191 are on it).
func TestThemesRowsCarrySermonsAndVerses(t *testing.T) {
	ts := newTestServer(t, loadedDB(t))
	by := themesOf(t, ts, "rom.5.8")
	res := ts.do("GET", "/v1/themes?ref=rom.5.8")
	var raw struct {
		Rows []map[string]any `json:"rows"`
	}
	if err := json.Unmarshal([]byte(body(t, res)), &raw); err != nil {
		t.Fatal(err)
	}
	if len(raw.Rows) != 2 || len(by["themes-demo"]) != 2 {
		t.Fatalf("rows: %v", raw.Rows)
	}
	for _, r := range raw.Rows {
		want := map[string][2]float64{"atonement": {2, 7}, "love": {1, 3}}[r["topic"].(string)]
		if r["sermons"] != want[0] || r["verses"] != want[1] {
			t.Errorf("%v: sermons %v verses %v, want %v", r["topic"], r["sermons"], r["verses"], want)
		}
	}
}
