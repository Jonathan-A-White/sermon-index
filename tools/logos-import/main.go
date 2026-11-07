// Command logos-import writes a sermon source from the index file of one of
// the Governor's Logos sermon archives.
//
//	go run ./tools/logos-import --in <slug>-sermons.jsonl --id <source id> --out sources/<source id>
//
// A source made of several books gives --in more than once (the rows follow
// the files in order); --dedupe-titles then drops a line whose title is the
// title of a row of an earlier file (Wesley's Works repeat his Sermons on
// Several Occasions). A book archive's index (the parser's fallback form:
// ids like 2~14522, cited passages, no date) is read as well as an index
// section's.
//
// The input is the index the Mayor made of an archive (the vault's
// plans/sermons-2026-11-05/<slug>-sermons.jsonl, one sermon a line: id,
// title, preacher, date, year, ref, bibleRefs, series, part, url). The tool
// turns each line into a row of rows.jsonl in the format of docs/sources.md:
// the ref is OSIS, read from Logos's data-reference (bible.<book>.<chapter>
// in Logos's book numbering) and else from the label; the other references
// are cited; kind is link; the url is the ref.ly address. Nothing of a
// sermon's words is read or written. --title-form reads more out of a title
// where the archive prints it there: "Basil (329–379): The Creation of the
// World", "Christ’s Desire for His People—By William Henry Green (John 17:24)",
// "XXIII: The Claims of the Christian Ministry" (see titles.go);
// --title-from-ref adds a row's ref to a title that is only "Sermon N". It writes REPORT.md beside rows.jsonl
// (what it dropped and what it could not place). A run over the same input
// writes the same bytes. source.toml is by hand (docs/adding-a-source.md).
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// stringList is a flag given more than once.
type stringList []string

func (l *stringList) String() string     { return strings.Join(*l, ",") }
func (l *stringList) Set(v string) error { *l = append(*l, v); return nil }

func main() {
	var ins stringList
	flag.Var(&ins, "in", "an archive's index file, <slug>-sermons.jsonl (more than once: one source from several books, in order)")
	dedupe := flag.Bool("dedupe-titles", false, "drop a line whose title is that of a row of an earlier --in file")
	id := flag.String("id", "", "the source's id, as in sources/<id>")
	out := flag.String("out", "", "the source directory to write rows.jsonl and REPORT.md in")
	preacher := flag.String("preacher", "", "the preacher of every row, where the index names the book's editor or translator instead")
	titleForm := flag.String("title-form", "", "read preacher, years, passage or number from each title: "+strings.Join(TitleForms, ", "))
	var notImported stringList
	flag.Var(&notImported, "not-imported", "a book of the order this source leaves out and why, \"<book>: <why>\" (more than once); REPORT.md names it")
	note := flag.String("note", "", "a sentence added to the notes of every row (the kind of the book's pieces, where the index does not say)")
	titleFromRef := flag.Bool("title-from-ref", false, "a title that is only \"Sermon N\" becomes \"Sermon N: <its ref>\" (Moberly's)")
	refsFromTitle := flag.Bool("refs-from-title", false, "a line with no Logos reference and no label that reads takes its ref from the passages its title names (Augustine's titles)")
	report := flag.String("report", "", "where to write the report (default <out>/REPORT.md)")
	flag.Parse()
	if len(ins) == 0 || *id == "" || *out == "" || flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "usage: logos-import --in <slug>-sermons.jsonl [--in <more>] [--dedupe-titles] [--preacher <name>] [--refs-from-title] [--title-form <form>] [--title-from-ref] [--note <text>] [--not-imported <book: why>] --id <source id> --out <source dir> [--report <file>]")
		os.Exit(2)
	}
	opts := Options{Dedupe: *dedupe, Preacher: *preacher, RefsFromTitle: *refsFromTitle, TitleForm: *titleForm, Note: *note, NotImported: notImported, TitleFromRef: *titleFromRef}
	if err := run(ins, *id, *out, *report, opts, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "logos-import:", err)
		os.Exit(1)
	}
}

// Options are the flags that shape the rows, as the report repeats them.
type Options struct {
	Dedupe        bool     // --dedupe-titles
	Preacher      string   // --preacher
	RefsFromTitle bool     // --refs-from-title
	TitleForm     string   // --title-form
	Note          string   // --note
	NotImported   []string // --not-imported
	TitleFromRef  bool     // --title-from-ref
}

func run(ins []string, id, out, reportPath string, opts Options, stdout io.Writer) error {
	if opts.TitleForm != "" && !validTitleForm(opts.TitleForm) {
		return fmt.Errorf("--title-form %q: want one of %s", opts.TitleForm, strings.Join(TitleForms, ", "))
	}
	var files []File
	var names []string
	for _, in := range ins {
		f, err := os.Open(in)
		if err != nil {
			return err
		}
		inputs, err := ReadInput(f)
		f.Close()
		if err != nil {
			return fmt.Errorf("%s: %w", in, err)
		}
		base := filepath.Base(in)
		names = append(names, base)
		files = append(files, File{Name: strings.TrimSuffix(base, "-sermons.jsonl"), Inputs: inputs, Preacher: opts.Preacher, RefsFromTitle: opts.RefsFromTitle, TitleForm: opts.TitleForm, Note: opts.Note, TitleFromRef: opts.TitleFromRef})
	}
	rows, rep := BuildSource(files, opts.Dedupe)
	rep.Preacher, rep.RefsFromTitle, rep.TitleForm, rep.Note, rep.NotImported = opts.Preacher, opts.RefsFromTitle, opts.TitleForm, opts.Note, opts.NotImported
	rep.TitleFromRef = opts.TitleFromRef
	if len(rows) == 0 {
		return fmt.Errorf("%s: no row to write (%d lines read)", strings.Join(ins, ", "), rep.Input)
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		return err
	}
	if err := writeRows(filepath.Join(out, "rows.jsonl"), rows); err != nil {
		return err
	}
	if reportPath == "" {
		reportPath = filepath.Join(out, "REPORT.md")
	}
	if err := os.WriteFile(reportPath, []byte(rep.Markdown(id, names)), 0o644); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "%s\n", id)
	fmt.Fprint(stdout, rep.Summary())
	for _, d := range rep.Dropped {
		fmt.Fprintf(stdout, "dropped %s\n", d)
	}
	return nil
}
