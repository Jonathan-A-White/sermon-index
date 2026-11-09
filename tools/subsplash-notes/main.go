// Command subsplash-notes reads the sermon-notes alerts Postern's alerts mail
// keeps for the sermon index's own family (the Governor's forwarded Subsplash
// notes mail, parsed there into a title, a passage and a date) and writes each
// sermon's Sunday, title and passage against the Brookfield First YouTube row
// it belongs to:
//
//	subsplash-notes --alerts alerts.json --rows sources/brookfield-first/rows.jsonl --out data/subsplash-notes.jsonl
//
// The alerts file is the body of GET /api/alerts, {"alerts": [...], "next": n}
// ("-" reads it from standard input). Only an alert of kind sermon-notes is
// read, and only its title, reference and date: the reader has no place for a
// mail's body or the notes PDF attached to it, and an alert that carries any
// field besides those is dropped and reported. Nothing is fetched or claimed
// here: the tool reads what it is given.
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
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Jonathan-A-White/sermon-index/internal/sources"
)

func main() {
	alertsPath := flag.String("alerts", "-", "the alerts JSON (the body of GET /api/alerts); - is standard input")
	rowsPath := flag.String("rows", "sources/brookfield-first/rows.jsonl", "the YouTube rows to match against")
	out := flag.String("out", "data/subsplash-notes.jsonl", "the records file to write; read first and merged with")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "usage: subsplash-notes [--alerts <file>|-] [--rows rows.jsonl] [--out records.jsonl]")
		os.Exit(2)
	}
	if err := run(*alertsPath, *rowsPath, *out, os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "subsplash-notes:", err)
		os.Exit(1)
	}
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
	records, report, err := Run(alerts, rows, existing)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	for _, r := range records {
		enc := json.NewEncoder(&buf)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(r); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(out, buf.Bytes(), 0o644); err != nil {
		return err
	}
	for _, line := range report {
		fmt.Fprintln(stderr, line)
	}
	fmt.Fprintf(stdout, "%d records in %s (%d before), %d reported\n", len(records), out, len(existing), len(report))
	return nil
}

// Run reads the alerts, matches each note to a row and merges the records
// with the existing ones by row id. report lists, one line each, what was
// dropped or not guessed.
func Run(alerts io.Reader, rows []sources.Row, existing []Record) (records []Record, report []string, err error) {
	notes, problems, err := ReadAlerts(alerts)
	if err != nil {
		return nil, nil, err
	}
	for _, p := range problems {
		report = append(report, fmt.Sprintf("alert %s: %s", p.AlertID, p.Why))
	}
	byID := map[string]Record{}
	for _, r := range existing {
		byID[r.ID] = r
	}
	for _, n := range notes {
		rec, err := Match(n, rows)
		if err != nil {
			report = append(report, fmt.Sprintf("alert %s: not written: %v", n.AlertID, err))
			continue
		}
		if have, ok := byID[rec.ID]; ok {
			if have != rec {
				report = append(report, fmt.Sprintf("alert %s: disagrees with the record kept for row %s (kept '%s', %s, %s); not replaced", n.AlertID, rec.ID, have.Title, have.Ref, have.Date))
			}
			continue
		}
		byID[rec.ID] = rec
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
	return records, report, nil
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
