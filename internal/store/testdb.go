package store

// The tests' database: an embedded Postgres (fergusstrange/embedded-postgres)
// started once per test binary on a free port, with a fresh database for
// each test. No Docker and no installed Postgres are needed. The Postgres
// binaries are downloaded from Maven Central on the first run and cached
// under the user's cache directory (~/.cache/sermon-index/embedded-postgres
// on Linux); later runs work offline.
//
// The tests never read DATABASE_URL, so a set DATABASE_URL naming a live
// server is never touched by `go test`.

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestPostgresVersion is the Postgres the tests run, the major version
// production runs.
const TestPostgresVersion = embeddedpostgres.V17

type server struct {
	once sync.Once
	err  error
	pg   *embeddedpostgres.EmbeddedPostgres
	port uint32
	root string // the server's runtime and data directories are under it
	log  bytes.Buffer
	n    atomic.Int64 // databases made so far
}

var testServer server

// RunTests runs a package's tests and stops the embedded server, if one was
// started, when they end. Use it from TestMain:
//
//	func TestMain(m *testing.M) { os.Exit(store.RunTests(m)) }
func RunTests(m *testing.M) int {
	code := m.Run()
	if testServer.pg != nil {
		if err := testServer.pg.Stop(); err != nil {
			fmt.Fprintf(os.Stderr, "stopping the test Postgres: %v\n", err)
		}
	}
	if testServer.root != "" {
		os.RemoveAll(testServer.root)
	}
	return code
}

// TestDB gives the test a fresh, empty database on the embedded server and
// closes its pool when the test ends. A package that uses it must run its
// tests through RunTests.
func TestDB(t testing.TB) *pgxpool.Pool {
	t.Helper()
	testServer.once.Do(func() { testServer.err = testServer.start() })
	if testServer.err != nil {
		t.Fatalf("starting the test Postgres: %v\n%s", testServer.err, testServer.log.String())
	}
	ctx := context.Background()
	name := fmt.Sprintf("t%d", testServer.n.Add(1))

	admin, err := pgxpool.New(ctx, testServer.url("postgres"))
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	if _, err := admin.Exec(ctx, "create database "+name); err != nil {
		t.Fatalf("create database %s: %v", name, err)
	}

	db, err := pgxpool.New(ctx, testServer.url(name))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(db.Close)
	return db
}

func (s *server) url(db string) string {
	return fmt.Sprintf("postgres://postgres:postgres@127.0.0.1:%d/%s?sslmode=disable", s.port, db)
}

func (s *server) start() error {
	cache, err := cacheDir()
	if err != nil {
		return err
	}
	root, err := os.MkdirTemp("", "sermon-index-pg-")
	if err != nil {
		return err
	}
	s.root = root
	port, err := freePort()
	if err != nil {
		return err
	}
	s.port = port

	// Two test binaries run at once (go test ./... runs packages in
	// parallel); a lock on the cache stops them downloading the same file
	// into it together.
	unlock, err := lockFile(filepath.Join(cache, ".lock"))
	if err != nil {
		return err
	}
	defer unlock()

	s.pg = embeddedpostgres.NewDatabase(embeddedpostgres.DefaultConfig().
		Version(TestPostgresVersion).
		Port(port).
		CachePath(cache).
		RuntimePath(filepath.Join(root, "runtime")).
		DataPath(filepath.Join(root, "data")).
		StartTimeout(60 * time.Second).
		Logger(&s.log))
	if err := s.pg.Start(); err != nil {
		s.pg = nil
		return err
	}
	return nil
}

// cacheDir is where the Postgres binaries are kept between runs.
func cacheDir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("finding the cache directory: %w", err)
	}
	dir := filepath.Join(base, "sermon-index", "embedded-postgres")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

// freePort asks the kernel for a free TCP port on the loopback.
func freePort() (uint32, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return uint32(l.Addr().(*net.TCPAddr).Port), nil
}

// lockFile takes an exclusive lock on path, waiting for it.
func lockFile(path string) (func(), error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		f.Close()
		return nil, fmt.Errorf("locking %s: %w", path, err)
	}
	return func() {
		syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		f.Close()
	}, nil
}
