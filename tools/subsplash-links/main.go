// Command subsplash-links files the Subsplash links the Governor shared
// against the sermons they are, as each row's alt_url in
// sources/brookfield-first/rows.jsonl.
//
//	subsplash-links [--links <file>] [--pdf <file>]... [--rows rows.jsonl]
//	                [--notes data/subsplash-notes.jsonl] [--cache <file>] [--progress <file>] [--dry-run]
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
// that record's row id finds the row (the record's date is its Sunday, a row
// may be dated the Saturday or the Monday); a record with no id, or an id no
// row has, finds the row by its exact date. Two rows on one date, or none, are
// reported, not written. The row's alt_url gets the full subsplash.com
// address; url (YouTube) is never changed. A row that holds another alt_url
// is a conflict and is left alone. A second run changes nothing.
//
// The report is printed to standard output and ends with "N of M linked": the
// rows that hold a Subsplash address, of all the rows.
//
// --compare <service url> writes nothing and makes no request but GETs of the
// sermons service (never Subsplash): it asks the service for the rows of
// --source (default brookfield-first) and prints
//
//	compare: N of M rows differ from the service
//	ids: a b c d e
//
// M is the rows of --rows that have a passage (the service serves a row only
// through one, its ref or a cited reference; a row with neither, or a series,
// cannot be asked about), N those
// whose title, ref, date, preacher, url or alt_url differ from the service's
// answer, or that it lacks, plus any row it serves that --rows does not hold;
// the ids line (at most five) is left out when N is 0. It exits 0 whenever it
// could compare, 1 when it could not (the service down, a bad answer).
// scripts/subsplash-hourly.sh reads it to load what the service lacks.
//
// --feed <file> --service <url> writes the job feed Postern's Jobs page shows
// (feed.go; Postern's docs/jobs.md) and nothing else, with GETs of the sermons
// service only: one item for each row of --rows, newest first, with the checks
// mail (a record of --notes names the row), link (the row holds a Subsplash
// address) and live (the service serves that address), and "here", the oldest
// row with both mail and link. It holds titles and dates and the Subsplash
// address, no notes text and no mail address. The file is written whole to a
// temporary name and renamed; if the service cannot be asked the old file is
// left as it was and the exit is 1. scripts/subsplash-hourly.sh runs it last,
// into $POSTERN_DATA/jobs/subsplash-links.json.
//
// --read-messages <inbox> --key <hex> reads the Governor's messages of a
// sermon with no notes (nonotes.go) and writes them to --passages and
// --no-notes-file, with no request at all; the run of tools/brookfield-youtube
// --merge-only that follows puts them on their rows.
//
// --read-notes writes the title and passage of the notes records (--notes) to
// --passages, as a line with the row's id and "notes" in its last column, for
// each row that has no passage and only a generic title; the run of
// tools/brookfield-youtube --merge-only that follows puts them on the row. The
// records file is git-ignored, the list is tracked, so this is what makes the
// notes' words stay on the row (notesread.go). A row that has a passage, a row
// his message or an earlier run has a line for, and a record with no passage
// are left as they are, the ones to look at named; no request is made.
//
// --progress <file> appends one JSON line to the file after a real run (never
// after a dry run), so that a page can show how far the linking has got:
//
//	{"time":"2026-11-11T19:00:00Z","records":19,"links":205,"rows_linked":29,
//	 "rows_total":354,"new":0,"oldest":"2026-02-01","newest":"2026-11-09"}
//
// records is the lines of the notes file; links the distinct sermon links read
// (short links and full addresses); rows_linked the rows that hold a Subsplash
// address, of rows_total; new the rows this run wrote; oldest and newest the
// dates of the linked rows (null when none is linked).
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
	"path/filepath"
	"strings"
	"time"

	"github.com/Jonathan-A-White/sermon-index/internal/sources"
)

const (
	defaultRows  = "sources/brookfield-first/rows.jsonl"
	defaultNotes = "data/subsplash-notes.jsonl"
	defaultCache = "data/subsplash-links-cache.jsonl"
	defaultList  = "sources/brookfield-first/passages.tsv"
	defaultNoNot = "sources/brookfield-first/no-notes.tsv"
)

// deps are the parts a test replaces.
type deps struct {
	expander func(*Cache) *Expander
	pdfToXML func(path string) ([]byte, error)
	now      func() time.Time
}

func (d deps) withDefaults() deps {
	if d.expander == nil {
		d.expander = NewExpander
	}
	if d.pdfToXML == nil {
		d.pdfToXML = pdftohtml
	}
	if d.now == nil {
		d.now = time.Now
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
	progressPath := fs.String("progress", "", "append one JSON line of progress to this file after a real run")
	dry := fs.Bool("dry-run", false, "write nothing and make no request")
	compare := fs.String("compare", "", "the sermons service's address: compare the rows with what it serves, write nothing")
	source := fs.String("source", "brookfield-first", "with --compare, the source id the service knows the rows by")
	feed := fs.String("feed", "", "write the Jobs page's feed to this file (with --service)")
	service := fs.String("service", "", "with --feed, the sermons service's address")
	messages := fs.String("read-messages", "", "the Postern inbox: read the Governor's 'No notes' messages into --passages and --no-notes-file")
	key := fs.String("key", "", "with --read-messages, the Governor's public key (hex); only messages signed by it are read")
	readNotes := fs.Bool("read-notes", false, "write the title and passage of the notes records to --passages for the rows that lack them")
	passages := fs.String("passages", defaultList, "with --read-messages, the church's list (passages.tsv)")
	noNotesFile := fs.String("no-notes-file", defaultNoNot, "no-notes.tsv, written by --read-messages and read by --feed")
	usage := "usage: subsplash-links [--links <file>] [--pdf <file>]... [--rows rows.jsonl] [--notes records.jsonl] [--cache <file>] [--progress <file>] [--dry-run]\n       subsplash-links --compare <service url> [--rows rows.jsonl] [--source brookfield-first]\n       subsplash-links --feed <file> --service <url> [--rows rows.jsonl] [--notes records.jsonl] [--no-notes-file no-notes.tsv] [--source brookfield-first]\n       subsplash-links --read-notes [--notes records.jsonl] [--rows rows.jsonl] [--passages passages.tsv]\n       subsplash-links --read-messages <inbox> --key <hex> [--rows rows.jsonl] [--passages passages.tsv] [--no-notes-file no-notes.tsv]"
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *readNotes {
		if *messages != "" || *key != "" || *feed != "" || *service != "" || *compare != "" || fs.NArg() != 0 || *linksPath != "" || len(pdfs) > 0 {
			fmt.Fprintln(stderr, usage)
			return 2
		}
		res, err := RunNotes(*notesPath, *passages, *rowsPath)
		if err != nil {
			fmt.Fprintln(stderr, "subsplash-links: read-notes:", strings.Join(strings.Fields(err.Error()), " "))
			return 1
		}
		for _, p := range res.Problems {
			fmt.Fprintln(stdout, "notes: not added:", p)
		}
		fmt.Fprintf(stdout, "notes: %d record(s), %d added, %d already there, %d not added\n", res.Records, res.Added, res.Already, len(res.Problems))
		return 0
	}
	if *messages != "" {
		if *key == "" || *feed != "" || *service != "" || *compare != "" || fs.NArg() != 0 || *linksPath != "" || len(pdfs) > 0 {
			fmt.Fprintln(stderr, usage)
			return 2
		}
		res, err := RunNoNotes(*messages, *key, *passages, *noNotesFile, *rowsPath)
		if err != nil {
			fmt.Fprintln(stderr, "subsplash-links: read-messages:", strings.Join(strings.Fields(err.Error()), " "))
			return 1
		}
		for _, p := range res.Problems {
			fmt.Fprintln(stdout, "no-notes: not read:", p)
		}
		fmt.Fprintf(stdout, "no-notes: %d sermon(s) read, %d added, %d already there, %d not read\n", res.Read, res.Added, res.Already, len(res.Problems))
		return 0
	}
	if *feed != "" || *service != "" {
		if *feed == "" || *service == "" || *compare != "" || fs.NArg() != 0 || *linksPath != "" || len(pdfs) > 0 {
			fmt.Fprintln(stderr, usage)
			return 2
		}
		if err := feedService(*rowsPath, *notesPath, *noNotesFile, *service, *source, *feed, d.now()); err != nil {
			fmt.Fprintln(stderr, "subsplash-links: feed:", strings.Join(strings.Fields(err.Error()), " "))
			return 1
		}
		return 0
	}
	if *compare != "" {
		if fs.NArg() != 0 || *linksPath != "" || len(pdfs) > 0 {
			fmt.Fprintln(stderr, usage)
			return 2
		}
		if err := compareService(*rowsPath, *compare, *source, stdout); err != nil {
			fmt.Fprintln(stderr, "subsplash-links: compare:", strings.Join(strings.Fields(err.Error()), " "))
			return 1
		}
		return 0
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
		if *progressPath != "" {
			codes := map[string]bool{}
			for c := range known {
				codes[c] = true
			}
			for _, c := range in.Codes {
				codes[c] = true
			}
			if err := appendProgress(*progressPath, progressLine(d.now(), len(recs), len(codes), rows, len(res.Set))); err != nil {
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

// progress is one line of the --progress file.
type progress struct {
	Time       string  `json:"time"`
	Records    int     `json:"records"`
	Links      int     `json:"links"`
	RowsLinked int     `json:"rows_linked"`
	RowsTotal  int     `json:"rows_total"`
	New        int     `json:"new"`
	Oldest     *string `json:"oldest"`
	Newest     *string `json:"newest"`
}

// progressLine counts the rows (as they stand after the run) that hold a
// Subsplash address, and the oldest and newest date among them.
func progressLine(now time.Time, records, links int, rows []sources.Row, written int) progress {
	p := progress{Time: now.UTC().Format(time.RFC3339), Records: records, Links: links, RowsTotal: len(rows), New: written}
	for _, r := range rows {
		if r.AltURL == nil || !strings.HasPrefix(*r.AltURL, addressPrefix) {
			continue
		}
		p.RowsLinked++
		if r.Date == nil || *r.Date == "" {
			continue
		}
		d := *r.Date
		if p.Oldest == nil || d < *p.Oldest {
			p.Oldest = &d
		}
		if p.Newest == nil || d > *p.Newest {
			p.Newest = &d
		}
	}
	return p
}

// appendProgress adds one line to the file, making its directory (0700) and
// the file (0600) if they are not there.
func appendProgress(path string, p progress) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	b, err := json.Marshal(p)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if _, err := f.Write(append(b, '\n')); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
