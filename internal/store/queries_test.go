package store_test

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Jonathan-A-White/sermon-index/internal/migrate"
	"github.com/Jonathan-A-White/sermon-index/internal/passage"
	"github.com/Jonathan-A-White/sermon-index/internal/sources"
	"github.com/Jonathan-A-White/sermon-index/internal/store"
)

const (
	spurgeon20 = "../../sources/_fixtures/spurgeon-20"
	themesDemo = "../../sources/_fixtures/themes-demo"
)

// loaded is a migrated test database with the sources given loaded.
func loaded(t *testing.T, dirs ...string) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()
	db := store.TestDB(t)
	if _, err := migrate.Up(ctx, db); err != nil {
		t.Fatal(err)
	}
	for _, d := range dirs {
		if _, err := sources.Load(ctx, db, d); err != nil {
			t.Fatalf("load %s: %v", d, err)
		}
	}
	return db
}

// sermonSource writes sources/<id> in a temp directory: a sermon source
// with the rows given.
func sermonSource(t *testing.T, id string, rows ...string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), id)
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	toml := `id = "` + id + `"
name = "Source ` + id + `"
kind = "sermons"
licence = "Public domain."
credit = "Someone"
homepage = "https://example.org"
terms_url = "https://example.org/terms"
added = 2026-11-05
on_by_default = true
`
	if err := os.WriteFile(filepath.Join(dir, "source.toml"), []byte(toml), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "rows.jsonl"), []byte(strings.Join(rows, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func ref(t *testing.T, s string) passage.Range {
	t.Helper()
	r, err := passage.Parse(s)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// keys is each row as "source/id role ref width".
func keys(rows []store.SermonRow) []string {
	var out []string
	for _, r := range rows {
		out = append(out, r.Source+"/"+r.ID+" "+r.Role+" "+r.Ref+" "+itoa(r.Width))
	}
	return out
}

func itoa(n int) string {
	const digits = "0123456789"
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(digits[n%10]) + s
		n /= 10
	}
	return s
}

func TestCounts(t *testing.T) {
	db := loaded(t, spurgeon20, themesDemo)
	got, err := store.Count(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	if got != (store.Totals{Sources: 2, Sermons: 20}) {
		t.Errorf("Count = %+v", got)
	}
}

func TestSourcesListsBothKindsWithTheirRows(t *testing.T) {
	db := loaded(t, themesDemo, spurgeon20)
	got, err := store.Sources(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("%d sources", len(got))
	}
	s, th := got[0], got[1]
	if s.ID != "spurgeon-20" || s.Kind != "sermons" || s.Rows != 20 || s.OnByDefault ||
		!strings.HasPrefix(s.Licence, "CCEL.org website") || s.Homepage != "https://ccel.org/ccel/spurgeon" {
		t.Errorf("spurgeon-20: %+v", s)
	}
	if th.ID != "themes-demo" || th.Kind != "themes" || th.Rows != 4 || th.Name != "Demo themes (a fixture)" {
		t.Errorf("themes-demo: %+v", th)
	}
}

// The order: preaching text before cited; narrower before wider; then the
// source, then the id (a number as a number); one row a sermon.
func TestSermonsOrder(t *testing.T) {
	a := sermonSource(t, "a",
		`{"id": "10", "title": "Ten", "ref": "Rom.5.8", "url": "https://example.org/10"}`,
		`{"id": "9", "title": "Nine", "ref": "Rom.5.8", "url": "https://example.org/9", "cited": ["Rom.5.8", "Rom.5"]}`,
		`{"id": "wide", "title": "Chapter", "ref": "Rom.5", "url": "https://example.org/w"}`,
		`{"id": "peri", "title": "Pericope", "ref": "Rom.5.1-Rom.5.11", "url": "https://example.org/p"}`,
		`{"id": "cross", "title": "Across", "ref": "Rom.5.6-Rom.6.4", "url": "https://example.org/x"}`,
		`{"id": "cites", "title": "Cites", "ref": "John.3.16", "url": "https://example.org/c", "cited": ["Rom.5", "Rom.5.8"]}`,
		`{"id": "none", "title": "Elsewhere", "ref": "John.3.16", "url": "https://example.org/n"}`,
	)
	b := sermonSource(t, "b",
		`{"id": "1", "title": "B one", "ref": "Rom.5.8", "url": "https://example.org/b1"}`,
		`{"id": "2", "title": "B cites", "url": "https://example.org/b2", "cited": ["Rom.5.1-Rom.5.11"]}`,
	)
	db := loaded(t, b, a)
	ctx := context.Background()

	total, rows, err := store.Sermons(ctx, db, store.SermonQuery{Span: ref(t, "rom.5.8"), Limit: 50})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"a/9 text Rom.5.8 1",
		"a/10 text Rom.5.8 1",
		"b/1 text Rom.5.8 1",
		"a/peri text Rom.5.1-Rom.5.11 11",
		"a/wide text Rom.5 21",
		"a/cross text Rom.5.6-Rom.6.4 20",
		"a/cites cited Rom.5.8 1",
		"b/2 cited Rom.5.1-Rom.5.11 11",
	}
	if !reflect.DeepEqual(keys(rows), want) || total != len(want) {
		t.Errorf("rom.5.8: total %d\n got %q\nwant %q", total, keys(rows), want)
	}

	// source=b limits to b; limit and offset page, total counts them all.
	total, rows, err = store.Sermons(ctx, db, store.SermonQuery{Span: ref(t, "Rom.5.8"), Sources: []string{"b", "nosuch"}, Limit: 50})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"b/1 text Rom.5.8 1", "b/2 cited Rom.5.1-Rom.5.11 11"}; !reflect.DeepEqual(keys(rows), want) || total != 2 {
		t.Errorf("source=b: %d %q", total, keys(rows))
	}
	total, rows, err = store.Sermons(ctx, db, store.SermonQuery{Span: ref(t, "Rom.5.8"), Limit: 2, Offset: 2})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"b/1 text Rom.5.8 1", "a/peri text Rom.5.1-Rom.5.11 11"}; !reflect.DeepEqual(keys(rows), want) || total != 8 {
		t.Errorf("limit 2 offset 2: %d %q", total, keys(rows))
	}
	total, rows, err = store.Sermons(ctx, db, store.SermonQuery{Span: ref(t, "Rom.5.8"), Limit: 2, Offset: 100})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 || rows == nil || total != 8 {
		t.Errorf("offset past the end: %d %v (want an empty, non-nil slice)", total, rows)
	}
}

func TestSermonsOnTheFixture(t *testing.T) {
	db := loaded(t, spurgeon20)
	ctx := context.Background()
	_, rows, err := store.Sermons(ctx, db, store.SermonQuery{Span: ref(t, "rom.5.8"), Limit: 50})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Title != "Love's Commendation" || rows[0].AltURL == nil ||
		*rows[0].AltURL != "https://www.spurgeongems.org/sermon/chs104.pdf" || rows[0].Date != nil {
		t.Errorf("rom.5.8: %+v", rows)
	}
	_, rows, err = store.Sermons(ctx, db, store.SermonQuery{Span: ref(t, "Rom.5"), Limit: 50})
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, r := range rows {
		ids = append(ids, r.ID)
	}
	if want := []string{"37", "104", "330", "1191", "2544", "3392"}; !reflect.DeepEqual(ids, want) {
		t.Errorf("Rom.5: %v", ids)
	}
}

func TestThemesNarrowerFirstGroupedBySource(t *testing.T) {
	other := filepath.Join(t.TempDir(), "themes-other")
	os.Mkdir(other, 0o755)
	b, err := os.ReadFile(filepath.Join(themesDemo, "source.toml"))
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(other, "source.toml"), []byte(strings.Replace(string(b), `id = "themes-demo"`, `id = "themes-other"`, 1)), 0o644)
	os.WriteFile(filepath.Join(other, "topics.jsonl"), []byte(`{"id": "mercy", "label": "Mercy"}`+"\n"), 0o644)
	os.WriteFile(filepath.Join(other, "links.jsonl"), []byte(`{"topic": "mercy", "ref": "Rom.5"}`+"\n"+`{"topic": "mercy", "ref": "Rom.5.8"}`+"\n"), 0o644)

	db := loaded(t, other, themesDemo)
	rows, err := store.Themes(context.Background(), db, ref(t, "Rom.5.8"))
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, r := range rows {
		part := ""
		if r.Part != nil {
			part = "(" + itoa(*r.Part) + " " + *r.PartLabel + ")"
		}
		got = append(got, r.Source+" "+r.Topic+part+" "+r.Ref+" see="+strings.Join(r.See, ","))
	}
	want := []string{
		"themes-demo love(1 Of God) Rom.5.8 see=grace",
		"themes-demo atonement Rom.5.6-Rom.5.11 see=",
		"themes-other mercy Rom.5.8 see=",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("themes on Rom.5.8:\n got %q\nwant %q", got, want)
	}
	for _, r := range rows {
		if r.See == nil {
			t.Errorf("%s: see is nil, want []", r.Topic)
		}
	}
}

func TestSearchSermonsAndThemes(t *testing.T) {
	db := loaded(t, spurgeon20, themesDemo)
	ctx := context.Background()
	hits, err := store.Search(ctx, db, "spurgeon grace")
	if err != nil {
		t.Fatal(err)
	}
	var titles []string
	for _, h := range hits {
		titles = append(titles, h.Kind+" "+h.Title)
	}
	if len(hits) == 0 || !contains(titles, "sermon Justification by Grace") || contains(titles, "theme Grace") {
		t.Errorf("spurgeon grace: %q", titles)
	}
	for i := 1; i < len(hits); i++ {
		if hits[i].Rank > hits[i-1].Rank {
			t.Errorf("not by rank: %v", hits)
		}
	}

	hits, err = store.Search(ctx, db, "grace")
	if err != nil {
		t.Fatal(err)
	}
	var theme *store.Hit
	for i := range hits {
		if hits[i].Kind == "theme" {
			theme = &hits[i]
		}
	}
	if theme == nil || theme.Source != "themes-demo" || theme.ID != "grace" || theme.Ref != nil || theme.URL != nil {
		t.Errorf("grace: no theme hit, or a wrong one: %+v", hits)
	}
	// A sermon hit carries its preaching text; one with none has a null ref.
	for _, h := range hits {
		if h.Kind == "sermon" && h.ID == "126" && (h.Ref == nil || *h.Ref != "Rom.3.24") {
			t.Errorf("126's ref: %v", h.Ref)
		}
		if h.Kind == "sermon" && h.ID == "385" && h.Ref != nil {
			t.Errorf("385 has no preaching text, ref %v", *h.Ref)
		}
	}
	if !contains(titles, "sermon Exposition of the Doctrines of Grace") {
		t.Errorf("385 not found by 'spurgeon grace': %q", titles)
	}

	// Words found nowhere, and stop words alone, answer nothing, not an error.
	for _, q := range []string{"zzyzx", "the", `"unbalanced`} {
		hits, err := store.Search(ctx, db, q)
		if err != nil || len(hits) != 0 {
			t.Errorf("%q: %v %v", q, hits, err)
		}
	}
}

func contains(xs []string, x string) bool {
	for _, y := range xs {
		if y == x {
			return true
		}
	}
	return false
}

func TestVerses(t *testing.T) {
	sum := 0
	for _, b := range passage.Books() {
		for _, last := range b.Last {
			sum += last
		}
	}
	for _, c := range []struct {
		ref  string
		want int
	}{{"Rom.5.8", 1}, {"Rom.5", 21}, {"Rom.5.12-Rom.6.4", 14}, {"Mal.4-Matt.1", 6 + 25}, {"Gen-Rev", sum}} {
		r := ref(t, c.ref)
		if got := store.Verses(r.Lo, r.Hi); got != c.want {
			t.Errorf("Verses(%s) = %d, want %d", c.ref, got, c.want)
		}
	}
}

// themeSource writes a theme source in a temp directory: three topics that
// all link Rom.5.8, of 2, 34 and over 1,000 verses.
func themeSource(t *testing.T, id string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), id)
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(themesDemo, "source.toml"))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"source.toml": strings.Replace(string(b), `id = "themes-demo"`, `id = "`+id+`"`, 1),
		"topics.jsonl": `{"id": "narrow", "label": "Narrow", "parts": [{"n": 1, "label": "Cross"}, {"n": 2, "label": "Love"}]}` + "\n" +
			`{"id": "mid", "label": "Middle"}` + "\n" +
			`{"id": "wide", "label": "Wide"}` + "\n",
		"links.jsonl": `{"topic": "narrow", "part": 2, "ref": "Rom.5.8"}` + "\n" +
			`{"topic": "narrow", "part": 1, "ref": "Isa.53.5"}` + "\n" +
			`{"topic": "mid", "ref": "Rom.5.8"}` + "\n" +
			`{"topic": "mid", "ref": "Luke.15"}` + "\n" +
			`{"topic": "mid", "ref": "John.6.37"}` + "\n" +
			`{"topic": "wide", "ref": "Rom.5.8"}` + "\n" +
			`{"topic": "wide", "ref": "Isa"}` + "\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func tierSources(t *testing.T) (string, string) {
	a := sermonSource(t, "a",
		`{"id": "1", "title": "On the verse", "ref": "Rom.5.8", "url": "https://example.org/1"}`,
		`{"id": "2", "title": "Stripes", "ref": "Isa.53.5", "url": "https://example.org/2"}`,
		`{"id": "3", "title": "Prodigal", "ref": "Luke.15.17", "url": "https://example.org/3"}`,
		`{"id": "4", "title": "Come", "ref": "John.6.37", "url": "https://example.org/4"}`,
		`{"id": "5", "title": "Hear", "ref": "Isa.1.1", "url": "https://example.org/5"}`,
		`{"id": "6", "title": "Only cites", "ref": "John.1.1", "url": "https://example.org/6", "cited": ["Isa.53.5", "Luke.15"]}`,
		`{"id": "9", "title": "Nine", "ref": "John.6.37", "url": "https://example.org/9"}`,
		`{"id": "10", "title": "Ten", "ref": "John.6.37", "url": "https://example.org/10"}`,
	)
	return a, themeSource(t, "tx")
}

func themeKeys(rows []store.ThemeSermonRow) []string {
	var out []string
	for _, r := range rows {
		part := ""
		if r.Theme.Part != nil {
			part = " (" + itoa(*r.Theme.Part) + " " + *r.Theme.PartLabel + ")"
		}
		out = append(out, r.Source+"/"+r.ID+" "+r.Role+" "+r.Ref+" via "+r.Theme.Source+":"+r.Theme.Topic+" "+r.Theme.Label+part)
	}
	return out
}

// The theme tier: sermons whose preaching text meets a link of a topic that
// also links the ref, the topic at most 300 verses, each sermon once with
// its narrowest topic, the passage rows left out.
func TestThemeSermons(t *testing.T) {
	a, tx := tierSources(t)
	db := loaded(t, a, tx)
	ctx := context.Background()
	q := store.ThemeQuery{Span: ref(t, "Rom.5.8"), Limit: 50}

	total, rows, err := store.ThemeSermons(ctx, db, q)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"a/2 text Isa.53.5 via tx:narrow Narrow (1 Cross)",
		"a/3 text Luke.15.17 via tx:mid Middle",
		"a/4 text John.6.37 via tx:mid Middle",
		"a/9 text John.6.37 via tx:mid Middle",
		"a/10 text John.6.37 via tx:mid Middle",
	}
	if got := themeKeys(rows); total != 5 || !reflect.DeepEqual(got, want) {
		t.Errorf("rom.5.8: total %d\n got %q\nwant %q", total, got, want)
	}
	if rows[0].Width != 1 || rows[0].Title != "Stripes" || rows[0].URL == "" {
		t.Errorf("the row is a sermon row: %+v", rows[0])
	}

	// A topic past the cap is never used, and Verses counts what a topic covers.
	if store.ThemeMaxVerses != 300 {
		t.Errorf("ThemeMaxVerses = %d, want 300", store.ThemeMaxVerses)
	}

	q.Themes = []string{"tx:mid"}
	total, rows, _ = store.ThemeSermons(ctx, db, q)
	if want := []string{"a/3 text Luke.15.17 via tx:mid Middle", "a/4 text John.6.37 via tx:mid Middle", "a/9 text John.6.37 via tx:mid Middle", "a/10 text John.6.37 via tx:mid Middle"}; total != 4 || !reflect.DeepEqual(themeKeys(rows), want) {
		t.Errorf("theme=tx:mid: %d %q", total, themeKeys(rows))
	}
	q.Themes = []string{"tx:wide"}
	if total, rows, _ = store.ThemeSermons(ctx, db, q); total != 0 || len(rows) != 0 || rows == nil {
		t.Errorf("theme=tx:wide is past the cap: %d %v", total, rows)
	}
	q.Themes = nil

	q.Limit, q.Offset = 2, 1
	total, rows, _ = store.ThemeSermons(ctx, db, q)
	if want := []string{"a/3 text Luke.15.17 via tx:mid Middle", "a/4 text John.6.37 via tx:mid Middle"}; total != 5 || !reflect.DeepEqual(themeKeys(rows), want) {
		t.Errorf("limit 2 offset 1: %d %q", total, themeKeys(rows))
	}
	q.Limit, q.Offset = 50, 0
	q.Sources = []string{"nosuch"}
	if total, rows, _ = store.ThemeSermons(ctx, db, q); total != 0 || len(rows) != 0 {
		t.Errorf("source=nosuch: %d %v", total, rows)
	}

	// A ref no topic links answers nothing, and never nil.
	q = store.ThemeQuery{Span: ref(t, "Jude"), Limit: 50}
	if total, rows, _ = store.ThemeSermons(ctx, db, q); total != 0 || rows == nil || len(rows) != 0 {
		t.Errorf("Jude: %d %v", total, rows)
	}
}

// /v1/themes's rows carry the sermons on each topic by preaching text and
// the topic's verses (all of it, past the cap too).
func TestThemesCarrySermonsAndVerses(t *testing.T) {
	a, tx := tierSources(t)
	db := loaded(t, a, tx)
	rows, err := store.Themes(context.Background(), db, ref(t, "Rom.5.8"))
	if err != nil {
		t.Fatal(err)
	}
	isa := ref(t, "Isa")
	want := map[string][2]int{
		"narrow": {2, 2},
		"mid":    {5, 34},
		"wide":   {3, 1 + store.Verses(isa.Lo, isa.Hi)},
	}
	for _, r := range rows {
		w := want[r.Topic]
		if r.Sermons != w[0] || r.Verses != w[1] {
			t.Errorf("%s: %d sermons, %d verses; want %v", r.Topic, r.Sermons, r.Verses, w)
		}
		delete(want, r.Topic)
	}
	if len(want) != 0 {
		t.Errorf("topics not answered: %v", want)
	}
}
