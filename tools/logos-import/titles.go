package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

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

// romanValue reads a Roman numeral (lower case) as a number, or 0.
func romanValue(s string) int {
	if !isRoman(s) {
		return 0
	}
	vals := map[byte]int{'i': 1, 'v': 5, 'x': 10, 'l': 50, 'c': 100, 'd': 500, 'm': 1000}
	total := 0
	for i := 0; i < len(s); i++ {
		v := vals[s[i]]
		if i+1 < len(s) && vals[s[i+1]] > v {
			total -= v
		} else {
			total += v
		}
	}
	return total
}

// SermonNumber reads a title that begins "Sermon N" (N a number or a Roman
// numeral): its number, and whether the title is that alone with no title of
// its own after it ("Sermon I", "Sermon 3:"). ok is false for any other title.
func SermonNumber(title string) (n int, bare, ok bool) {
	s := strings.NewReplacer("’", "", "‘", "", "'", "").Replace(strings.ToLower(title))
	m := sermonPrefixRE.FindStringSubmatch(s)
	if m == nil {
		return 0, false, false
	}
	if m[1][0] >= '0' && m[1][0] <= '9' {
		v, err := strconv.Atoi(m[1])
		if err != nil || v == 0 {
			return 0, false, false
		}
		n = v
	} else if n = romanValue(m[1]); n == 0 {
		return 0, false, false
	}
	return n, strings.TrimSpace(notAlnumRE.ReplaceAllString(s[len(m[0]):], " ")) == "", true
}

var (
	titleDateRE = regexp.MustCompile(`\(\s*([A-Za-z]+)\.?\s+(?:the\s+)?(\d{1,2})(?:st|nd|rd|th|d)?,?\s+(\d{4})\s*\)`)
	monthNames  = []string{"january", "february", "march", "april", "may", "june", "july", "august", "september", "october", "november", "december"}
)

// TitleDate reads a date a title carries in brackets, "(July the 6th, 1645)"
// or "(August the 23d, 1645)" or "(April 4, 1742)", as YYYY-MM-DD; "" when
// the title has none or the day is not one the month has. A date outside
// brackets ("Sunday, April 4, 1742, Before ...") is not read.
func TitleDate(title string) string {
	m := titleDateRE.FindStringSubmatch(title)
	if m == nil {
		return ""
	}
	word := strings.ToLower(m[1])
	if word == "sept" {
		word = "september"
	}
	month := 0
	for i, name := range monthNames {
		if name == word || (len(word) == 3 && strings.HasPrefix(name, word)) {
			month = i + 1
			break
		}
	}
	day, _ := strconv.Atoi(m[2])
	if month == 0 || day == 0 {
		return ""
	}
	date := fmt.Sprintf("%s-%02d-%02d", m[3], month, day)
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return ""
	}
	return date
}

// TitleForms are the forms --title-form reads.
var TitleForms = []string{FormPreacherYears, FormByPreacherRef, FormNumbered, FormRefInBrackets, FormPtRef}

const (
	FormPreacherYears = "preacher-years"  // "Basil (329–379): Title" or "Title—Hooker (1586–1647)"
	FormByPreacherRef = "by-preacher-ref" // "Title—By William Henry Green (John 17:24)"
	FormNumbered      = "numbered"        // "XXIII: Title"
	FormRefInBrackets = "ref-in-brackets" // "Title (Reyel. 12:15, 16.)"
	FormPtRef         = "pt-ref"          // "I. Primeiro Sermão: Gênesis 25.12–22"
)

func validTitleForm(f string) bool {
	for _, x := range TitleForms {
		if x == f {
			return true
		}
	}
	return false
}

// TitleParts is what a title of one of the TitleForms carries.
type TitleParts struct {
	Title    string // the words of the title alone
	Preacher string // the name printed in the title, if any
	Years    string // "329–379" or "born 1851", the preacher's years as printed
	Label    string // the passage printed in the title, as a person writes it
	Numeral  string // the Roman numeral the title is numbered with
}

var (
	yearsRE = `(\d{3,4}\s?–\s?\d{3,4}|Born in \d{4})`
	// "Basil (329–379): The Creation of the World"
	yearsFirstRE = regexp.MustCompile(`^([^:()]+?)\s+\(` + yearsRE + `\):\s+(.+)$`)
	// "The Heavenly Footman—Bunyan (1628–1688)"
	yearsLastRE = regexp.MustCompile(`^(.+?)—([^—:()]+?)\s+\(` + yearsRE + `\)$`)
	// "Drifting—By Charles A. Aiken (Hebrews 2:1)"
	byRefRE = regexp.MustCompile(`^(.+)—By\s+([^()]+?)\s+\(([^()]+)\)$`)
	// "The Nature and Danger of Heresies (Reyel. 12:15, 16.)"
	refInBracketsRE = regexp.MustCompile(`^(.+?)\s*\(([^()]+)\)\.?$`)
	// "I. Primeiro Sermão: Gênesis 25.12–22": the passage after the last colon
	ptRefRE = regexp.MustCompile(`^(.+):\s+([^:]+)$`)
	// "XXIII: The Claims of the Christian Ministry"
	numberedRE = regexp.MustCompile(`^([IVXLCDM]+):\s+(.+)$`)
	// ", Jr." after a name: kept; any other comma between two words of a name is a slip.
	nameSuffixRE = regexp.MustCompile(`^(?i:jr\.?|sr\.?|[ivx]+|d\.\s?d\.|ll\.\s?d\.|m\.\s?d\.)$`)
)

// years says a preacher's years the way a note gives them: "329–379" as
// printed, "Born in 1851" as "born 1851".
func years(s string) string {
	s = strings.Join(strings.Fields(s), "")
	if strings.HasPrefix(s, "Bornin") {
		return "born " + strings.TrimPrefix(s, "Bornin")
	}
	return s
}

// corRE: "1 Cor." and "2 Cor.", how Princeton's titles print the epistles to
// the Corinthians, written out before the passage is read.
var corRE = regexp.MustCompile(`\b([12]) Cor\.`)

// CleanTitleLabel writes out the abbreviation of a passage printed in a
// title ("2 Cor. 3:6" is "2 Corinthians 3:6"); the book table reads the rest
// as it stands (LabelToOSIS reads a part-of-verse letter, "10a", as verse 10).
func CleanTitleLabel(label string) string {
	label = strings.TrimSpace(label)
	label = oldSpellingRE.ReplaceAllStringFunc(label, func(w string) string {
		return oldSpellings[strings.ToLower(strings.TrimRight(strings.TrimSpace(w), " ."))]
	})
	label = strings.TrimRight(label, ". ")
	return corRE.ReplaceAllString(label, "$1 Corinthians")
}

// oldSpellings are the books Sedgwick's titles name in a seventeenth-century
// hand or abbreviation ("Reyel. 12:15, 16", "Jerem . 4:3"), written out.
var oldSpellings = map[string]string{"reyel": "Revelation", "jerem": "Jeremiah"}

var oldSpellingRE = regexp.MustCompile(`(?i)\b(?:reyel|jerem)\b(?:\s?\.)?`)

// TitleLabelToOSIS reads the passage a title prints as OSIS references. It is
// LabelToOSIS over CleanTitleLabel, and then verses that follow one another in
// a chapter ("12:15, 16", "John 3:14, 15, 16") are one range: the text of
// the sermon is the whole of it, not 12:15 with 12:16 cited.
func TitleLabelToOSIS(label string) ([]string, error) {
	refs, err := LabelToOSIS(CleanTitleLabel(label))
	if err != nil {
		return nil, err
	}
	return mergeFollowing(refs), nil
}

// mergeFollowing joins a reference to the one before it when it starts at the
// verse after the other ends, in the same chapter.
func mergeFollowing(refs []string) []string {
	var out []string
	var prev passage.Range
	var start string
	for i, ref := range refs {
		r, err := passage.Parse(ref)
		if err != nil || i == 0 || len(out) == 0 || !followsInChapter(prev, r) {
			out = append(out, ref)
			prev, start = r, ref
			continue
		}
		end := ref
		if k := strings.Index(ref, "-"); k > 0 {
			end = ref[k+1:]
		}
		start = strings.SplitN(start, "-", 2)[0] + "-" + end
		out[len(out)-1] = start
		prev = passage.Range{OSIS: start, Lo: prev.Lo, Hi: r.Hi}
	}
	return out
}

// followsInChapter says whether b begins at the verse after a ends, in the
// same chapter (ordinals are book, chapter, verse in one number).
func followsInChapter(a, b passage.Range) bool {
	return b.Lo == a.Hi+1 && b.Lo/1000 == a.Hi/1000 && a.Lo%1000 != 0 && b.Lo%1000 != 0
}

// ReadableRef writes an OSIS reference as a title does: "Matthew 5:3",
// "Matthew 23:29–32", "Matthew 5:3–6:2", "Romans 8–9", "Jude".
func ReadableRef(osis string) string {
	ends := strings.SplitN(osis, "-", 2)
	type pt struct {
		book           *passage.Book
		chapter, verse string
	}
	var pts []pt
	for _, e := range ends {
		f := strings.Split(e, ".")
		b, ok := passage.Lookup(f[0])
		if !ok {
			return osis
		}
		p := pt{book: b}
		if len(f) > 1 {
			p.chapter = f[1]
		}
		if len(f) > 2 {
			p.verse = f[2]
		}
		pts = append(pts, p)
	}
	write := func(p pt) string {
		switch {
		case p.verse != "":
			return p.chapter + ":" + p.verse
		default:
			return p.chapter
		}
	}
	a := pts[0]
	out := a.book.Name
	if a.chapter != "" {
		out += " " + write(a)
	}
	if len(pts) == 2 {
		z := pts[1]
		switch {
		case z.book != a.book:
			out += "–" + z.book.Name
			if z.chapter != "" {
				out += " " + write(z)
			}
		case z.chapter == a.chapter && z.verse != "":
			out += "–" + z.verse
		default:
			out += "–" + write(z)
		}
	}
	return out
}

// CleanName mends a name set with a comma between two of its words
// ("William, Paxton" is William Paxton); a suffix ("John Smith, Jr.") stays.
func CleanName(name string) string {
	name = strings.Join(strings.Fields(name), " ")
	words := strings.Split(name, " ")
	for i := 0; i+1 < len(words); i++ {
		if w := words[i]; strings.HasSuffix(w, ",") && !nameSuffixRE.MatchString(words[i+1]) {
			words[i] = strings.TrimSuffix(w, ",")
		}
	}
	return strings.Join(words, " ")
}

// ParseTitle reads a title of the given form. ok is false when the title does
// not have the form (a heading, an index): the caller keeps it as it is.
//
//   - preacher-years: the World's Great Sermons print "Basil (329–379): The
//     Creation of the World" (also "Wagner (Born in 1851): ...") and, in
//     Volume 2, "The Heavenly Footman—Bunyan (1628–1688)". Preacher and years
//     come from the title; the name the index gives on every line is the
//     book's editor.
//   - by-preacher-ref: Princeton Sermons print "Christ’s Desire for His
//     People—By William Henry Green (John 17:24)": title, preacher and the
//     preaching text.
//   - ref-in-brackets: Sedgwick prints "The Nature and Danger of Heresies
//     (Reyel. 12:15, 16.)": the title and the preaching text, which must read
//     as a passage (a date in brackets, "(July the 6th, 1645)", does not).
//   - pt-ref: Calvin's Portuguese sermons print "I. Primeiro Sermão: Gênesis
//     25.12–22": the title is kept whole, in Portuguese, and the passage after
//     the colon is the preaching text (see PtLabel).
//   - numbered: Strong's collection prints "XXIII: The Claims of ...": the
//     number is taken off the title.
func ParseTitle(form, title string) (TitleParts, bool) {
	title = strings.TrimSpace(title)
	switch form {
	case FormPreacherYears:
		if m := yearsFirstRE.FindStringSubmatch(title); m != nil {
			return TitleParts{Title: strings.TrimSpace(m[3]), Preacher: CleanName(m[1]), Years: years(m[2])}, true
		}
		if m := yearsLastRE.FindStringSubmatch(title); m != nil {
			return TitleParts{Title: strings.TrimSpace(m[1]), Preacher: CleanName(m[2]), Years: years(m[3])}, true
		}
	case FormByPreacherRef:
		if m := byRefRE.FindStringSubmatch(title); m != nil {
			return TitleParts{Title: strings.TrimSpace(m[1]), Preacher: CleanName(m[2]), Label: strings.TrimSpace(m[3])}, true
		}
	case FormRefInBrackets:
		if m := refInBracketsRE.FindStringSubmatch(title); m != nil {
			if _, err := TitleLabelToOSIS(m[2]); err == nil {
				return TitleParts{Title: strings.TrimSpace(m[1]), Label: strings.TrimSpace(m[2])}, true
			}
		}
	case FormPtRef:
		if m := ptRefRE.FindStringSubmatch(title); m != nil {
			label := PtLabel(m[2])
			if _, err := TitleLabelToOSIS(label); err == nil {
				return TitleParts{Title: title, Label: label}, true
			}
		}
	case FormNumbered:
		if m := numberedRE.FindStringSubmatch(title); m != nil && isRoman(strings.ToLower(m[1])) {
			return TitleParts{Title: strings.TrimSpace(m[2]), Numeral: m[1]}, true
		}
	}
	return TitleParts{}, false
}
