package main

import (
	"fmt"
	"html"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/Jonathan-A-White/sermon-index/internal/passage"
)

// Episode is one article of a series archive page: a post of the series.
// Nothing of the post's text is read, only what the listing prints around it.
type Episode struct {
	PostID     int
	Title      string   // as the site prints it, on one line
	URL        string   // the post's address
	Slug       string   // the last part of URL
	YouTubeID  string   // the id of the post's own YouTube embed, "" when it has none
	DurationS  int      // the listing's "Duration", 0 when it prints none
	Categories []string // the slugs of the post's categories ("category-<slug>" classes)
}

var (
	articleSplit = regexp.MustCompile(`<article\s`)
	postIDRE     = regexp.MustCompile(`^[^>]*\bpost-(\d+)\b`)
	categoryRE   = regexp.MustCompile(`\bcategory-([A-Za-z0-9_-]+)`)
	titleLinkRE  = regexp.MustCompile(`(?s)<a class="entry-title-link"[^>]*href="([^"]+)"[^>]*>(.*?)</a>`)
	embedRE      = regexp.MustCompile(`youtube(?:-nocookie)?\.com/embed/([A-Za-z0-9_-]{11})\b`)
	durationRE   = regexp.MustCompile(`(?s)class="label duration">[^<]*</span>\s*<span>\s*([0-9:]+)\s*</span>`)
	nextRE       = regexp.MustCompile(`<(?:link|a)\s[^>]*\brel="next"[^>]*\bhref="([^"]+)"|<(?:link|a)\s[^>]*\bhref="([^"]+)"[^>]*\brel="next"`)
	tagRE        = regexp.MustCompile(`<[^>]*>`)
)

// ParseArchive reads one page of a series archive: its articles, and the
// address of the next page ("" on the last).
func ParseArchive(page []byte) (eps []Episode, next string, err error) {
	s := string(page)
	parts := articleSplit.Split(s, -1)
	for _, a := range parts[1:] {
		if i := strings.Index(a, "</article>"); i >= 0 {
			a = a[:i]
		}
		var e Episode
		m := postIDRE.FindStringSubmatch(a)
		if m == nil {
			return nil, "", fmt.Errorf("an article has no post-<id> class: %.80q", a)
		}
		e.PostID, _ = strconv.Atoi(m[1])
		if i := strings.Index(a, ">"); i >= 0 {
			for _, c := range categoryRE.FindAllStringSubmatch(a[:i], -1) {
				e.Categories = append(e.Categories, c[1])
			}
		}
		t := titleLinkRE.FindStringSubmatch(a)
		if t == nil {
			return nil, "", fmt.Errorf("post %d has no title link", e.PostID)
		}
		e.URL = html.UnescapeString(t[1])
		e.Title = strings.Join(strings.Fields(html.UnescapeString(tagRE.ReplaceAllString(t[2], ""))), " ")
		e.Slug = slugOf(e.URL)
		if m := embedRE.FindStringSubmatch(a); m != nil {
			e.YouTubeID = m[1]
		}
		if m := durationRE.FindStringSubmatch(a); m != nil {
			e.DurationS = parseDuration(m[1])
		}
		eps = append(eps, e)
	}
	if m := nextRE.FindStringSubmatch(s); m != nil {
		next = html.UnescapeString(m[1] + m[2])
	}
	return eps, next, nil
}

func slugOf(u string) string {
	u = strings.TrimRight(u, "/")
	return u[strings.LastIndex(u, "/")+1:]
}

// parseDuration reads "1:09:09" or "7:21" as seconds; anything else is 0.
func parseDuration(s string) int {
	if s == "" {
		return 0
	}
	total := 0
	for _, p := range strings.Split(s, ":") {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return 0
		}
		total = total*60 + n
	}
	return total
}

// verseRE reads "c:v", "c:v-v", "c:v-c:v" and a trailing ", v" the way a
// title prints a passage, with or without the book before it.
var verseRE = regexp.MustCompile(`\b(\d{1,3}):(\d{1,3})[a-z]?(?:\s?[-–]\s?(\d{1,3})(?::(\d{1,3}))?)?(?:,\s*(\d{1,3})\b)?`)

// RefFromTitle reads the passage a title prints for a post of the book's
// series: "(1:1-8)", "Mk 5:1-20", "1 Peter 1:3-7", "(15:40-16:8)", or a whole
// chapter ("Romans 6"). It returns "" when the title prints none, and an
// error when it prints one the book does not have. A verse after a comma
// ("Romans 8:29-30, 33") is a cited reference.
func RefFromTitle(book, title string) (ref string, cited []string, err error) {
	b, ok := passage.Lookup(book)
	if !ok {
		return "", nil, fmt.Errorf("no book %q", book)
	}
	ms := verseRE.FindAllStringSubmatch(title, -1)
	if len(ms) > 0 {
		m := ms[len(ms)-1]
		ref, err = passage.RangeOSIS(b.OSIS, m[1], m[2], m[3], m[4])
		if err != nil {
			return "", nil, fmt.Errorf("%q: %v", title, err)
		}
		if err := checkRef(b, ref); err != nil {
			return "", nil, fmt.Errorf("%q: %v", title, err)
		}
		if m[5] != "" {
			c := b.OSIS + "." + m[1] + "." + m[5]
			if m[4] != "" {
				c = b.OSIS + "." + m[3] + "." + m[5]
			}
			if checkRef(b, c) == nil {
				cited = []string{c}
			}
		}
		return ref, cited, nil
	}
	name := strings.ReplaceAll(regexp.QuoteMeta(b.Name), `\ `, `\s+`)
	if m := regexp.MustCompile(`\b` + name + `\s+(\d{1,3})\b`).FindStringSubmatch(title); m != nil {
		ref = b.OSIS + "." + m[1]
		if err := checkRef(b, ref); err != nil {
			return "", nil, fmt.Errorf("%q: %v", title, err)
		}
		return ref, nil, nil
	}
	return "", nil, nil
}

// checkRef refuses a reference the book does not have: a chapter or verse
// past the book's table, or a range that runs backwards.
func checkRef(b *passage.Book, ref string) error {
	r, err := passage.Parse(ref)
	if err != nil {
		return err
	}
	if r.Lo > r.Hi {
		return fmt.Errorf("%s runs backwards", ref)
	}
	return nil
}

var (
	partOfSlug = map[string]bool{"pt": true, "part": true}
	isDigits   = regexp.MustCompile(`^\d+$`)
)

// RefsFromSlug reads the passage a post's slug ends in ("...-1-peter-13-7" is
// 1 Peter 1:3-7), where the chapter and the verse are written run together.
// It returns every passage the digits can mean in the book (a chapter that
// exists and a verse that chapter has), sorted; the slug is plain when there
// is one. A series part number ("-pt-9", "-part-5") is not a passage.
func RefsFromSlug(book, slug string) []string {
	b, ok := passage.Lookup(book)
	if !ok {
		return nil
	}
	toks := strings.Split(slug, "-")
	for len(toks) > 0 && toks[len(toks)-1] == "again" {
		toks = toks[:len(toks)-1]
	}
	i := len(toks)
	for i > 0 && isDigits.MatchString(toks[i-1]) {
		i--
	}
	run := toks[i:]
	if i > 0 && partOfSlug[toks[i-1]] && len(run) > 0 {
		run = run[1:] // the part number
	}
	if len(run) == 0 || len(run) > 2 {
		return nil
	}
	set := map[string]bool{}
	for _, c := range cvSplits(b, run[0]) {
		if len(run) == 1 {
			set[fmt.Sprintf("%s.%d.%d", b.OSIS, c[0], c[1])] = true
			continue
		}
		// the second number is a last verse of the same chapter or, when
		// the chapter has no such verse, chapter and verse run together
		// ("328-413": 3:28-4:13)
		if v, err := strconv.Atoi(run[1]); err == nil && v >= c[1] && v <= b.Last[c[0]-1] {
			set[fmt.Sprintf("%s.%d.%d-%s.%d.%d", b.OSIS, c[0], c[1], b.OSIS, c[0], v)] = true
			continue
		}
		for _, d := range cvSplits(b, run[1]) {
			if d[0] > c[0] {
				set[fmt.Sprintf("%s.%d.%d-%s.%d.%d", b.OSIS, c[0], c[1], b.OSIS, d[0], d[1])] = true
			}
		}
	}
	if len(run) == 1 {
		if n, _ := strconv.Atoi(run[0]); n >= 1 && n <= len(b.Last) && run[0][0] != '0' {
			set[fmt.Sprintf("%s.%d", b.OSIS, n)] = true
		}
	}
	var out []string
	for r := range set {
		out = append(out, r)
	}
	sort.Strings(out)
	return out
}

// cvSplits cuts digits into chapter and verse in every way the book allows.
func cvSplits(b *passage.Book, digits string) [][2]int {
	var out [][2]int
	for k := 1; k < len(digits); k++ {
		if digits[0] == '0' || digits[k] == '0' {
			continue
		}
		c, _ := strconv.Atoi(digits[:k])
		v, _ := strconv.Atoi(digits[k:])
		if c >= 1 && c <= len(b.Last) && v >= 1 && v <= b.Last[c-1] {
			out = append(out, [2]int{c, v})
		}
	}
	return out
}

var (
	bodyStarts = []string{`<div class="swp-hidden-panel-wrap"`, `<div class="hjs-section prev-next"`, `</article>`}

	// scriptureRE reads a book's name (any spelling passage.BookOSIS reads),
	// a chapter and, after a colon, a verse or a span of verses.
	scriptureRE = regexp.MustCompile(`(?i)(?:^|[^\pL\d])(` + passage.BookNamePattern() + `)\.?\s*(\d{1,3})(?::(\d{1,3})[a-z]?(?:\s?[-–]\s?(\d{1,3})(?::(\d{1,3}))?)?)?`)

	// notAChapter are words that follow a number in prose: "Mark 5 minutes".
	notAChapter = regexp.MustCompile(`^\s+(?:minutes?|mins?|seconds?|hours?|days?|weeks?|months?|years?|times?|people|percent)\b`)
)

// BodyText is the text of a post's own words: what the page prints in the
// entry-content before the share buttons and the "Additional Videos In This
// Series" links, which are other posts' addresses and titles. Tags are
// removed, entities read and white space made single; "" when the page has no
// entry-content.
func BodyText(page []byte) string {
	s := string(page)
	i := strings.Index(s, `<div class="entry-content">`)
	if i < 0 {
		return ""
	}
	s = s[i+len(`<div class="entry-content">`):]
	end := len(s)
	for _, m := range bodyStarts {
		if j := strings.Index(s, m); j >= 0 && j < end {
			end = j
		}
	}
	s = tagRE.ReplaceAllString(s[:end], " ")
	return strings.Join(strings.Fields(html.UnescapeString(s)), " ")
}

// RefsInText returns the passages a text names, in order and each once, as
// OSIS references: a book written with a capital (full or abbreviated, as
// passage.BookOSIS reads it), a chapter and a verse or span of verses
// ("Isaiah 53:1-12", "1 John 4:8", "Genesis 1:26-2:3"). A chapter and verse the
// book does not have are not a passage. With bare, a full book name and a
// chapter alone ("Psalm 23") is a passage too, unless a word of time follows
// ("Mark 5 minutes").
func RefsInText(text string, bare bool) []string {
	var out []string
	seen := map[string]bool{}
	for _, m := range scriptureRE.FindAllStringSubmatchIndex(text, -1) {
		g := func(n int) string {
			if m[2*n] < 0 {
				return ""
			}
			return text[m[2*n]:m[2*n+1]]
		}
		name := g(1)
		if r := []rune(name)[0]; !unicode.IsUpper(r) && !unicode.IsDigit(r) {
			continue // "mark 3:4" is a verb, not a book
		}
		osis, ok := passage.BookOSIS(name)
		if !ok {
			continue
		}
		if g(3) == "" {
			if !bare {
				continue
			}
			if _, full := passage.FullBookOSIS(name); !full || notAChapter.MatchString(text[m[1]:]) {
				continue
			}
		}
		ref, err := passage.RangeOSIS(osis, g(2), g(3), g(4), g(5))
		if err != nil {
			continue
		}
		if r, err := passage.Parse(ref); err != nil || r.Lo > r.Hi {
			continue
		}
		if !seen[ref] {
			seen[ref] = true
			out = append(out, ref)
		}
	}
	return out
}

// RefsFromSlugAnyBook is RefsFromSlug for a post of no one book: when the
// address ends in a book's name and digits ("...-isaiah-531-12"), the
// passages those digits can mean in that book; nil when it does not.
func RefsFromSlugAnyBook(slug string) []string {
	toks := strings.Split(slug, "-")
	for len(toks) > 0 && toks[len(toks)-1] == "again" {
		toks = toks[:len(toks)-1]
	}
	var best *passage.Book
	bestN := 0
	for _, b := range passage.Books() {
		b := b
		names := []string{b.Name}
		if b.OSIS == "Ps" {
			names = append(names, "Psalm") // "psalm-23"
		}
		for _, name := range names {
			nt := strings.Split(strings.ToLower(name), " ")
			for k := 0; k+len(nt) < len(toks); k++ {
				if !equalToks(toks[k:k+len(nt)], nt) || len(toks)-(k+len(nt)) > 3 {
					continue
				}
				rest := toks[k+len(nt):]
				if len(rest) > 1 && rest[0] == "pt" || len(rest) > 1 && rest[0] == "part" {
					rest = rest[1:]
				}
				digits := len(rest) > 0
				for _, r := range rest {
					digits = digits && isDigits.MatchString(r)
				}
				if digits && len(nt) > bestN {
					best, bestN = &b, len(nt)
				}
			}
		}
	}
	if best == nil {
		return nil
	}
	return RefsFromSlug(best.OSIS, strings.Join(toks, "-"))
}

func equalToks(a, b []string) bool {
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return len(a) == len(b)
}
