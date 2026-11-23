package main

import (
	"fmt"
	"html"
	"regexp"
	"sort"
	"strconv"
	"strings"

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
