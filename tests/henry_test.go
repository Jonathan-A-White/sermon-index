package tests

// A test of scripts/fetch-henry-ocr.sh with curl replaced by a fake: nothing
// here reaches the Internet Archive.

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// fakeCurlArchive logs the User-Agent each call sends (empty where none is
// given, which is curl's own) and writes the file named after -o.
const fakeCurlArchive = `#!/bin/sh
agent= out=
while [ $# -gt 0 ]; do
	case "$1" in
	-A|--user-agent) agent=$2; shift ;;
	-o) out=$2; shift ;;
	esac
	shift
done
echo "agent=$agent" >> "$FAKE_LOG"
[ -n "$out" ] || exit 2
echo "<DjVuXML/>" > "$out"
`

// The recipe (docs/adding-a-source.md): a User-Agent names the project.
func TestFetchHenryOCRNamesTheProject(t *testing.T) {
	r := newRig(t)
	if err := os.WriteFile(filepath.Join(r.stubs, "curl"), []byte(fakeCurlArchive), 0o755); err != nil {
		t.Fatal(err)
	}
	out, code := r.run("scripts/fetch-henry-ocr.sh", "SERMON_INDEX_ARCHIVE_CACHE="+t.TempDir())
	if code != 0 {
		t.Fatalf("exit %d:\n%s", code, out)
	}
	calls := strings.Split(strings.TrimSpace(r.calls()), "\n")
	if len(calls) != 2 {
		t.Fatalf("%d curl calls, want one for each of the two scans: %q", len(calls), calls)
	}
	for _, c := range calls {
		if !strings.HasPrefix(c, "agent=sermon-index") {
			t.Errorf("User-Agent %q does not name the project", strings.TrimPrefix(c, "agent="))
		}
	}
}

// fakeCurlTimed logs the clock, in nanoseconds, when each call starts and
// when it ends, and writes the file named after -o.
const fakeCurlTimed = `#!/bin/sh
start=$(date +%s%N)
out=
while [ $# -gt 0 ]; do
	case "$1" in
	-o) out=$2; shift ;;
	esac
	shift
done
[ -n "$out" ] || exit 2
echo "<DjVuXML/>" > "$out"
echo "$start $(date +%s%N)" >> "$FAKE_LOG"
`

// The recipe (docs/adding-a-source.md): one request at a time, a second apart.
// The second download waits at least that long after the first has ended.
func TestFetchHenryOCRSpacesItsRequests(t *testing.T) {
	r := newRig(t)
	if err := os.WriteFile(filepath.Join(r.stubs, "curl"), []byte(fakeCurlTimed), 0o755); err != nil {
		t.Fatal(err)
	}
	out, code := r.run("scripts/fetch-henry-ocr.sh", "SERMON_INDEX_ARCHIVE_CACHE="+t.TempDir())
	if code != 0 {
		t.Fatalf("exit %d:\n%s", code, out)
	}
	calls := strings.Split(strings.TrimSpace(r.calls()), "\n")
	if len(calls) != 2 {
		t.Fatalf("%d curl calls, want one for each of the two scans: %q", len(calls), calls)
	}
	at := func(call, field int) time.Time {
		f := strings.Fields(calls[call])
		if len(f) != 2 {
			t.Fatalf("call %q, want a start and an end time", calls[call])
		}
		ns, err := strconv.ParseInt(f[field], 10, 64)
		if err != nil {
			t.Fatal(err)
		}
		return time.Unix(0, ns)
	}
	if gap := at(1, 0).Sub(at(0, 1)); gap < time.Second {
		t.Errorf("the second download started %v after the first ended, want at least a second", gap)
	}
}
