package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/Jonathan-A-White/sermon-index/internal/passage"
)

// logosBook is Logos's own book number as an OSIS book name. Logos numbers
// the Old Testament 1 to 39 in the Protestant order and its New Testament
// 61 (Matthew) to 87 (Revelation); 40 to 60 are not among the sixty-six
// (Logos keeps other books there), so they are refused. The table was read
// from the archives themselves: bible.66.9 is Romans 9, bible.65.27 is Acts
// 27, bible.87.* is Revelation, and every row whose label could be parsed
// agrees (REPORT.md counts the rows where it does not).
func logosBook(n int) (string, bool) {
	var lampas int
	switch {
	case n >= 1 && n <= 39:
		lampas = n
	case n >= 61 && n <= 87:
		lampas = n - 21 // Matthew is Lampas's 40
	default:
		return "", false
	}
	return passage.Books()[lampas-1].OSIS, true
}

// LogosToOSIS reads a Logos data-reference, bible.<book>.<chapter>[.<verse>]
// or a range of two (bible.1.1-1.3, bible.19.51.1-19.51.19, the end written
// without the bible. prefix), as an OSIS reference checked against the book
// table. Logos writes a whole chapter as book.chapter.
func LogosToOSIS(s string) (string, error) {
	fail := func(why string) (string, error) { return "", fmt.Errorf("%q: %s", s, why) }
	// A trailing + or ++ ("bible.83.3.5++") is Logos's "and the verses after".
	rest, ok := strings.CutPrefix(strings.TrimRight(s, "+"), "bible.")
	if !ok {
		return fail("not a Logos bible reference")
	}
	sides := strings.Split(rest, "-")
	if len(sides) > 2 {
		return fail("more than one range")
	}
	var osis []string
	for _, side := range sides {
		nums := strings.Split(side, ".")
		if len(nums) < 2 || len(nums) > 3 {
			return fail("want book.chapter or book.chapter.verse")
		}
		var ns []int
		for _, p := range nums {
			n, err := strconv.Atoi(p)
			if err != nil || n < 0 {
				return fail("not a number: " + p)
			}
			ns = append(ns, n)
		}
		book, ok := logosBook(ns[0])
		if !ok {
			return fail(fmt.Sprintf("Logos book %d is not one of the 66", ns[0]))
		}
		parts := []string{book, strconv.Itoa(ns[1])}
		if len(ns) == 3 {
			parts = append(parts, strconv.Itoa(ns[2]))
		}
		osis = append(osis, strings.Join(parts, "."))
	}
	// A chapter to a verse of a later chapter of the same book
	// (bible.87.4-87.5.14) starts at the chapter's first verse, written by
	// passage.RangeOSIS like every other reader's "Revelation 4-5:14".
	if len(osis) == 2 {
		a, b := strings.Split(osis[0], "."), strings.Split(osis[1], ".")
		if len(a) == 2 && len(b) == 3 && a[0] == b[0] {
			o, err := passage.RangeOSIS(a[0], a[1], "", b[1], b[2])
			if err != nil {
				return fail(err.Error())
			}
			osis = []string{o}
		}
	}
	r, err := passage.Parse(strings.Join(osis, "-"))
	if err != nil {
		return fail(err.Error())
	}
	return r.OSIS, nil
}

// bookPrefix finds the book a label starts with (the longest name, followed
// by a space or the end) and returns its OSIS name and the rest.
func bookPrefix(s string) (osis, rest string, ok bool) {
	return findBook(s, false)
}

// labelBookPrefix is bookPrefix for a label: it also knows the old and the
// ordinary abbreviations and a leading "St.", and it takes a point or a colon
// after the book, with a space before it, as part of the book ("Math . 5:6",
// "Math : 5:4", "Hebr. 11:1", "St. Matthew 5:10").
func labelBookPrefix(s string) (osis, rest string, ok bool) {
	return findBook(s, true)
}

var saintRE = regexp.MustCompile(`(?i)^(?:st|saint)\.?\s+`)

func findBook(s string, tolerant bool) (osis, rest string, ok bool) {
	skip := 0
	if tolerant {
		if m := saintRE.FindString(s); m != "" {
			skip = len(m)
		}
	}
	text := s[skip:]
	best := 0
	// Each end of a word of the text is tried as the end of the book's name; the
	// longest name that passage reads (passage.BookOSIS, or only a full name
	// for a title: passage.FullBookOSIS) is the book.
	for e := 1; e <= len(text); e++ {
		if e < len(text) && text[e] != ' ' {
			continue
		}
		name := text[:e]
		var o string
		var found bool
		end := e
		if tolerant {
			o, found = passage.BookOSIS(name)
			// A space, then a point or a colon, may follow the name.
			j := e
			for j < len(text) && text[j] == ' ' {
				j++
			}
			if j < len(text) && (text[j] == '.' || text[j] == ':') {
				end = j + 1
			}
			if !found && strings.HasSuffix(name, ":") {
				o, found = passage.BookOSIS(strings.TrimSuffix(name, ":"))
			}
		} else {
			o, found = passage.FullBookOSIS(name)
		}
		if found && end < len(text) && text[end] != ' ' {
			found = false
		}
		if found && end > best {
			best, osis = end, o
		}
	}
	if best == 0 {
		return "", s, false
	}
	return osis, strings.TrimSpace(s[skip+best:]), true
}

// A piece of a label after its book: 3, 3:16, 3:1-13, 3-4, 3:1-4:5, 4-5:14.
// A number may carry a half verse's letter (3:1-13b); passage.VerseNumber and
// passage.ChapterNumber say where it is allowed.
var tokenRE = regexp.MustCompile(`^(` + passage.NumberPattern + `)(?::(` + passage.NumberPattern + `))?(?:-(` + passage.NumberPattern + `)(?::(` + passage.NumberPattern + `))?)?$`)

// LabelToOSIS reads a reference as a person writes it ("Romans 9",
// "John 3:1–13", "Acts 1:12–15; 2:1, 7–13", "2 Samuel 11; 12:1–31; Psalm
// 51:1–19") as OSIS references, one for each piece, checked against the book
// table. A piece with no book takes the one before it; a bare number after a
// comma, where the piece before had a verse, is a verse of the same chapter.
func LabelToOSIS(label string) ([]string, error) {
	s := strings.NewReplacer("–", "-", "—", "-", " ", " ").Replace(strings.TrimSpace(CleanLabel(label)))
	fail := func(why string) ([]string, error) { return nil, fmt.Errorf("%q: %s", label, why) }
	if s == "" {
		return fail("empty")
	}
	var out []string
	add := func(ref string) error {
		r, err := passage.Parse(ref)
		if err != nil {
			return err
		}
		out = append(out, r.OSIS)
		return nil
	}
	var book string
	for _, piece := range strings.Split(s, ";") {
		piece = strings.TrimSpace(piece)
		chapter := ""
		verseCtx := false
		if o, rest, ok := labelBookPrefix(piece); ok {
			book, piece = o, rest
			if piece == "" {
				if err := add(book); err != nil {
					return fail(err.Error())
				}
				continue
			}
		}
		if book == "" {
			return fail("no book")
		}
		for i, tok := range strings.Split(piece, ",") {
			tok = strings.ReplaceAll(strings.TrimSpace(tok), " ", "")
			m := tokenRE.FindStringSubmatch(tok)
			if m == nil {
				return fail("cannot read " + strconv.Quote(tok))
			}
			c1, v1, x, y := m[1], m[2], m[3], m[4]
			// Each number is a chapter or a verse by its place: a verse may
			// carry a letter (dropped), a chapter may not (refused).
			var bad error
			chap := func(n *string) {
				if v, err := passage.ChapterNumber(*n); err != nil {
					bad = err
				} else {
					*n = v
				}
			}
			verse := func(n *string) {
				if v, err := passage.VerseNumber(*n); err != nil {
					bad = err
				} else {
					*n = v
				}
			}
			var ref string
			switch {
			case v1 == "" && i > 0 && verseCtx:
				// A bare number after a verse: verses of the same chapter.
				if y != "" {
					return fail("cannot read " + strconv.Quote(tok))
				}
				verse(&c1)
				if x != "" {
					verse(&x)
				}
				if bad != nil {
					return fail("cannot read " + strconv.Quote(tok))
				}
				ref = fmt.Sprintf("%s.%s.%s", book, chapter, c1)
				if x != "" {
					ref += fmt.Sprintf("-%s.%s.%s", book, chapter, x)
				}
			case v1 == "":
				// A chapter, chapters, or "4-5:14": passage.RangeOSIS writes
				// them all, as every other reader of a range does.
				r, err := passage.RangeOSIS(book, c1, "", x, y)
				if err != nil {
					return fail("cannot read " + strconv.Quote(tok))
				}
				ref = r
				chapter, verseCtx = c1, false
			default:
				chap(&c1)
				verse(&v1)
				if x != "" && y == "" {
					verse(&x)
				} else if x != "" {
					chap(&x)
					verse(&y)
				}
				if bad != nil {
					return fail("cannot read " + strconv.Quote(tok))
				}
				ref = fmt.Sprintf("%s.%s.%s", book, c1, v1)
				chapter, verseCtx = c1, true
				switch {
				case x != "" && y == "":
					ref += fmt.Sprintf("-%s.%s.%s", book, c1, x)
				case x != "":
					ref += fmt.Sprintf("-%s.%s.%s", book, x, y)
				}
			}
			if err := add(ref); err != nil {
				return fail(err.Error())
			}
		}
	}
	return out, nil
}
