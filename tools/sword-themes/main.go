// Command sword-themes writes a theme source (topics.jsonl and links.jsonl)
// from a CrossWire SWORD module of a topical index: TCR (Thompson Chain
// Topics, the 1934 subject index) or Torrey (R. A. Torrey's New Topical
// Textbook).
//
//	go run ./tools/sword-themes --module TCR --cache <dir> --out sources/themes-tcr1934 [--report tools/sword-themes/REPORT.md]
//	go run ./tools/sword-themes --module Torrey --cache <dir> --out sources/themes-torrey [--report tools/sword-themes/REPORT.md]
//
// It fetches the module's package (packages/rawzip/<Module>.zip) and its
// mods.d/<module>.conf from https://www.crosswire.org/ftpmirror/pub/sword
// into the cache directory (a file already there is not fetched again),
// reads the RawLD dictionary (a topic a key, ThML in the entry) and writes
// the two files. source.toml is written by hand: it holds the licence and
// what was found about it. Two runs over the same cache write the same bytes.
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	baseURL   = "https://www.crosswire.org/ftpmirror/pub/sword"
	userAgent = "sermon-index-sword-themes (+https://github.com/Jonathan-A-White/sermon-index)"
)

type kind int

const (
	kindThompson kind = iota
	kindTorrey
)

// Module is a SWORD module this tool reads.
type Module struct {
	Name     string // as --module: "TCR"
	Kind     kind
	ZipName  string // under packages/rawzip/
	ConfName string // under raw/mods.d/
	DataPath string // in the zip, without .idx or .dat
	Title    string // for the report
}

var modules = map[string]Module{
	"TCR": {
		Name: "TCR", Kind: kindThompson, ZipName: "TCR.zip", ConfName: "tcr.conf",
		DataPath: "modules/lexdict/rawld/tcr/tcr",
		Title:    "Thompson Chain Topics (the 1934 subject index)",
	},
	"Torrey": {
		Name: "Torrey", Kind: kindTorrey, ZipName: "Torrey.zip", ConfName: "torrey.conf",
		DataPath: "modules/lexdict/rawld/torrey/torrey",
		Title:    "Torrey's New Topical Textbook",
	},
}

func main() {
	module := flag.String("module", "", "the SWORD module: TCR or Torrey")
	cache := flag.String("cache", "", "directory for the fetched module (not committed)")
	out := flag.String("out", "", "the source directory to write topics.jsonl and links.jsonl in")
	report := flag.String("report", "", "write the report (the unparsed references, the see terms that name no topic) to this file; a module's section replaces its own")
	flag.Parse()
	if *module == "" || *cache == "" || *out == "" || flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "usage: sword-themes --module TCR|Torrey --cache <dir> --out <source dir> [--report <file>]")
		os.Exit(2)
	}
	client := &http.Client{Timeout: 5 * time.Minute}
	if err := runModule(context.Background(), client, baseURL, *module, *cache, *out, *report, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "sword-themes:", err)
		os.Exit(1)
	}
}

func runModule(ctx context.Context, client *http.Client, base, name, cache, out, reportPath string, stdout io.Writer) error {
	m, ok := modules[name]
	if !ok {
		names := make([]string, 0, len(modules))
		for n := range modules {
			names = append(names, n)
		}
		sort.Strings(names)
		return fmt.Errorf("unknown module '%s' (the modules are %s)", name, strings.Join(names, ", "))
	}
	if err := fetchModule(ctx, client, base, m, cache, stdout); err != nil {
		return err
	}
	entries, err := ReadRawLD(filepath.Join(cache, m.ZipName), m.DataPath)
	if err != nil {
		return err
	}
	b := Build(m, entries)

	var topics, links bytes.Buffer
	if err := b.WriteJSONL(&topics, &links); err != nil {
		return err
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(out, "topics.jsonl"), topics.Bytes(), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(out, "links.jsonl"), links.Bytes(), 0o644); err != nil {
		return err
	}
	if reportPath != "" {
		old, _ := os.ReadFile(reportPath)
		doc := mergeReport(string(old), m.Name, reportSection(m.Name, b.Markdown()))
		if err := os.WriteFile(reportPath, []byte(doc), 0o644); err != nil {
			return err
		}
	}
	fmt.Fprint(stdout, b.Summary())
	fmt.Fprintf(stdout, "topics.jsonl: %d bytes; links.jsonl: %d bytes\n", topics.Len(), links.Len())
	return nil
}

// fetchModule fetches the module's package and its conf into the cache,
// each unless it is there already.
func fetchModule(ctx context.Context, client *http.Client, base string, m Module, cache string, log io.Writer) error {
	if err := os.MkdirAll(cache, 0o755); err != nil {
		return err
	}
	files := []struct{ url, name string }{
		{base + "/packages/rawzip/" + m.ZipName, m.ZipName},
		{base + "/raw/mods.d/" + m.ConfName, m.ConfName},
	}
	for i, f := range files {
		path := filepath.Join(cache, f.name)
		if st, err := os.Stat(path); err == nil && st.Size() > 0 {
			continue
		}
		fmt.Fprintf(log, "fetching %s\n", f.url)
		if err := fetchOne(ctx, client, f.url, path, i == 0); err != nil {
			return err
		}
	}
	return nil
}

func fetchOne(ctx context.Context, client *http.Client, u, path string, wantZip bool) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: %s", u, resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("%s: %w", u, err)
	}
	if wantZip && !bytes.HasPrefix(body, []byte("PK\x03\x04")) {
		return fmt.Errorf("%s: not a zip", u)
	}
	tmp := path + ".part"
	if err := os.WriteFile(tmp, body, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Summary is the counts, as printed by the tool.
func (b *Built) Summary() string {
	badGroups := map[string]bool{}
	for _, u := range b.Unparsed {
		badGroups[u.Key+"\x00"+u.Group] = true
	}
	var s strings.Builder
	fmt.Fprintf(&s, "module:                        %s\n", b.Module.Name)
	fmt.Fprintf(&s, "topics:                        %d\n", len(b.Topics))
	fmt.Fprintf(&s, "parts:                         %d\n", b.Parts)
	fmt.Fprintf(&s, "topics with a part renumbered: %d\n", len(b.Renumbered))
	fmt.Fprintf(&s, "reference groups read:         %d\n", b.Groups)
	fmt.Fprintf(&s, "  groups with an item unread:  %d (%d items)\n", len(badGroups), len(b.Unparsed))
	fmt.Fprintf(&s, "links (ranges):                %d (%d more read twice and kept once)\n", len(b.Links), b.Duplicates)
	fmt.Fprintf(&s, "verses in them:                %d\n", b.Verses)
	fmt.Fprintf(&s, "see terms read:                %d\n", b.SeeTerms)
	fmt.Fprintf(&s, "  see links kept:              %d\n", b.SeeKept)
	fmt.Fprintf(&s, "  terms naming no topic:       %d\n", len(b.SeeUnresolved))
	return s.String()
}

// Markdown is one module's section of REPORT.md: what the module does not
// settle. Only keys, references and counts, never a topic's text.
func (b *Built) Markdown() string {
	var s strings.Builder
	fmt.Fprintf(&s, "### Counts\n\n```\n%s```\n\n", b.Summary())
	list := func(title, why string, items []string) {
		fmt.Fprintf(&s, "### %s (%d)\n\n%s\n\n", title, len(items), why)
		for _, it := range items {
			fmt.Fprintf(&s, "- %s\n", it)
		}
		if len(items) == 0 {
			s.WriteString("None.\n")
		}
		s.WriteString("\n")
	}
	var un []string
	for _, u := range b.Unparsed {
		un = append(un, fmt.Sprintf("%s: %s (group: %s)", u.Key, u.Item, u.Group))
	}
	list("Reference items not read", "An item of a `<scripRef>` group that the reader leaves out, with the topic's key and the reason. The rest of its group is kept.", un)
	list("See terms that name no topic", "A `<term>` whose text is empty or is not the key of a topic of the module (after case and punctuation are set aside). It is read and not kept.", b.SeeUnresolved)
	var rn []string
	for _, r := range b.Renumbered {
		rn = append(rn, fmt.Sprintf("%s: %d part(s)", r.Key, r.Moved))
	}
	list("Topics whose part numbers repeat", "The lettered sections of a topic ((A), (B), ...) each begin again at (1), and a part's number is unique in a topic. The first part with a number keeps it; a repeat takes a number past the topic's printed ones and its label begins with its section's name.", rn)
	list("Parts printed with no label", "A number with nothing after it (`<b>(1)</b>`); the part's label is \"Part N\".", b.Unlabelled)
	list("Keys whose id is another key's", "The id is the key's slug; a second key with the same slug takes `-2`.", b.Collisions)
	list("Entries skipped", "An entry with no key.", b.Skipped)
	return s.String()
}

const reportHeader = "# SWORD themes: what the modules do not settle\n\n<!-- toc: none -->\n\nWritten by `go run ./tools/sword-themes --module <TCR|Torrey> --cache <dir> --out <source dir> --report tools/sword-themes/REPORT.md`; each run replaces its module's section. Do not edit by hand. Only keys, references and counts are listed.\n\n"

// reportSection wraps a module's report in the markers mergeReport finds.
func reportSection(module, body string) string {
	title := module
	if m, ok := modules[module]; ok {
		title = fmt.Sprintf("%s: %s", m.Name, m.Title)
	}
	return fmt.Sprintf("<!-- module:%s -->\n## %s\n\n%s<!-- /module:%s -->\n", module, title, body, module)
}

// mergeReport puts section in the report doc in place of the module's
// earlier one, the modules in order of name.
func mergeReport(doc, module, section string) string {
	sections := map[string]string{}
	rest := doc
	for {
		i := strings.Index(rest, "<!-- module:")
		if i < 0 {
			break
		}
		rest = rest[i:]
		name, _, _ := strings.Cut(strings.TrimPrefix(rest, "<!-- module:"), " -->")
		end := "<!-- /module:" + name + " -->\n"
		j := strings.Index(rest, end)
		if j < 0 {
			break
		}
		sections[name] = rest[:j+len(end)]
		rest = rest[j+len(end):]
	}
	sections[module] = section
	names := make([]string, 0, len(sections))
	for n := range sections {
		names = append(names, n)
	}
	sort.Strings(names)
	var out strings.Builder
	out.WriteString(reportHeader)
	for i, n := range names {
		if i > 0 {
			out.WriteString("\n")
		}
		out.WriteString(sections[n])
	}
	return out.String()
}
