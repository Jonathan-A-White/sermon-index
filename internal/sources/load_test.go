package sources

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Jonathan-A-White/sermon-index/internal/migrate"
	"github.com/Jonathan-A-White/sermon-index/internal/passage"
	"github.com/Jonathan-A-White/sermon-index/internal/store"
)

func TestMain(m *testing.M) { os.Exit(store.RunTests(m)) }

func migrated(t *testing.T) *pgxpool.Pool {
	t.Helper()
	db := store.TestDB(t)
	if _, err := migrate.Up(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	return db
}

func count(t *testing.T, db *pgxpool.Pool, sql string, args ...any) int {
	t.Helper()
	var n int
	if err := db.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
	return n
}

func TestLoadTheFixtureTwiceGivesTheSameCounts(t *testing.T) {
	ctx := context.Background()
	db := migrated(t)

	first, err := Load(ctx, db, fixture)
	if err != nil {
		t.Fatal(err)
	}
	if got := first.String(); got != "spurgeon-20: 20 rows, 19 passages, 0 removed" {
		t.Errorf("first load: %s", got)
	}
	sermons := count(t, db, "select count(*) from sermons")
	passages := count(t, db, "select count(*) from sermon_passages")
	if sermons != 20 || passages != 19 {
		t.Errorf("after the first load: %d sermons, %d passages", sermons, passages)
	}

	second, err := Load(ctx, db, fixture)
	if err != nil {
		t.Fatal(err)
	}
	if second != first {
		t.Errorf("second load %s, first %s", second, first)
	}
	if n := count(t, db, "select count(*) from sermons"); n != sermons {
		t.Errorf("sermons %d after the second load, %d after the first", n, sermons)
	}
	if n := count(t, db, "select count(*) from sermon_passages"); n != passages {
		t.Errorf("passages %d after the second load, %d after the first", n, passages)
	}
	if n := count(t, db, "select count(*) from sources"); n != 1 {
		t.Errorf("%d sources", n)
	}

	var licence string
	if err := db.QueryRow(ctx, "select licence from sources where id = 'spurgeon-20'").Scan(&licence); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(licence, "These books may be used for personal, educational, or non-profit purposes.") {
		t.Errorf("licence not kept verbatim: %q", licence)
	}
}

func TestRomans5v8OverlapAnswersLovesCommendation(t *testing.T) {
	ctx := context.Background()
	db := migrated(t)
	if _, err := Load(ctx, db, fixture); err != nil {
		t.Fatal(err)
	}
	q, err := passage.Parse("rom.5.8")
	if err != nil {
		t.Fatal(err)
	}
	rows, err := db.Query(ctx, `select s.id, s.title from sermons s
		join sermon_passages p on p.source_id = s.source_id and p.sermon_id = s.id
		where p.span && $1::int4range order by s.id`, q.PG())
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for rows.Next() {
		var id, title string
		if err := rows.Scan(&id, &title); err != nil {
			t.Fatal(err)
		}
		got = append(got, id+" "+title)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if want := []string{"104 Love's Commendation"}; !reflect.DeepEqual(got, want) {
		t.Errorf("span && Rom.5.8 gave %q, want %q", got, want)
	}

	// The chapter: Romans 5's six preaching texts, and nothing of Romans 3 or 4.
	ch, _ := passage.Parse("Rom.5")
	var ids []string
	r2, err := db.Query(ctx, `select p.sermon_id from sermon_passages p where p.span && $1::int4range order by p.sermon_id::int`, ch.PG())
	if err != nil {
		t.Fatal(err)
	}
	for r2.Next() {
		var id string
		if err := r2.Scan(&id); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	if want := []string{"37", "104", "330", "1191", "2544", "3392"}; !reflect.DeepEqual(ids, want) {
		t.Errorf("span && Rom.5 gave %v, want %v", ids, want)
	}
	if n := count(t, db, `select count(*) from sermon_passages where span && $1::int4range and not span <@ $1::int4range`, ch.PG()); n != 0 {
		t.Errorf("%d passages reach outside Romans 5", n)
	}
}

// copyFixture copies the fixture into a temp sources/spurgeon-20 so a test
// may change it.
func copyFixture(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "spurgeon-20")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"source.toml", "rows.jsonl"} {
		b, err := os.ReadFile(filepath.Join(fixture, f))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, f), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestLoadUpdatesAndRemoves(t *testing.T) {
	ctx := context.Background()
	db := migrated(t)
	dir := copyFixture(t)
	if _, err := Load(ctx, db, dir); err != nil {
		t.Fatal(err)
	}

	b, err := os.ReadFile(filepath.Join(dir, "rows.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	var kept []string
	for _, line := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		switch {
		case strings.Contains(line, `"id": "1000"`): // dropped
		case strings.Contains(line, `"id": "104"`):
			kept = append(kept, strings.Replace(line, `"ref": "Rom.5.8"`, `"ref": "Rom.5.6-Rom.5.8", "cited": ["John.3.16"]`, 1))
		default:
			kept = append(kept, strings.Replace(line, "Love's Commendation", "Love's Commendation (again)", 1))
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "rows.jsonl"), []byte(strings.Join(kept, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	c, err := Load(ctx, db, dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := c.String(); got != "spurgeon-20: 19 rows, 19 passages, 1 removed" {
		t.Errorf("reload: %s", got)
	}
	if n := count(t, db, "select count(*) from sermons where id = '1000'"); n != 0 {
		t.Errorf("sermon 1000 not removed")
	}
	if n := count(t, db, "select count(*) from sermon_passages where sermon_id = '1000'"); n != 0 {
		t.Errorf("sermon 1000's passage not removed")
	}
	if n := count(t, db, "select count(*) from sermon_passages where sermon_id = '104' and role = 'text' and ref_osis = 'Rom.5.6-Rom.5.8'"); n != 1 {
		t.Errorf("sermon 104's new text not loaded")
	}
	if n := count(t, db, "select count(*) from sermon_passages where sermon_id = '104' and ref_osis = 'Rom.5.8'"); n != 0 {
		t.Errorf("sermon 104's old text kept")
	}
	if n := count(t, db, "select count(*) from sermon_passages where sermon_id = '104' and role = 'cited' and ref_osis = 'John.3.16'"); n != 1 {
		t.Errorf("sermon 104's cited ref not loaded")
	}
}

func TestLoadRefusesAFaultySourceAndChangesNothing(t *testing.T) {
	ctx := context.Background()
	db := migrated(t)
	dir := copyFixture(t)
	if _, err := Load(ctx, db, dir); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(filepath.Join(dir, "rows.jsonl"), os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString(`{"id": "9999", "title": "Bad", "ref": "Rom.5.99", "url": "https://example.org/9999"}` + "\n")
	f.Close()

	_, err = Load(ctx, db, dir)
	var faults Faults
	if !errors.As(err, &faults) {
		t.Fatalf("Load gave %v, want Faults", err)
	}
	if got := faults.Error(); !strings.Contains(got, "rows.jsonl:21: ref 'Rom.5.99': verse past the chapter") {
		t.Errorf("faults: %s", got)
	}
	if n := count(t, db, "select count(*) from sermons"); n != 20 {
		t.Errorf("%d sermons after a refused load", n)
	}
}

func TestLoadNeedsTheSchema(t *testing.T) {
	db := store.TestDB(t)
	_, err := Load(context.Background(), db, fixture)
	if err == nil || !strings.Contains(err.Error(), "sermons migrate") {
		t.Errorf("Load on an unmigrated database: %v", err)
	}
}

func TestLoadAllSkipsUnderscoreDirectories(t *testing.T) {
	ctx := context.Background()
	db := migrated(t)
	root := t.TempDir()
	for _, sub := range []string{"_template", "_fixtures", "demo"} {
		if err := os.MkdirAll(filepath.Join(root, sub), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	os.WriteFile(filepath.Join(root, "demo", "source.toml"), []byte(goodToml), 0o644)
	os.WriteFile(filepath.Join(root, "demo", "rows.jsonl"), []byte(goodRow+"\n"), 0o644)
	os.WriteFile(filepath.Join(root, "_template", "source.toml"), []byte("not toml ="), 0o644)
	os.WriteFile(filepath.Join(root, "README.md"), []byte("a file, not a source"), 0o644)

	cs, err := LoadAll(ctx, db, root)
	if err != nil {
		t.Fatal(err)
	}
	if len(cs) != 1 || cs[0].String() != "demo: 1 rows, 1 passages, 0 removed" {
		t.Errorf("LoadAll: %v", cs)
	}
}

func TestLoadAThemeSource(t *testing.T) {
	ctx := context.Background()
	db := migrated(t)
	topics := `{"id": "love", "label": "Love-Hatred", "parts": [{"n": 4, "label": "Of God"}], "see": ["grace"]}
{"id": "grace", "label": "Grace, Divine"}
`
	links := `{"topic": "love", "part": 4, "ref": "Rom.5.8"}
{"topic": "grace", "ref": "Rom.5.15-Rom.5.17"}
{"topic": "grace", "ref": "Eph.1.7", "via": "sermon", "from": "spurgeon-ccel:330"}
`
	dir := writeSource(t, map[string]string{"source.toml": goodThemeToml, "topics.jsonl": topics, "links.jsonl": links})
	c, err := Load(ctx, db, dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := c.String(); got != "demo: 2 topics, 1 parts, 3 links, 0 removed" {
		t.Errorf("load: %s", got)
	}
	again, err := Load(ctx, db, dir)
	if err != nil {
		t.Fatal(err)
	}
	if again != c {
		t.Errorf("second load %s, first %s", again, c)
	}
	if n := count(t, db, "select count(*) from theme_links"); n != 3 {
		t.Errorf("%d links after two loads", n)
	}
	q, _ := passage.Parse("Rom.5.8")
	var label string
	var part *string
	if err := db.QueryRow(ctx, `select t.label, p.label from theme_links l
		join themes t on t.source_id = l.source_id and t.id = l.theme_id
		left join theme_parts p on p.theme_source_id = l.source_id and p.theme_id = l.theme_id and p.n = l.part_n
		where l.span && $1::int4range`, q.PG()).Scan(&label, &part); err != nil {
		t.Fatal(err)
	}
	if label != "Love-Hatred" || part == nil || *part != "Of God" {
		t.Errorf("Rom.5.8's theme: %s / %v", label, part)
	}
	var see []string
	if err := db.QueryRow(ctx, "select see from themes where id = 'love'").Scan(&see); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(see, []string{"grace"}) {
		t.Errorf("see = %v", see)
	}
	if n := count(t, db, "select count(*) from theme_sources where id = 'demo'"); n != 1 {
		t.Errorf("theme_sources does not list the theme source")
	}

	// Dropping a topic removes it, its parts and its links.
	links2 := `{"topic": "grace", "ref": "Rom.5.15-Rom.5.17"}` + "\n"
	topics2 := `{"id": "grace", "label": "Grace, Divine"}` + "\n"
	os.WriteFile(filepath.Join(dir, "topics.jsonl"), []byte(topics2), 0o644)
	os.WriteFile(filepath.Join(dir, "links.jsonl"), []byte(links2), 0o644)
	c3, err := Load(ctx, db, dir)
	if err != nil {
		t.Fatal(err)
	}
	// Removed: the topic "love" and the two links no longer in the files.
	if got := c3.String(); got != "demo: 1 topics, 0 parts, 1 links, 3 removed" {
		t.Errorf("reload: %s", got)
	}
	if n := count(t, db, "select count(*) from theme_links"); n != 1 {
		t.Errorf("%d links after the reload", n)
	}
	if n := count(t, db, "select count(*) from theme_parts"); n != 0 {
		t.Errorf("%d parts after the reload", n)
	}
}
