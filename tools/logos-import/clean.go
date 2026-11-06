package main

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/Jonathan-A-White/sermon-index/internal/passage"
)

// CleanLabel mends a printed reference before it is read: a comma the book's
// name was set with ("Matthew, 5:20", "1 Kings, 18:2l" -> "Matthew 5:20",
// "1 Kings 18:21") and an OCR lower-case l for a 1 in a number ("2l" is 21,
// "l9" is 19). Anything else is left as it is.
func CleanLabel(label string) string {
	pieces := strings.Split(label, ";")
	for i, piece := range pieces {
		if k := strings.Index(piece, ","); k > 0 {
			head, tail := strings.TrimSpace(piece[:k]), strings.TrimSpace(piece[k+1:])
			if _, rest, ok := bookPrefix(head); ok && rest == "" && tail != "" && tail[0] >= '0' && tail[0] <= '9' {
				lead := piece[:len(piece)-len(strings.TrimLeft(piece, " "))]
				piece = lead + head + " " + tail
			}
		}
		pieces[i] = ocrNumRE.ReplaceAllStringFunc(piece, func(w string) string {
			if strings.ContainsAny(w, "0123456789") {
				return strings.ReplaceAll(w, "l", "1")
			}
			return w
		})
	}
	return strings.Join(pieces, ";")
}

// "’s Gospel" and "Chapter " in a title's passage, taken out before it is read.
var gospelRE = regexp.MustCompile(`[’']s\s+Gospel|Chapter\s+`)

// A word made only of digits and l: a number whose 1 was read as l.
var ocrNumRE = regexp.MustCompile(`\b[0-9l]+\b`)

// titleRefRE finds a book, then a chapter or chapter and verses, in a title:
// "Mark 8:1–9", "Luke’s Gospel, 13:6–17", "John, Chapter 9", "Luke 10:17, 20".
var titleRefRE = func() *regexp.Regexp {
	var names []string
	for _, b := range passage.Books() {
		names = append(names, b.Name)
	}
	names = append(names, "Psalms", "Psalm")
	sort.Slice(names, func(i, j int) bool { return len(names[i]) > len(names[j]) })
	for i, n := range names {
		names[i] = regexp.QuoteMeta(n)
	}
	const dash = `[–—-]`
	return regexp.MustCompile(`\b(?:` + strings.Join(names, "|") + `)(?:[’']s\s+Gospel)?,?\s+(?:Chapter\s+)?\d+` +
		`(?::\d+(?:\s?` + dash + `\s?\d+)?(?:,\s*\d+(?:` + dash + `\d+)?)*|\s?` + dash + `\s?\d+)?\b`)
}()

// RefsInTitle reads the passages named in a title, in the order they come, as
// OSIS references: Augustine's titles carry the Gospel passage they preach on
// ("Sermon 95: On the Words of the Gospel, Mark 8:1–9, Where the Miracle ...")
// and the archive's index has no reference for it. The first is the sermon's
// text, the others are cited. A bare "13:23" with no book is not read; a
// passage the book table refuses is left out.
func RefsInTitle(title string) []string {
	var out []string
	for _, m := range titleRefRE.FindAllString(title, -1) {
		text := gospelRE.ReplaceAllString(m, "")
		refs, err := LabelToOSIS(text)
		if err != nil {
			continue
		}
		for _, r := range refs {
			out = appendOnce(out, r)
		}
	}
	return out
}

// NonSermon says why a line of an archive's index is not a sermon (a table,
// an index, the translator's introduction, a book's argument, a preface or an
// epistle to the reader, a list of contents), or "" when it may be one. Titles are compared as NormTitle reads them: lower case, a
// leading Roman numeral off, marks ignored.
func NonSermon(title string) string {
	n := NormTitle(title)
	switch n {
	case "chronological table":
		return "a chronological table"
	case "new testament", "old testament":
		return "an index of Scripture"
	case "introductory discourse", "prolegomena",
		"st chrysostom as a homilist",
		"the place of chrysostom in the history of exegesis",
		"extent and character of chrysostoms exegetical labors":
		return "prolegomena"
	case "argument", "the argument":
		return "the argument of the work"
	case "preface", "the preface", "authors preface", "translators preface", "general editors introduction",
		"foreword", "to the reader", "to the christian reader",
		"epistle to the reader", "the epistle to the reader":
		return "front matter"
	case "contents", "table of contents":
		return "a table of contents"
	}
	switch {
	case strings.HasPrefix(n, "the contents of ") || strings.HasPrefix(n, "an exact alphabetical table") ||
		strings.HasPrefix(n, "the table of the chief heads"):
		return "a table of contents"
	case strings.HasPrefix(n, "argument ") || strings.HasPrefix(n, "the argument "):
		return "the argument of the work"
	case strings.HasSuffix(n, " index of texts") || n == "index of texts" || strings.HasPrefix(n, "index of "):
		return "an index"
	}
	return ""
}

func nonSermonNote(id, title, why string) string {
	return fmt.Sprintf("%s: %q (%s)", id, title, why)
}
