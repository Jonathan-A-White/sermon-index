// Package tests holds the tests that are not about one Go package: here,
// scripts/install-laptop.sh, uninstall-laptop.sh and smoke.sh driven with
// SERMON_INDEX_HOME set to a temporary directory and docker, systemctl, go
// and curl replaced by fakes first on PATH. Nothing here starts a real
// service, container or build, and nothing reads the real home directory.
package tests

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// repoRoot is the directory holding scripts/ and deploy/.
func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Dir(wd)
}

const fakeDocker = `#!/bin/sh
echo "docker $*" >> "$FAKE_LOG"
case "$*" in
*pg_isready*) [ "$FAKE_PG" = down ] && exit 2; exit 0 ;;
esac
exit 0
`

// fakeSystemctl keeps what is enabled and active as files under
// $FAKE_STATE, so a second run can see the first's work.
const fakeSystemctl = `#!/bin/sh
echo "systemctl $*" >> "$FAKE_LOG"
[ "$1" = "--user" ] && shift
cmd=$1; shift
mkdir -p "$FAKE_STATE"
case "$cmd" in
enable)
	now=0; units=
	for a in "$@"; do case "$a" in --now) now=1 ;; *) units="$units $a" ;; esac; done
	for u in $units; do touch "$FAKE_STATE/enabled-$u"; [ $now = 1 ] && touch "$FAKE_STATE/active-$u"; done ;;
disable)
	units=
	for a in "$@"; do case "$a" in --now) ;; *) units="$units $a" ;; esac; done
	for u in $units; do rm -f "$FAKE_STATE/enabled-$u" "$FAKE_STATE/active-$u"; done ;;
is-enabled) [ -e "$FAKE_STATE/enabled-$1" ] || exit 1 ;;
is-active) [ -e "$FAKE_STATE/active-$1" ] || exit 3 ;;
restart|start) touch "$FAKE_STATE/active-$1" ;;
esac
exit 0
`

// fakeGo answers "go build -ldflags X -o OUT ./cmd/sermons" by writing a
// script at OUT that logs its arguments, the same bytes every time.
const fakeGo = `#!/bin/sh
echo "go $*" >> "$FAKE_LOG"
out=
while [ $# -gt 0 ]; do
	[ "$1" = "-o" ] && out=$2
	shift
done
[ -n "$out" ] || exit 2
cat > "$out" <<'BIN'
#!/bin/sh
echo "sermons $*" >> "$FAKE_LOG"
exit 0
BIN
chmod +x "$out"
`

// fakeCurl answers the health and sermons calls smoke.sh makes. The body
// comes first and the status last, as curl -w '\n%{http_code}' prints.
const fakeCurl = `#!/bin/sh
echo "curl $*" >> "$FAKE_LOG"
for last; do :; done
case "$FAKE_SMOKE:$last" in
bad:*/v1/health) printf '{"status":"ok","version":"x","sources":1,"sermons":3}\n200\n' ;;
down:*) printf '\n000\n'; exit 7 ;;
*:*/v1/health) printf '{\n  "status": "ok",\n  "version": "x",\n  "sources": 4,\n  "sermons": 3563\n}\n200\n' ;;
bad2:*/v1/sermons*) printf '{"ref":"Rom.5.8","total":1,"rows":[{"title":"Another Sermon"}]}\n200\n' ;;
*:*/v1/sermons*) printf '{\n  "ref": "Rom.5.8",\n  "total": 2,\n  "rows": [\n    {\n      "title": "Love'"'"'s Commendation"\n    },\n    {\n      "title": "Other"\n    }\n  ]\n}\n200\n' ;;
esac
exit 0
`

// rig is one temporary home and the fakes.
type rig struct {
	t     *testing.T
	root  string // the repo
	home  string // SERMON_INDEX_HOME
	stubs string
	log   string
	state string
	env   []string
}

func newRig(t *testing.T) *rig {
	t.Helper()
	tmp := t.TempDir()
	r := &rig{t: t, root: repoRoot(t), home: filepath.Join(tmp, "home"), stubs: filepath.Join(tmp, "stubs"),
		log: filepath.Join(tmp, "calls.log"), state: filepath.Join(tmp, "state")}
	for _, d := range []string{r.home, r.stubs} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for name, body := range map[string]string{"docker": fakeDocker, "systemctl": fakeSystemctl, "go": fakeGo, "curl": fakeCurl} {
		if err := os.WriteFile(filepath.Join(r.stubs, name), []byte(body), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	r.env = []string{
		"PATH=" + r.stubs + ":" + os.Getenv("PATH"),
		"SERMON_INDEX_HOME=" + r.home,
		"FAKE_LOG=" + r.log,
		"FAKE_STATE=" + r.state,
		"SMOKE_WAIT=0",
		"PG_WAIT=3",
	}
	return r
}

// run runs a script of the repo and returns its output (stdout and stderr
// together) and exit code.
func (r *rig) run(script string, extraEnv ...string) (string, int) {
	r.t.Helper()
	cmd := exec.Command("sh", filepath.Join(r.root, script))
	cmd.Dir = r.root
	cmd.Env = append(append([]string{}, r.env...), extraEnv...)
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	err := cmd.Run()
	code := 0
	if err != nil {
		ee, ok := err.(*exec.ExitError)
		if !ok {
			r.t.Fatalf("%s: %v", script, err)
		}
		code = ee.ExitCode()
	}
	return out.String(), code
}

func (r *rig) calls() string {
	b, _ := os.ReadFile(r.log)
	return string(b)
}

func (r *rig) clearCalls() { _ = os.Remove(r.log) }

func (r *rig) path(rel string) string { return filepath.Join(r.home, rel) }

func (r *rig) read(rel string) string {
	r.t.Helper()
	b, err := os.ReadFile(r.path(rel))
	if err != nil {
		r.t.Fatal(err)
	}
	return string(b)
}

func (r *rig) install() string {
	r.t.Helper()
	out, code := r.run("scripts/install-laptop.sh")
	if code != 0 {
		r.t.Fatalf("install-laptop.sh exited %d:\n%s\ncalls:\n%s", code, out, r.calls())
	}
	return out
}

func TestUnitsHaveWhatTheStoryNames(t *testing.T) {
	root := repoRoot(t)
	read := func(name string) string {
		b, err := os.ReadFile(filepath.Join(root, "deploy/laptop", name))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	pg := read("sermon-index-pg.service")
	for _, want := range []string{
		"docker run --rm --name sermon-index-pg",
		"-v %h/.local/state/sermon-index/pg:/var/lib/postgresql/data",
		"-p 127.0.0.1:5433:5432",
		"--env-file %h/.config/sermon-index/pg.env",
		"postgres:17",
		"ExecStop=/usr/bin/docker stop sermon-index-pg",
		"Restart=on-failure",
	} {
		if !strings.Contains(pg, want) {
			t.Errorf("sermon-index-pg.service lacks %q", want)
		}
	}
	svc := read("sermon-index.service")
	for _, want := range []string{
		"ExecStart=%h/.local/bin/sermons serve",
		"EnvironmentFile=%h/.config/sermon-index/env",
		"After=sermon-index-pg.service",
		"Requires=sermon-index-pg.service",
		"Restart=on-failure",
		"WantedBy=default.target",
	} {
		if !strings.Contains(svc, want) {
			t.Errorf("sermon-index.service lacks %q", want)
		}
	}
	// deploy/laptop/README.md documents both files line by line: every
	// directive of both units appears in it.
	readme := read("README.md")
	for _, unit := range []string{pg, svc} {
		for _, line := range strings.Split(unit, "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "[") {
				continue
			}
			key := line[:strings.Index(line, "=")]
			if !strings.Contains(readme, "`"+key+"`") {
				t.Errorf("deploy/laptop/README.md does not document %s", key)
			}
		}
	}
}

func TestInstallWritesWhatItSays(t *testing.T) {
	r := newRig(t)
	out := r.install()

	// The binary, built with the version stamped in.
	if fi, err := os.Stat(r.path(".local/bin/sermons")); err != nil || fi.Mode()&0o100 == 0 {
		t.Fatalf("no executable ~/.local/bin/sermons: %v", err)
	}
	if c := r.calls(); !regexp.MustCompile(`go build -ldflags -X main\.version=\S+ -o \S+ \./cmd/sermons`).MatchString(c) {
		t.Errorf("go build was not called with the version in -ldflags:\n%s", c)
	}

	// The two env files, 0600, one password, never printed.
	pgEnv := r.read(".config/sermon-index/pg.env")
	env := r.read(".config/sermon-index/env")
	for _, f := range []string{"pg.env", "env"} {
		fi, err := os.Stat(r.path(".config/sermon-index/" + f))
		if err != nil || fi.Mode().Perm() != 0o600 {
			t.Errorf("%s mode = %v (%v), want 0600", f, fi.Mode().Perm(), err)
		}
	}
	m := regexp.MustCompile(`(?m)^POSTGRES_PASSWORD=([0-9a-f]{48})$`).FindStringSubmatch(pgEnv)
	if m == nil {
		t.Fatalf("pg.env has no 48-hex POSTGRES_PASSWORD:\n%s", pgEnv)
	}
	pw := m[1]
	for _, want := range []string{"POSTGRES_USER=sermons", "POSTGRES_DB=sermons"} {
		if !strings.Contains(pgEnv, want) {
			t.Errorf("pg.env lacks %s", want)
		}
	}
	for _, want := range []string{
		"DATABASE_URL=postgres://sermons:" + pw + "@127.0.0.1:5433/sermons",
		"SERMONS_ADDR=10.88.0.2:8790",
		"SERMONS_ORIGINS=https://lampas.allmymind.org,http://localhost:5173",
	} {
		if !strings.Contains(env, want) {
			t.Errorf("env lacks %s:\n%s", want, strings.ReplaceAll(env, pw, "<pw>"))
		}
	}
	if strings.Contains(out, pw) {
		t.Error("the install printed the password")
	}

	// The units, the data directory.
	for _, u := range []string{"sermon-index-pg.service", "sermon-index.service"} {
		want, _ := os.ReadFile(filepath.Join(r.root, "deploy/laptop", u))
		if got := r.read(".config/systemd/user/" + u); got != string(want) {
			t.Errorf("installed %s differs from deploy/laptop/%s", u, u)
		}
	}
	if fi, err := os.Stat(r.path(".local/state/sermon-index/pg")); err != nil || !fi.IsDir() {
		t.Errorf("data directory not made: %v", err)
	}

	// The steps, in order.
	c := r.calls()
	steps := []string{
		"systemctl --user daemon-reload",
		"systemctl --user enable --now sermon-index-pg.service",
		"docker exec sermon-index-pg pg_isready -h 127.0.0.1 -U sermons -d sermons",
		"sermons migrate",
		"sermons load --all " + filepath.Join(r.root, "sources"),
		"systemctl --user enable --now sermon-index.service",
		"curl ",
	}
	pos := 0
	for _, s := range steps {
		i := strings.Index(c[pos:], s)
		if i < 0 {
			t.Fatalf("step %q missing or out of order in:\n%s", s, c)
		}
		pos += i + len(s)
	}

	// The way back is printed, naming what is left.
	for _, want := range []string{"scripts/uninstall-laptop.sh", ".local/state/sermon-index/pg", ".config/sermon-index"} {
		if !strings.Contains(out, want) {
			t.Errorf("the closing text does not name %q:\n%s", want, out)
		}
	}
}

func TestSecondInstallChangesNothingAndSaysSo(t *testing.T) {
	r := newRig(t)
	first := r.install()
	if strings.Contains(first, "nothing changed") {
		t.Error("the first install said nothing changed")
	}
	files := []string{".local/bin/sermons", ".config/sermon-index/pg.env", ".config/sermon-index/env",
		".config/systemd/user/sermon-index-pg.service", ".config/systemd/user/sermon-index.service"}
	before := map[string]string{}
	for _, f := range files {
		before[f] = r.read(f)
	}
	r.clearCalls()
	second := r.install()
	if !strings.Contains(second, "nothing changed") {
		t.Errorf("the second install did not say nothing changed:\n%s", second)
	}
	for _, f := range files {
		if r.read(f) != before[f] {
			t.Errorf("%s changed on the second run", f)
		}
	}
	for _, bad := range []string{"enable", "restart", "daemon-reload"} {
		if strings.Contains(r.calls(), "systemctl --user "+bad) {
			t.Errorf("the second run called systemctl %s:\n%s", bad, r.calls())
		}
	}
	// migrate and load still run: both are idempotent, and the smoke test.
	for _, want := range []string{"sermons migrate", "sermons load --all", "curl "} {
		if !strings.Contains(r.calls(), want) {
			t.Errorf("the second run did not call %q", want)
		}
	}
}

func TestInstallKeepsAnExistingPassword(t *testing.T) {
	r := newRig(t)
	dir := r.path(".config/sermon-index")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	pgEnv := "POSTGRES_USER=sermons\nPOSTGRES_PASSWORD=feedfeed\nPOSTGRES_DB=sermons\n"
	if err := os.WriteFile(filepath.Join(dir, "pg.env"), []byte(pgEnv), 0o600); err != nil {
		t.Fatal(err)
	}
	r.install()
	if got := r.read(".config/sermon-index/pg.env"); got != pgEnv {
		t.Errorf("pg.env was rewritten:\n%s", got)
	}
	if env := r.read(".config/sermon-index/env"); !strings.Contains(env, "sermons:feedfeed@127.0.0.1:5433") {
		t.Errorf("env does not use pg.env's password:\n%s", env)
	}
}

func TestInstallStopsWhenPostgresNeverAnswers(t *testing.T) {
	r := newRig(t)
	out, code := r.run("scripts/install-laptop.sh", "FAKE_PG=down", "PG_WAIT=1")
	if code == 0 {
		t.Fatalf("install succeeded with Postgres down:\n%s", out)
	}
	if !strings.Contains(out, "Postgres") {
		t.Errorf("no word about Postgres:\n%s", out)
	}
	if strings.Contains(r.calls(), "sermons migrate") {
		t.Error("migrate ran with Postgres down")
	}
}

func TestInstallStopsWhenSmokeFails(t *testing.T) {
	r := newRig(t)
	out, code := r.run("scripts/install-laptop.sh", "FAKE_SMOKE=bad")
	if code == 0 {
		t.Fatalf("install succeeded though the smoke test failed:\n%s", out)
	}
}

func TestUninstallRemovesTheUnitsAndTheBinaryOnly(t *testing.T) {
	r := newRig(t)
	r.install()
	r.clearCalls()
	out, code := r.run("scripts/uninstall-laptop.sh")
	if code != 0 {
		t.Fatalf("uninstall exited %d:\n%s", code, out)
	}
	for _, gone := range []string{".local/bin/sermons", ".config/systemd/user/sermon-index-pg.service", ".config/systemd/user/sermon-index.service"} {
		if _, err := os.Stat(r.path(gone)); !os.IsNotExist(err) {
			t.Errorf("%s is still there", gone)
		}
	}
	for _, kept := range []string{".config/sermon-index/env", ".config/sermon-index/pg.env", ".local/state/sermon-index/pg"} {
		if _, err := os.Stat(r.path(kept)); err != nil {
			t.Errorf("%s was removed: %v", kept, err)
		}
		if !strings.Contains(out, filepath.Dir(r.path(kept))) && !strings.Contains(out, r.path(kept)) {
			t.Errorf("uninstall does not name the kept %s:\n%s", kept, out)
		}
	}
	c := r.calls()
	for _, want := range []string{"systemctl --user disable --now sermon-index.service", "systemctl --user disable --now sermon-index-pg.service", "systemctl --user daemon-reload"} {
		if !strings.Contains(c, want) {
			t.Errorf("uninstall did not call %q:\n%s", want, c)
		}
	}
	// The service stops before Postgres.
	if strings.Index(c, "disable --now sermon-index.service") > strings.Index(c, "disable --now sermon-index-pg.service") {
		t.Error("Postgres was stopped before the service")
	}
	// A second uninstall has nothing to do and still succeeds.
	if out, code := r.run("scripts/uninstall-laptop.sh"); code != 0 {
		t.Errorf("second uninstall exited %d:\n%s", code, out)
	}
}

func TestSmoke(t *testing.T) {
	r := newRig(t)
	if out, code := r.run("scripts/smoke.sh", "SERMONS_URL=http://x"); code != 0 {
		t.Errorf("smoke failed on a good service (%d):\n%s", code, out)
	}
	cases := map[string]string{
		"bad":  "fewer than two sources",
		"bad2": "wrong first sermon",
		"down": "service down",
	}
	for mode := range cases {
		out, code := r.run("scripts/smoke.sh", "SERMONS_URL=http://x", "FAKE_SMOKE="+mode)
		if code != 1 {
			t.Errorf("FAKE_SMOKE=%s: exit %d, want 1:\n%s", mode, code, out)
		}
	}
}
