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
	r, err := passage.Parse(strings.Join(osis, "-"))
	if err != nil {
		return fail(err.Error())
	}
	return r.OSIS, nil
}

var aliases = map[string]string{
	"psalm":         "Ps",
	"psalms":        "Ps",
	"song of songs": "Song",
}

// bookPrefix finds the book a label starts with (the longest name, followed
// by a space or the end) and returns its OSIS name and the rest.
func bookPrefix(s string) (osis, rest string, ok bool) {
	low := strings.ToLower(s)
	best := 0
	try := func(name, o string) {
		n := strings.ToLower(name)
		if len(n) > best && strings.HasPrefix(low, n) && (len(low) == len(n) || low[len(n)] == ' ') {
			best, osis = len(n), o
		}
	}
	for _, b := range passage.Books() {
		try(b.Name, b.OSIS)
	}
	for a, o := range aliases {
		try(a, o)
	}
	if best == 0 {
		return "", s, false
	}
	return osis, strings.TrimSpace(s[best:]), true
}

// A piece of a label after its book: 3, 3:16, 3:1-13, 3-4, 3:1-4:5, 4-5:14.
var tokenRE = regexp.MustCompile(`^(\d+)(?::(\d+))?(?:-(\d+)(?::(\d+))?)?$`)

// LabelToOSIS reads a reference as a person writes it ("Romans 9",
// "John 3:1–13", "Acts 1:12–15; 2:1, 7–13", "2 Samuel 11; 12:1–31; Psalm
// 51:1–19") as OSIS references, one for each piece, checked against the book
// table. A piece with no book takes the one before it; a bare number after a
// comma, where the piece before had a verse, is a verse of the same chapter.
func LabelToOSIS(label string) ([]string, error) {
	s := strings.NewReplacer("–", "-", "—", "-", " ", " ").Replace(strings.TrimSpace(label))
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
		if o, rest, ok := bookPrefix(piece); ok {
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
			var ref string
			switch {
			case v1 == "" && i > 0 && verseCtx:
				// A bare number after a verse: verses of the same chapter.
				if y != "" {
					return fail("cannot read " + strconv.Quote(tok))
				}
				ref = fmt.Sprintf("%s.%s.%s", book, chapter, c1)
				if x != "" {
					ref += fmt.Sprintf("-%s.%s.%s", book, chapter, x)
				}
			case v1 == "":
				ref = fmt.Sprintf("%s.%s", book, c1)
				chapter, verseCtx = c1, false
				if x != "" {
					ref += fmt.Sprintf("-%s.%s", book, x)
					if y != "" {
						ref += "." + y
					}
				}
			default:
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
