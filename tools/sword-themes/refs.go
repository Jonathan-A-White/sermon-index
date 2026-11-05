package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/Jonathan-A-White/sermon-index/internal/passage"
)

// bookAbbr is the spellings of a book the two modules use (lower case) and
// the OSIS name of each. Jud is Judges and Jude is Jude; Joe is Joel; Mr is
// Mark; Php is Philippians and Phm Philemon. The modules also spell a book
// in capitals ("EX", "MT", "2KI") and a few as two letters ("He", "Mi").
var bookAbbr = map[string]string{
	"co": "Col", "ge": "Gen", "ex": "Exod", "le": "Lev", "nu": "Num", "de": "Deut",
	"jos": "Josh", "jud": "Judg", "ru": "Ruth",
	"1sa": "1Sam", "2sa": "2Sam", "1ki": "1Kgs", "2ki": "2Kgs",
	"1ch": "1Chr", "2ch": "2Chr", "ezr": "Ezra", "ne": "Neh", "es": "Esth",
	"job": "Job", "ps": "Ps", "pr": "Prov", "ec": "Eccl", "so": "Song",
	"isa": "Isa", "jer": "Jer", "la": "Lam", "lam": "Lam", "eze": "Ezek",
	"da": "Dan", "dan": "Dan", "ho": "Hos", "joe": "Joel", "am": "Amos",
	"ob": "Obad", "jon": "Jonah", "mic": "Mic", "mi": "Mic", "na": "Nah",
	"hab": "Hab", "zep": "Zeph", "hag": "Hag", "zec": "Zech", "mal": "Mal",
	"mt": "Matt", "mr": "Mark", "lu": "Luke", "joh": "John", "ac": "Acts",
	"ro": "Rom", "1co": "1Cor", "2co": "2Cor", "ga": "Gal", "gal": "Gal",
	"eph": "Eph", "php": "Phil", "col": "Col", "1th": "1Thess", "2th": "2Thess",
	"1ti": "1Tim", "2ti": "2Tim", "tit": "Titus", "phm": "Phlm",
	"he": "Heb", "heb": "Heb", "jas": "Jas", "1pe": "1Pet", "2pe": "2Pet",
	"1jo": "1John", "2jo": "2John", "3jo": "3John", "jude": "Jude", "re": "Rev",
}

// bookToken is a book at the start of an item: an optional 1, 2 or 3, then
// letters. An item that begins with a number and a colon has no book.
var bookToken = regexp.MustCompile(`^((?:[1-3]\s?)?[A-Za-z]+)\.?\s*(.*)$`)

// ParseGroup reads the text of one <scripRef>: items separated by ';',
// each an optional book, then chapter and verses ("Ex 7:1,7,12; 12:1; Le
// 10:6"). It returns the OSIS references, in order, and one string a
// failing item, which says the item and why. The good items of a group with
// a bad one are still returned.
func ParseGroup(group string) (refs []string, bad []string) {
	var book *passage.Book
	chapter := 0
	for _, item := range splitItems(group) {
		if item == "" {
			continue
		}
		rest := item
		if m := bookToken.FindStringSubmatch(item); m != nil {
			name := strings.ToLower(strings.ReplaceAll(m[1], " ", ""))
			osis, ok := bookAbbr[name]
			if !ok {
				bad = append(bad, fmt.Sprintf("'%s': unknown book '%s'", item, m[1]))
				book, chapter = nil, 0
				continue
			}
			b, ok := passage.Lookup(osis)
			if !ok {
				bad = append(bad, fmt.Sprintf("'%s': the table's book '%s' is not Lampas's", item, osis))
				continue
			}
			book, chapter, rest = b, 0, m[2]
		}
		if book == nil {
			bad = append(bad, fmt.Sprintf("'%s': no book (none in the item and none before it)", item))
			continue
		}
		got, err := parseItem(book, &chapter, rest)
		if err != "" {
			bad = append(bad, fmt.Sprintf("'%s': %s", item, err))
			continue
		}
		for _, r := range got {
			if _, perr := passage.Parse(r); perr != nil {
				bad = append(bad, fmt.Sprintf("'%s': %s", item, perr.(*passage.Error).Reason))
				continue
			}
			refs = append(refs, r)
		}
	}
	return refs, bad
}

// startsWithBook is an element that begins with a book: letters, or a 1, 2
// or 3 and letters.
var startsWithBook = regexp.MustCompile(`^(?:[1-3]\s?)?[A-Za-z]`)

// splitItems cuts a group into items. ';' separates them; so does a ',' that
// is followed by a book ("Re 21:1, Joh 14:1"), which some groups use.
func splitItems(group string) []string {
	var out []string
	for _, item := range strings.Split(group, ";") {
		item = strings.Join(strings.Fields(item), " ")
		start := 0
		parts := strings.Split(item, ",")
		for i := 1; i < len(parts); i++ {
			if startsWithBook.MatchString(strings.TrimSpace(parts[i])) {
				out = append(out, strings.TrimSpace(strings.Join(parts[start:i], ",")))
				start = i
			}
		}
		out = append(out, strings.TrimSpace(strings.Join(parts[start:], ",")))
	}
	return out
}

// parseItem reads what follows a book: "5:8", "5:8-11", "5:8,10-12", "5:8-6:2",
// "23", "40-42", or, after a comma, "10:5" (a new chapter) or "12" (a
// verse). chapter is the chapter the last reference was in, kept for the
// verses that follow it in a comma list.
func parseItem(b *passage.Book, chapter *int, rest string) ([]string, string) {
	rest = strings.TrimSpace(rest)
	// Slips of the keying: a trailing '.' or ',', and a ',' after the colon
	// ("De 28:,34" is 28:34).
	rest = strings.TrimSpace(strings.TrimRight(rest, ".,"))
	rest = strings.ReplaceAll(rest, ":,", ":")
	if rest == "" {
		return []string{b.OSIS}, ""
	}
	oneChapter := len(b.Last) == 1
	var out []string
	inVerses := false // a verse list is being read: a bare number is a verse
	for _, el := range strings.Split(rest, ",") {
		el = strings.ReplaceAll(strings.TrimSpace(el), " ", "")
		if el == "" {
			return nil, "an empty element in a list"
		}
		lo, hi, ok := strings.Cut(el, "-")
		loC, loV, loColon := splitCV(lo)
		switch {
		case loColon || (inVerses && !strings.Contains(lo, ":")) || (oneChapter && !loColon):
			// Verses: C:V, C:V-V2, C:V-C2:V2, or a bare V (V-V2) in a list.
			c, v := loC, loV
			if !loColon {
				c, v = *chapter, atoi(lo)
				if oneChapter {
					c = 1
				}
			}
			if c == 0 || v == 0 {
				return nil, fmt.Sprintf("'%s' is not a verse of a chapter", el)
			}
			*chapter = c
			inVerses = true
			start := fmt.Sprintf("%s.%d.%d", b.OSIS, c, v)
			if !ok {
				out = append(out, start)
				continue
			}
			hiBook := b
			if m := endBook.FindStringSubmatch(hi); m != nil {
				name := strings.ToLower(m[1])
				osis, known := bookAbbr[name]
				hb, found := passage.Lookup(osis)
				if !known || !found {
					return nil, fmt.Sprintf("'%s': unknown book '%s' at the end of a range", el, m[1])
				}
				hiBook, hi = hb, m[2]
			}
			hiC, hiV, hiColon := splitCV(hi)
			if hiColon {
				out = append(out, start+fmt.Sprintf("-%s.%d.%d", hiBook.OSIS, hiC, hiV))
				continue
			}
			end := atoi(hi)
			if end == 0 {
				return nil, fmt.Sprintf("'%s': the end of a range is not a number", el)
			}
			if end == v {
				out = append(out, start)
			} else {
				out = append(out, start+fmt.Sprintf("-%s.%d.%d", b.OSIS, c, end))
			}
		default:
			// Chapters: C or C-C2.
			c := atoi(lo)
			if c == 0 {
				return nil, fmt.Sprintf("'%s' is not a chapter", el)
			}
			*chapter = c
			if !ok {
				out = append(out, fmt.Sprintf("%s.%d", b.OSIS, c))
				continue
			}
			c2 := atoi(hi)
			if c2 == 0 {
				return nil, fmt.Sprintf("'%s': the end of a range is not a number", el)
			}
			out = append(out, fmt.Sprintf("%s.%d-%s.%d", b.OSIS, c, b.OSIS, c2))
		}
	}
	return out, ""
}

// endBook is a range's end that begins with a book ("1Ki2:11", spaces gone).
var endBook = regexp.MustCompile(`^([1-3]?[A-Za-z]+)(\d.*)$`)

// splitCV reads "5:8" as chapter 5, verse 8 with colon true; anything else
// has colon false.
func splitCV(s string) (c, v int, colon bool) {
	cs, vs, ok := strings.Cut(s, ":")
	if !ok {
		return 0, 0, false
	}
	return atoi(cs), atoi(vs), true
}

// atoi is the number s is, or 0 when s is not only digits.
func atoi(s string) int {
	if s == "" {
		return 0
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0
		}
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}
