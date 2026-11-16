// Command subsplash-notes reads the sermon-notes alerts Postern's alerts mail
// keeps for the sermon index's own family (the Governor's forwarded Subsplash
// notes mail, parsed there into a title, a passage and a date) and writes each
// sermon's Sunday, title and passage against the Brookfield First YouTube row
// it belongs to.
//
//	subsplash-notes keygen [--key <file>]
//	subsplash-notes address [--key <file>] [--postern <url>]
//	subsplash-notes fetch [--key <file>] [--postern <url>] [--since n] [--rows rows.jsonl] [--out records.jsonl] [--notes <dir>]
//	subsplash-notes forward-confirm [--key <file>] [--postern <url>]
//	subsplash-notes browse [--cdp <url>] [--rows rows.jsonl] [--out records.jsonl] [--notes <dir>] [--links-out <file>] [--rephil <file>] [--feed <file>] [--pace 10s] [--limit n] [--verify n] [--series <url>] [--series-only] [--episode-pages]
//	subsplash-notes [--alerts <file>|-] [--rows rows.jsonl] [--out records.jsonl]
//
// keygen makes the family's key once, at ~/.config/sermon-index/postern-alerts.key
// by default (mode 0600, never replaced), and prints only its public key, which
// the Governor issues a licence to on Postern's Key screen. The key never goes
// in the repo, which is public, and is never printed.
//
// address asks Postern for the family's alert address (POST
// /api/alerts/address, made on the first ask) and prints it: the address the
// Gmail filter forwards the notes mail to. fetch lists the family's alerts
// (GET /api/alerts?since=n), writes the records, and only once the records
// file is in place claims (DELETE /api/alerts/{id}) each alert whose record it
// holds; an alert that is dropped, unmatched or disagrees with a kept record
// is reported and left unclaimed, and Postern forgets it after 30 days. Every
// request is signed by the key with a Postern2 proof (Postern's docs/api.md,
// Authentication). --postern defaults to $POSTERN_URL.
//
// forward-confirm lists the family's alerts the same way and prints the link of
// the newest forward-confirm alert (Gmail's confirmation of a forwarding
// address), which the Governor opens to confirm the address; with none it says
// so and exits 1. It prints only a link
// https://<host>/mail/vf-<token> on one of Postern's confirm hosts (see
// confirmHosts; any other link is refused and not printed), makes no request but
// the read of the alerts, and neither visits the link nor claims the alert.
//
// browse is the second way in, through the signed-in browser `mw browser open`
// starts on the Laptop (browser.go, browse.go, cdp.go): see browse below.
//
// With no subcommand the tool reads a saved reply of GET /api/alerts, {"alerts":
// [...], "next": n} ("-" reads it from standard input), and claims nothing.
//
// Only an alert of kind sermon-notes is read, and only its title, reference,
// date and, when it has notes, notes_sha256 and notes_size: the reader has no
// place for a mail's body, and an alert that carries any field besides those
// is dropped and reported.
//
// The notes PDF. An alert with notes names them by notes_sha256 and
// notes_size, and their bytes are at the signed GET /api/alerts/{id}/notes.
// For each alert fetch will record it downloads them, checks the size and the
// sha256, and keeps the PDF as <row id>.pdf in the notes directory (--notes;
// default ~/.local/state/sermon-index/notes, mode 0700, files 0600). That is
// on the Laptop alone and outside the checkout: the PDF holds the church's
// notes and the Governor's own fill-ins, so it is NEVER COMMITTED, the repo
// being public and keeping no sermon's words, and a notes directory inside a
// git work tree is refused. The order is: the PDF on disk, then the records
// file in place, and only then the alert claimed. A failed download, a wrong
// size or a wrong sha256 is reported, writes no record for that alert and
// leaves it unclaimed; the other alerts go on. A file already there with the
// same sha256 is not downloaded again. One with another sha256 is the notes of
// a second mail for the same sermon (the same title, passage and Sunday as the
// record kept for the row, the church's PDF made again): the newer mail's
// notes are kept as <row id>.pdf and the older beside them as
// <row id>.prev.pdf. A mail for another sermon that falls on the row is
// reported with both titles and writes nothing, and so is a file with no
// record to say it is the same sermon's; each says what to do, and the
// summary line counts them ("n not replaced"). The no-subcommand reader
// accepts the two fields and downloads nothing.
//
// The date an alert gives can be the Saturday or the Monday of the Sunday it
// is about (the mail's date is when the notes went out), so a Saturday or a
// Monday is taken to the Sunday one day away; any other weekday is reported.
// The Sunday must be within one day of exactly one YouTube row; none, or more
// than one, is reported and the alert is not written. A second run over the
// same alerts writes the same bytes; a record already written is never
// replaced by an alert that says otherwise (that is reported).
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Jonathan-A-White/sermon-index/internal/sources"
)

const (
	defaultRows   = "sources/brookfield-first/rows.jsonl"
	defaultOut    = "data/subsplash-notes.jsonl"
	defaultLinks  = "data/subsplash-browser-links.txt"
	defaultRephil = "data/morning-rephil.jsonl"
)

func main() {
	os.Exit(cli(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

// cli runs one command line and returns its exit code: 0 done, 1 failed, 2
// a usage error.
func cli(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) > 0 {
		switch args[0] {
		case "keygen":
			return keygenCmd(args[1:], stdout, stderr)
		case "address":
			return addressCmd(args[1:], stdout, stderr)
		case "fetch":
			return fetchCmd(args[1:], stdout, stderr)
		case "forward-confirm":
			return forwardConfirmCmd(args[1:], stdout, stderr)
		case "browse":
			return browseCmd(args[1:], stdout, stderr)
		}
	}
	return readCmd(args, stdin, stdout, stderr)
}

// fail reports err in one line.
func fail(stderr io.Writer, err error) int {
	fmt.Fprintln(stderr, "subsplash-notes:", strings.Join(strings.Fields(err.Error()), " "))
	return 1
}

func flags(name string, stderr io.Writer) *flag.FlagSet {
	fs := flag.NewFlagSet(strings.TrimSpace("subsplash-notes "+name), flag.ContinueOnError)
	fs.SetOutput(stderr)
	return fs
}

func parse(fs *flag.FlagSet, args []string, stderr io.Writer, usage string) bool {
	if err := fs.Parse(args); err != nil {
		return false
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "usage: "+usage)
		return false
	}
	return true
}

func keygenCmd(args []string, stdout, stderr io.Writer) int {
	fs := flags("keygen", stderr)
	keyPath := fs.String("key", DefaultKeyPath(), "where to write the new key (never replaced)")
	if !parse(fs, args, stderr, "subsplash-notes keygen [--key <file>]") {
		return 2
	}
	pub, err := Keygen(*keyPath)
	if err != nil {
		return fail(stderr, err)
	}
	fmt.Fprintln(stdout, pub)
	return 0
}

// postern is the client for --postern with the key at keyPath.
func postern(base, keyPath string) (*Client, error) {
	if base == "" {
		return nil, errors.New("no Postern named: give --postern <url> or set POSTERN_URL")
	}
	key, err := LoadKey(keyPath)
	if err != nil {
		return nil, err
	}
	return &Client{Base: base, Key: key, HTTP: &http.Client{Timeout: 30 * time.Second}}, nil
}

func addressCmd(args []string, stdout, stderr io.Writer) int {
	fs := flags("address", stderr)
	keyPath := fs.String("key", DefaultKeyPath(), "the key keygen wrote")
	base := fs.String("postern", os.Getenv("POSTERN_URL"), "Postern's backend (default $POSTERN_URL)")
	if !parse(fs, args, stderr, "subsplash-notes address [--key <file>] [--postern <url>]") {
		return 2
	}
	c, err := postern(*base, *keyPath)
	if err != nil {
		return fail(stderr, err)
	}
	addr, err := c.Address(context.Background())
	if err != nil {
		return fail(stderr, err)
	}
	fmt.Fprintln(stdout, addr)
	return 0
}

// forwardConfirmCmd prints the link of the newest forward-confirm alert, so the
// forwarding address the Governor added in Gmail can be confirmed. It reads
// the alerts (GET /api/alerts) and nothing else: it does not visit the link,
// claim the alert or change anything.
func forwardConfirmCmd(args []string, stdout, stderr io.Writer) int {
	fs := flags("forward-confirm", stderr)
	keyPath := fs.String("key", DefaultKeyPath(), "the key keygen wrote")
	base := fs.String("postern", os.Getenv("POSTERN_URL"), "Postern's backend (default $POSTERN_URL)")
	if !parse(fs, args, stderr, "subsplash-notes forward-confirm [--key <file>] [--postern <url>]") {
		return 2
	}
	c, err := postern(*base, *keyPath)
	if err != nil {
		return fail(stderr, err)
	}
	body, err := c.Alerts(context.Background(), 0)
	if err != nil {
		return fail(stderr, err)
	}
	link, _, err := ForwardConfirmLink(body)
	if err != nil {
		return fail(stderr, err)
	}
	fmt.Fprintln(stdout, link)
	return 0
}

func fetchCmd(args []string, stdout, stderr io.Writer) int {
	fs := flags("fetch", stderr)
	keyPath := fs.String("key", DefaultKeyPath(), "the key keygen wrote")
	base := fs.String("postern", os.Getenv("POSTERN_URL"), "Postern's backend (default $POSTERN_URL)")
	since := fs.Int("since", 0, "list the alerts above this seq (a claimed alert is gone, so 0 lists what is left)")
	rowsPath := fs.String("rows", defaultRows, "the YouTube rows to match against")
	out := fs.String("out", defaultOut, "the records file to write; read first and merged with")
	notesDir := fs.String("notes", DefaultNotesDir(), "where the notes PDFs are kept: outside the checkout, never committed")
	if !parse(fs, args, stderr, "subsplash-notes fetch [--key <file>] [--postern <url>] [--since n] [--rows rows.jsonl] [--out records.jsonl] [--notes <dir>]") {
		return 2
	}
	c, err := postern(*base, *keyPath)
	if err != nil {
		return fail(stderr, err)
	}
	store := &NotesStore{Dir: *notesDir, Client: c}
	if err := store.checkDir(); err != nil {
		return fail(stderr, err)
	}
	rows, err := readRows(*rowsPath)
	if err != nil {
		return fail(stderr, err)
	}
	existing, err := readRecords(*out)
	if err != nil {
		return fail(stderr, err)
	}
	ctx := context.Background()
	body, err := c.Alerts(ctx, *since)
	if err != nil {
		return fail(stderr, err)
	}
	// An alert with notes is recorded only once its PDF is on disk.
	records, report, recorded, refused, err := merge(bytes.NewReader(body), rows, existing, func(n Note, rec Record, same bool) (string, error) {
		if n.NotesSHA256 == "" {
			return "", nil
		}
		return store.Keep(ctx, n, rec.ID, same)
	})
	if err != nil {
		return fail(stderr, err)
	}
	// The PDFs and then the records are on disk before any alert is claimed: a
	// claimed alert is gone from Postern, so neither must be lost with it.
	if err := writeRecords(*out, records); err != nil {
		return fail(stderr, fmt.Errorf("%w; no alert claimed", err))
	}
	for _, line := range report {
		fmt.Fprintln(stderr, line)
	}
	claimed, code := 0, 0
	for _, id := range recorded {
		if err := c.Claim(ctx, id); err != nil {
			code = fail(stderr, fmt.Errorf("alert %s recorded but not claimed: %w", id, err))
			continue
		}
		claimed++
	}
	fmt.Fprintf(stdout, "%d records in %s (%d before), %d reported, %d not replaced, %d claimed\n", len(records), *out, len(existing), len(report), refused, claimed)
	if store.Asked > 0 {
		fmt.Fprintf(stdout, "%d notes downloaded to %s\n", store.Asked, store.Dir)
	}
	return code
}

// browseCmd is subsplash-notes browse. THE TERMS: Subsplash's terms bar
// automated means; the Governor allowed this one use, on his own signed-in
// account in his own browser, at a polite pace, accepting the account's risk
// (his B on card direct:09581219, 2026-11-16). So it drives that one tab and
// no other, at least 10 s between page loads, one load at a time, reads only
// the sermon list, each sermon's page and its notes, and stops at the first
// login prompt, captcha, HTTP 429 or 403, or page it did not expect.
//
// It walks the list newest to oldest. A sermon already sent by hand (a record
// by its title, and its address on a row or in the links file) is not read;
// any other is: its page, its notes, and then the record the mail would have
// made in --out, the notes' text as <row id>.txt in --notes beside the PDFs
// (never committed), and the sermon's full address in --links-out, which
// `subsplash-links --links <file>` takes and files with no request (it holds
// the address already). A Morning RePhil episode is written to --rephil
// (oldest first, number, title, date, share link). --feed is the Jobs page's
// file. --verify n reads the newest n sermons that already have a mail-made
// record, says how each compares, and writes nothing.
func browseCmd(args []string, stdout, stderr io.Writer) int {
	fs := flags("browse", stderr)
	endpoint := fs.String("cdp", DefaultCDP, "the browser's DevTools address (mw browser status): this machine only")
	rowsPath := fs.String("rows", defaultRows, "the YouTube rows to match against")
	out := fs.String("out", defaultOut, "the records file to write; read first and merged with")
	notesDir := fs.String("notes", DefaultNotesDir(), "where the notes are kept: outside the checkout, never committed")
	linksOut := fs.String("links-out", defaultLinks, "the file of sermon addresses for subsplash-links --links")
	rephil := fs.String("rephil", defaultRephil, "the Morning RePhil episodes file")
	feed := fs.String("feed", DefaultFeed(), "the Jobs page's feed file ('' for none)")
	pace := fs.Duration("pace", MinPace, "the time between two page loads (never under 10s)")
	limit := fs.Int("limit", 0, "the most sermon and episode pages to read (0: all)")
	verify := fs.Int("verify", 0, "compare the newest n sermons already recorded by mail, writing nothing")
	series := fs.String("series", seriesURL, "the Morning RePhil series page to read for every episode ('' for none)")
	seriesOnly := fs.Bool("series-only", false, "read the series page and not the sermon list")
	episodePages := fs.Bool("episode-pages", false, "also load each new episode's own page, to keep any text it carries")
	if !parse(fs, args, stderr, "subsplash-notes browse [--cdp <url>] [--rows rows.jsonl] [--out records.jsonl] [--notes <dir>] [--links-out <file>] [--rephil <file>] [--feed <file>] [--pace 10s] [--limit n] [--verify n] [--series <url>] [--series-only] [--episode-pages]") {
		return 2
	}
	rows, err := readRows(*rowsPath)
	if err != nil {
		return fail(stderr, err)
	}
	b, err := NewCDP(*endpoint)
	if err != nil {
		return fail(stderr, err)
	}
	defer b.Close()
	notes := *notesDir
	cfg := Config{Rows: rows, Out: *out, NotesDir: notes, LinksOut: *linksOut, Rephil: *rephil, Feed: *feed,
		Interval: *pace, Limit: *limit, Verify: *verify, Series: *series, SeriesOnly: *seriesOnly, EpisodePages: *episodePages, PDFText: func(id string) (string, error) { return pdfText(filepath.Join(notes, id+".pdf")) }}
	res, err := Browse(context.Background(), b, cfg, stdout)
	fmt.Fprintf(stdout, "%d page loads, %d sermon and episode pages read; %d recorded, %d sermon links, %d episodes", res.Loads, res.Visited, res.Recorded, res.Linked, res.Episodes)
	if *verify > 0 {
		fmt.Fprintf(stdout, "; %d compared, %d differ", res.Verified, res.Differ)
	}
	fmt.Fprintln(stdout)
	var st Stop
	if errors.As(err, &st) {
		fmt.Fprintln(stderr, "subsplash-notes browse:", st.Error(), "(nothing more was written; sign in again if asked, then run it again: it resumes here)")
		return 1
	}
	if err != nil {
		return fail(stderr, err)
	}
	return 0
}

func readCmd(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flags("", stderr)
	alertsPath := fs.String("alerts", "-", "the alerts JSON (the body of GET /api/alerts); - is standard input")
	rowsPath := fs.String("rows", defaultRows, "the YouTube rows to match against")
	out := fs.String("out", defaultOut, "the records file to write; read first and merged with")
	if !parse(fs, args, stderr, "subsplash-notes [keygen|address|fetch|forward-confirm] ... or subsplash-notes [--alerts <file>|-] [--rows rows.jsonl] [--out records.jsonl]") {
		return 2
	}
	if err := run(*alertsPath, *rowsPath, *out, stdin, stdout, stderr); err != nil {
		return fail(stderr, err)
	}
	return 0
}

func run(alertsPath, rowsPath, out string, stdin io.Reader, stdout, stderr io.Writer) error {
	var alerts io.Reader = stdin
	if alertsPath != "-" {
		f, err := os.Open(alertsPath)
		if err != nil {
			return err
		}
		defer f.Close()
		alerts = f
	}
	rows, err := readRows(rowsPath)
	if err != nil {
		return err
	}
	existing, err := readRecords(out)
	if err != nil {
		return err
	}
	records, report, _, refused, err := merge(alerts, rows, existing, nil)
	if err != nil {
		return err
	}
	if err := writeRecords(out, records); err != nil {
		return err
	}
	for _, line := range report {
		fmt.Fprintln(stderr, line)
	}
	fmt.Fprintf(stdout, "%d records in %s (%d before), %d reported, %d not replaced\n", len(records), out, len(existing), len(report), refused)
	return nil
}

// writeRecords writes the records, one JSON object a line, to a temporary
// file beside out, syncs it and renames it into place: out is either the old
// file or the whole new one, and on a nil return the new one is on disk.
func writeRecords(out string, records []Record) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	for _, r := range records {
		if err := enc.Encode(r); err != nil {
			return err
		}
	}
	dir := filepath.Dir(out)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("writing the records: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".subsplash-notes-*")
	if err != nil {
		return fmt.Errorf("writing the records: %w", err)
	}
	defer os.Remove(tmp.Name())
	_, err = tmp.Write(buf.Bytes())
	if err == nil {
		err = tmp.Chmod(0o644)
	}
	if err == nil {
		err = tmp.Sync()
	}
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Rename(tmp.Name(), out)
	}
	if err != nil {
		return fmt.Errorf("writing the records to %s: %w", out, err)
	}
	if d, err := os.Open(dir); err == nil {
		d.Sync()
		d.Close()
	}
	return nil
}

// Run reads the alerts, matches each note to a row and merges the records
// with the existing ones by row id. report lists, one line each, what was
// dropped or not guessed.
func Run(alerts io.Reader, rows []sources.Row, existing []Record) (records []Record, report []string, err error) {
	records, report, _, _, err = merge(alerts, rows, existing, nil)
	return records, report, err
}

// merge is Run, also naming the alerts whose record the merged records hold
// (written now, or kept already and the same) in the order they came: the
// alerts that may be claimed once the records are on disk. refused counts the
// alerts left as they were because a record or a file was there and would not
// be written over (the summary line says how many). keep, when not nil, is
// asked of each note about to be recorded (its record new or already kept and
// the same; same says which) and an error from it is reported, the alert
// neither recorded nor named: fetch keeps the notes PDF there. A non-empty
// line it returns is reported too, as news and not as a problem.
func merge(alerts io.Reader, rows []sources.Row, existing []Record, keep func(n Note, rec Record, same bool) (string, error)) (records []Record, report, recorded []string, refused int, err error) {
	notes, problems, err := ReadAlerts(alerts)
	if err != nil {
		return nil, nil, nil, 0, err
	}
	for _, p := range problems {
		report = append(report, reported(p.AlertID, p.Why, p.Says))
	}
	byID := map[string]Record{}
	for _, r := range existing {
		byID[r.ID] = r
	}
	for _, n := range notes {
		rec, how, err := MatchNote(n, rows)
		if err != nil {
			report = append(report, reported(n.AlertID, "not written: "+err.Error(), n.Says()))
			continue
		}
		if how != "" {
			report = append(report, reported(n.AlertID, how, n.Says()))
		}
		have, same := byID[rec.ID]
		if same && have != rec {
			refused++
			report = append(report, reported(n.AlertID, fmt.Sprintf("disagrees with the record kept for row %s (kept '%s', %s, %s): another sermon, so its notes and record are not written over that row's; not replaced (if the kept record is the wrong one, delete its line from the records file and the next run writes this one)", rec.ID, have.Title, have.Ref, have.Date), n.Says()))
			continue
		}
		news, err := keepNotes(keep, n, rec, same)
		if err != nil {
			var nr notReplaced
			if errors.As(err, &nr) {
				refused++
			}
			report = append(report, reported(n.AlertID, "not recorded: "+err.Error(), n.Says()))
			continue
		}
		if news != "" {
			report = append(report, reported(n.AlertID, news, n.Says()))
		}
		if !same {
			byID[rec.ID] = rec
		}
		recorded = append(recorded, n.AlertID)
	}
	for _, r := range byID {
		records = append(records, r)
	}
	sort.Slice(records, func(i, j int) bool {
		if records[i].Date != records[j].Date {
			return records[i].Date < records[j].Date
		}
		return records[i].ID < records[j].ID
	})
	return records, report, recorded, refused, nil
}

// reported is one line of the report: the alert, what happened to it and,
// when there is any, what the alert itself says (its title, passage and mail
// date), so the line alone tells which sermon's notes it carries.
func reported(id, what, says string) string {
	if says == "" {
		return fmt.Sprintf("alert %s: %s", id, what)
	}
	return fmt.Sprintf("alert %s: %s [alert says: %s]", id, what, says)
}

func keepNotes(keep func(Note, Record, bool) (string, error), n Note, rec Record, same bool) (string, error) {
	if keep == nil {
		return "", nil
	}
	return keep(n, rec, same)
}

func readRows(path string) ([]sources.Row, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var rows []sources.Row
	sc := bufio.NewScanner(bytes.NewReader(b))
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var r sources.Row
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			return nil, fmt.Errorf("%s line %d: %w", path, n, err)
		}
		rows = append(rows, r)
	}
	return rows, sc.Err()
}

func readRecords(path string) ([]Record, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var recs []Record
	for n, line := range strings.Split(string(b), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var r Record
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			return nil, fmt.Errorf("%s line %d: %w", path, n+1, err)
		}
		recs = append(recs, r)
	}
	return recs, nil
}
