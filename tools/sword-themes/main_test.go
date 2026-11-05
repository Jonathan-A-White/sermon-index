package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/binary"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// rawZip is a SWORD RawLD package of the entries, as CrossWire's rawzip.
func rawZip(t *testing.T, m Module, entries []Entry) []byte {
	t.Helper()
	var idx, dat bytes.Buffer
	for _, e := range entries {
		text := e.Key + "\r\n" + e.Body + "\n"
		var rec [6]byte
		binary.LittleEndian.PutUint32(rec[:4], uint32(dat.Len()))
		binary.LittleEndian.PutUint16(rec[4:], uint16(len(text)))
		idx.Write(rec[:])
		dat.WriteString(text)
	}
	var zb bytes.Buffer
	zw := zip.NewWriter(&zb)
	for name, data := range map[string][]byte{m.DataPath + ".idx": idx.Bytes(), m.DataPath + ".dat": dat.Bytes()} {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		w.Write(data)
	}
	zw.Close()
	return zb.Bytes()
}

const confText = "[TCR]\nDistributionLicense=Public Domain\n"

func moduleServer(t *testing.T, m Module, entries []Entry) (*httptest.Server, *[]string) {
	t.Helper()
	zipBytes := rawZip(t, m, entries)
	var mu sync.Mutex
	var got []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		got = append(got, r.URL.Path+" "+r.UserAgent())
		mu.Unlock()
		switch r.URL.Path {
		case "/packages/rawzip/" + m.ZipName:
			w.Write(zipBytes)
		case "/raw/mods.d/" + m.ConfName:
			w.Write([]byte(confText))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, &got
}

func TestFetchModuleSkipsWhatIsCached(t *testing.T) {
	m := modules["TCR"]
	srv, got := moduleServer(t, m, fixture(t, "tcr-three.txt"))
	cache := t.TempDir()
	var log bytes.Buffer
	if err := fetchModule(context.Background(), srv.Client(), srv.URL, m, cache, &log); err != nil {
		t.Fatal(err)
	}
	if len(*got) != 2 {
		t.Fatalf("requests: %v", *got)
	}
	for _, g := range *got {
		if !strings.Contains(g, "sermon-index-sword-themes") {
			t.Errorf("no user agent naming the tool: %q", g)
		}
	}
	for _, f := range []string{"TCR.zip", "tcr.conf"} {
		if st, err := os.Stat(filepath.Join(cache, f)); err != nil || st.Size() == 0 {
			t.Errorf("%s not cached: %v", f, err)
		}
	}
	if err := fetchModule(context.Background(), srv.Client(), srv.URL, m, cache, &log); err != nil {
		t.Fatal(err)
	}
	if len(*got) != 2 {
		t.Errorf("a cached module was fetched again: %v", *got)
	}
}

func TestFetchModuleRefusesWhatIsNotAZip(t *testing.T) {
	m := modules["TCR"]
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("<html>not a zip</html>"))
	}))
	defer srv.Close()
	err := fetchModule(context.Background(), srv.Client(), srv.URL, m, t.TempDir(), &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "not a zip") {
		t.Errorf("err = %v", err)
	}
}

func TestRunWritesTheSameBytesTwice(t *testing.T) {
	m := modules["TCR"]
	srv, _ := moduleServer(t, m, fixture(t, "tcr-three.txt"))
	cache, out := t.TempDir(), filepath.Join(t.TempDir(), "themes-tcr1934")
	report := filepath.Join(t.TempDir(), "REPORT.md")
	var first [3][]byte
	for run := 0; run < 2; run++ {
		var stdout bytes.Buffer
		if err := runModule(context.Background(), srv.Client(), srv.URL, "TCR", cache, out, report, &stdout); err != nil {
			t.Fatal(err)
		}
		for i, f := range []string{"topics.jsonl", "links.jsonl"} {
			b, err := os.ReadFile(filepath.Join(out, f))
			if err != nil {
				t.Fatal(err)
			}
			if run == 0 {
				first[i] = b
			} else if !bytes.Equal(first[i], b) {
				t.Errorf("%s differs on the second run", f)
			}
		}
		rb, _ := os.ReadFile(report)
		if run == 0 {
			first[2] = rb
		} else if !bytes.Equal(first[2], rb) {
			t.Error("REPORT.md differs on the second run")
		}
		if !strings.Contains(stdout.String(), "topics:") || !strings.Contains(stdout.String(), "3") {
			t.Errorf("summary: %q", stdout.String())
		}
	}
	if len(first[0]) == 0 || len(first[1]) == 0 {
		t.Error("empty output")
	}
	// source.toml is not the tool's to write.
	if _, err := os.Stat(filepath.Join(out, "source.toml")); err == nil {
		t.Error("the tool wrote source.toml")
	}
}

func TestUnknownModule(t *testing.T) {
	err := runModule(context.Background(), http.DefaultClient, "http://127.0.0.1:1", "Nave", t.TempDir(), t.TempDir(), "", &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "unknown module") {
		t.Errorf("err = %v", err)
	}
}

func TestReportMergesModulesInOrder(t *testing.T) {
	a := reportSection("Torrey", "torrey body\n")
	b := reportSection("TCR", "tcr body\n")
	doc := mergeReport("", "Torrey", a)
	doc = mergeReport(doc, "TCR", b)
	if strings.Index(doc, "tcr body") > strings.Index(doc, "torrey body") || !strings.Contains(doc, "tcr body") {
		t.Errorf("order:\n%s", doc)
	}
	// A second run of a module replaces its section and keeps the other's.
	doc2 := mergeReport(doc, "TCR", reportSection("TCR", "tcr again\n"))
	if strings.Contains(doc2, "tcr body") || !strings.Contains(doc2, "tcr again") || !strings.Contains(doc2, "torrey body") {
		t.Errorf("replace:\n%s", doc2)
	}
	if strings.Count(doc2, "<!-- toc: none -->") != 1 {
		t.Errorf("the header repeats:\n%s", doc2)
	}
}
