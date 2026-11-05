// Package migrate applies the schema's SQL migrations, which are embedded in
// the binary.
//
// A migration is sql/NNNN_name.sql; its version is NNNN, and versions run
// 1, 2, 3 with no gaps. Each is applied once, in its own transaction, and
// recorded in schema_migrations; a run takes a transaction-scoped advisory
// lock first, so two `sermons migrate` at once apply each migration once.
// A migration that has been applied is never edited: a change is a new file.
package migrate

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed sql/*.sql
var files embed.FS

// lockKey is the advisory lock key a migration run holds ("sermons" as
// ASCII digits, an arbitrary constant).
const lockKey = 115101114109111110

// Migration is one embedded SQL file.
type Migration struct {
	Version int
	Name    string // the file name without .sql, "0001_init"
	SQL     string
}

var fileName = regexp.MustCompile(`^(\d{4})_[a-z0-9_]+\.sql$`)

// All returns the embedded migrations in version order.
func All() ([]Migration, error) {
	entries, err := fs.ReadDir(files, "sql")
	if err != nil {
		return nil, err
	}
	var ms []Migration
	for _, e := range entries {
		m := fileName.FindStringSubmatch(e.Name())
		if m == nil {
			return nil, fmt.Errorf("migration %s: not named NNNN_name.sql", e.Name())
		}
		v, _ := strconv.Atoi(m[1])
		body, err := files.ReadFile("sql/" + e.Name())
		if err != nil {
			return nil, err
		}
		ms = append(ms, Migration{Version: v, Name: e.Name()[:len(e.Name())-len(".sql")], SQL: string(body)})
	}
	sort.Slice(ms, func(i, j int) bool { return ms[i].Version < ms[j].Version })
	for i, m := range ms {
		if m.Version != i+1 {
			return nil, fmt.Errorf("migration %s: version %d, want %d (no gaps, no repeats)", m.Name, m.Version, i+1)
		}
	}
	return ms, nil
}

const createTable = `CREATE TABLE IF NOT EXISTS schema_migrations (
	version    integer PRIMARY KEY,
	name       text NOT NULL,
	applied_at timestamptz NOT NULL DEFAULT now()
)`

// Up applies every migration not yet applied and returns their names. A
// run with nothing to apply is a no-op that returns none.
func Up(ctx context.Context, db *pgxpool.Pool) ([]string, error) {
	ms, err := All()
	if err != nil {
		return nil, err
	}
	var done []string
	for _, m := range ms {
		ran, err := apply(ctx, db, m)
		if err != nil {
			return done, err
		}
		if ran {
			done = append(done, m.Name)
		}
	}
	return done, nil
}

// apply runs one migration in its own transaction, unless it has been
// applied already; it reports whether it ran.
func apply(ctx context.Context, db *pgxpool.Pool, m Migration) (bool, error) {
	tx, err := db.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", int64(lockKey)); err != nil {
		return false, fmt.Errorf("taking the migration lock: %w", err)
	}
	if _, err := tx.Exec(ctx, createTable); err != nil {
		return false, fmt.Errorf("schema_migrations: %w", err)
	}
	var n int
	if err := tx.QueryRow(ctx, "SELECT count(*) FROM schema_migrations WHERE version = $1", m.Version).Scan(&n); err != nil {
		return false, err
	}
	if n > 0 {
		return false, nil
	}
	if _, err := tx.Exec(ctx, m.SQL); err != nil {
		return false, fmt.Errorf("migration %s: %w", m.Name, err)
	}
	if _, err := tx.Exec(ctx, "INSERT INTO schema_migrations (version, name) VALUES ($1, $2)", m.Version, m.Name); err != nil {
		return false, err
	}
	return true, tx.Commit(ctx)
}

// Pending returns the names of the migrations not yet applied.
func Pending(ctx context.Context, db *pgxpool.Pool) ([]string, error) {
	ms, err := All()
	if err != nil {
		return nil, err
	}
	var exists bool
	if err := db.QueryRow(ctx, "SELECT to_regclass('schema_migrations') IS NOT NULL").Scan(&exists); err != nil {
		return nil, err
	}
	have := map[int]bool{}
	if exists {
		rows, err := db.Query(ctx, "SELECT version FROM schema_migrations")
		if err != nil {
			return nil, err
		}
		vs, err := pgx.CollectRows(rows, pgx.RowTo[int32])
		if err != nil {
			return nil, err
		}
		for _, v := range vs {
			have[int(v)] = true
		}
	}
	var out []string
	for _, m := range ms {
		if !have[m.Version] {
			out = append(out, m.Name)
		}
	}
	return out, nil
}
