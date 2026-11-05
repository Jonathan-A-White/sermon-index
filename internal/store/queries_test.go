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
