package main

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
	if code, _, _ := runCmd(t, nil, "nonsense"); code != 2 {
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
	if code != 0 || out != "applied 0001_init\napplied 0002_search\napplied 0003_theme_verses\n" {
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
	// --all loads no directory beginning with '_': a root holding only the
	// fixtures and the template (a copy, so that the real sources/ may grow)
	// has nothing to load.
	root := t.TempDir()
	for _, name := range []string{"_fixtures", "_template"} {
		if err := os.CopyFS(filepath.Join(root, name), os.DirFS(filepath.Join("../../sources", name))); err != nil {
			t.Fatal(err)
		}
	}
	code, out, errOut = runCmd(t, env, "load", "--all", root)
	if code != 0 || out != "no source directories under "+root+"\n" {
		t.Errorf("load --all: %d %q %q", code, out, errOut)
	}
}

func TestServe(t *testing.T) {
	db := store.TestDB(t)
	env := map[string]string{"DATABASE_URL": db.Config().ConnString()}
	if code, _, errOut := runCmd(t, env, "migrate"); code != 0 {
		t.Fatalf("migrate: %s", errOut)
	}
	if code, _, errOut := runCmd(t, env, "load", "../../sources/_fixtures/spurgeon-20"); code != 0 {
		t.Fatalf("load: %s", errOut)
	}
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().String()
	l.Close()
	env["SERMONS_ADDR"] = addr

	ctx, cancel := context.WithCancel(context.Background())
	var out, errOut bytes.Buffer
	done := make(chan int)
	go func() {
		done <- run(ctx, []string{"serve"}, &out, &errOut, func(k string) string { return env[k] })
	}()
	var res *http.Response
	for i := 0; i < 100; i++ {
		res, err = http.Get("http://" + addr + "/v1/health")
		if err == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if err != nil {
		cancel()
		code := <-done
		t.Fatalf("serve never answered: %v (exit %d)\n%s", err, code, errOut.String())
	}
	b, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != 200 || !strings.Contains(string(b), `"sermons": 20`) || !strings.Contains(string(b), `"version": "dev"`) {
		t.Errorf("health: %d %s", res.StatusCode, b)
	}
	res, err = http.Get("http://" + addr + "/v1/sources")
	if err != nil || res.Header.Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("sources: %v", err)
	}
	cancel()
	if code := <-done; code != 0 {
		t.Errorf("serve exited %d: %s", code, errOut.String())
	}
	if !strings.Contains(errOut.String(), "sermons serve: listening on "+addr) ||
		!strings.Contains(errOut.String(), "GET /v1/health 200 ") {
		t.Errorf("stderr: %q", errOut.String())
	}
}

func TestServeRefusesAnUnmigratedDatabase(t *testing.T) {
	db := store.TestDB(t)
	env := map[string]string{"DATABASE_URL": db.Config().ConnString(), "SERMONS_ADDR": "127.0.0.1:0"}
	code, _, errOut := runCmd(t, env, "serve")
	if code != 1 || !strings.Contains(errOut, "sermons migrate") {
		t.Errorf("serve on an empty database: %d %q", code, errOut)
	}
	if code, _, errOut := runCmd(t, map[string]string{}, "serve"); code != 1 || !strings.Contains(errOut, "DATABASE_URL") {
		t.Errorf("serve with no DATABASE_URL: %d %q", code, errOut)
	}
}

func TestCheckPrintsWarningsAndStillExits0(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	os.Mkdir(dir, 0o755)
	b, _ := os.ReadFile("../../sources/_template/source.toml")
	os.WriteFile(filepath.Join(dir, "source.toml"), []byte(strings.Replace(string(b), `id = "example-preacher"`, `id = "demo"`, 1)), 0o644)
	os.WriteFile(filepath.Join(dir, "rows.jsonl"), []byte(
		`{"id": "1", "title": "One", "ref": "Acts.1.1-Jude.1.25", "url": "https://example.org/1"}`+"\n"), 0o644)
	code, out, errOut := runCmd(t, nil, "check", dir)
	if code != 0 || !strings.HasPrefix(out, "rows.jsonl:1: warning: ref 'Acts.1.1-Jude.1.25' spans 149 chapters: ") ||
		!strings.HasSuffix(out, "ok: 1 source(s), 1 warning(s)\n") {
		t.Errorf("check: %d %q %q", code, out, errOut)
	}
	// --all says the same, naming the source's directory.
	code, out, _ = runCmd(t, nil, "check", "--all", filepath.Dir(dir))
	if code != 0 || !strings.Contains(out, "demo/rows.jsonl:1: warning: ") {
		t.Errorf("check --all: %d %q", code, out)
	}
}

func TestProposePrintsTheStory(t *testing.T) {
	code, out, errOut := runCmd(t, nil, "propose", "../../sources/_fixtures/spurgeon-20")
	if code != 0 || !strings.Contains(out, "20 rows") || !strings.Contains(out, "Size: S") || errOut != "" {
		t.Errorf("propose: %d %q %q", code, out, errOut)
	}
	if code, _, _ := runCmd(t, nil, "propose"); code != 2 {
		t.Errorf("propose with no directory: %d", code)
	}
	dir := filepath.Join(t.TempDir(), "demo")
	os.Mkdir(dir, 0o755)
	code, out, errOut = runCmd(t, nil, "propose", dir)
	if code != 1 || !strings.Contains(out, "source.toml: missing") || !strings.Contains(errOut, "fault(s)") {
		t.Errorf("propose of a broken dir: %d %q %q", code, out, errOut)
	}
}
