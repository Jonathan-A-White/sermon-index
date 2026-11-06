// Command logos-import writes a sermon source from the index file of one of
// the Governor's Logos sermon archives.
//
//	go run ./tools/logos-import --in <slug>-sermons.jsonl --id <source id> --out sources/<source id>
//
// The input is the index the Mayor made of an archive (the vault's
// plans/sermons-2026-11-05/<slug>-sermons.jsonl, one sermon a line: id,
// title, preacher, date, year, ref, bibleRefs, series, part, url). The tool
// turns each line into a row of rows.jsonl in the format of docs/sources.md:
// the ref is OSIS, read from Logos's data-reference (bible.<book>.<chapter>
// in Logos's book numbering) and else from the label; the other references
// are cited; kind is link; the url is the ref.ly address. Nothing of a
// sermon's words is read or written. It writes REPORT.md beside rows.jsonl
// (what it dropped and what it could not place). A run over the same input
// writes the same bytes. source.toml is by hand (docs/adding-a-source.md).
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func main() {
	in := flag.String("in", "", "the archive's index file, <slug>-sermons.jsonl")
	id := flag.String("id", "", "the source's id, as in sources/<id>")
	out := flag.String("out", "", "the source directory to write rows.jsonl and REPORT.md in")
	report := flag.String("report", "", "where to write the report (default <out>/REPORT.md)")
	flag.Parse()
	if *in == "" || *id == "" || *out == "" || flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "usage: logos-import --in <slug>-sermons.jsonl --id <source id> --out <source dir> [--report <file>]")
		os.Exit(2)
	}
	if err := run(*in, *id, *out, *report, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "logos-import:", err)
		os.Exit(1)
	}
}

func run(in, id, out, reportPath string, stdout io.Writer) error {
	f, err := os.Open(in)
	if err != nil {
		return err
	}
	inputs, err := ReadInput(f)
	f.Close()
	if err != nil {
		return fmt.Errorf("%s: %w", in, err)
	}
	rows, rep := BuildRows(inputs)
	if len(rows) == 0 {
		return fmt.Errorf("%s: no row to write (%d lines read)", in, len(inputs))
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
	if err := os.WriteFile(reportPath, []byte(rep.Markdown(id, filepath.Base(in))), 0o644); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "%s\n", id)
	fmt.Fprint(stdout, rep.Summary())
	for _, d := range rep.Dropped {
		fmt.Fprintf(stdout, "dropped %s\n", d)
	}
	return nil
}
