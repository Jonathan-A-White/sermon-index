package sources

import (
	"strings"
	"unicode"
)

// TitleSep is what a YouTube row's title puts between its parts: "Title |
// Passage | Preacher".
const TitleSep = " | "

// TitlePart is a row title's sermon-title part: what comes before the first
// TitleSep, or the whole title when it has none.
func TitlePart(title string) string {
	head, _, _ := strings.Cut(title, TitleSep)
	return head
}

// FoldTitle is a title with case, punctuation and spacing folded away.
func FoldTitle(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(unicode.ToLower(r))
		}
	}
	return b.String()
}

// SameTitle is whether a row's title is the mail's title: the row's whole
// title, or its part before TitleSep, equals the mail's with case, punctuation
// and spacing folded. A title with nothing to fold to is never the same.
func SameTitle(rowTitle, mailTitle string) bool {
	want := FoldTitle(mailTitle)
	if want == "" {
		return false
	}
	return FoldTitle(rowTitle) == want || FoldTitle(TitlePart(rowTitle)) == want
}
