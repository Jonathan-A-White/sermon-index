// Command subsplash-links files the Subsplash links the Governor shared
// against the sermons they are, as each row's alt_url in
// sources/brookfield-first/rows.jsonl.
//
//	subsplash-links [--links <file>] [--pdf <file>]... [--rows rows.jsonl]
//	                [--notes data/subsplash-notes.jsonl] [--cache <file>] [--dry-run]
//
// The links come from --links (a file of short links, one a line, like
// https://brookfieldfirst.subspla.sh/q59d572; a full subsplash.com sermon
// address is also taken) and from --pdf (a search-result PDF from any engine,
// read with pdftohtml -stdout -xml -i, whose addresses are already in full).
//
// THE TERMS. Subsplash's terms (subsplash.com/legal/terms-and-conditions,
// A.5(d)) bar automated means to download data from the Services, and
// docs/adding-a-source.md step 2 says no crawler. The Governor allowed this
// one narrow use and no other: ONE HTTP HEAD for each short link he shared,
// to read where it points, nothing downloaded. So the tool
//
//   - requests only https://brookfieldfirst.subspla.sh/<code>, for a code in
//     the input; any other host is refused before a request is made;
//   - sends HEAD, never follows the redirect, never reads a body, and sends
//     no GET;
//   - waits at least a second between requests, and stops after a 403 or 429;
//   - sends a generic User-Agent;
//   - keeps code -> full address in the cache file, so that no link is ever
//     requested twice (an address a PDF gave in full needs no request);
//   - makes no request at all under --dry-run.
//
// The redirect must lead to subsplash.com/u/brookfieldfirst/media/d/<code>-<slug>;
// anything else (a series share) is reported and written nowhere.
//
// MATCHING. The slug is matched to the slug of a row's title; failing that,
// to the slug of a title in the notes records (tools/subsplash-notes), and
// that record's date finds the row. Two rows on one date, or none, are
// reported, not written. The row's alt_url gets the full subsplash.com
// address; url (YouTube) is never changed. A row that holds another alt_url
// is a conflict and is left alone. A second run changes nothing.
//
// The report is printed to standard output and ends with "N of M linked": the
// rows that hold a Subsplash address, of all the rows.
//
// The link file, the PDFs and the cache are private: never commit them.
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/Jonathan-A-White/sermon-index/internal/sources"
)

const (
	defaultRows  = "sources/brookfield-first/rows.jsonl"
	defaultNotes = "data/subsplash-notes.jsonl"
	defaultCache = "data/subsplash-links-cache.jsonl"
)

// deps are the parts a test replaces.
type deps struct {
	expander func(*Cache) *Expander
	pdfToXML func(path string) ([]byte, error)
}

func (d deps) withDefaults() deps {
	if d.expander == nil {
		d.expander = NewExpander
	}
	if d.pdfToXML == nil {
		d.pdfToXML = pdftohtml
	}
	return d
}

// pdftohtml is the XML of a PDF, from poppler's pdftohtml.
func pdftohtml(path string) ([]byte, error) {
	out, err := exec.Command("pdftohtml", "-stdout", "-xml", "-i", path).Output()
	if err != nil {
		return nil, fmt.Errorf("pdftohtml %s: %w", path, err)
	}
	return out, nil
}

func main() {
	os.Exit(cli(os.Args[1:], os.Stdout, os.Stderr, deps{}))
}

type listFlag []string

func (l *listFlag) String() string     { return strings.Join(*l, ",") }
func (l *listFlag) Set(v string) error { *l = append(*l, v); return nil }

// cli runs one command line: 0 done, 1 something failed, 2 a usage error.
func cli(args []string, stdout, stderr io.Writer, d deps) int {
	d = d.withDefaults()
	fs := flag.NewFlagSet("subsplash-links", flag.ContinueOnError)
	fs.SetOutput(stderr)
	linksPath := fs.String("links", "", "a file of short links, one a line")
	var pdfs listFlag
	fs.Var(&pdfs, "pdf", "a search-result PDF (repeatable)")
	rowsPath := fs.String("rows", defaultRows, "the rows to write alt_url to")
	notesPath := fs.String("notes", defaultNotes, "the records file of tools/subsplash-notes (may be missing)")
	cachePath := fs.String("cache", defaultCache, "the cache of code -> full address")
	dry := fs.Bool("dry-run", false, "write nothing and make no request")
	usage := "usage: subsplash-links [--links <file>] [--pdf <file>]... [--rows rows.jsonl] [--notes records.jsonl] [--cache <file>] [--dry-run]"
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 || (*linksPath == "" && len(pdfs) == 0) {
		fmt.Fprintln(stderr, usage)
		return 2
	}
	fail := func(err error) int {
		fmt.Fprintln(stderr, "subsplash-links:", strings.Join(strings.Fields(err.Error()), " "))
		return 1
	}

	var in Inputs
	if *linksPath != "" {
		f, err := os.Open(*linksPath)
		if err != nil {
			return fail(err)
		}
		in = ReadLinkFile(f)
		f.Close()
	}
	var report []string
	report = append(report, in.Problems...)
	addrs := append([]Address(nil), in.Addresses...)
	for _, p := range pdfs {
		xml, err := d.pdfToXML(p)
		if err != nil {
			return fail(err)
		}
		found, other := AddressesInXML(xml)
		addrs = append(addrs, found...)
		for _, o := range other {
			report = append(report, fmt.Sprintf("%s: %s: not a sermon page (a series?): not written", p, o))
		}
	}

	rf, err := readRowsFile(*rowsPath)
	if err != nil {
		return fail(err)
	}
	recs, err := ReadRecords(*notesPath)
	if err != nil {
		return fail(err)
	}
	cache, err := LoadCache(*cachePath)
	if err != nil {
		return fail(err)
	}

	// An address given in full needs no request: the cache keeps it.
	cacheChanged := false
	known := map[string]bool{}
	for _, a := range addrs {
		known[a.Code] = true
		if got, ok := cache.Get(a.Code); !ok || got != a.URL {
			cache.Put(a.Code, a.URL)
			cacheChanged = true
		}
	}

	code := 0
	var ex *Expander
	if !*dry {
		ex = d.expander(cache)
	}
	for _, c := range in.Codes {
		if known[c] {
			continue
		}
		dest, ok := cache.Get(c)
		if !ok {
			if *dry {
				report = append(report, fmt.Sprintf("%s: not in the cache, and a dry run makes no request: not expanded", c))
				continue
			}
			dest, err = ex.Expand(c)
			if err != nil {
				report = append(report, fmt.Sprintf("%s: %v", c, err))
				code = 1
				continue
			}
		}
		a, err := ParseAddress(dest)
		if err != nil || a.Code != c {
			report = append(report, fmt.Sprintf("%s: points to %s, not to a sermon page of the church (a series?): not written", c, dest))
			continue
		}
		addrs = append(addrs, a)
	}

	res := Link(rf.rows, recs, addrs)
	report = append(report, res.Report...)

	rows := rf.apply(res.Set)
	linked := 0
	for _, r := range rows {
		if r.AltURL != nil && strings.HasPrefix(*r.AltURL, addressPrefix) {
			linked++
		}
	}

	for _, line := range report {
		fmt.Fprintln(stdout, line)
	}
	fmt.Fprintf(stdout, "%d rows to write, %d already linked\n", len(res.Set), res.Already)
	if *dry {
		fmt.Fprintln(stdout, "dry run: nothing written, no request made")
	} else {
		if len(res.Set) > 0 {
			if err := writeFile(*rowsPath, rf.bytes(res.Set)); err != nil {
				return fail(err)
			}
		}
		if cacheChanged {
			if err := cache.Save(); err != nil {
				return fail(err)
			}
		}
	}
	fmt.Fprintf(stdout, "%d of %d linked\n", linked, len(rows))
	return code
}

// rowsFile is rows.jsonl read line by line, so that a line that is not
// changed is written back byte for byte.
type rowsFile struct {
	lines []string
	rows  []sources.Row
	at    []int // rows[i] is lines[at[i]]
}

func readRowsFile(path string) (*rowsFile, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	rf := &rowsFile{}
	sc := bufio.NewScanner(bytes.NewReader(b))
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for n := 0; sc.Scan(); n++ {
		text := sc.Text()
		rf.lines = append(rf.lines, text)
		if strings.TrimSpace(text) == "" {
			continue
		}
		// An unknown field is refused: a rewritten line must lose nothing.
		dec := json.NewDecoder(strings.NewReader(text))
		dec.DisallowUnknownFields()
		var r sources.Row
		if err := dec.Decode(&r); err != nil {
			return nil, fmt.Errorf("%s line %d: %w", path, n+1, err)
		}
		rf.rows = append(rf.rows, r)
		rf.at = append(rf.at, n)
	}
	return rf, sc.Err()
}

// apply is the rows with the alt_url of set (row id -> address) given.
func (rf *rowsFile) apply(set map[string]string) []sources.Row {
	out := make([]sources.Row, len(rf.rows))
	copy(out, rf.rows)
	for i := range out {
		if u, ok := set[out[i].ID]; ok {
			u := u
			out[i].AltURL = &u
		}
	}
	return out
}

// bytes is the file with the rows of set rewritten and every other line as read.
func (rf *rowsFile) bytes(set map[string]string) []byte {
	lines := append([]string(nil), rf.lines...)
	for i, r := range rf.rows {
		u, ok := set[r.ID]
		if !ok {
			continue
		}
		r.AltURL = &u
		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		enc.SetEscapeHTML(false)
		enc.Encode(r)
		lines[rf.at[i]] = strings.TrimRight(buf.String(), "\n")
	}
	return []byte(strings.Join(lines, "\n") + "\n")
}

// readRows is the rows of a rows.jsonl.
func readRows(path string) ([]sources.Row, error) {
	rf, err := readRowsFile(path)
	if err != nil {
		return nil, err
	}
	return rf.rows, nil
}
