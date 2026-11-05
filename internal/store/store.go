// Package store opens the sermon index's Postgres database.
//
// The binary reads the connection URL from DATABASE_URL (cmd/sermons); the
// tests never do: they start their own server (testdb.go).
package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Open connects to the database at url and checks that it answers.
func Open(ctx context.Context, url string) (*pgxpool.Pool, error) {
	if url == "" {
		return nil, errors.New("no database URL: set DATABASE_URL (postgres://user:password@host:port/db)")
	}
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("DATABASE_URL: %w", err)
	}
	db, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connecting: %w", err)
	}
	if err := db.Ping(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("connecting to %s:%d: %w", cfg.ConnConfig.Host, cfg.ConnConfig.Port, err)
	}
	return db, nil
}
