package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Jonathan-A-White/sermon-index/internal/passage"
	"github.com/Jonathan-A-White/sermon-index/internal/sources"
)

// Input is one line of an archive's index file, as bin/logos-sermons-index.mjs
// wrote it into the vault (plans/sermons-2026-11-05/<slug>-sermons.jsonl).
// Only ids, titles, references, dates and addresses: no word of a sermon.
type Input struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Preacher  string   `json:"preacher"`
	Date      *string  `json:"date"`
	Year      *int     `json:"year"`
	Ref       *string  `json:"ref"`
	Refs      []string `json:"refs"`
	BibleRefs []string `json:"bibleRefs"`
	Series    *string  `json:"series"`
	Part      *int     `json:"part"`
	URL       string   `json:"url"`
	Web       *string  `json:"web"`

	// The book-archive record form (the parser's fallbackRecord, for a
	// resource with no index section): the passages the sermon cites, where
	// the entry begins in the book, whether the text came from an article,
	// and whether the contents list splits the entry into parts, with its
	// length. Absent in an archive's index-section form.
	Cited           []string `json:"cited"`
	Offset          *int     `json:"offset"`
	TextFromArticle *bool    `json:"textFromArticle"`
	Multipart       *bool    `json:"multipart"`
	Length          *int     `json:"length"`
}

// ReadInput reads the index file, one JSON object a line; a blank line is
// skipped and a field the tool does not know is an error, so a change in the
// index's shape is seen and not lost.
func ReadInput(r io.Reader) ([]Input, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	var in []Input
	for n, line := range bytes.Split(data, []byte("\n")) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		dec := json.NewDecoder(bytes.NewReader(line))
		dec.DisallowUnknownFields()
		var x Input
		if err := dec.Decode(&x); err != nil {
			return nil, fmt.Errorf("line %d: %w", n+1, err)
		}
		in = append(in, x)
	}
	return in, nil
}

// Report is what a run found: the counts the tool prints and the lists
// REPORT.md carries. Every list is in input order.
type Report struct {
	Input    int // lines read
	Rows     int // rows written
	WithRef  int // rows with a ref
	FromText int // rows whose ref was read from the label, not from a Logos reference
	Cited    int // rows with cited references
	Dated    int // rows with a full or partial date of their own
	YearOnly int // rows whose date is the year the archive files them under
	Series   int // rows marked series (a ref over more than 3 chapters)

	Dedupe     bool        // titles of an earlier input were dropped
	Files      []FileCount // one for each input, in order
	Duplicates []string    // "ID: "title" repeats ID", rows dropped for a title already in an earlier input
	Retitled   int         // rows whose title was made readable
	Multipart  []string    // ids of entries the contents list splits into parts

	Dropped  []string // "ID: why", lines not written
	NoRef    []string // ids of rows written with no ref
	Refused  []string // "ID: why", references the book table refused
	Disagree []string // "ID: Logos says X, the label says Y"
	Wide     []string // "ID ref (N chapters)"
}

// FileCount is what one input file gave: its lines, the rows it would give
// with no title taken out, the rows dropped as repeats of an earlier input's,
// and the rows left.
type FileCount struct {
	Name                             string
	Lines, Before, Duplicates, After int
}

// File is one input: a name for the report and its lines.
type File struct {
	Name   string
	Inputs []Input
}

var (
	idRE   = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
	resRE  = regexp.MustCompile(`LLS:([^?/]+)`)
	dateRE = regexp.MustCompile(`^\d{4}(-\d{2}(-\d{2})?)?$`)
)

func validDate(s string) bool {
	switch len(s) {
	case 4, 7, 10:
	default:
		return false
	}
	if !dateRE.MatchString(s) {
		return false
	}
	if len(s) == 10 {
		_, err := time.Parse("2006-01-02", s)
		return err == nil
	}
	if len(s) == 7 {
		_, err := time.Parse("2006-01", s)
		return err == nil
	}
	return true
}

// BuildRows turns one index file's lines into rows: BuildSource over a
// single file, with no title taken out.
func BuildRows(in []Input) ([]sources.Row, *Report) {
	return BuildSource([]File{{Inputs: in}}, false)
}

// rowID is the id a line gets in rows.jsonl. An index-section archive's ids
// (YEAR.1996.6) are kept. A book archive's are an offset in the book,
// "2~14522", which the format refuses and which two books can share, so
// they take the resource first: MYSRMNOTES01.2-14522.
func rowID(x Input) string {
	id := strings.TrimSpace(x.ID)
	if strings.Contains(id, "~") {
		if m := resRE.FindStringSubmatch(x.URL); m != nil {
			return m[1] + "." + strings.ReplaceAll(id, "~", "-")
		}
	}
	return id
}

// BuildSource turns the index's lines, file after file, into the rows of one
// source, in order. A line with neither a reference nor a title is dropped
// (it says nothing), as is one with no id, no address, a title missing, or
// an id already used; each drop is listed in the report with its reason. A
// reference the book table refuses is left out of the row and listed; the
// row stays. With dedupeTitles, a line whose title (NormTitle) is the title
// of a row of an EARLIER file is dropped as a repeat and listed; titles that
// repeat inside one file are all kept.
func BuildSource(files []File, dedupeTitles bool) ([]sources.Row, *Report) {
	rep := &Report{Dedupe: dedupeTitles}
	var rows []sources.Row
	type seenAt struct {
		file string
		line int
	}
	firstLine := map[string]seenAt{}
	titles := map[string]string{} // NormTitle -> id of the row of an earlier file
	for _, file := range files {
		fc := FileCount{Name: file.Name, Lines: len(file.Inputs)}
		rep.Input += len(file.Inputs)
		where := ""
		if len(files) > 1 {
			where = " of " + file.Name
		}
		var thisFile [][2]string // NormTitle, id of this file's rows
		for i, x := range file.Inputs {
			line := i + 1
			id := rowID(x)
			title := strings.TrimSpace(x.Title)
			label := ""
			if x.Ref != nil {
				label = strings.TrimSpace(*x.Ref)
			}
			drop := func(why string) {
				if id == "" {
					id = fmt.Sprintf("line %d", line)
				}
				rep.Dropped = append(rep.Dropped, fmt.Sprintf("%s: %s (line %d%s)", id, why, line, where))
			}
			switch {
			case id == "":
				drop("no id")
				continue
			case !idRE.MatchString(id):
				drop("id has a character the format refuses")
				continue
			case title == "" && label == "" && len(x.BibleRefs) == 0:
				drop("neither a ref nor a title")
				continue
			case title == "":
				drop("no title")
				continue
			case strings.TrimSpace(x.URL) == "":
				drop("no url")
				continue
			}
			if prev, ok := firstLine[id]; ok {
				at := fmt.Sprintf("line %d", prev.line)
				if len(files) > 1 {
					at += " of " + prev.file
				}
				rep.Dropped = append(rep.Dropped, fmt.Sprintf("%s: id repeats %s (line %d%s)", id, at, line, where))
				continue
			}
			fc.Before++
			norm := NormTitle(title)
			if dedupeTitles && norm != "" {
				if first, ok := titles[norm]; ok {
					rep.Duplicates = append(rep.Duplicates, fmt.Sprintf("%s: %q repeats %s", id, title, first))
					fc.Duplicates++
					continue
				}
			}
			firstLine[id] = seenAt{file.Name, line}
			thisFile = append(thisFile, [2]string{norm, id})

			r := sources.Row{ID: id, Title: title, URL: strings.TrimSpace(x.URL), Kind: ptr(sources.RowLink)}
			if p := strings.TrimSpace(x.Preacher); p != "" {
				r.Preacher = ptr(p)
			}

			// The references: Logos's own first, the label where Logos gave none.
			var problems []string
			var osis []string
			for _, b := range x.BibleRefs {
				o, err := LogosToOSIS(b)
				if err != nil {
					problems = append(problems, err.Error())
					continue
				}
				osis = appendOnce(osis, o)
			}
			var notes []string
			var fromLabel []string
			var labelErr error
			if label != "" {
				fromLabel, labelErr = LabelToOSIS(label)
			}
			if len(osis) == 0 && label != "" {
				if labelErr != nil {
					problems = append(problems, labelErr.Error())
				} else {
					for _, o := range fromLabel {
						osis = appendOnce(osis, o)
					}
					notes = append(notes, fmt.Sprintf("The ref is read from the label %q; the archive's index has no Logos reference for it.", label))
					rep.FromText++
				}
			} else if len(osis) > 0 && labelErr == nil && label != "" && !sameRefs(osis, fromLabel) {
				rep.Disagree = append(rep.Disagree, fmt.Sprintf("%s: Logos says %s, the label %q says %s", id, strings.Join(osis, "; "), label, strings.Join(fromLabel, "; ")))
			}
			// A book archive also lists the passages a sermon cites. They
			// are cited whether or not the sermon has a text of its own: none
			// of them is taken to be the text.
			var citedMore []string
			for _, c := range x.Cited {
				o, err := LogosToOSIS(c)
				if err != nil {
					problems = append(problems, err.Error())
					continue
				}
				citedMore = appendOnce(citedMore, o)
			}
			if len(problems) > 0 {
				rep.Refused = append(rep.Refused, fmt.Sprintf("%s: %s", id, strings.Join(problems, "; ")))
			}
			var cited []string
			if len(osis) > 0 {
				r.Ref = ptr(osis[0])
				for _, o := range append(append([]string(nil), osis[1:]...), citedMore...) {
					if o != osis[0] {
						cited = appendOnce(cited, o)
					}
				}
				rep.WithRef++
				if rg, err := passage.Parse(osis[0]); err == nil {
					if n := rg.Chapters(); n > sources.WideChapters {
						r.Series = ptr(true)
						rep.Series++
						rep.Wide = append(rep.Wide, fmt.Sprintf("%s %s (%d chapters)", id, osis[0], n))
						notes = append(notes, fmt.Sprintf("A text of %d chapters: a series row, found by its title and matched by no passage.", n))
					}
				}
			} else {
				cited = citedMore
				rep.NoRef = append(rep.NoRef, id)
			}
			if len(cited) > 0 {
				r.Cited = cited
				rep.Cited++
			}

			// A contents entry that reads "Chapter I: Gen. 19:15" is titled
			// by its passage, or by the book's heading.
			ref := ""
			if r.Ref != nil {
				ref = *r.Ref
			}
			if t, chap := ReadableTitle(title, ref); chap != "" {
				if t != title {
					rep.Retitled++
				}
				r.Title = t
				notes = append([]string{chap + "."}, notes...)
			}
			if x.Multipart != nil && *x.Multipart {
				rep.Multipart = append(rep.Multipart, id)
			}

			// Series and part, in the notes.
			var sp string
			switch {
			case x.Series != nil && strings.TrimSpace(*x.Series) != "" && x.Part != nil:
				sp = fmt.Sprintf("Series: %s, part %d.", strings.TrimSpace(*x.Series), *x.Part)
			case x.Series != nil && strings.TrimSpace(*x.Series) != "":
				sp = fmt.Sprintf("Series: %s.", strings.TrimSpace(*x.Series))
			case x.Part != nil:
				sp = fmt.Sprintf("Part %d.", *x.Part)
			}
			if sp != "" {
				notes = append([]string{sp}, notes...)
			}
			if len(notes) > 0 {
				r.Notes = ptr(strings.Join(notes, " "))
			}

			// The date: the sermon's own where the index has one, else the year
			// the archive files it under.
			switch {
			case x.Date != nil && validDate(*x.Date):
				r.Date = ptr(*x.Date)
				rep.Dated++
			case x.Year != nil && *x.Year >= 1000 && *x.Year <= 9999:
				r.Date = ptr(strconv.Itoa(*x.Year))
				rep.YearOnly++
			}
			rows = append(rows, r)
			fc.After++
		}
		// This file's titles count against the files after it, not its own lines.
		for _, t := range thisFile {
			if t[0] != "" {
				if _, ok := titles[t[0]]; !ok {
					titles[t[0]] = t[1]
				}
			}
		}
		rep.Files = append(rep.Files, fc)
	}
	rep.Rows = len(rows)
	return rows, rep
}

func sameRefs(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func appendOnce(list []string, s string) []string {
	for _, x := range list {
		if x == s {
			return list
		}
	}
	return append(list, s)
}

func ptr[T any](v T) *T { return &v }

// writeRows writes rows.jsonl: one object a line, the same bytes for the
// same rows.
func writeRows(path string, rows []sources.Row) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	for _, r := range rows {
		if err := enc.Encode(r); err != nil {
			return err
		}
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

// Summary is the counts, as the tool prints them.
func (r *Report) Summary() string {
	var b strings.Builder
	fmt.Fprintf(&b, "lines read:                          %d\n", r.Input)
	fmt.Fprintf(&b, "rows written:                        %d\n", r.Rows)
	fmt.Fprintf(&b, "lines dropped:                       %d\n", len(r.Dropped))
	fmt.Fprintf(&b, "rows with a ref:                     %d\n", r.WithRef)
	fmt.Fprintf(&b, "  of them read from the label:       %d\n", r.FromText)
	fmt.Fprintf(&b, "  of them with cited references:     %d\n", r.Cited)
	fmt.Fprintf(&b, "rows with no ref:                    %d\n", len(r.NoRef))
	fmt.Fprintf(&b, "series rows (ref over >%d chapters):  %d\n", sources.WideChapters, r.Series)
	fmt.Fprintf(&b, "rows with their own date:            %d\n", r.Dated)
	fmt.Fprintf(&b, "rows dated by the year alone:        %d\n", r.YearOnly)
	fmt.Fprintf(&b, "references the book table refused:   %d\n", len(r.Refused))
	fmt.Fprintf(&b, "Logos and the label disagree:        %d\n", len(r.Disagree))
	if r.Dedupe {
		fmt.Fprintf(&b, "duplicates dropped (earlier input):  %d\n", len(r.Duplicates))
	}
	if r.Retitled > 0 {
		fmt.Fprintf(&b, "titles made readable:                %d\n", r.Retitled)
	}
	if len(r.Multipart) > 0 {
		fmt.Fprintf(&b, "contents entries split in parts:     %d\n", len(r.Multipart))
	}
	return b.String()
}

// Markdown is REPORT.md: what was counted and everything the run could not
// place, so a person sees what is left. inNames are the input files, in the
// order they were given.
func (r *Report) Markdown(id string, inNames []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s: what the import did not settle\n\n<!-- toc: none -->\n\n", id)
	cmd := "go run ./tools/logos-import"
	for _, n := range inNames {
		cmd += " --in <vault>/plans/sermons-2026-11-05/" + n
	}
	if r.Dedupe {
		cmd += " --dedupe-titles"
	}
	fmt.Fprintf(&b, "Written by `%s --id %s --out sources/%s`; do not edit by hand. Only ids, titles, references and addresses are listed.\n\n", cmd, id, id)
	b.WriteString("## Counts\n\n```\n" + r.Summary() + "```\n\n")
	if len(r.Files) > 1 {
		b.WriteString("## Rows before and after, by input\n\n")
		if r.Dedupe {
			b.WriteString("A line whose title (lower case, a leading Roman numeral or \"Sermon N\" taken off, marks ignored) is the title of a row of an earlier input is dropped as a repeat. \"Before\" is the rows each input gives with no title taken out, \"after\" the rows written.\n\n")
		}
		b.WriteString("| input | lines | rows before | duplicates dropped | rows after |\n|---|---|---|---|---|\n")
		var l, bef, d, aft int
		for _, f := range r.Files {
			fmt.Fprintf(&b, "| %s | %d | %d | %d | %d |\n", f.Name, f.Lines, f.Before, f.Duplicates, f.After)
			l, bef, d, aft = l+f.Lines, bef+f.Before, d+f.Duplicates, aft+f.After
		}
		fmt.Fprintf(&b, "| **all** | %d | %d | %d | %d |\n\n", l, bef, d, aft)
	}
	list := func(title, why string, items []string) {
		fmt.Fprintf(&b, "## %s (%d)\n\n%s\n\n", title, len(items), why)
		for _, it := range items {
			fmt.Fprintf(&b, "- %s\n", it)
		}
		if len(items) == 0 {
			b.WriteString("None.\n")
		}
		b.WriteString("\n")
	}
	list("Lines dropped", "A line with neither a ref nor a title, or with no id, no title, no address or an id already used, is not a row.", r.Dropped)
	if r.Dedupe {
		list("Duplicates dropped", "The title is the title of a row of an earlier input (the first input is the one kept).", r.Duplicates)
	}
	list("References the book table refused", "A Logos reference to a book outside the sixty-six, or a chapter or verse past the end; it is left out of the row, and the row stays.", r.Refused)
	list("Series rows", "The ref spans more than three chapters (a whole book or a long text): the row is kept, found by its title and listed by its source, and matched by no passage.", r.Wide)
	list("Logos and the label disagree", "Both were read and differ; the row carries Logos's reference.", r.Disagree)
	if len(r.Multipart) > 0 {
		list("Contents entries split in parts", "The book's contents list splits the entry into parts, so the link opens the book where the entry begins (often a heading, with the sermon in the parts after it).", r.Multipart)
	}
	ids := append([]string(nil), r.NoRef...)
	sort.Strings(ids)
	fmt.Fprintf(&b, "## Rows with no ref (%d)\n\nNeither a Logos reference nor a label that could be read: the row is kept, found by its title and its source, and by no passage. Ids, in the file's order:\n\n", len(r.NoRef))
	if len(r.NoRef) == 0 {
		b.WriteString("None.\n")
	} else {
		b.WriteString(strings.Join(r.NoRef, ", ") + "\n")
	}
	return b.String()
}
