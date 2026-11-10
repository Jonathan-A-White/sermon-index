package migrate

import (
	"context"
	"os"
	"reflect"
	"testing"

	"github.com/Jonathan-A-White/sermon-index/internal/store"
)

func TestMain(m *testing.M) { os.Exit(store.RunTests(m)) }

func TestMigrationsAreNumberedInOrder(t *testing.T) {
	ms, err := All()
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) == 0 || ms[0].Version != 1 || ms[0].Name != "0001_init" {
		t.Fatalf("migrations: %+v", ms)
	}
	for i, m := range ms {
		if m.Version != i+1 {
			t.Errorf("migration %s has version %d, want %d", m.Name, m.Version, i+1)
		}
	}
}

func TestUpAppliesOnceAndASecondRunIsANoOp(t *testing.T) {
	ctx := context.Background()
	db := store.TestDB(t)

	pending, err := Pending(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(pending, []string{"0001_init", "0002_search", "0003_theme_verses", "0004_source_language", "0005_sermon_topics"}) {
		t.Errorf("pending before = %v", pending)
	}

	applied, err := Up(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(applied, []string{"0001_init", "0002_search", "0003_theme_verses", "0004_source_language", "0005_sermon_topics"}) {
		t.Errorf("first Up applied %v", applied)
	}
	again, err := Up(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	if len(again) != 0 {
		t.Errorf("second Up applied %v, want nothing", again)
	}
	var n int
	if err := db.QueryRow(ctx, "select count(*) from schema_migrations").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 5 {
		t.Errorf("schema_migrations has %d rows, want 5", n)
	}
	pending, err = Pending(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 0 {
		t.Errorf("pending after = %v", pending)
	}
}

func TestSchemaV1(t *testing.T) {
	ctx := context.Background()
	db := store.TestDB(t)
	if _, err := Up(ctx, db); err != nil {
		t.Fatal(err)
	}

	want := map[string][]string{
		"sources":         {"id", "name", "kind", "licence", "licence_note", "credit", "homepage", "terms_url", "added", "on_by_default"},
		"sermons":         {"source_id", "id", "title", "preacher", "url", "alt_url", "kind", "date", "duration_s", "notes"},
		"sermon_passages": {"source_id", "sermon_id", "ref_osis", "role", "span"},
		"themes":          {"source_id", "id", "label", "see"},
		"theme_parts":     {"theme_source_id", "theme_id", "n", "label"},
		"theme_links":     {"source_id", "theme_id", "part_n", "ref_osis", "span", "via", "from_ref"},
		"theme_sources":   {"id", "name", "licence", "credit"},
		"sermon_topics":   {"source_id", "sermon_id", "theme_source_id", "theme_id"},
	}
	for table, cols := range want {
		for _, col := range cols {
			var n int
			err := db.QueryRow(ctx, `select count(*) from information_schema.columns
				where table_schema = 'public' and table_name = $1 and column_name = $2`, table, col).Scan(&n)
			if err != nil {
				t.Fatal(err)
			}
			if n != 1 {
				t.Errorf("%s.%s missing", table, col)
			}
		}
	}

	var spanType string
	if err := db.QueryRow(ctx, `select data_type from information_schema.columns
		where table_name = 'sermon_passages' and column_name = 'span'`).Scan(&spanType); err != nil {
		t.Fatal(err)
	}
	if spanType != "int4range" {
		t.Errorf("sermon_passages.span is %s", spanType)
	}

	for _, table := range []string{"sermon_passages", "theme_links"} {
		var n int
		if err := db.QueryRow(ctx, `select count(*) from pg_indexes
			where tablename = $1 and indexdef like '%USING gist (span)%'`, table).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 1 {
			t.Errorf("%s has %d GiST indexes on span, want 1", table, n)
		}
	}

	var pk string
	if err := db.QueryRow(ctx, `select pg_get_constraintdef(oid) from pg_constraint
		where conrelid = 'sermons'::regclass and contype = 'p'`).Scan(&pk); err != nil {
		t.Fatal(err)
	}
	if pk != "PRIMARY KEY (source_id, id)" {
		t.Errorf("sermons primary key: %s", pk)
	}
}

// TestSchemaRefusesBadRows: the database holds its own rules, not only the
// loader.
func TestSchemaRefusesBadRows(t *testing.T) {
	ctx := context.Background()
	db := store.TestDB(t)
	if _, err := Up(ctx, db); err != nil {
		t.Fatal(err)
	}
	mustExec := func(sql string, args ...any) {
		t.Helper()
		if _, err := db.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}
	mustExec(`insert into sources (id, name, kind, licence, credit, homepage, terms_url, added, on_by_default)
		values ('s', 'S', 'sermons', 'Public domain', 'C', 'https://example.org', 'https://example.org/terms', '2026-11-05', false)`)
	mustExec(`insert into sources (id, name, kind, licence, credit, homepage, terms_url, added, on_by_default)
		values ('t', 'T', 'themes', 'Public domain', 'C', 'https://example.org', 'https://example.org/terms', '2026-11-05', false)`)
	mustExec(`insert into sermons (source_id, id, title, url) values ('s', '1', 'One', 'https://example.org/1')`)

	for _, c := range []struct{ name, sql string }{
		{"empty licence", `insert into sources (id, name, kind, licence, credit, homepage, terms_url, added, on_by_default)
			values ('x', 'X', 'sermons', '', 'C', 'https://e.org', 'https://e.org', '2026-11-05', false)`},
		{"unknown kind", `insert into sources (id, name, kind, licence, credit, homepage, terms_url, added, on_by_default)
			values ('x', 'X', 'videos', 'PD', 'C', 'https://e.org', 'https://e.org', '2026-11-05', false)`},
		{"a sermon in a theme source", `insert into sermons (source_id, id, title, url) values ('t', '1', 'One', 'https://example.org/1')`},
		{"a theme in a sermon source", `insert into themes (source_id, id, label) values ('s', 'love', 'Love')`},
		{"http url", `insert into sermons (source_id, id, title, url) values ('s', '2', 'Two', 'http://example.org/2')`},
		{"sermon kind", `insert into sermons (source_id, id, title, url, kind) values ('s', '2', 'Two', 'https://example.org/2', 'video')`},
		{"duplicate key", `insert into sermons (source_id, id, title, url) values ('s', '1', 'Again', 'https://example.org/1')`},
		{"passage of no sermon", `insert into sermon_passages (source_id, sermon_id, ref_osis, role, span) values ('s', '99', 'Rom.5.8', 'text', '[45005008,45005009)')`},
		{"passage role", `insert into sermon_passages (source_id, sermon_id, ref_osis, role, span) values ('s', '1', 'Rom.5.8', 'quoted', '[45005008,45005009)')`},
		{"empty span", `insert into sermon_passages (source_id, sermon_id, ref_osis, role, span) values ('s', '1', 'Rom.5.8', 'text', 'empty')`},
		{"date shape", `insert into sermons (source_id, id, title, url, date) values ('s', '3', 'Three', 'https://example.org/3', '5 Nov 1855')`},
		{"empty string for absent", `insert into sermons (source_id, id, title, url, preacher) values ('s', '4', 'Four', 'https://example.org/4', '')`},
		{"topic link of no sermon", `insert into sermon_topics (source_id, sermon_id, theme_source_id, theme_id) values ('s', '99', 't', 'love')`},
	} {
		if _, err := db.Exec(ctx, c.sql); err == nil {
			t.Errorf("%s: accepted", c.name)
		}
	}
}

// sermon_topics: a direct link needs its sermon and goes with it, but needs
// no topic (a sermon source may load before its theme sources).
func TestSermonTopicsKeys(t *testing.T) {
	ctx := context.Background()
	db := store.TestDB(t)
	if _, err := Up(ctx, db); err != nil {
		t.Fatal(err)
	}
	for _, sql := range []string{
		`insert into sources (id, name, kind, licence, credit, homepage, terms_url, added, on_by_default)
			values ('s', 'S', 'sermons', 'Public domain', 'C', 'https://example.org', 'https://example.org/terms', '2026-11-05', false)`,
		`insert into sermons (source_id, id, title, url) values ('s', '1', 'One', 'https://example.org/1')`,
		`insert into sermon_topics (source_id, sermon_id, theme_source_id, theme_id) values ('s', '1', 'themes-none', 'love')`,
	} {
		if _, err := db.Exec(ctx, sql); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}
	if _, err := db.Exec(ctx, `insert into sermon_topics (source_id, sermon_id, theme_source_id, theme_id) values ('s', '1', 'themes-none', 'love')`); err == nil {
		t.Error("a repeated link was accepted")
	}
	if _, err := db.Exec(ctx, `insert into sermon_topics (source_id, sermon_id, theme_source_id, theme_id) values ('s', '2', 'themes-none', 'love')`); err == nil {
		t.Error("a link to a sermon not in sermons was accepted")
	}
	if _, err := db.Exec(ctx, `delete from sermons where source_id = 's' and id = '1'`); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := db.QueryRow(ctx, `select count(*) from sermon_topics`).Scan(&n); err != nil || n != 0 {
		t.Errorf("deleting the sermon left %d links (%v)", n, err)
	}
	var idx int
	if err := db.QueryRow(ctx, `select count(*) from pg_indexes where tablename = 'sermon_topics'
		and indexdef like '%(theme_source_id, theme_id)%'`).Scan(&idx); err != nil || idx != 1 {
		t.Errorf("sermon_topics has %d indexes on (theme_source_id, theme_id) (%v)", idx, err)
	}
}

// 0005 applies over a database loaded at 0004, keeping its rows.
func TestSermonTopicsMigrationKeepsLoadedRows(t *testing.T) {
	ctx := context.Background()
	db := store.TestDB(t)
	ms, err := All()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(ctx, createTable); err != nil {
		t.Fatal(err)
	}
	for _, m := range ms[:4] {
		if _, err := db.Exec(ctx, m.SQL); err != nil {
			t.Fatalf("%s: %v", m.Name, err)
		}
		if _, err := db.Exec(ctx, `insert into schema_migrations (version, name) values ($1, $2)`, m.Version, m.Name); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(ctx, `insert into sources (id, name, kind, licence, credit, homepage, terms_url, added, on_by_default)
			values ('s', 'S', 'sermons', 'Public domain', 'C', 'https://example.org', 'https://example.org/terms', '2026-11-05', false)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(ctx, `insert into sermons (source_id, id, title, url) values ('s', '1', 'One', 'https://example.org/1')`); err != nil {
		t.Fatal(err)
	}
	applied, err := Up(ctx, db)
	if err != nil || !reflect.DeepEqual(applied, []string{"0005_sermon_topics"}) {
		t.Fatalf("Up over 0004: %v %v", applied, err)
	}
	var n int
	if err := db.QueryRow(ctx, `select count(*) from sermons`).Scan(&n); err != nil || n != 1 {
		t.Errorf("sermons after 0005: %d (%v)", n, err)
	}
	if again, err := Up(ctx, db); err != nil || len(again) != 0 {
		t.Errorf("second Up: %v %v", again, err)
	}
}
