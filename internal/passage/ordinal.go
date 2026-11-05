// Package passage turns an OSIS reference into a range of verse ordinals.
//
// A verse ordinal is book*1000000 + chapter*1000 + verse, with the book
// numbered 1 to 66 in Lampas's order (books.go) and the chapter and verse
// numbered as Lampas numbers them. A reference is a single verse
// (Rom.5.8), a chapter (Rom.5), a book (Jude), or a range of two of those
// (Rom.5.1-Rom.5.11, Rom.5.12-Rom.6.4, Rom.5-Rom.6). Its range is the first
// verse's ordinal to the last verse's, both inclusive; Range.PG gives the
// Postgres int4range literal, whose upper bound is exclusive.
package passage

import (
	"fmt"
	"strconv"
	"strings"
)

// Book is one of the 66 books.
type Book struct {
	N    int    // 1 to 66, Lampas's order
	Code string // Lampas's code, lower case ("rom")
	OSIS string // the OSIS book name ("Rom")
	Name string // the English name ("Romans")
	Last []int  // Last[c-1] is the number of chapter c's last verse
}

var byName = func() map[string]*Book {
	m := make(map[string]*Book, 132)
	for i := range books {
		b := &books[i]
		m[strings.ToLower(b.OSIS)] = b
		m[b.Code] = b
	}
	return m
}()

// Lookup finds a book by its OSIS name or its Lampas code, in any case.
func Lookup(name string) (*Book, bool) {
	b, ok := byName[strings.ToLower(name)]
	return b, ok
}

// Books returns the 66 books in order.
func Books() []Book { return books[:] }

// Ordinal is a verse's ordinal: book*1000000 + chapter*1000 + verse.
func Ordinal(book, chapter, verse int) int {
	return book*1000000 + chapter*1000 + verse
}

// Range is a parsed reference: OSIS is its canonical form, Lo and Hi the
// ordinals of its first and last verse (both inclusive).
type Range struct {
	OSIS   string
	Lo, Hi int
}

// PG is the range as a Postgres int4range literal in canonical form,
// "[Lo,Hi+1)".
func (r Range) PG() string { return fmt.Sprintf("[%d,%d)", r.Lo, r.Hi+1) }

// Error is a reference that does not parse; its message names the
// reference and the reason, as "ref 'Rom.5.99': verse past the chapter".
type Error struct {
	Ref    string
	Reason string
}

func (e *Error) Error() string { return fmt.Sprintf("ref '%s': %s", e.Ref, e.Reason) }

const notOSIS = "not an OSIS reference (Book.Chapter.Verse)"

// point is one end of a reference: a book, and a chapter and verse where
// given (0 when not).
type point struct {
	book       *Book
	chapter    int
	verse      int
	osisString string
}

// Parse reads an OSIS reference or range. Book names are OSIS names or
// Lampas's codes, in any case; the canonical form uses the OSIS name.
func Parse(ref string) (Range, error) {
	s := strings.TrimSpace(ref)
	fail := func(reason string) (Range, error) { return Range{}, &Error{Ref: ref, Reason: reason} }
	if s == "" {
		return fail("empty")
	}
	parts := strings.Split(s, "-")
	if len(parts) > 2 {
		return fail(notOSIS)
	}
	start, reason := parsePoint(parts[0])
	if reason != "" {
		return fail(reason)
	}
	end := start
	if len(parts) == 2 {
		if parts[1] == "" {
			return fail(notOSIS)
		}
		if isNumbers(parts[1]) {
			return fail(fmt.Sprintf("the end of a range is a whole reference (%s-%s.%s)",
				parts[0], parentOf(start), parts[1]))
		}
		end, reason = parsePoint(parts[1])
		if reason != "" {
			return fail(reason)
		}
	}
	lo := start.first()
	hi := end.last()
	if hi < lo {
		return fail("the range ends before it starts")
	}
	osis := start.osisString
	if len(parts) == 2 && end.osisString != start.osisString {
		osis += "-" + end.osisString
	}
	return Range{OSIS: osis, Lo: lo, Hi: hi}, nil
}

// parentOf is the reference one level above a point's last part, used to
// suggest the whole form of a short range end ("Rom.5" for Rom.5.1).
func parentOf(p point) string {
	switch {
	case p.verse != 0:
		return fmt.Sprintf("%s.%d", p.book.OSIS, p.chapter)
	default:
		return p.book.OSIS
	}
}

func isNumbers(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if (r < '0' || r > '9') && r != '.' {
			return false
		}
	}
	return true
}

// parsePoint reads Book, Book.C or Book.C.V; the reason is "" when it parses.
func parsePoint(s string) (point, string) {
	fields := strings.Split(s, ".")
	if len(fields) > 3 || fields[0] == "" {
		return point{}, notOSIS
	}
	for _, f := range fields[1:] {
		if f == "" || !isNumbers(f) {
			return point{}, notOSIS
		}
	}
	if strings.ContainsAny(fields[0], " :,;") {
		return point{}, notOSIS
	}
	b, ok := Lookup(fields[0])
	if !ok {
		return point{}, fmt.Sprintf("unknown book '%s'", fields[0])
	}
	p := point{book: b, osisString: b.OSIS}
	if len(fields) >= 2 {
		c, err := strconv.Atoi(fields[1])
		if err != nil {
			return point{}, notOSIS
		}
		if c == 0 {
			return point{}, "chapter 0 is not a chapter"
		}
		if c > len(b.Last) {
			return point{}, "chapter past the book"
		}
		p.chapter = c
		p.osisString += "." + strconv.Itoa(c)
	}
	if len(fields) == 3 {
		v, err := strconv.Atoi(fields[2])
		if err != nil {
			return point{}, notOSIS
		}
		if v == 0 {
			return point{}, "verse 0 is not a verse (a superscription is not numbered)"
		}
		if v > b.Last[p.chapter-1] {
			return point{}, "verse past the chapter"
		}
		p.verse = v
		p.osisString += "." + strconv.Itoa(v)
	}
	return p, ""
}

// first is the ordinal of the point's first verse.
func (p point) first() int {
	c, v := p.chapter, p.verse
	if c == 0 {
		c = 1
	}
	if v == 0 {
		v = 1
	}
	return Ordinal(p.book.N, c, v)
}

// last is the ordinal of the point's last verse.
func (p point) last() int {
	c, v := p.chapter, p.verse
	if c == 0 {
		c = len(p.book.Last)
	}
	if v == 0 {
		v = p.book.Last[c-1]
	}
	return Ordinal(p.book.N, c, v)
}

// Chapters is how many chapters the range touches: 1 for a verse, a
// chapter or a passage within one, 16 for the whole of Romans. A range
// across books counts the chapters of each.
func (r Range) Chapters() int {
	book := func(ord int) (*Book, int) { // the book and chapter an ordinal is in
		b := &books[ord/1000000-1]
		return b, ord / 1000 % 1000
	}
	b1, c1 := book(r.Lo)
	b2, c2 := book(r.Hi)
	if b1 == b2 {
		return c2 - c1 + 1
	}
	n := len(b1.Last) - c1 + 1 + c2
	for i := b1.N; i < b2.N-1; i++ { // the books between
		n += len(books[i].Last)
	}
	return n
}
