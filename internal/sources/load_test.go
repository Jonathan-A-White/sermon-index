package sources

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
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
// 52 rows (7 read from the 1847 Complete Works, 45 from the 1830
// Miscellaneous Works), 46 of them with a preaching text, and Matthew 16:26
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
	if n := count(t, db, "select count(*) from sermons where source_id = 'henry-sermons'"); n != 52 {
		t.Errorf("%d rows, want 52", n)
	}
	if n := count(t, db, "select count(*) from sermons where source_id = 'henry-sermons' and preacher <> 'Matthew Henry'"); n != 0 {
		t.Errorf("%d rows by someone else than Matthew Henry", n)
	}
	if n := count(t, db, "select count(*) from sermon_passages where role = 'text' and sermon_id in (select id from sermons where source_id = 'henry-sermons')"); n != 46 {
		t.Errorf("%d preaching texts, want 46", n)
	}
	for item, want := range map[string]int{"miscellaneouswor00henr": 45, "completeworksofr0001revm": 7} {
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
		{"burroughs-logos", 62, 57, 0, 5},
		{"keach-logos", 17, 17, 0, 0},
		{"maclaren-logos", 30, 0, 0, 30},
		{"kuyper-logos", 7, 0, 0, 7},
		{"broadus-logos", 21, 0, 0, 21},
		{"worlds-great-sermons-logos", 113, 0, 0, 113},
		{"princeton-sermons-logos", 16, 16, 0, 0},
		{"mcpherson-logos", 8, 0, 0, 8},
		{"strong-logos", 49, 0, 0, 49},
		{"sedgwick-logos", 7, 4, 0, 3},
		{"moberly-logos", 15, 15, 0, 0},
		{"eyton-logos", 17, 0, 0, 17},
		{"chapin-logos", 8, 0, 0, 8},
		{"watson-logos", 8, 8, 0, 0},
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
	// YEAR.2006.30, YEAR.2010.2); no other archive has a text there.
	rom58, err := passage.Parse("Rom.5.8")
	if err != nil {
		t.Fatal(err)
	}
	if n := count(t, db, "select count(distinct (p.source_id, p.sermon_id)) from sermon_passages p where p.role = 'text' and p.source_id like '%-logos' and p.span && $1::int4range", rom58.PG()); n != 3 {
		t.Errorf("%d Logos rows on Romans 5:8, want 3", n)
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
	// "Chapter I" is "The First Beatitude" (with no preaching text); no front matter or
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
	if n := count(t, db, "select count(*) from sermons s where s.source_id = 'maclaren-logos' and s.id = 'BTTDSTHRSRMNS.1-4253' and s.title = 'The First Beatitude' and not exists (select 1 from sermon_passages p where p.source_id = s.source_id and p.sermon_id = s.id and p.role = 'text')"); n != 1 {
		t.Errorf("Maclaren's First Beatitude: %d, want once", n)
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
}
