package tests

// Tests of the hourly Subsplash linking: scripts/subsplash-hourly.sh, its
// share of the lock with scripts/refresh-brookfield.sh, and the two units in
// deploy/laptop that run it. It runs in a temporary clone with a bare
// repository as its origin. The real tools/subsplash-links is run (built once
// a test, its cache already holding every address so that it makes no
// request); `go run ./tools/subsplash-notes`, `sermons` and the Postern inbox
// are fakes.

import (
	"bufio"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// fakeGoHourly answers the two `go run` commands of the script: the notes
// fetch (it writes FAKE_NOTES_RECORDS as the records file, or with
// FAKE_NOTES=fail fails as a Postern that is down does) and the links tool
// (the real one, built to $LINKS_BIN; it logs the link list it was given).
const fakeGoHourly = `#!/bin/sh
echo "go $*" >> "$FAKE_LOG"
[ "$1" = run ] || exit 2
case "$2" in
./tools/subsplash-notes)
	if [ "$FAKE_NOTES" = fail ]; then echo "subsplash-notes: postern is down" >&2; exit 1; fi
	[ -n "$FAKE_NOTES_RECORDS" ] && printf '%s\n' "$FAKE_NOTES_RECORDS" > data/subsplash-notes.jsonl
	exit 0 ;;
./tools/subsplash-links)
	shift 2
	prev=
	for a in "$@"; do
		[ "$prev" = --links ] && sed 's/^/link: /' "$a" >> "$FAKE_LOG"
		[ "$prev" = --pdf ] && echo "pdf: $a" >> "$FAKE_LOG"
		prev=$a
	done
	exec "$LINKS_BIN" "$@" ;;
esac
exit 2
`

// fakeSermonsHourly answers "sermons load <dir>" and logs what it saw.
const fakeSermonsHourly = `#!/bin/sh
echo "sermons $*" >> "$FAKE_LOG"
[ "$1" = load ] || exit 2
[ -n "$DATABASE_URL" ] && echo "database-url-in-environment" >> "$FAKE_LOG"
[ "$(git -C "$HOURLY_ROOT" rev-parse HEAD)" = "$(git -C "$HOURLY_ROOT" rev-parse origin/main)" ] && echo "load-after-push" >> "$FAKE_LOG"
if [ "$FAKE_LOAD" = fail ]; then echo "sermons: load: the database refused" >&2; exit 1; fi
# A load puts the file in the fake service, unless the service ignores loads.
[ "$FAKE_LOAD_IGNORED" = 1 ] || cp "$HOURLY_ROOT/sources/brookfield-first/rows.jsonl" "$FAKE_SERVICE"
echo "loaded"
`

const hourlyRows = `{"id":"v1","title":"The Garden Gate","ref":"Gen.2.8-Gen.2.15","url":"https://www.youtube.com/watch?v=v1","kind":"link","date":"2031-03-02"}
{"id":"v2","title":"Brookfield First - Online Service | March 9, 2031","ref":"Luke.13.22-Luke.13.30","url":"https://www.youtube.com/watch?v=v2","kind":"link","date":"2031-03-09"}
{"id":"v5","title":"God's Quiet Work","ref":"Mark.1.1-Mark.1.8","url":"https://www.youtube.com/watch?v=v5","kind":"link","date":"2031-04-06"}
{"id":"v6","title":"No Passage Named","url":"https://www.youtube.com/watch?v=v6","kind":"link","date":"2031-04-13"}
`

const hourlyRecord = `{"id":"v2","date":"2031-03-09","title":"The Narrow Door","ref":"Luke.13.22-Luke.13.30"}`

const hourlyCache = `{"code":"aaa1111","address":"https://subsplash.com/u/brookfieldfirst/media/d/aaa1111-the-garden-gate"}
{"code":"bbb2222","address":"https://subsplash.com/u/brookfieldfirst/media/d/bbb2222-the-narrow-door"}
{"code":"ccc3333","address":"https://subsplash.com/u/brookfieldfirst/media/d/ccc3333-gods-quiet-work"}
`

type hourlyRig struct {
	*refreshRig
	inbox   string
	service string // the file the fake service serves: a rows.jsonl as the last load left it
}

// fakeService answers GET /v1/sermons as the real service does for a row with
// a passage: the rows of the file at path that have a ref and are not a
// series, paged by limit and offset. A row with no ref is never served, as in
// the real service.
func fakeService(t *testing.T, path string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sermons" || r.URL.Query().Get("source") != "brookfield-first" || r.URL.Query().Get("ref") == "" {
			http.Error(w, `{"error":"bad_ref"}`, http.StatusBadRequest)
			return
		}
		b, err := os.ReadFile(path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		var all []map[string]any
		for _, line := range strings.Split(string(b), "\n") {
			if strings.TrimSpace(line) == "" {
				continue
			}
			var m map[string]any
			if err := json.Unmarshal([]byte(line), &m); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if m["ref"] == nil || m["series"] == true {
				continue
			}
			all = append(all, map[string]any{"source": "brookfield-first", "id": m["id"], "title": m["title"], "preacher": m["preacher"],
				"date": m["date"], "url": m["url"], "alt_url": m["alt_url"], "kind": m["kind"], "ref": m["ref"], "role": "text", "width": 1})
		}
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		if limit < 1 || limit > 200 {
			limit = 50
		}
		page := []map[string]any{}
		for i := offset; i < len(all) && i < offset+limit; i++ {
			page = append(page, all[i])
		}
		json.NewEncoder(w).Encode(map[string]any{"ref": r.URL.Query().Get("ref"), "total": len(all), "rows": page})
	}))
	t.Cleanup(srv.Close)
	return srv
}

func newHourlyRig(t *testing.T) *hourlyRig {
	t.Helper()
	r := newRefreshRig(t)
	root := repoRoot(t)
	bin := filepath.Join(t.TempDir(), "subsplash-links")
	build := exec.Command("go", "build", "-o", bin, "./tools/subsplash-links")
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build subsplash-links: %v\n%s", err, out)
	}
	for name, body := range map[string]string{"go": fakeGoHourly, "sermons": fakeSermonsHourly} {
		if err := os.WriteFile(filepath.Join(r.stubs, name), []byte(body), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	h := &hourlyRig{refreshRig: r, inbox: filepath.Join(r.home, ".local/state/mw/postern/inbox")}
	if err := os.MkdirAll(h.inbox, 0o755); err != nil {
		t.Fatal(err)
	}
	// The private files the tools keep are git-ignored, as in the real repo.
	for rel, body := range map[string]string{
		"sources/brookfield-first/rows.jsonl": hourlyRows,
		".gitignore":                          "data/subsplash-links-cache.jsonl\ndata/subsplash-notes.jsonl\n",
	} {
		if err := os.WriteFile(filepath.Join(r.clone, rel), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	git(t, r.clone, "add", ".")
	git(t, r.clone, "commit", "-m", "rows")
	git(t, r.clone, "push", "-q", "origin", "main")
	if err := os.WriteFile(filepath.Join(r.clone, "data/subsplash-links-cache.jsonl"), []byte(hourlyCache), 0o644); err != nil {
		t.Fatal(err)
	}
	// The fake service holds what the last load left: the committed rows.
	h.service = filepath.Join(t.TempDir(), "service-rows.jsonl")
	if err := os.WriteFile(h.service, []byte(hourlyRows), 0o644); err != nil {
		t.Fatal(err)
	}
	srv := fakeService(t, h.service)
	r.env = append(r.env, "HOURLY_ROOT="+r.clone, "LINKS_BIN="+bin, "FAKE_NOTES_RECORDS="+hourlyRecord, "SERMONS_URL="+srv.URL, "FAKE_SERVICE="+h.service)
	h.message("m1", "https://brookfieldfirst.subspla.sh/aaa1111")
	h.message("m2", "Look at this one https://brookfieldfirst.subspla.sh/bbb2222 and a stranger https://example.org/zzz")
	return h
}

func (h *hourlyRig) message(name, text string) {
	h.t.Helper()
	b, _ := json.Marshal(map[string]any{"seq": 1, "class": "message", "thread": "general", "text": text})
	if err := os.WriteFile(filepath.Join(h.inbox, name+".message.json"), b, 0o644); err != nil {
		h.t.Fatal(err)
	}
}

func (h *hourlyRig) hourly(extra ...string) (string, int) {
	h.t.Helper()
	return h.run("scripts/subsplash-hourly.sh", extra...)
}

func (h *hourlyRig) clearLog() { _ = os.Remove(h.log) }

func (h *hourlyRig) progressLines() []map[string]any {
	h.t.Helper()
	f, err := os.Open(filepath.Join(h.home, ".local/state/sermon-index/subsplash-progress.jsonl"))
	if err != nil {
		h.t.Fatalf("no progress file: %v", err)
	}
	defer f.Close()
	var out []map[string]any
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var m map[string]any
		if err := json.Unmarshal(sc.Bytes(), &m); err != nil {
			h.t.Fatalf("progress line %q: %v", sc.Text(), err)
		}
		out = append(out, m)
	}
	return out
}

func (h *hourlyRig) altURLs() int {
	return strings.Count(h.file("sources/brookfield-first/rows.jsonl"), `"alt_url":"https://subsplash.com/u/brookfieldfirst/media/d/`)
}

func (h *hourlyRig) lastMessage() string { return git(h.t, h.clone, "log", "-1", "--format=%s") }

func (h *hourlyRig) pushed() bool {
	return git(h.t, h.clone, "rev-parse", "HEAD") == git(h.t, h.origin, "rev-parse", "main")
}

// Acceptance 1, 2 and 4: a run does the steps in order, commits and pushes the
// rows it changed with the message the story names, loads brookfield-first,
// and writes a progress line; a second run with nothing new changes nothing and
// loads nothing; a new link in the inbox is linked by the third.
func TestHourlyRunsInOrderCommitsPushesLoadsAndTheSecondRunDoesNothing(t *testing.T) {
	h := newHourlyRig(t)
	before := h.commits()
	out, code := h.hourly()
	if code != 0 {
		t.Fatalf("exit %d:\n%s", code, out)
	}
	if h.altURLs() != 2 {
		t.Errorf("%d rows hold an alt_url, want 2 (v1 by its title, v2 by its record):\n%s", h.altURLs(), h.file("sources/brookfield-first/rows.jsonl"))
	}
	if got := h.lastMessage(); got != "Subsplash links: 2 new (hourly)" {
		t.Errorf("commit message %q", got)
	}
	if h.commits() == before {
		t.Error("no commit")
	}
	if st := git(t, h.clone, "status", "--porcelain"); st != "" {
		t.Errorf("the tree is not clean:\n%s", st)
	}
	if !h.pushed() {
		t.Error("the commit was not pushed")
	}
	if files := git(t, h.clone, "show", "--name-only", "--format=", "HEAD"); strings.TrimSpace(files) != "sources/brookfield-first/rows.jsonl" {
		t.Errorf("the commit holds more than rows.jsonl:\n%s", files)
	}
	log := h.calls()
	notes := strings.Index(log, "go run ./tools/subsplash-notes fetch")
	links := strings.Index(log, "go run ./tools/subsplash-links")
	load := strings.Index(log, "sermons load "+h.clone+"/sources/brookfield-first")
	if notes < 0 || links < notes || load < links {
		t.Errorf("want notes fetch, then links, then load, in order (%d, %d, %d):\n%s", notes, links, load, log)
	}
	if !strings.Contains(log, "link: https://brookfieldfirst.subspla.sh/aaa1111") || !strings.Contains(log, "link: https://brookfieldfirst.subspla.sh/bbb2222") {
		t.Errorf("the link list was not built from the inbox:\n%s", log)
	}
	if strings.Contains(log, "example.org") {
		t.Errorf("a stranger's link reached the link list:\n%s", log)
	}
	if !strings.Contains(log, "database-url-in-environment") || !strings.Contains(log, "load-after-push") {
		t.Errorf("the load lacked the env file or ran before the push:\n%s", log)
	}
	if strings.Contains(log, "sermons load --all") {
		t.Errorf("the load was of everything:\n%s", log)
	}
	p := h.progressLines()
	if len(p) != 1 {
		t.Fatalf("%d progress lines, want 1", len(p))
	}
	for k, v := range map[string]any{"records": 1.0, "links": 2.0, "rows_linked": 2.0, "rows_total": 4.0, "new": 2.0, "oldest": "2031-03-02", "newest": "2031-03-09"} {
		if p[0][k] != v {
			t.Errorf("progress %s = %v, want %v: %v", k, p[0][k], v, p[0])
		}
	}

	// A second run with nothing new changes nothing and loads nothing.
	h.clearLog()
	commits := h.commits()
	out, code = h.hourly()
	if code != 0 {
		t.Fatalf("second run exit %d:\n%s", code, out)
	}
	if h.commits() != commits {
		t.Error("a second run with nothing new committed")
	}
	if strings.Contains(h.calls(), "sermons") {
		t.Errorf("a second run with nothing new loaded:\n%s", h.calls())
	}
	if st := git(t, h.clone, "status", "--porcelain"); st != "" {
		t.Errorf("the tree is not clean after the second run:\n%s", st)
	}
	p = h.progressLines()
	if len(p) != 2 || p[1]["new"] != 0.0 || p[1]["rows_linked"] != 2.0 {
		t.Errorf("the second run's progress line: %v", p)
	}

	// A link he sends later is linked by the next run.
	h.message("m3", "https://brookfieldfirst.subspla.sh/ccc3333")
	h.clearLog()
	if out, code = h.hourly(); code != 0 {
		t.Fatalf("third run exit %d:\n%s", code, out)
	}
	if got := h.lastMessage(); got != "Subsplash links: 1 new (hourly)" {
		t.Errorf("third run commit message %q", got)
	}
	if h.altURLs() != 3 || !h.pushed() {
		t.Errorf("third run: %d linked, pushed %v", h.altURLs(), h.pushed())
	}
	if n := strings.Count(h.calls(), "sermons load"); n != 1 {
		t.Errorf("third run loaded %d times, want 1:\n%s", n, h.calls())
	}
}

// seedAltURLs writes alt_url on two committed rows and pushes, as a story or
// a hand edit would, without the hourly run's own links: the service (still
// holding the rows without them) lacks them. No inbox message, no record.
func seedAltURLs(t *testing.T, h *hourlyRig) {
	t.Helper()
	rows := h.file("sources/brookfield-first/rows.jsonl")
	rows = strings.Replace(rows, `"kind":"link","date":"2031-03-02"`, `"alt_url":"https://subsplash.com/u/brookfieldfirst/media/d/aaa1111-the-garden-gate","kind":"link","date":"2031-03-02"`, 1)
	rows = strings.Replace(rows, `"kind":"link","date":"2031-04-06"`, `"alt_url":"https://subsplash.com/u/brookfieldfirst/media/d/ccc3333-gods-quiet-work","kind":"link","date":"2031-04-06"`, 1)
	if err := os.WriteFile(filepath.Join(h.clone, "sources/brookfield-first/rows.jsonl"), []byte(rows), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, h.clone, "commit", "-qam", "alt_url written by something else")
	git(t, h.clone, "push", "-q", "origin", "main")
	for _, f := range []string{"m1.message.json", "m2.message.json"} {
		os.Remove(filepath.Join(h.inbox, f))
	}
}

// markLoaded writes the hash of rows.jsonl as the file of what the last load
// put in the database, as the refresh does after its load.
func markLoaded(t *testing.T, h *hourlyRig) {
	t.Helper()
	state := filepath.Join(h.home, ".local/state/sermon-index")
	os.MkdirAll(state, 0o755)
	os.WriteFile(filepath.Join(state, "subsplash-loaded"), []byte(git(t, h.clone, "hash-object", "sources/brookfield-first/rows.jsonl")+"\n"), 0o644)
}

// The story's acceptance 1: rows.jsonl holds alt_urls the service lacks, and
// the run found nothing new from mail or links (it commits nothing); it loads
// brookfield-first all the same, and the service then holds them. A second run
// finds the two equal and loads nothing (acceptance 2).
func TestHourlyLoadsWhatDiffersFromTheServiceWithNothingNewOfItsOwn(t *testing.T) {
	h := newHourlyRig(t)
	seedAltURLs(t, h)
	h.env = append(h.env, "FAKE_NOTES_RECORDS=")
	// The hash of what the last load put there says "nothing to load", as it
	// did when 59.3's landing wrote the rows: the hash is not what decides.
	markLoaded(t, h)
	commits := h.commits()
	out, code := h.hourly()
	if code != 0 {
		t.Fatalf("exit %d:\n%s", code, out)
	}
	if h.commits() != commits {
		t.Error("the run committed though it had nothing new")
	}
	if n := strings.Count(h.calls(), "sermons load "+h.clone+"/sources/brookfield-first"); n != 1 {
		t.Fatalf("loaded %d times, want 1 (the service lacked two alt_urls):\n%s\n%s", n, h.calls(), out)
	}
	if !strings.Contains(out, "2 of 3") && !strings.Contains(out, "2 rows") {
		t.Logf("journal:\n%s", out)
	}
	if got, _ := os.ReadFile(h.service); !strings.Contains(string(got), "aaa1111-the-garden-gate") {
		t.Errorf("the service does not hold the alt_url after the load:\n%s", got)
	}

	// Acceptance 2: with the two equal, nothing is loaded.
	h.clearLog()
	if out, code = h.hourly(); code != 0 {
		t.Fatalf("second run exit %d:\n%s", code, out)
	}
	if strings.Contains(h.calls(), "sermons") {
		t.Errorf("a run with the service equal to rows.jsonl loaded:\n%s", h.calls())
	}
}

// Every field the service serves counts: a title, a date, a preacher, a url or
// a ref that differs is loaded, and a row with no passage (never served) does
// not make the run load forever.
func TestHourlyComparesEveryServedField(t *testing.T) {
	for name, edit := range map[string][2]string{
		"title":    {`"title":"The Garden Gate"`, `"title":"The Garden Gate, Revised"`},
		"date":     {`"date":"2031-03-02"`, `"date":"2031-03-03"`},
		"url":      {`v=v1"`, `v=v1x"`},
		"ref":      {`"ref":"Gen.2.8-Gen.2.15"`, `"ref":"Gen.2.8-Gen.2.16"`},
		"preacher": {`"kind":"link","date":"2031-03-02"`, `"preacher":"Pat Doe","kind":"link","date":"2031-03-02"`},
	} {
		t.Run(name, func(t *testing.T) {
			h := newHourlyRig(t)
			seedAltURLs(t, h)
			rows := strings.Replace(h.file("sources/brookfield-first/rows.jsonl"), edit[0], edit[1], 1)
			os.WriteFile(filepath.Join(h.clone, "sources/brookfield-first/rows.jsonl"), []byte(rows), 0o644)
			git(t, h.clone, "commit", "-qam", "edit")
			git(t, h.clone, "push", "-q", "origin", "main")
			// The service already has the alt_urls; only the one field differs.
			os.WriteFile(h.service, []byte(strings.Replace(rows, edit[1], edit[0], 1)), 0o644)
			h.env = append(h.env, "FAKE_NOTES_RECORDS=")
			markLoaded(t, h)
			if out, code := h.hourly(); code != 0 {
				t.Fatalf("exit %d:\n%s", code, out)
			}
			if n := strings.Count(h.calls(), "sermons load"); n != 1 {
				t.Errorf("a differing %s was loaded %d times, want 1:\n%s", name, n, h.calls())
			}
		})
	}
}

// Acceptance 3: a load after which the service still differs (a fake service
// that ignores it) is one journal line naming the count and up to five row
// ids, and the run fails, so the unit's OnFailure= tells the Mayor.
func TestHourlyAlarmsWhenTheServiceStillDiffersAfterTheLoad(t *testing.T) {
	h := newHourlyRig(t)
	seedAltURLs(t, h)
	h.env = append(h.env, "FAKE_NOTES_RECORDS=", "FAKE_LOAD_IGNORED=1")
	out, code := h.hourly()
	if code == 0 {
		t.Fatalf("exit 0 though the service still differs after the load:\n%s", out)
	}
	var lines []string
	for _, l := range strings.Split(out, "\n") {
		if strings.Contains(l, "still differ") {
			lines = append(lines, l)
		}
	}
	if len(lines) != 1 {
		t.Fatalf("%d journal lines say the rows still differ, want 1:\n%s", len(lines), out)
	}
	for _, want := range []string{"2 ", "v1", "v5"} {
		if !strings.Contains(lines[0], want) {
			t.Errorf("the line lacks %q: %s", want, lines[0])
		}
	}
	if strings.Contains(lines[0], "v2") {
		t.Errorf("the line names a row that matches: %s", lines[0])
	}
	if n := strings.Count(h.calls(), "sermons load"); n != 1 {
		t.Errorf("loaded %d times, want 1 (no retry loop):\n%s", n, h.calls())
	}
}

// The line names at most five ids but counts them all.
func TestHourlyAlarmNamesFiveIDsAtMost(t *testing.T) {
	h := newHourlyRig(t)
	var rows strings.Builder
	for i := 1; i <= 8; i++ {
		n := strconv.Itoa(i)
		rows.WriteString(`{"id":"w` + n + `","title":"T` + n + `","ref":"Gen.1.` + n + `","url":"https://www.youtube.com/watch?v=w` + n + `","kind":"link","date":"2031-05-0` + n + `"}` + "\n")
	}
	os.WriteFile(filepath.Join(h.clone, "sources/brookfield-first/rows.jsonl"), []byte(rows.String()), 0o644)
	os.WriteFile(h.service, nil, 0o644) // the service holds none of the eight
	git(t, h.clone, "commit", "-qam", "eight rows")
	git(t, h.clone, "push", "-q", "origin", "main")
	h.env = append(h.env, "FAKE_NOTES_RECORDS=", "FAKE_LOAD_IGNORED=1")
	out, code := h.hourly()
	if code == 0 {
		t.Fatalf("exit 0:\n%s", out)
	}
	for _, l := range strings.Split(out, "\n") {
		if strings.Contains(l, "still differ") {
			if !strings.Contains(l, "8 ") || !strings.Contains(l, "w1 w2 w3 w4 w5") || strings.Contains(l, "w6") {
				t.Errorf("want the count 8 and the first five ids: %s", l)
			}
			return
		}
	}
	t.Errorf("no line says the rows still differ:\n%s", out)
}

// A service that cannot be asked is not a pass: the run still loads what the
// hash says is new, and it fails, since nothing was verified.
func TestHourlyFailsWhenTheServiceCannotBeAsked(t *testing.T) {
	h := newHourlyRig(t)
	h.env = append(h.env, "SERMONS_URL=http://127.0.0.1:1")
	out, code := h.hourly()
	if code == 0 || !strings.Contains(out, "cannot compare") {
		t.Errorf("exit %d, want non-zero and a line saying the comparison failed:\n%s", code, out)
	}
	if !strings.Contains(h.calls(), "sermons load") {
		t.Errorf("the hash did not trigger the load when the service was down:\n%s", h.calls())
	}
}

// Acceptance 2: a tree that is dirty stops the run before anything is done.
func TestHourlyStopsOnADirtyTree(t *testing.T) {
	for name, dirty := range map[string]func(*hourlyRig){
		"untracked": func(h *hourlyRig) {
			os.WriteFile(filepath.Join(h.clone, "notes.txt"), []byte("mine\n"), 0o644)
		},
		"edited": func(h *hourlyRig) {
			os.WriteFile(filepath.Join(h.clone, "README.md"), []byte("an edit\n"), 0o644)
		},
	} {
		t.Run(name, func(t *testing.T) {
			h := newHourlyRig(t)
			dirty(h)
			commits, status := h.commits(), git(t, h.clone, "status", "--porcelain")
			rows := h.file("sources/brookfield-first/rows.jsonl")
			out, code := h.hourly()
			if code == 0 {
				t.Fatalf("exit 0 over a dirty tree:\n%s", out)
			}
			if !strings.Contains(out, "not clean") || !strings.Contains(out, "nothing was done") {
				t.Errorf("the journal does not say why:\n%s", out)
			}
			if h.calls() != "" {
				t.Errorf("something ran:\n%s", h.calls())
			}
			if h.commits() != commits || git(t, h.clone, "status", "--porcelain") != status || h.file("sources/brookfield-first/rows.jsonl") != rows {
				t.Error("the run changed the checkout")
			}
		})
	}
}

// Acceptance 2: a branch behind origin stops the run before anything is done.
func TestHourlyStopsWhenBehindOrigin(t *testing.T) {
	h := newHourlyRig(t)
	other := filepath.Join(t.TempDir(), "other")
	git(t, filepath.Dir(other), "clone", "-q", h.origin, other)
	if err := os.WriteFile(filepath.Join(other, "README.md"), []byte("landed elsewhere\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, other, "commit", "-qam", "a landing")
	git(t, other, "push", "-q", "origin", "main")
	head := git(t, h.clone, "rev-parse", "HEAD")
	out, code := h.hourly()
	if code == 0 {
		t.Fatalf("exit 0 while behind origin:\n%s", out)
	}
	if !strings.Contains(out, "behind origin/main") || !strings.Contains(out, "nothing done") {
		t.Errorf("the journal does not say why:\n%s", out)
	}
	if h.calls() != "" {
		t.Errorf("something ran:\n%s", h.calls())
	}
	if git(t, h.clone, "rev-parse", "HEAD") != head || h.altURLs() != 0 {
		t.Error("the run changed the checkout")
	}
}

// The lock is shared: an hourly run that finds it held does nothing and exits
// 0 (the next hour links); the refresh that finds it held waits, and fails if
// it is never free.
func holdLock(t *testing.T, h *hourlyRig) func() {
	t.Helper()
	lock := filepath.Join(h.home, ".local/state/sermon-index/brookfield.lock")
	if err := os.MkdirAll(filepath.Dir(lock), 0o755); err != nil {
		t.Fatal(err)
	}
	holder := exec.Command("sh", "-c", `exec 9>"$1"; flock 9; echo held; exec sleep 60`, "sh", lock)
	pipe, err := holder.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := holder.Start(); err != nil {
		t.Fatal(err)
	}
	if line, _ := bufio.NewReader(pipe).ReadString('\n'); !strings.Contains(line, "held") {
		t.Fatalf("the lock holder said %q", line)
	}
	return func() { _ = holder.Process.Kill(); _ = holder.Wait() }
}

func TestHourlySkipsWhileTheRefreshHoldsTheLock(t *testing.T) {
	h := newHourlyRig(t)
	release := holdLock(t, h)
	defer release()
	out, code := h.hourly()
	if code != 0 || !strings.Contains(out, "holds") {
		t.Errorf("exit %d, want 0 and a line saying the lock is held:\n%s", code, out)
	}
	if h.calls() != "" || h.altURLs() != 0 {
		t.Errorf("a run without the lock did something:\n%s", h.calls())
	}
}

func TestRefreshWaitsForTheLockAndFailsWithoutRunningWhenItStaysHeld(t *testing.T) {
	h := newHourlyRig(t)
	release := holdLock(t, h)
	defer release()
	start := time.Now()
	out, code := h.run("scripts/refresh-brookfield.sh", "REFRESH_LOCK_WAIT=1", "FAKE_ROWS=new row")
	if code == 0 || !strings.Contains(out, "brookfield.lock") {
		t.Errorf("exit %d, want non-zero and the lock named:\n%s", code, out)
	}
	if time.Since(start) < time.Second {
		t.Error("the refresh did not wait for the lock")
	}
	if h.calls() != "" {
		t.Errorf("a refresh without the lock did something:\n%s", h.calls())
	}
}

// Once the lock is free again the refresh runs, and what it loaded is what
// the hourly run knows the database holds.
func TestRefreshTakesTheLockAndTellsTheHourlyRunWhatItLoaded(t *testing.T) {
	h := newHourlyRig(t)
	if err := os.WriteFile(filepath.Join(h.stubs, "go"), []byte(fakeGoRefresh), 0o755); err != nil {
		t.Fatal(err)
	}
	out, code := h.run("scripts/refresh-brookfield.sh", "FAKE_ROWS=new row")
	if code != 0 {
		t.Fatalf("exit %d:\n%s", code, out)
	}
	b, err := os.ReadFile(filepath.Join(h.home, ".local/state/sermon-index/subsplash-loaded"))
	if err != nil {
		t.Fatalf("the refresh did not note what it loaded: %v", err)
	}
	if want := git(t, h.clone, "hash-object", "sources/brookfield-first/rows.jsonl"); strings.TrimSpace(string(b)) != want {
		t.Errorf("subsplash-loaded = %q, want %q", b, want)
	}
}

// A load that fails is tried again next hour though nothing new came; a push
// that is refused is pushed next hour, and does not hold back the load.
func TestHourlyRetriesAFailedLoadAndAFailedPush(t *testing.T) {
	h := newHourlyRig(t)
	out, code := h.hourly("FAKE_LOAD=fail")
	if code == 0 || !strings.Contains(out, "load") {
		t.Fatalf("exit %d though the load failed:\n%s", code, out)
	}
	if !h.pushed() || h.altURLs() != 2 {
		t.Error("the commit was not kept and pushed before the load")
	}
	h.clearLog()
	commits := h.commits()
	if out, code = h.hourly(); code != 0 {
		t.Fatalf("second run exit %d:\n%s", code, out)
	}
	if h.commits() != commits || strings.Count(h.calls(), "sermons load") != 1 {
		t.Errorf("the second run should load once and commit nothing:\n%s", h.calls())
	}

	// A refused push.
	h.message("m3", "https://brookfieldfirst.subspla.sh/ccc3333")
	hook := filepath.Join(h.origin, "hooks/pre-receive")
	os.MkdirAll(filepath.Dir(hook), 0o755)
	os.WriteFile(hook, []byte("#!/bin/sh\necho refused >&2\nexit 1\n"), 0o755)
	h.clearLog()
	out, code = h.hourly()
	if code == 0 || !strings.Contains(out, "push") {
		t.Fatalf("exit %d though the push was refused:\n%s", code, out)
	}
	if strings.Count(h.calls(), "sermons load") != 1 {
		t.Errorf("a refused push held back the load:\n%s", h.calls())
	}
	if h.pushed() {
		t.Fatal("pushed through a refusing hook")
	}
	os.Remove(hook)
	h.clearLog()
	if out, code = h.hourly(); code != 0 {
		t.Fatalf("run after the hook went exit %d:\n%s", code, out)
	}
	if !h.pushed() || strings.Contains(h.calls(), "sermons") {
		t.Errorf("the waiting commit was not pushed, or loaded again:\n%s", h.calls())
	}
}

// A Postern that is down does not stop the inbox's links, but the run fails.
func TestHourlyGoesOnWhenPosternIsDown(t *testing.T) {
	h := newHourlyRig(t)
	out, code := h.hourly("FAKE_NOTES=fail", "FAKE_NOTES_RECORDS=")
	if code == 0 || !strings.Contains(out, "subsplash-notes fetch failed") {
		t.Fatalf("exit %d, want non-zero naming the fetch:\n%s", code, out)
	}
	if h.altURLs() != 1 || !h.pushed() || !strings.Contains(h.calls(), "sermons load") {
		t.Errorf("the link by title (1 row) was not committed, pushed and loaded: %d linked\n%s", h.altURLs(), h.calls())
	}
}

func TestHourlyUnits(t *testing.T) {
	root := repoRoot(t)
	read := func(name string) string {
		b, err := os.ReadFile(filepath.Join(root, "deploy/laptop", name))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	svc := read("sermon-index-subsplash-hourly.service")
	for _, l := range []string{"Type=oneshot", "OnFailure=mw-unit-failed@%n.service", "EnvironmentFile=-%h/.config/mw/dispatch.env",
		"Environment=SERMONS=%h/.local/bin/sermons", "Environment=POSTERN_URL=http://10.88.0.2:8787", "WorkingDirectory=%h/sermon-index",
		"ExecStart=/bin/sh %h/sermon-index/scripts/subsplash-hourly.sh", "TimeoutStartSec="} {
		if !strings.Contains(svc, l) {
			t.Errorf("the service lacks %q", l)
		}
	}
	timer := read("sermon-index-subsplash-hourly.timer")
	for _, l := range []string{"OnCalendar=hourly", "Persistent=true", "WantedBy=timers.target"} {
		if !strings.Contains(timer, l) {
			t.Errorf("the timer lacks %q", l)
		}
	}
	if strings.Contains(timer, "Unit=") {
		t.Error("the timer names a Unit=")
	}
}

func TestInstallEnablesTheHourlyTimerAndUninstallRemovesIt(t *testing.T) {
	r := newRig(t)
	r.install()
	for _, u := range []string{"sermon-index-subsplash-hourly.service", "sermon-index-subsplash-hourly.timer"} {
		want, err := os.ReadFile(filepath.Join(r.root, "deploy/laptop", u))
		if err != nil {
			t.Fatal(err)
		}
		if got := r.read(".config/systemd/user/" + u); got != string(want) {
			t.Errorf("installed %s differs from deploy/laptop/%s", u, u)
		}
	}
	if !strings.Contains(r.calls(), "systemctl --user enable --now sermon-index-subsplash-hourly.timer") {
		t.Errorf("the hourly timer was not enabled and started:\n%s", r.calls())
	}
	if strings.Contains(r.calls(), "enable --now sermon-index-subsplash-hourly.service") {
		t.Error("the hourly service was enabled")
	}
	r.clearCalls()
	if second := r.install(); !strings.Contains(second, "nothing changed") || strings.Contains(r.calls(), "enable --now sermon-index-subsplash") {
		t.Errorf("a second install changed something:\n%s\n%s", second, r.calls())
	}
	r.clearCalls()
	if out, code := r.run("scripts/uninstall-laptop.sh"); code != 0 {
		t.Fatalf("uninstall exited %d:\n%s", code, out)
	}
	if _, err := os.Stat(r.path(".config/systemd/user/sermon-index-subsplash-hourly.timer")); !os.IsNotExist(err) {
		t.Error("the hourly timer is still installed")
	}
	if !strings.Contains(r.calls(), "systemctl --user disable --now sermon-index-subsplash-hourly.timer") {
		t.Errorf("uninstall did not disable the hourly timer:\n%s", r.calls())
	}
}
