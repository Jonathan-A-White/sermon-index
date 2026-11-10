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
