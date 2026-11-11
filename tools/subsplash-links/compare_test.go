package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Jonathan-A-White/sermon-index/internal/sources"
)

func sp(s string) *string { return &s }

func TestDifferingComparesEveryServedField(t *testing.T) {
	row := func() sources.Row {
		return sources.Row{ID: "a", Title: "T", Ref: sp("Gen.1.1"), URL: "u", Date: sp("2031-01-05"), Preacher: sp("P"), AltURL: sp("x")}
	}
	served := func() servedRow {
		return servedRow{ID: "a", Title: "T", Ref: "Gen.1.1", URL: "u", Date: sp("2031-01-05"), Preacher: sp("P"), AltURL: sp("x"), Role: "text"}
	}
	if ids, n := Differing([]sources.Row{row()}, map[string]servedRow{"a": served()}); len(ids) != 0 || n != 1 {
		t.Errorf("equal rows: %v %d", ids, n)
	}
	for name, edit := range map[string]func(*servedRow){
		"title":    func(s *servedRow) { s.Title = "T2" },
		"ref":      func(s *servedRow) { s.Ref = "Gen.1.2" },
		"date":     func(s *servedRow) { s.Date = nil },
		"preacher": func(s *servedRow) { s.Preacher = sp("Q") },
		"url":      func(s *servedRow) { s.URL = "v" },
		"alt_url":  func(s *servedRow) { s.AltURL = nil },
	} {
		s := served()
		edit(&s)
		if ids, _ := Differing([]sources.Row{row()}, map[string]servedRow{"a": s}); len(ids) != 1 || ids[0] != "a" {
			t.Errorf("%s: not seen: %v", name, ids)
		}
	}
	// Absent in the service, extra in the service, no passage, a series.
	series := true
	rows := []sources.Row{row(), {ID: "b", Title: "no ref", URL: "u"}, {ID: "c", Title: "s", Ref: sp("Gen.1"), Series: &series, URL: "u"}}
	ids, n := Differing(rows, map[string]servedRow{"z": {ID: "z"}})
	if n != 1 || strings.Join(ids, ",") != "a,z" {
		t.Errorf("missing and extra: %v %d", ids, n)
	}
}

func TestCompareServicePagesAndReports(t *testing.T) {
	var rows, lines []string
	for i := 0; i < 450; i++ {
		id := "r" + strconv.Itoa(i)
		lines = append(lines, fmt.Sprintf(`{"id":%q,"title":"T","ref":"Gen.1.1","url":"u"}`, id))
		rows = append(rows, id)
	}
	path := filepath.Join(t.TempDir(), "rows.jsonl")
	os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644)
	var asked []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked = append(asked, r.URL.RawQuery)
		off, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		var page []map[string]any
		for i := off; i < 450 && i < off+200; i++ {
			// Each row twice, as a row with a cited passage answers.
			if i%2 == 0 {
				page = append(page, map[string]any{"id": rows[i], "title": "T", "ref": "Gen.1.1", "url": "u", "role": "text"})
			} else if i > 300 { // rows 301.. differ in the title
				page = append(page, map[string]any{"id": rows[i], "title": "other", "ref": "Gen.1.1", "url": "u", "role": "text"})
			} else {
				page = append(page, map[string]any{"id": rows[i], "title": "T", "ref": "Gen.1.1", "url": "u", "role": "text"})
			}
		}
		json.NewEncoder(w).Encode(map[string]any{"total": 450, "rows": page})
	}))
	defer srv.Close()
	var out bytes.Buffer
	if err := compareService(path, srv.URL, "brookfield-first", &out); err != nil {
		t.Fatal(err)
	}
	if len(asked) != 3 {
		t.Errorf("%d requests, want 3 pages: %v", len(asked), asked)
	}
	if !strings.HasPrefix(out.String(), "compare: 75 of 450 rows differ from the service\nids: r301 r303 r305 r307 r309\n") {
		t.Errorf("report:\n%s", out.String())
	}
	if !strings.Contains(asked[0], "source=brookfield-first") {
		t.Errorf("no source in %q", asked[0])
	}
}

func TestCompareServiceFailsWhenTheServiceCannotBeAsked(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rows.jsonl")
	os.WriteFile(path, []byte(`{"id":"a","title":"T","ref":"Gen.1.1","url":"u"}`+"\n"), 0o644)
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "no", 500) }))
	defer bad.Close()
	for _, base := range []string{bad.URL, "http://127.0.0.1:1"} {
		var out, errOut bytes.Buffer
		if code := cli([]string{"--compare", base, "--rows", path}, &out, &errOut, deps{}); code != 1 || out.Len() != 0 || !strings.Contains(errOut.String(), "compare") {
			t.Errorf("%s: code %d out %q err %q", base, code, out.String(), errOut.String())
		}
	}
}

// A row with cited references only is served through one of them (role
// "cited"), and its ref is not compared; a row with a ref must be served as
// its text.
func TestDifferingCitedOnlyRowsAndTextRefs(t *testing.T) {
	cited := sources.Row{ID: "c", Title: "T", URL: "u", Cited: []string{"Matt.5.7"}}
	servedCited := servedRow{ID: "c", Title: "T", URL: "u", Ref: "Matt.5.7", Role: "cited"}
	if ids, n := Differing([]sources.Row{cited}, map[string]servedRow{"c": servedCited}); len(ids) != 0 || n != 1 {
		t.Errorf("a cited-only row: %v %d", ids, n)
	}
	withRef := sources.Row{ID: "r", Title: "T", URL: "u", Ref: sp("Gen.1.1"), Cited: []string{"Matt.5.7"}}
	if ids, _ := Differing([]sources.Row{withRef}, map[string]servedRow{"r": {ID: "r", Title: "T", URL: "u", Ref: "Matt.5.7", Role: "cited"}}); len(ids) != 1 {
		t.Errorf("a row with a ref served only as cited is not the same: %v", ids)
	}
}

func TestCompareServiceWaitsOutARateLimit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rows.jsonl")
	os.WriteFile(path, []byte(`{"id":"a","title":"T","ref":"Gen.1.1","url":"u"}`+"\n"), 0o644)
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls < 3 {
			w.Header().Set("Retry-After", "7")
			http.Error(w, "slow down", http.StatusTooManyRequests)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"total": 1, "rows": []map[string]any{{"id": "a", "title": "T", "ref": "Gen.1.1", "url": "u", "role": "text"}}})
	}))
	defer srv.Close()
	var slept []time.Duration
	old := compareSleep
	compareSleep = func(d time.Duration) { slept = append(slept, d) }
	defer func() { compareSleep = old }()
	var out bytes.Buffer
	if err := compareService(path, srv.URL, "brookfield-first", &out); err != nil {
		t.Fatal(err)
	}
	if out.String() != "compare: 0 of 1 rows differ from the service\n" || len(slept) != 2 || slept[0] != 7*time.Second {
		t.Errorf("out %q slept %v", out.String(), slept)
	}
}
