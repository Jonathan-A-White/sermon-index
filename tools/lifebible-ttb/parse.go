package main

import (
	"encoding/xml"
	"html"
	"io"
	"regexp"
	"strings"

	"github.com/Jonathan-A-White/sermon-index/internal/passage"
)

const (
	site      = "https://www.lifebibleministry.com"
	ttbPrefix = site + "/through-the-bible"
	userAgent = "sermon-index-lifebible-ttb (+https://github.com/Jonathan-A-White/sermon-index)"
)

// disallowed are the paths the site's robots.txt (User-Agent: *) disallows,
// as read on 2026-11-23. The tool never asks for one.
var disallowed = []string{"/api/", "/auth/", "/console/", "/locked/", "/background"}

// Disallowed reports whether robots.txt disallows the path.
func Disallowed(path string) bool {
	for _, d := range disallowed {
		if strings.HasPrefix(path, d) {
			return true
		}
	}
	return false
}

// ParseSitemap returns, in order, every address of the sitemap that is the
// Through the Bible page or under it.
func ParseSitemap(r io.Reader) ([]string, error) {
	var doc struct {
		URLs []struct {
			Loc string `xml:"loc"`
		} `xml:"url"`
	}
	if err := xml.NewDecoder(r).Decode(&doc); err != nil {
		return nil, err
	}
	var out []string
	for _, u := range doc.URLs {
		loc := strings.TrimSpace(u.Loc)
		if loc == ttbPrefix || strings.HasPrefix(loc, ttbPrefix+"/") {
			out = append(out, loc)
		}
	}
	return out, nil
}

// slugOf is the address's path below /through-the-bible ("isaiah/1"), or ""
// for the list page itself.
func slugOf(u string) string {
	return strings.Trim(strings.TrimPrefix(u, ttbPrefix), "/")
}

// Page is what the tool keeps of a page: the title of its <title> without the
// site's name, and the id of the YouTube video it embeds ("" when none).
type Page struct {
	Title   string
	YouTube string
}

var (
	titleRE   = regexp.MustCompile(`(?s)<title[^>]*>(.*?)</title>`)
	embedURL  = regexp.MustCompile(`"embedUrl"\s*:\s*"https://www\.youtube(?:-nocookie)?\.com/embed/([A-Za-z0-9_-]{11})"`)
	embedAny  = regexp.MustCompile(`https://www\.youtube(?:-nocookie)?\.com/embed/([A-Za-z0-9_-]{11})`)
	siteSuffx = " | Life Bible Ministry"
)

// ParsePage reads the title and the YouTube id from a page's HTML. The id is
// the embedUrl of the page's VideoObject (its JSON-LD), else the first
// youtube.com/embed address in the page.
func ParsePage(body string) Page {
	var p Page
	if m := titleRE.FindStringSubmatch(body); m != nil {
		p.Title = siteTitle(m[1])
	}
	if m := embedURL.FindStringSubmatch(body); m != nil {
		p.YouTube = m[1]
	} else if m := embedAny.FindStringSubmatch(body); m != nil {
		p.YouTube = m[1]
	}
	return p
}

// siteTitle decodes the entities of a <title> and drops the site's name.
func siteTitle(raw string) string {
	t := strings.Join(strings.Fields(html.UnescapeString(raw)), " ")
	return strings.TrimSpace(strings.TrimSuffix(t, siteSuffx))
}

var (
	bookRE  = regexp.MustCompile(`(?i)^(` + passage.BookNamePattern() + `)\.?(?:\s+|$|:|•)`)
	partRE  = regexp.MustCompile(`(?i)^\s*\(Part\s*\d+\)`)
	chapRE  = regexp.MustCompile(`(?i)^(?:chapter\s+)?(\d{1,3})\b`)
	colonRE = regexp.MustCompile(`^\s*:\s*(\d{1,3}[a-z]?)\b`)
	bareVRE = regexp.MustCompile(`^\s+(\d{1,3}[a-z]?)\b`)
	rangeRE = regexp.MustCompile(`(?i)^\s*(?:[-–—]|\bthru\b|\bthrough\b|\bto\b|\band\b)\s*`)
	restRE  = regexp.MustCompile(`^\s*(?:•|:|[-–—])?\s*`)
)

// point is a place a passage starts or ends at: a chapter, and a verse or "".
type point struct {
	c, v string
	word bool // the chapter was written "chapter 6"
}

// endpoint reads "4", "chapter 4", "4:2", "3 (Part 1)" or "3 (Part 1) 1-17"
// (a verse after a part, with no colon) from the start of s.
func endpoint(s string) (p point, rest string, ok bool) {
	m := chapRE.FindStringSubmatch(s)
	if m == nil {
		return p, s, false
	}
	p.c = m[1]
	p.word = strings.HasPrefix(strings.ToLower(m[0]), "chapter")
	s = s[len(m[0]):]
	part := false
	if m := partRE.FindString(s); m != "" {
		part = true
		s = s[len(m):]
	}
	if m := colonRE.FindStringSubmatch(s); m != nil {
		p.v = m[1]
		s = s[len(m[0]):]
	} else if m := bareVRE.FindStringSubmatch(s); m != nil && part {
		p.v = m[1]
		s = s[len(m[0]):]
	}
	return p, s, true
}

// ParseTitle reads the passage a page's title opens with as an OSIS
// reference: "Isaiah 1-4:1: God confronts ..." (Isa.1.1-Isa.4.1), "Nehemiah
// 3–4: ..." (Neh.3-Neh.4), "Matthew 7 (Part 1) :1-5 • Judging Others",
// "Genesis 28–Genesis 30 (Part 1): ...", "Isaiah chapter 4:2 thru chapter 6:
// ...", or a book's name alone ("Obadiah: God's Judgment on Edom": the whole
// book). It returns the reference and the title as the site prints it (a
// title the page prints twice, "X • Y: Y", once). series is true for a whole
// book of more than 3 chapters, whose ref is then its first chapter. ok is
// false when the title opens with no passage, one the book table does not
// have, or a book and a part with no chapter ("Romans (Part 1) :1-17").
func ParseTitle(title string) (ref, shown string, series, ok bool) {
	m := bookRE.FindStringSubmatch(title)
	if m == nil {
		return "", "", false, false
	}
	book, found := passage.BookOSIS(m[1])
	if !found {
		return "", "", false, false
	}
	s := title[len(m[0]):]
	// The separator after a book's name alone belongs to what follows it.
	switch {
	case strings.HasSuffix(m[0], ":"):
		s = ":" + s
	case strings.HasSuffix(m[0], "•"):
		s = "•" + s
	}
	var from, to point
	var haveTo bool
	if p, r, good := endpoint(s); good {
		from, s = p, r
		if m := rangeRE.FindString(s); m != "" {
			r := s[len(m):]
			bookAgain := false
			if bm := bookRE.FindStringSubmatch(r); bm != nil {
				if b2, found := passage.BookOSIS(bm[1]); found && b2 == book {
					r = r[len(bm[0]):]
					bookAgain = true
				}
			}
			if p2, r2, good := endpoint(r); good {
				to, s, haveTo = p2, r2, true
				if from.v != "" && to.v == "" && !bookAgain && !to.word {
					// "3:1-5": the number after the dash is a verse of the same chapter.
					to = point{c: from.c, v: to.c}
				}
			}
		}
	} else if strings.HasPrefix(strings.TrimSpace(s), ":") || strings.HasPrefix(strings.TrimSpace(s), "•") || strings.TrimSpace(s) == "" {
		// A book's name alone: the whole book. "Genesis: Through the Bible" is a
		// book's list page, not a teaching.
		if strings.HasSuffix(title, ": Through the Bible") {
			return "", "", false, false
		}
		rg, err := passage.Parse(book)
		if err != nil {
			return "", "", false, false
		}
		ref = book
		switch n := rg.Chapters(); {
		case n == 1:
			ref = book + ".1"
		case n > 3:
			series = true
			ref = book + ".1"
		}
		return ref, dedupe(title), series, true
	} else {
		return "", "", false, false
	}

	start := book + "." + from.c
	if from.v != "" {
		start += "." + from.v
	}
	ref = start
	switch {
	case haveTo:
		end := book + "." + to.c
		if to.v != "" {
			end += "." + to.v
			if from.v == "" {
				ref = start + ".1"
			}
		}
		ref += "-" + end
	case from.v != "":
	}
	if _, err := passage.Parse(ref); err != nil {
		return "", "", false, false
	}
	return ref, dedupe(title), false, true
}

// dedupe prints a title the page gives twice ("X • Y: Y") once.
func dedupe(title string) string {
	if i := strings.Index(title, " • "); i >= 0 {
		rest := title[i+len(" • "):]
		if j := strings.LastIndex(rest, ": "); j >= 0 && rest[:j] == rest[j+2:] {
			return title[:i] + " • " + rest[:j]
		}
	}
	return title
}
