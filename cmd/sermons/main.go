// Command sermons is the sermon index's one binary.
//
//	sermons migrate                    apply the schema's migrations to DATABASE_URL
//	sermons check [--urls N] <dir>     check a source directory; exit 1 on any fault
//	sermons check --all <root>         check every source directory under root
//	sermons load <dir>                 load a source directory into DATABASE_URL
//	sermons load --all <root>          load every source directory under root
//
// DATABASE_URL is a Postgres URL (postgres://user:password@host:port/db).
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
	"net/http"
	"os"
	"time"

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
`

func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Stdout, os.Stderr, os.Getenv))
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
