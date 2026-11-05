// Command sermons is the sermon index's one binary.
//
//	sermons migrate                    apply the schema's migrations to DATABASE_URL
//	sermons check [--urls N] <dir>     check a source directory; exit 1 on any fault
//	sermons check --all <root>         check every source directory under root
//	sermons load <dir>                 load a source directory into DATABASE_URL
//	sermons load --all <root>          load every source directory under root
//	sermons serve                      answer the read-only HTTP API (docs/api.md)
//
// DATABASE_URL is a Postgres URL (postgres://user:password@host:port/db).
// serve also reads SERMONS_ADDR (default 127.0.0.1:8790) and
// SERMONS_ORIGINS (the origins CORS allows, comma-separated).
// A directory under root whose name begins with '_' (the template, the test
// fixtures) is never checked or loaded by --all.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Jonathan-A-White/sermon-index/internal/api"
	"github.com/Jonathan-A-White/sermon-index/internal/migrate"
	"github.com/Jonathan-A-White/sermon-index/internal/sources"
	"github.com/Jonathan-A-White/sermon-index/internal/store"
)

const usage = `usage:
  sermons migrate                  apply the schema's migrations to DATABASE_URL
  sermons check [--urls N] <dir>   check a source directory; exit 1 on any fault
  sermons check --all <root>       check every source directory under root
  sermons load <dir>               load a source directory into DATABASE_URL
  sermons load --all <root>        load every source directory under root
  sermons serve                    answer the HTTP API on SERMONS_ADDR (default 127.0.0.1:8790)
`

// version is the build's git describe, set by make build:
//
//	go build -ldflags "-X main.version=$(git describe --tags --always --dirty)"
//
// /v1/health answers it; a plain go build or go run says "dev".
var version = "dev"

// defaultAddr is SERMONS_ADDR when it is not set.
const defaultAddr = "127.0.0.1:8790"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	os.Exit(run(ctx, os.Args[1:], os.Stdout, os.Stderr, os.Getenv))
}

// run is the command; it returns the exit status: 0 done, 1 a fault or a
// failure, 2 a usage error.
func run(ctx context.Context, args []string, stdout, stderr io.Writer, getenv func(string) string) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return 2
	}
	switch args[0] {
	case "migrate":
		return runMigrate(ctx, args[1:], stdout, stderr, getenv)
	case "check":
		return runCheck(ctx, args[1:], stdout, stderr)
	case "load":
		return runLoad(ctx, args[1:], stdout, stderr, getenv)
	case "serve":
		return runServe(ctx, args[1:], stderr, getenv)
	case "help", "-h", "--help":
		fmt.Fprint(stdout, usage)
		return 0
	}
	fmt.Fprintf(stderr, "sermons: unknown command %q\n%s", args[0], usage)
	return 2
}

func newFlags(name string, stderr io.Writer) *flag.FlagSet {
	fs := flag.NewFlagSet("sermons "+name, flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { fmt.Fprint(stderr, usage) }
	return fs
}

func runMigrate(ctx context.Context, args []string, stdout, stderr io.Writer, getenv func(string) string) int {
	fs := newFlags("migrate", stderr)
	if err := fs.Parse(args); err != nil || fs.NArg() != 0 {
		return 2
	}
	db, err := store.Open(ctx, getenv("DATABASE_URL"))
	if err != nil {
		fmt.Fprintf(stderr, "sermons migrate: %v\n", err)
		return 1
	}
	defer db.Close()
	applied, err := migrate.Up(ctx, db)
	for _, name := range applied {
		fmt.Fprintf(stdout, "applied %s\n", name)
	}
	if err != nil {
		fmt.Fprintf(stderr, "sermons migrate: %v\n", err)
		return 1
	}
	if len(applied) == 0 {
		fmt.Fprintln(stdout, "nothing to apply: the schema is up to date")
	}
	return 0
}

func runCheck(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := newFlags("check", stderr)
	all := fs.Bool("all", false, "check every source directory under the root given")
	urls := fs.Int("urls", 0, "HEAD a random N of the urls and report any not 200 (uses the network)")
	if err := fs.Parse(args); err != nil || fs.NArg() != 1 {
		if err == nil {
			fmt.Fprint(stderr, usage)
		}
		return 2
	}
	if *urls < 0 {
		fmt.Fprintln(stderr, "sermons check: --urls must be 0 or more")
		return 2
	}
	dir := fs.Arg(0)

	var faults []sources.Fault
	var dirs []string
	if *all {
		dirs, faults = sources.CheckAll(dir)
	} else {
		dirs, faults = []string{dir}, sources.Check(dir)
	}
	if *urls > 0 && len(faults) == 0 {
		client := &http.Client{Timeout: 30 * time.Second}
		rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
		for _, d := range dirs {
			src, _ := sources.Read(d)
			faults = append(faults, sources.CheckURLs(ctx, src, *urls, client, rnd)...)
		}
	}
	for _, f := range faults {
		fmt.Fprintln(stdout, f.String())
	}
	if len(faults) > 0 {
		fmt.Fprintf(stderr, "%d fault(s)\n", len(faults))
		return 1
	}
	fmt.Fprintf(stdout, "ok: %d source(s)\n", len(dirs))
	return 0
}

func runLoad(ctx context.Context, args []string, stdout, stderr io.Writer, getenv func(string) string) int {
	fs := newFlags("load", stderr)
	all := fs.Bool("all", false, "load every source directory under the root given")
	if err := fs.Parse(args); err != nil || fs.NArg() != 1 {
		if err == nil {
			fmt.Fprint(stderr, usage)
		}
		return 2
	}
	dir := fs.Arg(0)

	// Check before connecting, so a fault is told even with no database.
	var faults []sources.Fault
	if *all {
		_, faults = sources.CheckAll(dir)
	} else {
		faults = sources.Check(dir)
	}
	if len(faults) > 0 {
		for _, f := range faults {
			fmt.Fprintln(stdout, f.String())
		}
		fmt.Fprintf(stderr, "sermons load: %d fault(s); nothing loaded\n", len(faults))
		return 1
	}

	db, err := store.Open(ctx, getenv("DATABASE_URL"))
	if err != nil {
		fmt.Fprintf(stderr, "sermons load: %v\n", err)
		return 1
	}
	defer db.Close()

	var counts []sources.Counts
	if *all {
		counts, err = sources.LoadAll(ctx, db, dir)
	} else {
		var c sources.Counts
		c, err = sources.Load(ctx, db, dir)
		if err == nil {
			counts = append(counts, c)
		}
	}
	for _, c := range counts {
		fmt.Fprintln(stdout, c.String())
	}
	if err == nil && *all && len(counts) == 0 {
		fmt.Fprintf(stdout, "no source directories under %s\n", dir)
	}
	if err != nil {
		var faults sources.Faults
		if errors.As(err, &faults) {
			fmt.Fprintln(stdout, faults.Error())
		}
		fmt.Fprintf(stderr, "sermons load: %v\n", err)
		return 1
	}
	return 0
}

// runServe answers the API until ctx ends (SIGINT or SIGTERM), then lets
// requests in flight finish for up to 10 seconds.
func runServe(ctx context.Context, args []string, stderr io.Writer, getenv func(string) string) int {
	fs := newFlags("serve", stderr)
	if err := fs.Parse(args); err != nil || fs.NArg() != 0 {
		return 2
	}
	url := getenv("DATABASE_URL")
	if url == "" {
		fmt.Fprintln(stderr, "sermons serve: no database URL: set DATABASE_URL (postgres://user:password@host:port/db)")
		return 1
	}
	addr := getenv("SERMONS_ADDR")
	if addr == "" {
		addr = defaultAddr
	}
	origins := getenv("SERMONS_ORIGINS")
	if strings.TrimSpace(origins) == "" {
		origins = api.DefaultOrigins
	}

	// The pool connects when asked, so serve starts with the database
	// down and answers 503 until it is up.
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		fmt.Fprintf(stderr, "sermons serve: DATABASE_URL: %v\n", err)
		return 1
	}
	db, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		fmt.Fprintf(stderr, "sermons serve: %v\n", err)
		return 1
	}
	defer db.Close()
	pctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	pending, err := migrate.Pending(pctx, db)
	cancel()
	switch {
	case err != nil:
		fmt.Fprintf(stderr, "sermons serve: warning: the database at %s:%d is not answering (%v); /v1/health answers 503 until it does\n",
			cfg.ConnConfig.Host, cfg.ConnConfig.Port, err)
	case len(pending) > 0:
		fmt.Fprintf(stderr, "sermons serve: the database lacks migrations %s: run sermons migrate first\n", strings.Join(pending, ", "))
		return 1
	}

	l, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Fprintf(stderr, "sermons serve: %v\n", err)
		return 1
	}
	srv := &http.Server{
		Handler: api.New(api.Config{
			DB:      db,
			Version: version,
			Origins: api.ParseOrigins(origins),
			Log:     stderr,
		}),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       2 * time.Minute,
		MaxHeaderBytes:    16 << 10,
	}
	fmt.Fprintf(stderr, "sermons serve: listening on %s (version %s)\n", l.Addr(), version)
	errc := make(chan error, 1)
	go func() { errc <- srv.Serve(l) }()
	select {
	case err := <-errc:
		fmt.Fprintf(stderr, "sermons serve: %v\n", err)
		return 1
	case <-ctx.Done():
	}
	sctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(sctx); err != nil {
		fmt.Fprintf(stderr, "sermons serve: shutting down: %v\n", err)
		return 1
	}
	fmt.Fprintln(stderr, "sermons serve: stopped")
	return 0
}
