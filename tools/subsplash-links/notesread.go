package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Jonathan-A-White/sermon-index/internal/passage"
	"github.com/Jonathan-A-White/sermon-index/internal/sources"
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
// for (his message's, or an earlier run's) and a record no single row holds are
// left as they are. A record whose mail names no passage takes the one its
// notes PDF's header prints (pdfPassage); when the PDF holds none the row is
// named. The ones that call for a person's look are named with why.

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

// listHoldsTitle reports whether the list has a line, for the record's row (by
// id, or by date when it has no row column), that carries the record's title
// and a passage.
func listHoldsTitle(entries []listEntry, rows []sources.Row, rec Record) bool {
	at := recordRow(rows, rec)
	if at < 0 || strings.TrimSpace(rec.Title) == "" {
		return false
	}
	for _, e := range entries {
		if (e.Row == rows[at].ID || (e.Row == "" && e.Date == rec.Date)) && e.Passage != "" && strings.EqualFold(e.Title, tsvField(rec.Title)) {
			return true
		}
	}
	return false
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

// speakerRE reads the line of a notes PDF's header that names its speaker:
// "Pastor Phil Morgan". pdftotext breaks a word in the middle at some kerned
// letters ("Pas tor"), so one space inside the title word is allowed.
var speakerRE = regexp.MustCompile(`^Pas ?tor\s+([A-Z][A-Za-z.'’-]+(?:\s+[A-Z][A-Za-z.'’-]+){1,3})\s*$`)

// notesSpeaker is the speaker a notes PDF's header names, from the text of its
// first page (pdftotext): a line "Pastor <Name>" above the line that begins
// "Part of" (the series) and nowhere else, since the body of the notes is
// the Governor's and the pastor's own words and says nothing of who preached.
// Empty when the header names none; a speaker is never guessed.
func notesSpeaker(text string) string {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Part of ") {
			return ""
		}
		if m := speakerRE.FindStringSubmatch(line); m != nil {
			return strings.Join(strings.Fields(m[1]), " ")
		}
	}
	return ""
}

// pdfPlaceRE reads a line that is one passage and nothing else, in the grammar
// the notes mail's reader uses (book, chapter, verses, built on
// passage.NumberPattern and written by passage.RangeOSIS).
var pdfPlaceRE = regexp.MustCompile(`(?i)^(` + passage.BookNamePattern() + `)\.?\s+(\d{1,3})(?::(` + passage.NumberPattern + `))?(?:\s*[-–—]\s*(` + passage.NumberPattern + `)(?::(` + passage.NumberPattern + `))?)?$`)

// pdfOSIS is the OSIS reference of a string that is one passage and nothing
// else, or false.
func pdfOSIS(s string) (string, bool) {
	m := pdfPlaceRE.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return "", false
	}
	book, ok := passage.BookOSIS(m[1])
	if !ok {
		return "", false
	}
	osis, err := passage.RangeOSIS(book, m[2], m[3], m[4], m[5])
	if err != nil {
		return "", false
	}
	if r, err := passage.Parse(osis); err != nil || r.OSIS != osis {
		return "", false
	}
	return osis, true
}

var (
	pdfDateRE    = regexp.MustCompile(`^[A-Z][A-Za-z ]+ \d{1,2}, \d{4}$`)
	pdfPastorRE  = regexp.MustCompile(`\s*\bPas ?tor\s.*$`)
	pdfNoDashRE  = strings.NewReplacer("-", "", "–", "", "—", "", " ", "", "\u00a0", "")
	pdfQuoteCuts = "\"“”"
)

// pdfPassage is the passage a notes PDF names, as OSIS, from the text of its
// first page (pdftotext). The header prints the title in quotes and the passage
// after it ("Title" — 2 Corinthians 8:10-15, or on a line of its own), then the
// speaker, then 'Part of <series>' and the date. A line wrap in the header
// drops the dash of a range (pdftotext gives "Luke 8:115" for 8:1-15), so the
// header's passage is taken as it is only when it keeps its dash; otherwise it
// is taken from a line of the notes' body that is one passage and reads the
// same without dashes (the heading above the outline: "Luke 8:1-15"), and
// when no such line confirms it there is no passage (a passage is never
// guessed). The body's other lines (a sub-passage, a verse quoted) are never
// read for themselves. False when there is none.
func pdfPassage(text string) (string, bool) {
	lines := strings.Split(text, "\n")
	var header []string
	n := 0
	for ; n < len(lines) && n < 10; n++ {
		l := strings.TrimSpace(lines[n])
		if l == "" || strings.HasPrefix(l, "Part of ") || pdfDateRE.MatchString(l) {
			break
		}
		header = append(header, l)
	}
	joined := strings.Join(header, " ")
	cut := strings.LastIndexAny(joined, pdfQuoteCuts)
	if cut < 0 {
		return "", false
	}
	// Drop the title (up to its closing quote), then a dash and the speaker.
	rest := joined[cut:]
	rest = strings.TrimLeft(rest, pdfQuoteCuts)
	rest = strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(rest), "—–-"))
	rest = strings.TrimSpace(pdfPastorRE.ReplaceAllString(rest, ""))
	if strings.ContainsAny(rest, "-–—") {
		return pdfOSIS(rest)
	}
	// No dash: the line may have lost one, so the string need not be a
	// passage that exists ("Luke 8:115"); a body line must say the same.
	if !pdfPlaceRE.MatchString(rest) {
		return "", false
	}
	for _, l := range lines[n:] {
		if b, ok := pdfOSIS(l); ok && pdfNoDashRE.Replace(strings.TrimSpace(l)) == pdfNoDashRE.Replace(rest) {
			return b, true
		}
	}
	return "", false
}

// pdftotextFirstPage is the text of a PDF's first page, from poppler's pdftotext.
func pdftotextFirstPage(path string) (string, error) {
	out, err := exec.Command("pdftotext", "-l", "1", path, "-").Output()
	if err != nil {
		return "", fmt.Errorf("pdftotext %s: %w", path, err)
	}
	return string(out), nil
}

// RunNotes writes the records of notesPath that need it to the list at
// passagesPath, for the rows of rowsPath. It names no speaker.
func RunNotes(notesPath, passagesPath, rowsPath string) (*NotesResult, error) {
	return RunNotesWithSpeakers(notesPath, passagesPath, rowsPath, "", nil)
}

// RunNotesWithSpeakers is RunNotes, and a line it writes takes the speaker the
// header of the row's notes PDF (<pdfDir>/<row id>.pdf, read with text) names.
// With no pdfDir, or no file, or a header that names none, the speaker is left
// empty.
func RunNotesWithSpeakers(notesPath, passagesPath, rowsPath, pdfDir string, text func(path string) (string, error)) (*NotesResult, error) {
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
			// The mail names no passage. Its title and a passage are on the
			// list when a person (or an earlier run) put them there; that is
			// not a problem. Otherwise the passage is read from the row's
			// notes PDF, and the record goes on as if the mail had named it.
			// A title and a date the mail always has (subsplash-notes drops a
			// mail without), so those are never read from the PDF.
			if listHoldsTitle(entries, rows, rec) {
				res.Already++
				continue
			}
			why := "no notes PDF to read (no directory)"
			if at := recordRow(rows, rec); at < 0 {
				why = "no one row holds this record, so there is no notes PDF to read"
			} else if pdfDir != "" && text != nil {
				t, err := text(filepath.Join(pdfDir, rows[at].ID+".pdf"))
				switch {
				case err != nil:
					why = fmt.Sprintf("its notes PDF cannot be read (%v)", err)
				default:
					if osis, ok := pdfPassage(t); ok {
						rec.Ref = osis
						why = ""
					} else {
						why = "its notes PDF's text holds no passage line"
					}
				}
			}
			if why != "" {
				problem(rec, "the mail names no passage and "+why+"; not added")
				continue
			}
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
		var speaker string
		if pdfDir != "" && text != nil {
			if t, err := text(filepath.Join(pdfDir, row.ID+".pdf")); err == nil {
				speaker = tsvField(notesSpeaker(t))
			}
		}
		fmt.Fprintf(&add, "\t%s\t%s\t%s\t%s\t%s\tnotes\n", rec.Date, title, speaker, tsvField(cell), row.ID)
		res.Added++
	}
	if add.Len() > 0 {
		if err := appendLines(passagesPath, "", add.String()); err != nil {
			return nil, err
		}
	}
	return res, nil
}
