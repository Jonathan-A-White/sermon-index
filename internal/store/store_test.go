package store

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestMain(m *testing.M) { os.Exit(RunTests(m)) }

// TestTestDBIgnoresDatabaseURL: the tests never touch the database
// DATABASE_URL names, even when it is set: they start their own server.
func TestTestDBIgnoresDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://live:secret@203.0.113.7:5432/sermons")
	db := TestDB(t)
	ctx := context.Background()

	var port int
	var dataDir string
	if err := db.QueryRow(ctx, "select current_setting('port')::int, current_setting('data_directory')").Scan(&port, &dataDir); err != nil {
		t.Fatal(err)
	}
	if uint32(port) != testServer.port {
		t.Errorf("connected to port %d, want the embedded server's %d", port, testServer.port)
	}
	if !strings.HasPrefix(dataDir, testServer.root) {
		t.Errorf("data directory %s is not under the embedded server's %s", dataDir, testServer.root)
	}
	if strings.Contains(db.Config().ConnString(), "203.0.113.7") {
		t.Errorf("the test pool was made from DATABASE_URL: %s", db.Config().ConnString())
	}
}

// TestTestDBGivesEachTestItsOwnDatabase: two tests never see each other's
// tables.
func TestTestDBGivesEachTestItsOwnDatabase(t *testing.T) {
	ctx := context.Background()
	a, b := TestDB(t), TestDB(t)
	if _, err := a.Exec(ctx, "create table only_in_a (x int)"); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := b.QueryRow(ctx, "select count(*) from pg_tables where tablename = 'only_in_a'").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("database b sees a's table")
	}
}

func TestOpenNeedsAURL(t *testing.T) {
	if _, err := Open(context.Background(), ""); err == nil {
		t.Error("Open(\"\") gave no error")
	}
}
