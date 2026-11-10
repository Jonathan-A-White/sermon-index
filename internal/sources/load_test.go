package sources

import (
	"bufio"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

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
	if got := first.String(); got != "spurgeon-20: 20 rows, 19 passages, 1 topic links, 0 removed" {
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

func TestLoadStoresTheSourcesLanguage(t *testing.T) {
	ctx := context.Background()
	db := migrated(t)
	lang := func(id string) string {
		var l string
		if err := db.QueryRow(ctx, `SELECT language FROM sources WHERE id = $1`, id).Scan(&l); err != nil {
			t.Fatal(err)
		}
		return l
	}
	body := map[string]string{"rows.jsonl": goodRow + "\n"}
	body["source.toml"] = goodToml
	en := writeSource(t, body)
	if _, err := Load(ctx, db, en); err != nil {
		t.Fatal(err)
	}
	if got := lang("demo"); got != "en" {
		t.Errorf("no language loads as %q, want en", got)
	}
	body["source.toml"] = goodToml + "language = \"pt\"\n"
	if _, err := Load(ctx, db, writeSource(t, body)); err != nil {
		t.Fatal(err)
	}
	if got := lang("demo"); got != "pt" {
		t.Errorf("language = pt loads as %q", got)
	}
	// Taking the key out again puts the source back to English.
	if _, err := Load(ctx, db, en); err != nil {
		t.Fatal(err)
	}
	if got := lang("demo"); got != "en" {
		t.Errorf("language left out loads as %q, want en", got)
	}
	body["source.toml"] = goodToml + "language = \"xx\"\n"
	if _, err := Load(ctx, db, writeSource(t, body)); err == nil || !strings.Contains(err.Error(), "language 'xx'") {
		t.Errorf("language xx: %v", err)
	}
	if got := lang("demo"); got != "en" {
		t.Errorf("a refused load changed the language to %q", got)
	}
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

// The real Spurgeon source loads in a few seconds with the counts the tool
// printed when it wrote rows.jsonl (tools/ccel-spurgeon): 3,543 rows, 3,467
// of them with a preaching text that resolves.
func TestLoadSpurgeonCCEL(t *testing.T) {
	const dir = "../../sources/spurgeon-ccel"
	if fs := Check(dir); len(fs) != 0 {
		t.Fatalf("check: %v", Faults(fs))
	}
	ctx := context.Background()
	db := migrated(t)

	start := time.Now()
	res, err := Load(ctx, db, dir)
	if err != nil {
		t.Fatal(err)
	}
	if took := time.Since(start); took > 10*time.Second {
		t.Errorf("load took %v, want under 10 s", took)
	}
	t.Log(res.String())

	if n := count(t, db, "select count(*) from sermons where source_id = 'spurgeon-ccel'"); n != 3543 {
		t.Errorf("%d sermons, want 3543", n)
	}
	if n := count(t, db, "select count(*) from sermon_passages where role = 'text'"); n != 3467 {
		t.Errorf("%d preaching texts, want 3467", n)
	}
	// Romans 5: the preaching text of 22 sermons lies in the chapter.
	rom5, err := passage.Parse("Rom.5")
	if err != nil {
		t.Fatal(err)
	}
	if n := count(t, db, "select count(distinct sermon_id) from sermon_passages where role = 'text' and span && $1::int4range", rom5.PG()); n != 22 {
		t.Errorf("%d sermons with their text in Romans 5, want 22", n)
	}
	if n := count(t, db, "select count(*) from sermons where source_id = 'spurgeon-ccel' and kind = 'exposition'"); n != 5 {
		t.Errorf("%d expositions, want 5", n)
	}
}

// A topic's verses is what its links cover together, written by load and
// kept right when the links change (the theme tier's 300-verse cap reads it).
func TestLoadWritesATopicsVerses(t *testing.T) {
	ctx := context.Background()
	db := migrated(t)
	topics := `{"id": "love", "label": "Love"}
{"id": "grace", "label": "Grace"}
{"id": "none", "label": "No links"}
`
	links := `{"topic": "love", "ref": "Rom.5.8"}
{"topic": "grace", "ref": "Rom.5.15-Rom.5.21"}
{"topic": "grace", "ref": "Rom.5.20"}
{"topic": "grace", "ref": "Rom.6.1-Rom.6.2"}
`
	verses := func() map[string]int {
		rows, err := db.Query(ctx, "select id, verses from themes order by id")
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		out := map[string]int{}
		for rows.Next() {
			var id string
			var n *int
			if err := rows.Scan(&id, &n); err != nil || n == nil {
				t.Fatalf("%s: %v %v", id, n, err)
			}
			out[id] = *n
		}
		return out
	}
	dir := writeSource(t, map[string]string{"source.toml": goodThemeToml, "topics.jsonl": topics, "links.jsonl": links})
	if _, err := Load(ctx, db, dir); err != nil {
		t.Fatal(err)
	}
	if got := verses(); got["love"] != 1 || got["grace"] != 9 || got["none"] != 0 {
		t.Errorf("verses after load: %v (want love 1, grace 7+2, none 0)", got)
	}
	dir = writeSource(t, map[string]string{"source.toml": goodThemeToml, "topics.jsonl": topics,
		"links.jsonl": `{"topic": "love", "ref": "Rom.5"}` + "\n" + `{"topic": "grace", "ref": "Rom.5.15"}` + "\n"})
	if _, err := Load(ctx, db, dir); err != nil {
		t.Fatal(err)
	}
	if got := verses(); got["love"] != 21 || got["grace"] != 1 {
		t.Errorf("verses after the links changed: %v", got)
	}
}

// The hand-made Matthew Henry source loads with the counts REPORT.md gives:
// 51 rows (7 read from the 1847 Complete Works, 44 from the 1830
// Miscellaneous Works), 45 of them with a preaching text, and Matthew 16:26
// answers the six sermons of The Worth of the Soul. Every url is a reader
// page of one of the two Archive items, never a page of Philip Henry's.
func TestLoadHenrySermons(t *testing.T) {
	const dir = "../../sources/henry-sermons"
	if fs := Check(dir); len(fs) != 0 {
		t.Fatalf("check: %v", Faults(fs))
	}
	ctx := context.Background()
	db := migrated(t)
	if _, err := Load(ctx, db, dir); err != nil {
		t.Fatal(err)
	}
	if n := count(t, db, "select count(*) from sermons where source_id = 'henry-sermons'"); n != 51 {
		t.Errorf("%d rows, want 51", n)
	}
	if n := count(t, db, "select count(*) from sermons where source_id = 'henry-sermons' and preacher <> 'Matthew Henry'"); n != 0 {
		t.Errorf("%d rows by someone else than Matthew Henry", n)
	}
	if n := count(t, db, "select count(*) from sermon_passages where role = 'text' and sermon_id in (select id from sermons where source_id = 'henry-sermons')"); n != 45 {
		t.Errorf("%d preaching texts, want 45", n)
	}
	for item, want := range map[string]int{"miscellaneouswor00henr": 44, "completeworksofr0001revm": 7} {
		prefix := "https://archive.org/details/" + item + "/page/n"
		if n := count(t, db, "select count(*) from sermons where source_id = 'henry-sermons' and url like $1", prefix+"%"); n != want {
			t.Errorf("%d rows at %s, want %d", n, item, want)
		}
	}
	if n := count(t, db, "select count(*) from sermons where source_id = 'henry-sermons' and url !~ '^https://archive.org/details/(miscellaneouswor00henr|completeworksofr0001revm)/page/n[0-9]+$'"); n != 0 {
		t.Errorf("%d rows whose url is not an Archive reader page of the two items", n)
	}
	// Matthew 16:26: the six sermons of The Worth of the Soul.
	mt, err := passage.Parse("Matt.16.26")
	if err != nil {
		t.Fatal(err)
	}
	if n := count(t, db, "select count(*) from sermon_passages p join sermons s on s.id = p.sermon_id where s.source_id = 'henry-sermons' and p.role = 'text' and s.title like 'The Worth of the Soul%' and p.span && $1::int4range", mt.PG()); n != 6 {
		t.Errorf("%d sermons of The Worth of the Soul on Matthew 16:26, want 6", n)
	}
	// A row without a preaching text says why.
	if n := count(t, db, "select count(*) from sermons s where s.source_id = 'henry-sermons' and not exists (select 1 from sermon_passages p where p.sermon_id = s.id and p.role = 'text') and coalesce(s.notes, '') not like 'no text printed%'"); n != 0 {
		t.Errorf("%d rows without a text and without the note 'no text printed'", n)
	}
}

// A `series` row is a sermon like any other, found by its title and listed
// by its source, but it has no passage: nothing a passage asks for matches it.
func TestLoadASeriesRowKeepsTheRowAndNoPassage(t *testing.T) {
	ctx := context.Background()
	db := migrated(t)
	rows := `{"id": "1", "title": "One Verse", "ref": "Rom.5.8", "url": "https://example.org/1"}
{"id": "2", "title": "Romans, Verse by Verse", "ref": "Rom.1.1-Rom.16.24", "cited": ["John.3.16"], "url": "https://example.org/2", "series": true}
`
	dir := writeSource(t, map[string]string{"source.toml": goodToml, "rows.jsonl": rows})
	c, err := Load(ctx, db, dir)
	if err != nil {
		t.Fatal(err)
	}
	if c.Rows != 2 || c.Passages != 1 {
		t.Errorf("counts %+v: want 2 rows and 1 passage (the series row has none)", c)
	}
	if n := count(t, db, "select count(*) from sermons where source_id = 'demo'"); n != 2 {
		t.Errorf("%d sermons, want 2", n)
	}
	if n := count(t, db, "select count(*) from sermon_passages where source_id = 'demo' and sermon_id = '2'"); n != 0 {
		t.Errorf("the series row has %d passages, want 0", n)
	}
	// Taking `series` off a row that was loaded as one gives it its passages
	// back, and putting it on takes them away.
	dir2 := writeSource(t, map[string]string{"source.toml": goodToml, "rows.jsonl": strings.Replace(rows, `, "series": true`, "", 1)})
	if _, err := Load(ctx, db, dir2); err != nil {
		t.Fatal(err)
	}
	if n := count(t, db, "select count(*) from sermon_passages where source_id = 'demo' and sermon_id = '2'"); n != 2 {
		t.Errorf("the row without series has %d passages, want 2", n)
	}
	if _, err := Load(ctx, db, dir); err != nil {
		t.Fatal(err)
	}
	if n := count(t, db, "select count(*) from sermon_passages where source_id = 'demo' and sermon_id = '2'"); n != 0 {
		t.Errorf("the series row has %d passages again, want 0", n)
	}
}

// The Governor's Logos sermon archives (tools/logos-import) load with the
// rows the index listed: 1,196, 248 and 334 for the three archives with an
// index section, and for batch 2's books (Spurgeon's My Sermon Notes 264,
// Wesley 164 once the Works' repeats of Sermons on Several Occasions are
// out, Whitefield 59, Moody 32). A `series` row has no passage, so the
// preaching texts are the rows with a ref less the series.
func TestLoadLogosArchives(t *testing.T) {
	archives := []struct {
		id            string
		rows, texts   int
		series, noRef int
	}{
		{"greg-laurie-logos", 1196, 1066, 25, 105},
		{"jd-greear-logos", 248, 206, 3, 39},
		{"skip-heitzig-logos", 334, 157, 1, 176},
		{"spurgeon-sermon-notes-logos", 264, 264, 0, 0},
		{"wesley-logos", 164, 137, 0, 27},
		{"whitefield-logos", 59, 47, 0, 12},
		{"moody-logos", 32, 8, 0, 24},
		{"augustine-logos", 70, 60, 0, 10},
		{"chrysostom-logos", 483, 479, 0, 4},
		{"piper-logos", 5, 0, 0, 5},
		{"bunyan-logos", 1, 0, 0, 1},
		{"charles-wesley-logos", 14, 12, 0, 2},
		{"jimmy-evans-logos", 156, 0, 0, 156},
		{"shedd-logos", 46, 46, 0, 0},
		{"burroughs-logos", 62, 58, 0, 4},
		{"keach-logos", 17, 17, 0, 0},
		{"maclaren-logos", 30, 20, 0, 10},
		{"kuyper-logos", 7, 0, 0, 7},
		{"broadus-logos", 21, 13, 0, 8},
		{"worlds-great-sermons-logos", 113, 0, 0, 113},
		{"princeton-sermons-logos", 16, 16, 0, 0},
		{"mcpherson-logos", 8, 0, 0, 8},
		{"strong-logos", 49, 0, 0, 49},
		{"sedgwick-logos", 7, 4, 0, 3},
		{"moberly-logos", 15, 15, 0, 0},
		{"eyton-logos", 17, 0, 0, 17},
		{"chapin-logos", 8, 0, 0, 8},
		{"watson-logos", 8, 8, 0, 0},
		{"pope-logos", 17, 0, 0, 17},
		{"smale-logos", 6, 0, 0, 6},
		{"wi-evans-logos", 10, 1, 0, 9},
		{"calvin-pt-logos", 13, 13, 0, 0},
		{"evangelistic-sermon-outlines-logos", 29, 29, 0, 0},
		{"sermon-outlines-for-growing-christians-logos", 27, 26, 1, 0},
		{"sermon-outlines-for-seekers-logos", 26, 26, 0, 0},
		{"sermon-outlines-for-special-days-logos", 28, 28, 0, 0},
		{"sermon-outlines-on-1-corinthians-logos", 28, 28, 0, 0},
		{"sermon-outlines-on-acts-logos", 38, 37, 1, 0},
		{"sermon-outlines-on-christian-living-logos", 30, 30, 0, 0},
		{"sermon-outlines-on-ephesians-logos", 26, 26, 0, 0},
		{"sermon-outlines-on-galatians-ephesians-philippians-colossians-logos", 30, 30, 0, 0},
		{"sermon-outlines-on-romans-logos", 28, 28, 0, 0},
		{"sermon-outlines-on-the-gospels-logos", 26, 26, 0, 0},
		{"sermon-starters-volume-1-logos", 200, 200, 0, 0},
		{"sermon-starters-volume-2-logos", 198, 198, 0, 0},
		{"sermon-starters-volume-3-logos", 199, 199, 0, 0},
		{"sermon-starters-volume-4-logos", 198, 198, 0, 0},
		{"sermon-starters-volume-5-logos", 199, 199, 0, 0},
	}
	ctx := context.Background()
	db := migrated(t)
	for _, a := range archives {
		dir := "../../sources/" + a.id
		faults, warnings := CheckReport(dir)
		if len(faults) != 0 || len(warnings) != 0 {
			t.Fatalf("%s: faults %v, warnings %v", a.id, Faults(faults), Faults(warnings))
		}
		if _, err := Load(ctx, db, dir); err != nil {
			t.Fatal(err)
		}
		q := func(where string, args ...any) int {
			return count(t, db, "select count(*) from sermons s where s.source_id = '"+a.id+"' "+where, args...)
		}
		if n := q(""); n != a.rows {
			t.Errorf("%s: %d rows, want %d", a.id, n, a.rows)
		}
		if n := count(t, db, "select count(*) from sermon_passages where role = 'text' and source_id = $1", a.id); n != a.texts {
			t.Errorf("%s: %d preaching texts, want %d", a.id, n, a.texts)
		}
		if n := q("and not exists (select 1 from sermon_passages p where p.source_id = s.source_id and p.sermon_id = s.id and p.role = 'text')"); n != a.series+a.noRef {
			t.Errorf("%s: %d rows without a preaching text, want %d (series %d, no ref %d)", a.id, n, a.series+a.noRef, a.series, a.noRef)
		}
		if n := q("and (kind <> 'link' or url !~ '^https://ref.ly/logosres/LLS:[A-Za-z0-9._]+\\?(art|off)=' or notes ~ 'Logos text')"); n != 0 {
			t.Errorf("%s: %d rows that are not a link to a ref.ly address", a.id, n)
		}
		if n := q("and preacher is null"); n != 0 {
			t.Errorf("%s: %d rows with no preacher", a.id, n)
		}
	}

	// Greg Laurie's "Who Is Man" (YEAR.1996.6) is under Genesis 1.
	gen1, err := passage.Parse("Gen.1")
	if err != nil {
		t.Fatal(err)
	}
	if n := count(t, db, "select count(*) from sermon_passages p join sermons s on s.source_id = p.source_id and s.id = p.sermon_id where p.role = 'text' and p.source_id = 'greg-laurie-logos' and s.id = 'YEAR.1996.6' and s.title = 'Who Is Man' and p.span && $1::int4range", gen1.PG()); n != 1 {
		t.Errorf("Who Is Man is under Genesis 1 %d times, want once", n)
	}
	// Romans 5:8: Greg Laurie preached on Romans 5 three times (YEAR.2000.8,
	// YEAR.2006.30, YEAR.2010.2); the only other texts there are two Sermon
	// Outlines (Special Days' Memorial Day, Romans 5:6-8, and Seekers' The
	// Great Therefore, Romans 5:1-10), two Sermon Outlines on Romans (Peace
	// on Earth? How?, 5:1-11, and The Verdict Changed, 5:6-11) and Sermon
	// Starters' God's Love (5:8).
	rom58, err := passage.Parse("Rom.5.8")
	if err != nil {
		t.Fatal(err)
	}
	if n := count(t, db, "select count(distinct (p.source_id, p.sermon_id)) from sermon_passages p where p.role = 'text' and p.source_id like '%-logos' and p.span && $1::int4range", rom58.PG()); n != 8 {
		t.Errorf("%d Logos rows on Romans 5:8, want 8", n)
	}
	if n := count(t, db, "select count(*) from sermon_passages p where p.role = 'text' and p.source_id like '%-logos' and p.sermon_id in ('YEAR.2000.8', 'YEAR.2006.30', 'YEAR.2010.2') and p.span && $1::int4range", rom58.PG()); n != 3 {
		t.Errorf("YEAR.2000.8, YEAR.2006.30 and YEAR.2010.2 on Romans 5:8: %d, want 3", n)
	}

	// Batch 2 (the book-archive form): Spurgeon's chapter "Chapter I: Gen.
	// 19:15" is "Genesis 19:15" under Genesis 19:15; Wesley's "Salvation by
	// Faith" is there once (the Works' copy is a repeat, dropped); his
	// "God's Love to Fallen Man" is under Romans 5:15; Moody's "Where Art
	// Thou?" is under Genesis 3:9. Each is a link to its offset in the book.
	for _, c := range []struct {
		source, id, title, ref string
	}{
		{"spurgeon-sermon-notes-logos", "MYSRMNOTES01.2-14522", "Genesis 19:15", "Gen.19.15"},
		{"wesley-logos", "36.0.11.2-1735602", "Sermon 59: God’s Love to Fallen Man", "Rom.5.15"},
		{"moody-logos", "12SELSERMMOODY.1-771", "“Where Art Thou?”", "Gen.3.9"},
	} {
		rg, err := passage.Parse(c.ref)
		if err != nil {
			t.Fatal(err)
		}
		if n := count(t, db, "select count(*) from sermon_passages p join sermons s on s.source_id = p.source_id and s.id = p.sermon_id where p.role = 'text' and p.source_id = $1 and s.id = $2 and s.title = $3 and s.kind = 'link' and s.url like 'https://ref.ly/logosres/LLS:%?off=%' and p.span && $4::int4range", c.source, c.id, c.title, rg.PG()); n != 1 {
			t.Errorf("%s %s %q under %s: %d, want once", c.source, c.id, c.title, c.ref, n)
		}
	}
	if n := count(t, db, "select count(*) from sermons where source_id = 'wesley-logos' and title like '%Salvation by Faith'"); n != 1 {
		t.Errorf("Wesley's Salvation by Faith is in the index %d times, want once", n)
	}
	// No two rows of Wesley's index have the same title once a Roman numeral
	// or "Sermon N" is off, bar the ones Sermons on Several Occasions has twice.
	if n := count(t, db, "select count(*) from sermons where source_id = 'wesley-logos' and id like 'WORKSWESLEY%' and title ~ '^Sermon [IVXLC]+: (Salvation by Faith|The Almost Christian|Satan.s Devices)'"); n != 0 {
		t.Errorf("%d Works rows that repeat Sermons on Several Occasions are in the index", n)
	}

	// Batch 3: Augustine's Sermon 95 is under Mark 8:1-9, read from the tail
	// of its title; a Chrysostom homily is under the verse it preaches on, by
	// John Chrysostom (the index names the editor); Charles Wesley's Sermon XII
	// is under 1 Kings 18:21, whose label the book prints "1 Kings, 18:2l".
	for _, c := range []struct {
		source, id, title, ref, preacher string
	}{
		{"augustine-logos", "AGSTNSRMNS04.1-279517", "Sermon 95: On the Words of the Gospel, Mark 8:1–9, Where the Miracle of the Seven Loaves Is Related", "Mark.8.1-Mark.8.9", "Augustine of Hippo"},
		{"chrysostom-logos", "6.60.10.3-267160", "Homily II", "Matt.1.1", "John Chrysostom"},
		{"charles-wesley-logos", "SERMSCWESLEY.1-247025", "Sermon XII", "1Kgs.18.21", "Charles Wesley"},
	} {
		rg, err := passage.Parse(c.ref)
		if err != nil {
			t.Fatal(err)
		}
		if n := count(t, db, "select count(*) from sermon_passages p join sermons s on s.source_id = p.source_id and s.id = p.sermon_id where p.role = 'text' and p.source_id = $1 and s.id = $2 and s.title = $3 and s.preacher = $4 and s.url like 'https://ref.ly/logosres/LLS:%?off=%' and p.span && $5::int4range", c.source, c.id, c.title, c.preacher, rg.PG()); n != 1 {
			t.Errorf("%s %s %q under %s: %d, want once", c.source, c.id, c.title, c.ref, n)
		}
	}
	// Chrysostom's non-sermon lines (the Indexes of Texts cite a thousand
	// passages each) are not in the index, and Augustine's Chronological
	// Table is not.
	if n := count(t, db, "select count(*) from sermons where source_id = 'chrysostom-logos' and (title like '%Index of Texts%' or title like '%Argument%' or title like '%Homilist%')"); n != 0 {
		t.Errorf("%d Chrysostom rows that are not homilies", n)
	}
	if n := count(t, db, "select count(*) from sermons where source_id = 'augustine-logos' and title in ('Chronological Table', 'New Testament')"); n != 0 {
		t.Errorf("%d Augustine rows that are not sermons", n)
	}
	// Jimmy Evans: dated, in a series, on no verse.
	if n := count(t, db, "select count(*) from sermons s where s.source_id = 'jimmy-evans-logos' and s.id = 'SER2004.1.1' and s.date = '2004-06-11' and s.notes = 'Series: Blessed Marriage Seminar, part 1.' and not exists (select 1 from sermon_passages p where p.source_id = s.source_id and p.sermon_id = s.id)"); n != 1 {
		t.Errorf("Jimmy Evans SER2004.1.1: %d, want once", n)
	}

	// Batch 4a: Shedd's chapter 1 of the Natural Man is "The Future State a
	// Self-Conscious State" under 1 Corinthians 13:12 ("Chapter 1: " taken
	// off); Burroughs's Spots sermons carry the date of their title; Maclaren's
	// "Chapter I" is "The First Beatitude" (its text inferred, below); no front matter or
	// table of contents is in the index.
	rg, err := passage.Parse("1Cor.13.12")
	if err != nil {
		t.Fatal(err)
	}
	if n := count(t, db, "select count(*) from sermon_passages p join sermons s on s.source_id = p.source_id and s.id = p.sermon_id where p.role = 'text' and p.source_id = 'shedd-logos' and s.id = 'SRMNATMAN.1-43544' and s.title = 'The Future State a Self-Conscious State' and s.preacher = 'William G. T. Shedd' and p.span && $1::int4range", rg.PG()); n != 1 {
		t.Errorf("Shedd chapter 1 under 1 Corinthians 13:12: %d, want once", n)
	}
	if n := count(t, db, "select count(*) from sermons where source_id = 'burroughs-logos' and id = 'WRKSBURROUGHS07.1-1510' and date = '1645-07-06' and title = 'The First Sermon (July the 6th, 1645)'"); n != 1 {
		t.Errorf("Burroughs's First Sermon of the Spots (1645-07-06): %d, want once", n)
	}
	if n := count(t, db, "select count(*) from sermons where source_id = 'burroughs-logos' and date is not null"); n != 4 {
		t.Errorf("Burroughs rows with a date: %d, want 4", n)
	}
	// --infer-text (mw-qtu3zvz.35): Maclaren's First Beatitude cites Matthew
	// 5:3 alone, so that is its text, marked inferred in its notes; Burroughs's
	// Sermon I of the Rare Jewel takes Philippians 4:11 from the sermons beside
	// it; Maclaren's Chapter X cites two passages and stays without a text.
	rg, err = passage.Parse("Matt.5.3")
	if err != nil {
		t.Fatal(err)
	}
	if n := count(t, db, "select count(*) from sermon_passages p join sermons s on s.source_id = p.source_id and s.id = p.sermon_id where p.role = 'text' and p.source_id = 'maclaren-logos' and s.id = 'BTTDSTHRSRMNS.1-4253' and s.title = 'The First Beatitude' and s.notes like '%The ref is inferred%' and p.span && $1::int4range", rg.PG()); n != 1 {
		t.Errorf("Maclaren's First Beatitude under Matthew 5:3, inferred: %d, want once", n)
	}
	if n := count(t, db, "select count(*) from sermons s where s.source_id = 'maclaren-logos' and s.id = 'BTTDSTHRSRMNS.1-144838' and not exists (select 1 from sermon_passages p where p.source_id = s.source_id and p.sermon_id = s.id and p.role = 'text')"); n != 1 {
		t.Errorf("Maclaren's Chapter X (two cited passages) without a text: %d, want once", n)
	}
	rg, err = passage.Parse("Phil.4.11")
	if err != nil {
		t.Fatal(err)
	}
	if n := count(t, db, "select count(*) from sermon_passages p join sermons s on s.source_id = p.source_id and s.id = p.sermon_id where p.role = 'text' and p.source_id = 'burroughs-logos' and s.id = 'WRKSBURROUGHS01.1-15584' and s.notes like '%The ref is inferred%' and p.span && $1::int4range", rg.PG()); n != 1 {
		t.Errorf("Burroughs's Sermon I under Philippians 4:11, inferred: %d, want once", n)
	}
	for _, id := range []string{"shedd-logos", "burroughs-logos", "keach-logos", "maclaren-logos", "kuyper-logos", "broadus-logos"} {
		if n := count(t, db, "select count(*) from sermons where source_id = $1 and (title ~ '^Chapter [IVXLC0-9]+: ' or title ~* '(^|[^a-z])(preface|epistle to the reader|contents of|alphabetical table|table of the chief heads)')", id); n != 0 {
			t.Errorf("%s: %d rows that are front matter or keep a Chapter prefix", id, n)
		}
	}

	// Batch 4b: the World's Great Sermons name the preacher in the title, not
	// the editor Grenville Kleiser (kept in the notes); Princeton's title
	// prints the preacher and the text; Strong's numbers are off the titles and
	// his index's General Index and catch-all heading are not rows.
	if n := count(t, db, "select count(*) from sermons where source_id = 'worlds-great-sermons-logos' and id = 'WRLDGRTSERM01.1-5589' and title = 'The Creation of the World' and preacher = 'Basil' and notes like '%Preacher’s years: 329–379.%' and notes like '%Book credit: Grenville Kleiser.%'"); n != 1 {
		t.Errorf("Basil's Creation of the World: %d, want once", n)
	}
	if n := count(t, db, "select count(*) from sermons where source_id = 'worlds-great-sermons-logos' and (preacher = 'Grenville Kleiser' or title ~ '\\([0-9]{3,4}–' or title ~ '^General Index')"); n != 0 {
		t.Errorf("World's Great Sermons: %d rows with the editor as preacher, years in the title or the General Index", n)
	}
	rg, err = passage.Parse("John.17.24")
	if err != nil {
		t.Fatal(err)
	}
	if n := count(t, db, "select count(*) from sermon_passages p join sermons s on s.source_id = p.source_id and s.id = p.sermon_id where p.role = 'text' and p.source_id = 'princeton-sermons-logos' and s.id = 'PRINCESERMON.1-5790' and s.title = 'Christ’s Desire for His People' and s.preacher = 'William Henry Green' and p.span && $1::int4range", rg.PG()); n != 1 {
		t.Errorf("Princeton's Christ's Desire for His People under John 17:24: %d, want once", n)
	}
	if n := count(t, db, "select count(*) from sermons where source_id = 'princeton-sermons-logos' and title like '%—By %'"); n != 0 {
		t.Errorf("Princeton: %d titles that keep the preacher", n)
	}
	if n := count(t, db, "select count(*) from sermons where source_id = 'strong-logos' and id = 'PHILANDREL.1-1485859' and title = 'Christianity and Political Economy' and notes like '%Number XLIV of the collection.%'"); n != 1 {
		t.Errorf("Strong's XLIV: %d, want once", n)
	}
	if n := count(t, db, "select count(*) from sermons where source_id = 'strong-logos' and (title ~ '^[IVXLC]+: ' or title ~* '^addresses to successive')"); n != 0 {
		t.Errorf("Strong: %d titles that keep a number or are the catch-all heading", n)
	}

	// Batch 5a: Moberly's "Sermon I" is titled with its text; Sedgwick's
	// Heresies is Revelation 12:15-16 (his title prints it, "Reyel. 12:15,
	// 16.", and Logos lists the two verses apart), with the words alone for
	// the title; Eyton and Chapin have no text.
	rg, err = passage.Parse("Matt.5.3")
	if err != nil {
		t.Fatal(err)
	}
	if n := count(t, db, "select count(*) from sermon_passages p join sermons s on s.source_id = p.source_id and s.id = p.sermon_id where p.role = 'text' and p.source_id = 'moberly-logos' and s.id = 'SRMNSBTTD.1-113776' and s.title = 'Sermon I: Matthew 5:3' and s.preacher = 'George Moberly' and p.span && $1::int4range", rg.PG()); n != 1 {
		t.Errorf("Moberly's Sermon I under Matthew 5:3: %d, want once", n)
	}
	if n := count(t, db, "select count(*) from sermons where source_id = 'moberly-logos' and title !~ '^Sermon [IVX]+: .+ [0-9]+:?[0-9–]*$'"); n != 0 {
		t.Errorf("Moberly: %d titles that are not a number and a text", n)
	}
	rg, err = passage.Parse("Rev.12.15-Rev.12.16")
	if err != nil {
		t.Fatal(err)
	}
	if n := count(t, db, "select count(*) from sermon_passages p join sermons s on s.source_id = p.source_id and s.id = p.sermon_id where p.role = 'text' and p.source_id = 'sedgwick-logos' and s.id = 'NTRDNGRHRSS.1-4734' and s.title = 'The Nature and Danger of Heresies' and s.preacher = 'Obadiah Sedgwick' and p.span = $1::int4range", rg.PG()); n != 1 {
		t.Errorf("Sedgwick's Heresies exactly Revelation 12:15-16: %d, want once", n)
	}

	// Batch 5b: Calvin's Portuguese sermons are on Genesis 25 to 27 from their
	// titles (the third, "Gênesis 25.21,22", is the two verses as one range),
	// with the Portuguese title kept; Evans's fourth chapter is on 1
	// Corinthians 14, "Chapter 4: " off its title; Pope and Smale have no text.
	rg, err = passage.Parse("Gen.25.21-Gen.25.22")
	if err != nil {
		t.Fatal(err)
	}
	if n := count(t, db, "select count(*) from sermon_passages p join sermons s on s.source_id = p.source_id and s.id = p.sermon_id where p.role = 'text' and p.source_id = 'calvin-pt-logos' and s.id = 'SRMSSBRLRPRV.1-96169' and s.title = 'III. Terceiro Sermão: Gênesis 25.21,22' and s.preacher = 'John Calvin' and p.span = $1::int4range", rg.PG()); n != 1 {
		t.Errorf("Calvin's third sermon exactly Genesis 25:21-22: %d, want once", n)
	}
	if n := count(t, db, "select count(*) from sermons where source_id = 'calvin-pt-logos' and (title ~* 'pref[aá]cio' or title ~* 'cal[uú]nias')"); n != 0 {
		t.Errorf("Calvin: %d rows for the preface or the appendix", n)
	}
	if n := count(t, db, "select count(*) from sources where id = 'calvin-pt-logos' and language = 'pt'"); n != 1 {
		t.Errorf("calvin-pt-logos is not stored with language pt")
	}
	if n := count(t, db, "select count(*) from sources where language <> 'en' and id <> 'calvin-pt-logos'"); n != 0 {
		t.Errorf("%d other sources are not English", n)
	}
	rg, err = passage.Parse("1Cor.14")
	if err != nil {
		t.Fatal(err)
	}
	if n := count(t, db, "select count(*) from sermon_passages p join sermons s on s.source_id = p.source_id and s.id = p.sermon_id where p.role = 'text' and p.source_id = 'wi-evans-logos' and s.title = 'Speaking in Tongues' and p.span = $1::int4range", rg.PG()); n != 1 {
		t.Errorf("Evans's Speaking in Tongues under 1 Corinthians 14: %d, want once", n)
	}
	if n := count(t, db, "select count(*) from sermons where source_id in ('wi-evans-logos', 'smale-logos') and title ~ '^Chapter '"); n != 0 {
		t.Errorf("Evans and Smale: %d titles that keep \"Chapter N: \"", n)
	}

	// Batch 6a (the Sermon Outlines books): the passage in a title is the ref
	// and is off the title ("Paul and the Corinthians" is 1 Corinthians
	// 1:1-9); Special Days keeps the bracket of the three titles whose
	// passage the book table does not read; Acts keeps one row for each
	// chapter outline (no numbered sub-point, no "Chapter N: ") and the
	// Soul Winning book is not a source.
	for _, c := range []struct {
		source, id, title, ref, preacher string
	}{
		{"sermon-outlines-on-1-corinthians-logos", "SO1COR.1-2867", "Paul and the Corinthians", "1Cor.1.1-1Cor.1.9", "Ben Merold"},
		{"sermon-outlines-for-growing-christians-logos", "SOGROWCHR.1-1913", "So Help Me, Me", "Heb.6.13-Heb.6.20", "Stephen M. Hooks"},
		{"sermon-outlines-for-seekers-logos", "SOSEEKERS.1-9766", "Seven Secrets to Spiritual Success", "Luke.5.1-Luke.5.11", "J. Michael Shannon"},
		{"sermon-outlines-for-special-days-logos", "SOSPECDAYS.1-8396", "Maundy Thursday: The Example of Jesus at Communion", "Luke.22.7-Luke.22.16", "Robert C. Shannon"},
		{"sermon-outlines-on-acts-logos", "SOACTS.1-129186", "The Holy Spirit in Conversion", "Acts.7.51-Acts.7.53", "C. C. Crawford"},
		{"evangelistic-sermon-outlines-logos", "SOEVANG.1-8054", "What Think Ye of Christ?", "Matt.22.41-Matt.22.45", "Burris Butler"},
	} {
		rg, err := passage.Parse(c.ref)
		if err != nil {
			t.Fatal(err)
		}
		if n := count(t, db, "select count(*) from sermon_passages p join sermons s on s.source_id = p.source_id and s.id = p.sermon_id where p.role = 'text' and p.source_id = $1 and s.id = $2 and s.title = $3 and s.preacher = $4 and s.url like 'https://ref.ly/logosres/LLS:%?off=%' and p.span = $5::int4range", c.source, c.id, c.title, c.preacher, rg.PG()); n != 1 {
			t.Errorf("%s %s %q exactly %s: %d, want once", c.source, c.id, c.title, c.ref, n)
		}
	}
	if n := count(t, db, "select count(*) from sermons where source_id = 'sermon-outlines-on-acts-logos' and (title ~ '^[0-9]+\\. ' or title ~ '^Chapter ' or title = 'Afterword')"); n != 0 {
		t.Errorf("Acts: %d rows that are a sub-point, keep \"Chapter N: \" or are the afterword", n)
	}
	if n := count(t, db, "select count(*) from sermons where source_id like 'sermon-outlines-%-logos' and source_id <> 'sermon-outlines-on-acts-logos' and title ~ '\\((Matthew|Mark|Luke|John|Acts|Hebrews|1 Corinthians) [0-9]+:[0-9–]+\\)$'"); n != 0 {
		t.Errorf("%d outline rows that keep a bracketed passage a book table can read", n)
	}
	if n := count(t, db, "select count(*) from sermons where source_id = 'sermon-outlines-for-special-days-logos' and title like '%(%'"); n != 3 {
		t.Errorf("Special Days: %d titles with a bracketed passage kept, want the 3 it does not read", n)
	}
	if n := count(t, db, "select count(*) from sermons where source_id like '%soul-winning%'"); n != 0 {
		t.Errorf("%d rows of the Soul Winning book, which is not imported", n)
	}

	// Batch 6b (five more Sermon Outlines books and Sermon Starters, Volume
	// 1): the passage in a title is the ref and is off the title, a bracketed
	// text over two chapters ("Galatians 1:1-2:21") included; a title's
	// "10a" is read to verse 10; the Scripture Index of Sermon Starters is not
	// a row (201 lines, 200 rows).
	for _, c := range []struct {
		source, id, title, ref, preacher string
	}{
		{"sermon-outlines-on-galatians-ephesians-philippians-colossians-logos", "SOGEPC.2-2070", "Paul and the Gospel", "Gal.1.1-Gal.2.21", "Sam E. Stone"},
		{"sermon-outlines-on-romans-logos", "SOROM.1-3975", "Condition? Critical!", "Rom.1.18-Rom.2.2", "Marshall W. Hayden"},
		{"sermon-outlines-on-christian-living-logos", "SOCHRLIV.1-25496", "Faithfulness in Hard Places", "2Pet.2.1-2Pet.2.10", "Burris Butler"},
		{"sermon-outlines-on-the-gospels-logos", "SOGOSP.1-7483", "The Best of the Good News", "John.3.16-John.3.17", "Mark Scott"},
		{"sermon-outlines-on-ephesians-logos", "SOEPH.1-7468", "Marks of the Church: Part 1", "Eph.1.15-Eph.1.23", "Floyd Strater"},
		{"sermon-starters-volume-1-logos", "SERMONSTARTERS01.2-7635", "Disinterest in God", "Gen.4.16", "John G. Butler"},
	} {
		rg, err := passage.Parse(c.ref)
		if err != nil {
			t.Fatal(err)
		}
		if n := count(t, db, "select count(*) from sermon_passages p join sermons s on s.source_id = p.source_id and s.id = p.sermon_id where p.role = 'text' and p.source_id = $1 and s.id = $2 and s.title = $3 and s.preacher = $4 and s.url like 'https://ref.ly/logosres/LLS:%?off=%' and p.span = $5::int4range", c.source, c.id, c.title, c.preacher, rg.PG()); n != 1 {
			t.Errorf("%s %s %q exactly %s: %d, want once", c.source, c.id, c.title, c.ref, n)
		}
	}
	if n := count(t, db, "select count(*) from sermons where source_id in ('sermon-outlines-on-christian-living-logos', 'sermon-outlines-on-ephesians-logos', 'sermon-outlines-on-galatians-ephesians-philippians-colossians-logos', 'sermon-outlines-on-romans-logos', 'sermon-outlines-on-the-gospels-logos', 'sermon-starters-volume-1-logos') and title ~ '\\((1 |2 )?[A-Z][a-z]+ [0-9]+[:0-9,–a-b ]*\\)$'"); n != 1 {
		t.Errorf("%d batch 6b rows that keep a bracketed passage, want the one with two passages (Pears or Prickle?)", n)
	}
	if n := count(t, db, "select count(*) from sermons where source_id = 'sermon-starters-volume-1-logos' and title ilike '%index%'"); n != 0 {
		t.Errorf("%d Sermon Starters rows that are the Scripture Index", n)
	}

	// Batch 6c (Sermon Starters, Volumes 2 to 5): the passage in a title is
	// the ref and is off the title; the Scripture Index of each volume and
	// volume 4's four numbered points ("First—The Purpose in the Message")
	// are not rows (199, 200, 203 and 200 lines; 198, 199, 198 and 199 rows).
	for _, c := range []struct {
		source, id, title, ref string
	}{
		{"sermon-starters-volume-2-logos", "SERMONSTARTERS02.2-7101", "God", "Gen.1.1"},
		{"sermon-starters-volume-2-logos", "SERMONSTARTERS02.2-576657", "Nosey", "John.21.21"},
		{"sermon-starters-volume-3-logos", "SERMONSTARTERS03.2-567840", "Giving", "Rom.15.27"},
		{"sermon-starters-volume-4-logos", "SERMONSTARTERS04.2-562939", "Encouragement", "Rev.1.9"},
		{"sermon-starters-volume-5-logos", "SERMONSTARTERS05.2-20122", "Judgment First", "Isa.2.6"},
	} {
		rg, err := passage.Parse(c.ref)
		if err != nil {
			t.Fatal(err)
		}
		if n := count(t, db, "select count(*) from sermon_passages p join sermons s on s.source_id = p.source_id and s.id = p.sermon_id where p.role = 'text' and p.source_id = $1 and s.id = $2 and s.title = $3 and s.preacher = 'John G. Butler' and s.url like 'https://ref.ly/logosres/LLS:SERMONSTARTERS0_?off=%' and p.span = $4::int4range", c.source, c.id, c.title, rg.PG()); n != 1 {
			t.Errorf("%s %s %q exactly %s: %d, want once", c.source, c.id, c.title, c.ref, n)
		}
	}
	if n := count(t, db, "select count(*) from sermons where source_id like 'sermon-starters-volume-%-logos' and (title ilike '%index%' or title ~ '^(First|Second|Third|Fourth)—' or title like '%(%)')"); n != 0 {
		t.Errorf("%d Sermon Starters rows that are an index, a numbered point or keep a bracket", n)
	}
	if n := count(t, db, "select count(*) from sermons s where source_id like 'sermon-starters-volume-%-logos' and not exists (select 1 from sermon_passages p where p.source_id = s.source_id and p.sermon_id = s.id and p.role = 'text')"); n != 0 {
		t.Errorf("%d Sermon Starters rows with no preaching text, want none", n)
	}
}

// The Archive leaf of every Matthew Henry row.
//
// A hand-made url https://archive.org/details/<item>/page/n<N> names a
// leaf of a scan, and the first version of the source had most of its urls
// wrong (the Worth of the Soul, Sermon I, opened in the middle of Sermon II):
// N is not the page name in the OCR file and not the printed page. REPORT.md
// holds a table, one line a row, of the leaf found by reading the Archive's
// own OCR, its printed page and the phrases of the sermon's head the OCR
// prints on it. TestHenryLeafTable keeps the table and rows.jsonl in step and
// always runs; TestHenryLeavesHoldTheirHeads reads the OCR itself and runs
// when the OCR is cached (sh scripts/fetch-henry-ocr.sh, or make henry-leaves).

const henryDir = "../../sources/henry-sermons"

type henryLeaf struct {
	item  string
	n     int
	heads []string
}

var henryRowLine = regexp.MustCompile("^\\| `([a-z0-9-]+)` \\| `([a-z0-9]+)` \\| ([0-9]+) \\| ([0-9]+) \\| (.+) \\|$")

// henryTable reads REPORT.md: the leaf table, and the ids of the rows it says
// were left out because the scan cannot place them.
func henryTable(t *testing.T) (leaves map[string]henryLeaf, notPlaced []string) {
	t.Helper()
	f, err := os.Open(filepath.Join(henryDir, "REPORT.md"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	leaves = map[string]henryLeaf{}
	inNotPlaced := false
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "## ") {
			inNotPlaced = strings.HasPrefix(line, "## Rows left out because the scan cannot place them")
			continue
		}
		if inNotPlaced && strings.HasPrefix(line, "- `") {
			if end := strings.Index(line[3:], "`"); end > 0 {
				notPlaced = append(notPlaced, line[3:3+end])
			}
			continue
		}
		m := henryRowLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		n, _ := strconv.Atoi(m[3])
		var heads []string
		for _, h := range strings.Split(m[5], ";") {
			if h = strings.TrimSpace(h); h != "" {
				heads = append(heads, h)
			}
		}
		if _, dup := leaves[m[1]]; dup {
			t.Errorf("REPORT.md: row %s twice in the leaf table", m[1])
		}
		leaves[m[1]] = henryLeaf{item: m[2], n: n, heads: heads}
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	return leaves, notPlaced
}

// henryRows reads rows.jsonl: id to url.
func henryRows(t *testing.T) map[string]string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(henryDir, "rows.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	rows := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		var r struct{ ID, URL string }
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			t.Fatalf("rows.jsonl: %v: %s", err, line)
		}
		rows[r.ID] = r.URL
	}
	return rows
}

var henryURL = regexp.MustCompile(`^https://archive\.org/details/([a-z0-9]+)/page/n([0-9]+)$`)

// TestHenryLeafTable: the table in REPORT.md and rows.jsonl name the same rows
// and the same leaves, and a row REPORT.md says could not be placed is not in
// the index. It needs no OCR.
func TestHenryLeafTable(t *testing.T) {
	leaves, notPlaced := henryTable(t)
	rows := henryRows(t)
	for id, url := range rows {
		l, ok := leaves[id]
		if !ok {
			t.Errorf("row %s has no line in the leaf table of REPORT.md", id)
			continue
		}
		if want := fmt.Sprintf("https://archive.org/details/%s/page/n%d", l.item, l.n); url != want {
			t.Errorf("row %s: url %s, but REPORT.md puts it at %s", id, url, want)
		}
		if len(l.heads) == 0 {
			t.Errorf("row %s: no head phrase in REPORT.md", id)
		}
	}
	for id := range leaves {
		if _, ok := rows[id]; !ok {
			t.Errorf("REPORT.md has a leaf for %s, which is not a row", id)
		}
	}
	if len(notPlaced) == 0 {
		t.Error("REPORT.md names no row left out because the scan cannot place it (funeral-sermon-4)")
	}
	for _, id := range notPlaced {
		if _, ok := rows[id]; ok {
			t.Errorf("row %s is in the index but REPORT.md says its leaf is not in the scan", id)
		}
	}
}

// norm lower-cases s and turns every run of anything but letters and digits
// into one space, with a space at each end, so a phrase is found by Contains
// on whole words and "sermon i" is not found in "SERMON II.".
func norm(s string) string {
	var b strings.Builder
	b.WriteByte(' ')
	space := true
	for _, r := range strings.ToLower(s) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
			space = false
		} else if !space {
			b.WriteByte(' ')
			space = true
		}
	}
	if !space {
		b.WriteByte(' ')
	}
	return b.String()
}

// djvuLeaves streams the Archive's _djvu.xml and returns the text of the
// leaves in want, by the 0-based index of the OBJECT, which is the reader's n,
// and the number of leaves the file holds.
func djvuLeaves(path string, want map[int]bool) (map[int]string, int, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()
	dec := xml.NewDecoder(bufio.NewReaderSize(f, 1<<20))
	dec.Strict = false
	out := map[int]string{}
	n, inWord := -1, false
	var cur strings.Builder
	flush := func() {
		if n >= 0 && want[n] {
			out[n] = cur.String()
		}
		cur.Reset()
	}
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, 0, err
		}
		switch e := tok.(type) {
		case xml.StartElement:
			switch e.Name.Local {
			case "OBJECT":
				flush()
				n++
			case "WORD":
				inWord = true
			}
		case xml.EndElement:
			if e.Name.Local == "WORD" {
				inWord = false
				cur.WriteByte(' ')
			}
		case xml.CharData:
			if inWord {
				cur.Write(e)
			}
		}
	}
	flush()
	return out, n + 1, nil
}

// TestHenryLeavesHoldTheirHeads: the OCR of the leaf each row's url opens prints
// that sermon's head. Skipped, with the command that fills it, when the OCR is
// not cached.
func TestHenryLeavesHoldTheirHeads(t *testing.T) {
	leaves, _ := henryTable(t)
	rows := henryRows(t)
	cache := os.Getenv("SERMON_INDEX_ARCHIVE_CACHE")
	if cache == "" {
		cache = "../../.archive-cache"
	}
	byItem := map[string]map[int]bool{}
	for id := range rows {
		m := henryURL.FindStringSubmatch(rows[id])
		if m == nil {
			t.Fatalf("row %s: url %s is not an Archive reader page", id, rows[id])
		}
		n, _ := strconv.Atoi(m[2])
		if byItem[m[1]] == nil {
			byItem[m[1]] = map[int]bool{}
		}
		byItem[m[1]][n] = true
		if n > 0 {
			byItem[m[1]][n-1] = true // the leaf before: the head must not stand there too
		}
	}
	text := map[string]map[int]string{}
	for item, want := range byItem {
		path := filepath.Join(cache, item+"_djvu.xml")
		if _, err := os.Stat(path); err != nil {
			t.Skipf("the Archive's OCR is not cached (%s): run  sh scripts/fetch-henry-ocr.sh  (or  make henry-leaves)", path)
		}
		got, total, err := djvuLeaves(path, want)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		for n := range want {
			if n >= total {
				t.Errorf("%s has %d leaves, a row asks for n%d", item, total, n)
			}
		}
		text[item] = got
	}
	ids := make([]string, 0, len(rows))
	for id := range rows {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		m := henryURL.FindStringSubmatch(rows[id])
		n, _ := strconv.Atoi(m[2])
		l, ok := leaves[id]
		if !ok {
			t.Errorf("row %s has no head phrases in REPORT.md", id)
			continue
		}
		page := norm(text[m[1]][n])
		for _, h := range l.heads {
			if !strings.Contains(page, norm(h)) {
				t.Errorf("row %s: leaf n%d of %s does not print %q", id, n, m[1], h)
			}
		}
		// A head that is also on the leaf before is a running head: the
		// sermon began earlier.
		if n > 0 {
			before, all := norm(text[m[1]][n-1]), true
			for _, h := range l.heads {
				all = all && strings.Contains(before, norm(h))
			}
			if all {
				t.Errorf("row %s: leaf n%d of %s prints the head, and so does n%d before it: the sermon begins earlier", id, n, m[1], n-1)
			}
		}
	}
}

// Direct links (sermon-topics.jsonl): loaded with the source, the same on a
// second load, a line taken out deleted, the file taken out all deleted; a
// sermon source loads before the theme source its links name, and its direct
// rows answer once that is loaded.
func TestLoadSermonTopics(t *testing.T) {
	ctx := context.Background()
	db := migrated(t)
	dir := demoWithTopics(t, `{"row": "1", "theme": "themes-x", "topic": "love"}`+"\n"+
		`{"row": "1", "theme": "themes-x", "topic": "grace"}`+"\n"+
		`{"row": "s", "theme": "themes-x", "topic": "grace"}`+"\n")
	links := func() []string {
		t.Helper()
		rows, err := db.Query(ctx, `select source_id || '/' || sermon_id || ' ' || theme_source_id || ':' || theme_id
			from sermon_topics order by 1`)
		if err != nil {
			t.Fatal(err)
		}
		var out []string
		for rows.Next() {
			var s string
			if err := rows.Scan(&s); err != nil {
				t.Fatal(err)
			}
			out = append(out, s)
		}
		return out
	}

	// The sermon source first: themes-x is not loaded.
	first, err := Load(ctx, db, dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := first.String(); got != "demo: 2 rows, 1 passages, 3 topic links, 0 removed" {
		t.Errorf("first load: %s", got)
	}
	want := []string{"demo/1 themes-x:grace", "demo/1 themes-x:love", "demo/s themes-x:grace"}
	if got := links(); !reflect.DeepEqual(got, want) {
		t.Errorf("links %q, want %q", got, want)
	}
	second, err := Load(ctx, db, dir)
	if err != nil || second != first {
		t.Errorf("second load %s (%v), first %s", second, err, first)
	}
	if got := links(); !reflect.DeepEqual(got, want) {
		t.Errorf("links after the second load %q", got)
	}

	// Its direct row answers once the theme source is loaded: Love links
	// Rom.5.8, where demo/1 is a passage row, and Grace links Eph.2.8.
	span, _ := passage.Parse("Eph.2.8")
	q := store.ThemeQuery{Span: span, Limit: 50}
	if total, _, err := store.ThemeSermons(ctx, db, q); err != nil || total != 0 {
		t.Errorf("before themes-x is loaded: %d %v", total, err)
	}
	if _, err := Load(ctx, db, filepath.Join(filepath.Dir(dir), "themes-x")); err != nil {
		t.Fatal(err)
	}
	total, rows, err := store.ThemeSermons(ctx, db, q)
	if err != nil || total != 2 || len(rows) != 2 || !rows[0].Direct || rows[0].ID != "1" || rows[1].ID != "s" || rows[0].Theme.Topic != "grace" {
		t.Errorf("after themes-x is loaded: %d %+v %v", total, rows, err)
	}
	// A theme source's reload never deletes a direct link.
	if _, err := Load(ctx, db, filepath.Join(filepath.Dir(dir), "themes-x")); err != nil {
		t.Fatal(err)
	}
	if got := links(); !reflect.DeepEqual(got, want) {
		t.Errorf("links after themes-x's reload %q", got)
	}

	// A line taken out is deleted on reload.
	if err := os.WriteFile(filepath.Join(dir, SermonTopicsFile), []byte(`{"row": "1", "theme": "themes-x", "topic": "love"}`+"\n"+
		`{"row": "s", "theme": "themes-x", "topic": "grace"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := Load(ctx, db, dir)
	if err != nil || c.String() != "demo: 2 rows, 1 passages, 2 topic links, 0 removed" {
		t.Errorf("a line out: %s %v", c, err)
	}
	if got, want := links(), []string{"demo/1 themes-x:love", "demo/s themes-x:grace"}; !reflect.DeepEqual(got, want) {
		t.Errorf("links after a line out %q, want %q", got, want)
	}

	// The file taken out: the source has no links, and its load line is as
	// it was before direct links.
	if err := os.Remove(filepath.Join(dir, SermonTopicsFile)); err != nil {
		t.Fatal(err)
	}
	c, err = Load(ctx, db, dir)
	if err != nil || c.String() != "demo: 2 rows, 1 passages, 0 removed" {
		t.Errorf("the file out: %s %v", c, err)
	}
	if got := links(); len(got) != 0 {
		t.Errorf("links after the file out %q", got)
	}
}

// A sermon taken out of rows.jsonl takes its links with it (and the check
// refuses a link to it left in the file).
func TestLoadSermonTopicsFollowTheirRow(t *testing.T) {
	ctx := context.Background()
	db := migrated(t)
	dir := demoWithTopics(t, `{"row": "s", "theme": "themes-x", "topic": "grace"}`+"\n")
	if _, err := Load(ctx, db, dir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, RowsFile), []byte(goodRow+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(ctx, db, dir); err == nil || !strings.Contains(err.Error(), "sermon-topics.jsonl:1: row 's': not in rows.jsonl") {
		t.Errorf("a link to a row taken out: %v", err)
	}
	if err := os.Remove(filepath.Join(dir, SermonTopicsFile)); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(ctx, db, dir); err != nil {
		t.Fatal(err)
	}
	if n := count(t, db, "select count(*) from sermon_topics"); n != 0 {
		t.Errorf("%d links left", n)
	}
}
