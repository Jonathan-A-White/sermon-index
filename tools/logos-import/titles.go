package main

import (
	"regexp"
	"strings"

	"github.com/Jonathan-A-White/sermon-index/internal/passage"
)

// isRoman says whether s (lower case) is a Roman numeral, so that a title
// like "Sermon mill" is not read as "Sermon" and a number.
var romanRE = regexp.MustCompile(`^m{0,3}(cm|cd|d?c{0,3})(xc|xl|l?x{0,3})(ix|iv|v?i{0,3})$`)

func isRoman(s string) bool { return s != "" && romanRE.MatchString(s) }

var (
	sermonPrefixRE = regexp.MustCompile(`^\s*sermon\s+([ivxlcdm]+|\d+)(?:\.|\s|:|$)\s*[:.\-]?\s*`)
	bareRomanRE    = regexp.MustCompile(`^\s*([ivxlcdm]+)\s*[:.]\s+`)
	notAlnumRE     = regexp.MustCompile(`[^a-z0-9]+`)
)

// NormTitle is a title as two books of the same author can be compared:
// lower case, a leading "Sermon N" (N a number or a Roman numeral, with any
// full stop or colon after it) or a bare Roman numeral ("XLI. ") taken off,
// apostrophes dropped and every other mark made a space. Wesley's Works
// call "Sermon 42: Satan’s Devices" "Sermon XLII: Satan's Devices"; both
// read "satans devices". A title that is only a prefix reads "".
func NormTitle(s string) string {
	s = strings.ToLower(s)
	s = strings.NewReplacer("’", "", "‘", "", "'", "").Replace(s)
	if m := sermonPrefixRE.FindStringSubmatch(s); m != nil && (m[1][0] >= '0' && m[1][0] <= '9' || isRoman(m[1])) {
		s = s[len(m[0]):]
	} else if m := bareRomanRE.FindStringSubmatch(s); m != nil && isRoman(m[1]) {
		s = s[len(m[0]):]
	}
	return strings.TrimSpace(notAlnumRE.ReplaceAllString(s, " "))
}

var (
	chapterRE = regexp.MustCompile(`^Chapter\s+([IVXLCDM]+|\d+)\s*:\s*(.+)$`)
	abbrevRE  = regexp.MustCompile(`^([1-3]\s?)?([A-Za-z]+)\.?\s+(\d.*)$`)
)

// ReadableTitle turns a book-archive title such as "Chapter I: Gen. 19:15"
// into one a person can read: the "Chapter N: " the contents list adds is
// taken off (and returned as note, "Chapter I"), and what is left is the
// passage with its book written out ("Genesis 19:15"), or, where it is not
// a passage, the book's own heading ("Love that Passeth Knowledge"). ref
// is the row's OSIS reference, if any: it says which of Phil. or Phlm. an
// abbreviation means. A title with no such prefix is returned as it is.
func ReadableTitle(title, ref string) (readable, note string) {
	m := chapterRE.FindStringSubmatch(strings.TrimSpace(title))
	if m == nil {
		return title, ""
	}
	note = "Chapter " + m[1]
	rest := strings.TrimSpace(m[2])
	if _, _, ok := bookPrefix(rest); ok {
		return rest, note
	}
	a := abbrevRE.FindStringSubmatch(rest)
	if a == nil {
		return rest, note
	}
	num, token, tail := strings.TrimSpace(a[1]), strings.ToLower(a[2]), a[3]
	var found []passage.Book
	for _, b := range passage.Books() {
		bnum, base := "", strings.ToLower(b.Name)
		if b.Name[0] >= '1' && b.Name[0] <= '3' {
			bnum, base = b.Name[:1], strings.ToLower(b.Name[2:])
		}
		if bnum == num && strings.HasPrefix(base, token) {
			found = append(found, b)
		}
	}
	if len(found) > 1 && ref != "" {
		own := strings.SplitN(ref, ".", 2)[0]
		var pick []passage.Book
		for _, b := range found {
			if b.OSIS == own {
				pick = append(pick, b)
			}
		}
		found = pick
	}
	if len(found) != 1 {
		return rest, note
	}
	return found[0].Name + " " + tail, note
}
