package tests

// Tests of the Brookfield First refresh: scripts/refresh-brookfield.sh,
// scripts/check-brookfield-stale.sh and the four units in deploy/laptop that
// run them. The refresh runs in a temporary clone with a bare repository as
// its origin, with `go` and `mw` replaced by fakes: nothing here reaches
// YouTube, the vault or the real home.

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

const fakeKey = "fake-key-for-test-7731"

// fakeGoRefresh answers "go run ./tools/brookfield-youtube" the way the tool
// does: it writes the three data files from FAKE_ROWS, or (FAKE_FETCH=fail)
// writes nothing and exits 1 as the tool does when the API refuses, or
// (FAKE_FETCH=partial) leaves a half-written file behind and exits 1. It logs
// that the key reached its environment, never the key.
const fakeGoRefresh = `#!/bin/sh
echo "go $*" >> "$FAKE_LOG"
[ "$1" = run ] && [ "$2" = ./tools/brookfield-youtube ] || exit 2
[ -n "$YOUTUBE_API_KEY" ] && echo "key-in-environment" >> "$FAKE_LOG"
case "$FAKE_FETCH" in
fail) echo "brookfield-youtube: playlistItems.list: 403 quota exceeded" >&2; exit 1 ;;
partial) echo garbage > sources/brookfield-first/rows.jsonl; echo "brookfield-youtube: disk full" >&2; exit 1 ;;
esac
printf '%s\n' "$FAKE_ROWS" > sources/brookfield-first/rows.jsonl
printf '{"number":1}\n' > data/morning-rephil.jsonl
echo "wrote $FAKE_ROWS"
`

// fakeMw answers "mw secrets get youtube_api_key" with the key, or (FAKE_MW=down)
// fails as the vault does when it cannot be read.
const fakeMw = `#!/bin/sh
echo "mw $*" >> "$FAKE_LOG"
[ "$1 $2 $3" = "secrets get youtube_api_key" ] || exit 2
[ "$FAKE_MW" = down ] && { echo "mw: the vault is locked" >&2; exit 1; }
echo "fake-key-for-test-7731"
`

type refreshRig struct {
	t      *testing.T
	script string // the repo's scripts directory
	clone  string
	origin string
	home   string
	log    string
	env    []string
}

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null",
		"GIT_AUTHOR_NAME=T", "GIT_AUTHOR_EMAIL=t@example.com", "GIT_COMMITTER_NAME=T", "GIT_COMMITTER_EMAIL=t@example.com")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func newRefreshRig(t *testing.T) *refreshRig {
	t.Helper()
	tmp := t.TempDir()
	r := &refreshRig{t: t, script: repoRoot(t), clone: filepath.Join(tmp, "clone"), origin: filepath.Join(tmp, "origin.git"),
		home: filepath.Join(tmp, "home"), log: filepath.Join(tmp, "calls.log")}
	stubs := filepath.Join(tmp, "stubs")
	for _, d := range []string{stubs, r.home, r.clone} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for name, body := range map[string]string{"go": fakeGoRefresh, "mw": fakeMw} {
		if err := os.WriteFile(filepath.Join(stubs, name), []byte(body), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	git(t, tmp, "init", "--bare", "-b", "main", r.origin)
	git(t, r.clone, "init", "-b", "main")
	git(t, r.clone, "remote", "add", "origin", r.origin)
	for rel, body := range map[string]string{
		"sources/brookfield-first/rows.jsonl":  "old row\n",
		"sources/brookfield-first/source.toml": "id = 'x'\n",
		"data/morning-rephil.jsonl":            "{\"number\":1}\n",
		"README.md":                            "readme\n",
	} {
		p := filepath.Join(r.clone, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	git(t, r.clone, "add", ".")
	git(t, r.clone, "commit", "-m", "first")
	git(t, r.clone, "push", "-u", "origin", "main")
	r.env = []string{
		"PATH=" + stubs + ":" + os.Getenv("PATH"),
		"SERMON_INDEX_HOME=" + r.home,
		"REFRESH_ROOT=" + r.clone,
		"FAKE_LOG=" + r.log,
		"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null",
		"GIT_AUTHOR_NAME=T", "GIT_AUTHOR_EMAIL=t@example.com", "GIT_COMMITTER_NAME=T", "GIT_COMMITTER_EMAIL=t@example.com",
	}
	return r
}

func (r *refreshRig) run(script string, extra ...string) (string, int) {
	r.t.Helper()
	cmd := exec.Command("sh", filepath.Join(r.script, script))
	cmd.Dir = r.script
	cmd.Env = append(append([]string{}, r.env...), extra...)
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	code := 0
	if err := cmd.Run(); err != nil {
		ee, ok := err.(*exec.ExitError)
		if !ok {
			r.t.Fatalf("%s: %v", script, err)
		}
		code = ee.ExitCode()
	}
	return out.String(), code
}

func (r *refreshRig) stamp() string {
	return filepath.Join(r.home, ".local/state/sermon-index/brookfield-refreshed")
}

func (r *refreshRig) writeStamp(at time.Time) {
	r.t.Helper()
	if err := os.MkdirAll(filepath.Dir(r.stamp()), 0o755); err != nil {
		r.t.Fatal(err)
	}
	if err := os.WriteFile(r.stamp(), []byte(strconv.FormatInt(at.Unix(), 10)+"\n"), 0o644); err != nil {
		r.t.Fatal(err)
	}
}

func (r *refreshRig) commits() string { return git(r.t, r.clone, "rev-list", "--count", "HEAD") }

func (r *refreshRig) file(rel string) string {
	b, err := os.ReadFile(filepath.Join(r.clone, rel))
	if err != nil {
		r.t.Fatal(err)
	}
	return string(b)
}

func TestRefreshCommitsAndPushesWhenTheOutputChanged(t *testing.T) {
	r := newRefreshRig(t)
	out, code := r.run("scripts/refresh-brookfield.sh", "FAKE_FETCH=ok", "FAKE_ROWS=new row")
	if code != 0 {
		t.Fatalf("exit %d:\n%s", code, out)
	}
	if got := r.commits(); got != "2" {
		t.Errorf("commits = %s, want 2 (a commit for the changed output)", got)
	}
	if got := r.file("sources/brookfield-first/rows.jsonl"); got != "new row\n" {
		t.Errorf("rows.jsonl = %q", got)
	}
	if st := git(t, r.clone, "status", "--porcelain"); st != "" {
		t.Errorf("the tree is not clean after the commit:\n%s", st)
	}
	if head, remote := git(t, r.clone, "rev-parse", "HEAD"), git(t, r.origin, "rev-parse", "main"); head != remote {
		t.Errorf("the commit was not pushed: HEAD %s, origin %s", head, remote)
	}
	msg := git(t, r.clone, "log", "-1", "--format=%B")
	if strings.Contains(strings.ToLower(msg), "co-authored") || strings.Contains(strings.ToLower(msg), "generated") {
		t.Errorf("the commit message carries an attribution line:\n%s", msg)
	}
	// Only the data files were committed.
	if files := git(t, r.clone, "show", "--name-only", "--format=", "HEAD"); strings.Contains(files, "README.md") {
		t.Errorf("the commit holds more than the data files:\n%s", files)
	}
	b, err := os.ReadFile(r.stamp())
	if err != nil {
		t.Fatalf("no stamp after a good run: %v", err)
	}
	at, err := strconv.ParseInt(strings.TrimSpace(string(b)), 10, 64)
	if err != nil || time.Since(time.Unix(at, 0)) > time.Minute {
		t.Errorf("stamp %q is not a recent epoch (%v)", b, err)
	}
	// The key reached the one go run, and is printed nowhere.
	logb, _ := os.ReadFile(r.log)
	if !strings.Contains(string(logb), "key-in-environment") {
		t.Errorf("the key did not reach go run's environment:\n%s", logb)
	}
	if strings.Contains(out, fakeKey) || strings.Contains(string(logb), fakeKey) {
		t.Error("the key was printed or logged")
	}
	if st := git(t, r.clone, "log", "--all", "-p"); strings.Contains(st, fakeKey) {
		t.Error("the key is in the repository")
	}
}

func TestRefreshWithTheSameBytesMakesNoCommit(t *testing.T) {
	r := newRefreshRig(t)
	out, code := r.run("scripts/refresh-brookfield.sh", "FAKE_FETCH=ok", "FAKE_ROWS=old row")
	if code != 0 {
		t.Fatalf("exit %d:\n%s", code, out)
	}
	if got := r.commits(); got != "1" {
		t.Errorf("commits = %s, want 1 (nothing changed, nothing to commit)", got)
	}
	if !strings.Contains(out, "unchanged") {
		t.Errorf("the run does not say the list was unchanged:\n%s", out)
	}
	// A refresh that found nothing new is still a good refresh: the 30 days
	// start again, so the stamp is written.
	if _, err := os.Stat(r.stamp()); err != nil {
		t.Errorf("no stamp after an unchanged run: %v", err)
	}
}

func TestRefreshWhenTheFetchFailsLeavesTheFilesAndFails(t *testing.T) {
	for _, mode := range []string{"fail", "partial"} {
		t.Run(mode, func(t *testing.T) {
			r := newRefreshRig(t)
			old := time.Now().Add(-10 * 24 * time.Hour)
			r.writeStamp(old)
			before := r.file("sources/brookfield-first/rows.jsonl")
			out, code := r.run("scripts/refresh-brookfield.sh", "FAKE_FETCH="+mode, "FAKE_ROWS=new row")
			if code == 0 {
				t.Fatalf("exit 0 though the fetch failed:\n%s", out)
			}
			if !strings.Contains(out, "brookfield-youtube") {
				t.Errorf("the failure does not say what failed:\n%s", out)
			}
			if got := r.file("sources/brookfield-first/rows.jsonl"); got != before {
				t.Errorf("rows.jsonl changed to %q", got)
			}
			if st := git(t, r.clone, "status", "--porcelain"); st != "" {
				t.Errorf("the tree is not clean after the failure:\n%s", st)
			}
			if got := r.commits(); got != "1" {
				t.Errorf("commits = %s, want 1", got)
			}
			b, _ := os.ReadFile(r.stamp())
			if strings.TrimSpace(string(b)) != strconv.FormatInt(old.Unix(), 10) {
				t.Errorf("a failed run moved the stamp to %q", b)
			}
		})
	}
}

func TestRefreshWhenTheKeyCannotBeReadDoesNotFetch(t *testing.T) {
	r := newRefreshRig(t)
	out, code := r.run("scripts/refresh-brookfield.sh", "FAKE_MW=down", "FAKE_ROWS=new row")
	if code == 0 {
		t.Fatalf("exit 0 though the key could not be read:\n%s", out)
	}
	logb, _ := os.ReadFile(r.log)
	if strings.Contains(string(logb), "go run") {
		t.Error("go run was started without a key")
	}
	if _, err := os.Stat(r.stamp()); err == nil {
		t.Error("a failed run wrote the stamp")
	}
}

func TestRefreshRefusesAnEditedDataFile(t *testing.T) {
	r := newRefreshRig(t)
	if err := os.WriteFile(filepath.Join(r.clone, "sources/brookfield-first/rows.jsonl"), []byte("someone's edit\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, code := r.run("scripts/refresh-brookfield.sh", "FAKE_ROWS=new row")
	if code == 0 {
		t.Fatalf("exit 0 over an edited data file:\n%s", out)
	}
	if got := r.file("sources/brookfield-first/rows.jsonl"); got != "someone's edit\n" {
		t.Errorf("the edit was overwritten: %q", got)
	}
}

func TestRefreshPushesAnEarlierCommitThatNeverLeft(t *testing.T) {
	r := newRefreshRig(t)
	// A first run whose push is refused commits and fails; the next run,
	// finding the same list, pushes what is waiting and succeeds.
	hook := filepath.Join(r.origin, "hooks/pre-receive")
	if err := os.MkdirAll(filepath.Dir(hook), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(hook, []byte("#!/bin/sh\necho refused >&2\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	out, code := r.run("scripts/refresh-brookfield.sh", "FAKE_ROWS=new row")
	if code == 0 {
		t.Fatalf("exit 0 though the push was refused:\n%s", out)
	}
	if _, err := os.Stat(r.stamp()); err == nil {
		t.Error("the stamp was written though the push failed")
	}
	_ = os.Remove(hook)
	out, code = r.run("scripts/refresh-brookfield.sh", "FAKE_ROWS=new row")
	if code != 0 {
		t.Fatalf("second run exit %d:\n%s", code, out)
	}
	if head, remote := git(t, r.clone, "rev-parse", "HEAD"), git(t, r.origin, "rev-parse", "main"); head != remote {
		t.Errorf("the waiting commit was not pushed: HEAD %s, origin %s", head, remote)
	}
	if got := r.commits(); got != "2" {
		t.Errorf("commits = %s, want 2", got)
	}
}

func TestStaleCheck(t *testing.T) {
	r := newRefreshRig(t)
	day := 24 * time.Hour
	// Missing: non-zero, and it says where the stamp should be.
	out, code := r.run("scripts/check-brookfield-stale.sh")
	if code == 0 || !strings.Contains(out, "brookfield-refreshed") {
		t.Errorf("missing stamp: exit %d, output %q", code, out)
	}
	// 26 days: still inside the 27.
	r.writeStamp(time.Now().Add(-26 * day))
	if out, code := r.run("scripts/check-brookfield-stale.sh"); code != 0 {
		t.Errorf("26 days old: exit %d:\n%s", code, out)
	}
	// 28 days: stale, and the message says so in days.
	r.writeStamp(time.Now().Add(-28 * day))
	out, code = r.run("scripts/check-brookfield-stale.sh")
	if code == 0 || !strings.Contains(out, "28 days") {
		t.Errorf("28 days old: exit %d, output %q", code, out)
	}
	// Just refreshed.
	r.writeStamp(time.Now())
	if out, code := r.run("scripts/check-brookfield-stale.sh"); code != 0 {
		t.Errorf("fresh: exit %d:\n%s", code, out)
	}
	// A stamp that is not a number is as good as none.
	if err := os.WriteFile(r.stamp(), []byte("yesterday\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, code := r.run("scripts/check-brookfield-stale.sh"); code == 0 {
		t.Errorf("garbage stamp: exit 0:\n%s", out)
	}
}

func TestRefreshUnits(t *testing.T) {
	root := repoRoot(t)
	read := func(name string) string {
		b, err := os.ReadFile(filepath.Join(root, "deploy/laptop", name))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	want := map[string][]string{
		"sermon-index-brookfield-refresh.service": {
			"Description=", "OnFailure=mw-unit-failed@%n.service", "Type=oneshot",
			"ExecStart=/bin/sh %h/sermon-index/scripts/refresh-brookfield.sh", "TimeoutStartSec=",
			"EnvironmentFile=-%h/.config/mw/dispatch.env",
		},
		"sermon-index-brookfield-stale.service": {
			"Description=", "OnFailure=mw-unit-failed@%n.service", "Type=oneshot",
			"ExecStart=/bin/sh %h/sermon-index/scripts/check-brookfield-stale.sh", "TimeoutStartSec=",
		},
		"sermon-index-brookfield-refresh.timer": {
			"Description=", "OnCalendar=*-*-01,26 04:00:00 UTC", "Persistent=true", "WantedBy=timers.target",
		},
		"sermon-index-brookfield-stale.timer": {
			"Description=", "OnCalendar=*-*-* 13:00:00 UTC", "Persistent=true", "WantedBy=timers.target",
		},
	}
	for name, lines := range want {
		body := read(name)
		for _, l := range lines {
			if !strings.Contains(body, l) {
				t.Errorf("%s lacks %q", name, l)
			}
		}
		if strings.Contains(body, "AI"+"za") || strings.Contains(body, "YOUTUBE_API_KEY=") {
			t.Errorf("%s holds a key", name)
		}
	}
	// A timer's unit is named by the timer's own name: the .timer needs no
	// Unit= line, and must not name another.
	for _, name := range []string{"sermon-index-brookfield-refresh.timer", "sermon-index-brookfield-stale.timer"} {
		if strings.Contains(read(name), "Unit=") {
			t.Errorf("%s names a Unit=", name)
		}
	}
}

func TestInstallEnablesTheRefreshTimersAndUninstallRemovesThem(t *testing.T) {
	r := newRig(t)
	r.install()
	units := []string{"sermon-index-brookfield-refresh.service", "sermon-index-brookfield-refresh.timer",
		"sermon-index-brookfield-stale.service", "sermon-index-brookfield-stale.timer"}
	for _, u := range units {
		want, err := os.ReadFile(filepath.Join(r.root, "deploy/laptop", u))
		if err != nil {
			t.Fatal(err)
		}
		if got := r.read(".config/systemd/user/" + u); got != string(want) {
			t.Errorf("installed %s differs from deploy/laptop/%s", u, u)
		}
	}
	for _, timer := range []string{"sermon-index-brookfield-refresh.timer", "sermon-index-brookfield-stale.timer"} {
		if !strings.Contains(r.calls(), "systemctl --user enable --now "+timer) {
			t.Errorf("%s was not enabled and started:\n%s", timer, r.calls())
		}
	}
	// The services are run by their timers, never enabled themselves.
	for _, svc := range []string{"sermon-index-brookfield-refresh.service", "sermon-index-brookfield-stale.service"} {
		if strings.Contains(r.calls(), "enable --now "+svc) {
			t.Errorf("%s was enabled", svc)
		}
	}
	// A second install enables nothing again.
	r.clearCalls()
	second := r.install()
	if !strings.Contains(second, "nothing changed") {
		t.Errorf("the second install did not say nothing changed:\n%s", second)
	}
	if strings.Contains(r.calls(), "enable --now sermon-index-brookfield") {
		t.Errorf("the second run enabled a timer again:\n%s", r.calls())
	}
	// The install never fetches or refreshes anything.
	if strings.Contains(r.calls(), "brookfield-youtube") || strings.Contains(r.calls(), "secrets") {
		t.Errorf("the install ran the refresh:\n%s", r.calls())
	}

	r.clearCalls()
	if out, code := r.run("scripts/uninstall-laptop.sh"); code != 0 {
		t.Fatalf("uninstall exited %d:\n%s", code, out)
	}
	for _, u := range units {
		if _, err := os.Stat(r.path(".config/systemd/user/" + u)); !os.IsNotExist(err) {
			t.Errorf("%s is still there", u)
		}
	}
	for _, timer := range []string{"sermon-index-brookfield-refresh.timer", "sermon-index-brookfield-stale.timer"} {
		if !strings.Contains(r.calls(), "systemctl --user disable --now "+timer) {
			t.Errorf("uninstall did not disable %s:\n%s", timer, r.calls())
		}
	}
}
