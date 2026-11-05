package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Jonathan-A-White/sermon-index/internal/store"
)

func TestMain(m *testing.M) { os.Exit(store.RunTests(m)) }

func runCmd(t *testing.T, env map[string]string, args ...string) (int, string, string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code := run(context.Background(), args, &out, &errOut, func(k string) string { return env[k] })
	return code, out.String(), errOut.String()
}

func TestCheckTheFixture(t *testing.T) {
	code, out, _ := runCmd(t, nil, "check", "../../sources/_fixtures/spurgeon-20")
	if code != 0 || out != "ok: 1 source(s)\n" {
		t.Errorf("check fixture: %d %q", code, out)
	}
}

func TestCheckABadRowExits1WithTheLine(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	os.Mkdir(dir, 0o755)
	b, _ := os.ReadFile("../../sources/_template/source.toml")
	os.WriteFile(filepath.Join(dir, "source.toml"), []byte(strings.Replace(string(b), `id = "example-preacher"`, `id = "demo"`, 1)), 0o644)
	os.WriteFile(filepath.Join(dir, "rows.jsonl"), []byte(
		`{"id": "1", "title": "One", "ref": "Rom.5.8", "url": "https://example.org/1"}`+"\n"+
			`{"id": "2", "title": "Two", "ref": "Rom.5.99", "url": "https://example.org/2"}`+"\n"), 0o644)
	code, out, _ := runCmd(t, nil, "check", dir)
	if code != 1 || out != "rows.jsonl:2: ref 'Rom.5.99': verse past the chapter\n" {
		t.Errorf("check: %d %q", code, out)
	}
	// load refuses it before it needs a database.
	code, out, _ = runCmd(t, nil, "load", dir)
	if code != 1 || !strings.Contains(out, "rows.jsonl:2:") {
		t.Errorf("load: %d %q", code, out)
	}
}

func TestUsage(t *testing.T) {
	if code, _, _ := runCmd(t, nil); code != 2 {
		t.Errorf("no args: %d", code)
	}
	if code, _, _ := runCmd(t, nil, "serve"); code != 2 {
		t.Errorf("unknown command: %d", code)
	}
	if code, _, errOut := runCmd(t, nil, "migrate"); code != 1 || !strings.Contains(errOut, "DATABASE_URL") {
		t.Errorf("migrate with no DATABASE_URL: %d %q", code, errOut)
	}
}

func TestMigrateAndLoadTwice(t *testing.T) {
	db := store.TestDB(t)
	env := map[string]string{"DATABASE_URL": db.Config().ConnString()}

	code, out, errOut := runCmd(t, env, "migrate")
	if code != 0 || out != "applied 0001_init\n" {
		t.Fatalf("migrate: %d %q %q", code, out, errOut)
	}
	if code, out, _ := runCmd(t, env, "migrate"); code != 0 || !strings.Contains(out, "nothing to apply") {
		t.Errorf("second migrate: %d %q", code, out)
	}
	for i := 0; i < 2; i++ {
		code, out, errOut := runCmd(t, env, "load", "../../sources/_fixtures/spurgeon-20")
		if code != 0 || out != "spurgeon-20: 20 rows, 19 passages, 0 removed\n" {
			t.Errorf("load %d: %d %q %q", i+1, code, out, errOut)
		}
	}
	// --all over sources/ loads no directory beginning with '_'.
	code, out, errOut = runCmd(t, env, "load", "--all", "../../sources")
	if code != 0 || out != "no source directories under ../../sources\n" {
		t.Errorf("load --all: %d %q %q", code, out, errOut)
	}
}
