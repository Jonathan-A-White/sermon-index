package api

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"testing"

	"github.com/Jonathan-A-White/sermon-index/internal/migrate"
	"github.com/Jonathan-A-White/sermon-index/internal/sources"
	"github.com/Jonathan-A-White/sermon-index/internal/store"
)

const (
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
