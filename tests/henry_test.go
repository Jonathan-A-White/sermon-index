package tests

// A test of scripts/fetch-henry-ocr.sh with curl replaced by a fake: nothing
// here reaches the Internet Archive.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
