package passage

import (
	"fmt"
	"regexp"
)

// NumberPattern is the regexp text of a chapter or verse number as a person
// writes it: one to three digits and, for a half verse ("10a", "12b"), a
// letter. A tool's reference grammar puts it where a number may carry a letter
// and then reads the captured text with VerseNumber or ChapterNumber, which
// decide whether the letter is allowed there.
const NumberPattern = `\d{1,3}[a-zA-Z]?`

var (
	verseNumberRE   = regexp.MustCompile(`^(\d{1,3})[a-zA-Z]?$`)
	chapterNumberRE = regexp.MustCompile(`^\d{1,3}$`)
)

// VerseNumber reads a verse number with an optional letter part ("10", "10a",
// "12b") as that verse ("10", "10", "12"): the verse is the unit the book
// table counts, and the letter names half of it. Anything else is an error.
func VerseNumber(s string) (string, error) {
	m := verseNumberRE.FindStringSubmatch(s)
	if m == nil {
		return "", fmt.Errorf("%q is not a verse number", s)
	}
	return m[1], nil
}

// ChapterNumber reads a chapter number. A chapter has no half, so a letter on
// one ("2a", "40b") is refused, not dropped.
func ChapterNumber(s string) (string, error) {
	if !chapterNumberRE.MatchString(s) {
		return "", fmt.Errorf("%q is not a chapter number", s)
	}
	return s, nil
}

// RangeOSIS writes one passage a person typed as an OSIS reference. The
// arguments are the pieces a tool's grammar (built on NumberPattern) captured:
// the book's OSIS name, the first chapter, the verse after its colon (or ""),
// the number after the dash (or "") and the number after that number's colon
// (or ""). The shapes are
//
//	4         chapter                   4:1       verse
//	4-5       chapters                  4:1-10    verses of one chapter
//	4:1-5:14  across chapters           4-5:14    chapter 4 verse 1 to 5:14
//
// A verse may carry a letter (VerseNumber); a chapter may not (ChapterNumber).
// A number that cannot be read is an error, never a shorter reference.
func RangeOSIS(book, c1, v1, x, y string) (string, error) {
	if _, err := ChapterNumber(c1); err != nil {
		return "", err
	}
	if v1 == "" {
		switch {
		case x == "" && y == "":
			return book + "." + c1, nil
		case y == "":
			c2, err := ChapterNumber(x)
			if err != nil {
				return "", err
			}
			return book + "." + c1 + "-" + book + "." + c2, nil
		}
		// "4-5:14": from the start of chapter 4 to verse 14 of chapter 5.
		v1 = "1"
	}
	v, err := VerseNumber(v1)
	if err != nil {
		return "", err
	}
	ref := book + "." + c1 + "." + v
	switch {
	case x != "" && y == "":
		v2, err := VerseNumber(x)
		if err != nil {
			return "", err
		}
		ref += "-" + book + "." + c1 + "." + v2
	case x != "":
		c2, err := ChapterNumber(x)
		if err != nil {
			return "", err
		}
		v2, err := VerseNumber(y)
		if err != nil {
			return "", err
		}
		ref += "-" + book + "." + c2 + "." + v2
	}
	return ref, nil
}
