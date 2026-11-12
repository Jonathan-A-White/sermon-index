package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/Jonathan-A-White/sermon-index/internal/passage"
)

// The notes mails give a sermon its title and passage, but the records file
// they are kept in (data/subsplash-notes.jsonl) is git-ignored, so nothing the
// refresh or a clone does can carry them to the sermon's row. RunNotes writes
// each record's title and passage to the church's list, passages.tsv, as a line
// with the row's id and "notes" in its last column ("from"); the run of
// tools/brookfield-youtube --merge-only that follows puts them on the row. The
// list is tracked, so the row keeps them through the next refresh.
//
// A record is written only for a row that has no passage and a generic title
// ("Brookfield First - Online Service | October 19, 2025"). A row that has a
// passage, a row that has a title of its own, a row the list already has a line
// for (his message's, or an earlier run's), a record with no passage and a
// record no single row holds are left as they are; the ones that call for a
// person's look are named with why.

// NotesResult is what RunNotes did.
type NotesResult struct {
	Records  int      // records read
	Added    int      // lines newly written to the list
	Already  int      // records whose row already holds the same words
	Problems []string // one line each: the row id, the title and why
}

// listEntry is a line of passages.tsv as RunNotes needs it.
type listEntry struct{ Date, Title, Passage, Row string }

// readListEntries reads every line of the list, by line, and requires the row column.
func readListEntries(path string) ([]listEntry, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out []listEntry
	header := false
	for n, line := range strings.Split(string(b), "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") {
			continue
		}
		f := strings.Split(line, "\t")
		if !header {
			if len(f) < 6 || strings.TrimSpace(f[5]) != "row" {
				return nil, fmt.Errorf("%s:%d: the header has no row column (series, date, title, speaker, passage, row)", path, n+1)
			}
			header = true
			continue
		}
		if len(f) < 5 {
			continue
		}
		e := listEntry{Date: strings.TrimSpace(f[1]), Title: strings.TrimSpace(f[2]), Passage: strings.TrimSpace(f[4])}
		if len(f) > 5 {
			e.Row = strings.TrimSpace(f[5])
		}
		out = append(out, e)
	}
	if !header {
		return nil, fmt.Errorf("%s: no header line", path)
	}
	return out, nil
}

// osisText writes an OSIS reference of a record ("2Cor.13", "Matt.6.9",
// "Eph.4.1-Eph.5.14") as the text the list takes ("2 Corinthians 13",
// "Matthew 6:9", "Ephesians 4:1-5:14"). A reference over two books, or whose
// book the book table does not read back, is an error.
func osisText(ref string) (string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", errors.New("no passage")
	}
	if _, err := passage.Parse(ref); err != nil {
		return "", err
	}
	type place struct{ book, c, v string }
	split := func(s string) (place, error) {
		p := strings.Split(s, ".")
		if len(p) < 2 || len(p) > 3 {
			return place{}, fmt.Errorf("passage %q is not book.chapter[.verse]", s)
		}
		pl := place{book: p[0], c: p[1]}
		if len(p) == 3 {
			pl.v = p[2]
		}
		return pl, nil
	}
	ends := strings.Split(ref, "-")
	if len(ends) > 2 {
		return "", fmt.Errorf("passage %q has more than one range", ref)
	}
	a, err := split(ends[0])
	if err != nil {
		return "", err
	}
	b, ok := passage.Lookup(a.book)
	if !ok {
		return "", fmt.Errorf("passage %q: no such book", ref)
	}
	if back, ok := passage.BookOSIS(b.Name); !ok || back != b.OSIS {
		return "", fmt.Errorf("passage %q: the book table does not read %q back", ref, b.Name)
	}
	text := b.Name + " " + a.c
	if a.v != "" {
		text += ":" + a.v
	}
	if len(ends) == 1 {
		return text, nil
	}
	z, err := split(ends[1])
	if err != nil {
		return "", err
	}
	if zb, ok := passage.Lookup(z.book); !ok || zb.OSIS != b.OSIS {
		return "", fmt.Errorf("passage %q runs from one book to another", ref)
	}
	switch {
	case a.v == "" && z.v == "":
		return text + "-" + z.c, nil
	case a.v != "" && z.v != "" && a.c == z.c:
		return text + "-" + z.v, nil
	case a.v != "" && z.v != "":
		return text + "-" + z.c + ":" + z.v, nil
	}
	return "", fmt.Errorf("passage %q mixes a chapter and a verse", ref)
}

// recordPassage is the passage cell of a record: its ref, then its also.
func recordPassage(rec Record) (string, error) {
	cell, err := osisText(rec.Ref)
	if err != nil {
		return "", err
	}
	if rec.Also != "" {
		more, err := osisText(rec.Also)
		if err != nil {
			return "", err
		}
		cell += "; " + more
	}
	return cell, nil
}

// RunNotes writes the records of notesPath that need it to the list at
// passagesPath, for the rows of rowsPath.
func RunNotes(notesPath, passagesPath, rowsPath string) (*NotesResult, error) {
	recs, err := ReadRecords(notesPath)
	if err != nil {
		return nil, err
	}
	res := &NotesResult{Records: len(recs)}
	if len(recs) == 0 {
		return res, nil
	}
	rows, err := readRows(rowsPath)
	if err != nil {
		return nil, err
	}
	entries, err := readListEntries(passagesPath)
	if err != nil {
		return nil, err
	}
	problem := func(rec Record, why string) {
		res.Problems = append(res.Problems, fmt.Sprintf("%s %q: %s", rec.ID, rec.Title, why))
	}
	taken := map[string]bool{} // row ids this run has written a line for
	var add strings.Builder
	for _, rec := range recs {
		if strings.TrimSpace(rec.Ref) == "" {
			problem(rec, "the mail names no passage; not added")
			continue
		}
		if strings.TrimSpace(rec.Title) == "" {
			problem(rec, "the mail has no title; not added")
			continue
		}
		at := recordRow(rows, rec)
		if at < 0 {
			problem(rec, fmt.Sprintf("no one row holds this record (id, or date %s); not added", rec.Date))
			continue
		}
		row := rows[at]
		if row.Ref != nil && *row.Ref != "" {
			// The row has its passage. A difference is said and nothing is changed.
			if row.Series == nil || !*row.Series {
				a, aerr := passage.Parse(*row.Ref)
				b, berr := passage.Parse(rec.Ref)
				if aerr != nil || berr != nil || a.OSIS != b.OSIS {
					problem(rec, fmt.Sprintf("row %s holds the passage %s and the mail says %s; the row is kept", row.ID, *row.Ref, rec.Ref))
					continue
				}
			}
			res.Already++
			continue
		}
		cell, err := recordPassage(rec)
		if err != nil {
			problem(rec, fmt.Sprintf("the passage is not written to the list: %v", err))
			continue
		}
		title := tsvField(rec.Title)
		found := false
		for _, e := range entries {
			if e.Row == row.ID || (e.Row == "" && e.Date == rec.Date) {
				found = true
				if e.Title != title || e.Passage != cell {
					problem(rec, fmt.Sprintf("passages.tsv already has a line for row %s with other words (%q, %q); it is not replaced", row.ID, e.Title, e.Passage))
				} else {
					res.Already++
				}
				break
			}
		}
		if found {
			continue
		}
		if !genericRow(row) {
			problem(rec, fmt.Sprintf("row %s has a title of its own (%q) and no passage; not added", row.ID, row.Title))
			continue
		}
		if taken[row.ID] {
			problem(rec, fmt.Sprintf("another record gave row %s other words; the first is kept", row.ID))
			continue
		}
		taken[row.ID] = true
		fmt.Fprintf(&add, "\t%s\t%s\t\t%s\t%s\tnotes\n", rec.Date, title, tsvField(cell), row.ID)
		res.Added++
	}
	if add.Len() > 0 {
		if err := appendLines(passagesPath, "", add.String()); err != nil {
			return nil, err
		}
	}
	return res, nil
}
