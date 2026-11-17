package main

// The browser way in (subsplash-notes browse): the Governor signs in to
// Subsplash himself in the browser `mw browser open` starts on the Laptop, and
// this reads, through that one signed-in tab, what the by-hand job reads: the
// sermon list, each sermon's page, its notes and its share link. This file is
// the part that holds no browser: the pages as text, what each is expected to
// look like, the pace, and the stop rules. cdp.go drives the real Chromium.

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// MinPace is the least time between two page loads. A caller asking for less
// is given this.
const MinPace = 10 * time.Second

const (
	listURL    = "https://subsplash.com/brookfieldfirst/media"
	sermonBase = "https://subsplash.com/u/brookfieldfirst/media/d/"
	shortBase  = "https://brookfieldfirst.subspla.sh/"
)

// Link is an anchor of a page.
type Link struct{ Text, Href string }

// Page is what one load showed: the address it ended on, the HTTP status of
// the load (0 when none was seen), the page's title and visible text, its
// anchors, whether it has a Notes button, and, for the list, how many pages the
// pager counts.
type Page struct {
	URL      string
	Status   int
	Title    string
	Text     string
	Links    []Link
	HasNotes bool
	Pages    int
	// SignedIn is whether the page shows the account menu of a signed-in
	// reader (some pages print "Logged in as ..." instead).
	SignedIn bool
	// Total and More are a series page's: how many episodes the series has,
	// and whether its View More button is still there.
	Total int
	More  bool
}

// Browser is the one signed-in tab. Open loads an address; OpenNotes presses
// the Notes button of the page it is on and gives the notes page that opens;
// ListPage presses page n of the list's pager. Each is one page load.
type Browser interface {
	Open(ctx context.Context, url string) (Page, error)
	OpenNotes(ctx context.Context) (Page, error)
	ListPage(ctx context.Context, n int) (Page, error)
	// More presses the series page's View More and gives the page with the
	// episodes it added.
	More(ctx context.Context) (Page, error)
}

// Stop is the reason a run ends where it is: a login prompt, a captcha, an
// HTTP 429 or 403, or a page that is not the one expected. Nothing is written
// after it, and the next run resumes where this one stopped.
type Stop struct{ Why string }

func (s Stop) Error() string { return "stopped: " + s.Why }

// Kind is which page a load is expected to give.
type Kind int

const (
	KindList Kind = iota
	KindSermon
	KindNotes
	KindEpisode
	KindSeries
	KindSeriesIndex
)

// seriesURL is the Morning RePhil series on the church's site, which lists
// every episode with its date and share code, twelve a View More.
const seriesURL = "https://brookfieldfirst.org/media/series/4fmk6cd/morning-rephil"

// seriesIndexURL is the church site's list of its series, fifteen to a page:
// page n after the first is seriesIndexURL + "page/n/". Subsplash's own sermon
// list holds only the newest 200 items, so the series pages (each lists every
// sermon of its series, twelve a View More) are the way to the older ones.
const seriesIndexURL = "https://brookfieldfirst.org/media/"

var (
	seriesPageRE  = regexp.MustCompile(`^https://brookfieldfirst\.org/media/series/([0-9a-z]+)/[a-z0-9-]+$`)
	seriesIndexRE = regexp.MustCompile(`^https://brookfieldfirst\.org/media/(?:page/[0-9]+/)?$`)
	sermonURLRE   = regexp.MustCompile(`^https://subsplash\.com/u/brookfieldfirst/media/d/([0-9a-z]+)-([a-z0-9-]+)$`)
	captchaWords  = []string{"captcha", "verify you are human", "verify you're human", "unusual traffic", "are you a robot", "not a robot"}
	loginWords    = []string{"sign in", "log in", "password"}
)

// CodeOf is the short-link code and slug of a sermon's address
// (https://subsplash.com/u/brookfieldfirst/media/d/3nxf2bb-unfinished-business),
// which carries the code of its share link.
func CodeOf(address string) (code, slug string, ok bool) {
	m := sermonURLRE.FindStringSubmatch(address)
	if m == nil {
		return "", "", false
	}
	return m[1], m[2], true
}

// readable is whether the tool may load the address: the sermon list, a
// sermon's own page, the church site's series index and a series' page, and
// nothing else.
func readable(address string) bool {
	if address == listURL || address == seriesURL || seriesIndexRE.MatchString(address) || seriesPageRE.MatchString(address) {
		return true
	}
	_, _, ok := CodeOf(address)
	return ok
}

// Check says whether the page is the one expected. A Stop names why not: an
// HTTP 429 or 403, a captcha, a login prompt (or a sermon page that does not
// say the reader is signed in), or any other page.
func Check(p Page, kind Kind) error {
	if p.Status == 429 || p.Status == 403 {
		return Stop{fmt.Sprintf("HTTP %d from %s", p.Status, p.URL)}
	}
	low := strings.ToLower(p.Text)
	for _, w := range captchaWords {
		if strings.Contains(low, w) {
			return Stop{"a captcha at " + p.URL}
		}
	}
	lowURL := strings.ToLower(p.URL)
	for _, w := range []string{"/login", "/signin", "/sign-in", "/sign_in"} {
		if strings.Contains(lowURL, w) {
			return Stop{"a login prompt at " + p.URL}
		}
	}
	if kind == KindSeries {
		if !seriesPageRE.MatchString(strings.TrimSuffix(p.URL, "/")) {
			return Stop{"unexpected page: " + p.URL + " is not a series of the church's site"}
		}
		return nil
	}
	if kind == KindSeriesIndex {
		if p.Status >= 400 || strings.Contains(low, "page not found") {
			return Stop{fmt.Sprintf("unexpected page: HTTP %d, not found at %s", p.Status, p.URL)}
		}
		if !seriesIndexRE.MatchString(p.URL) {
			return Stop{"unexpected page: " + p.URL + " is not the church's list of series"}
		}
		return nil
	}
	signedIn := p.SignedIn || strings.Contains(p.Text, "Logged in as ")
	if !signedIn {
		for _, w := range loginWords {
			if strings.Contains(low, w) && kind != KindNotes {
				return Stop{"a login prompt at " + p.URL}
			}
		}
		if kind == KindSermon || kind == KindEpisode {
			return Stop{"a login prompt: the page at " + p.URL + " does not say the reader is signed in"}
		}
	}
	if p.Status >= 400 {
		return Stop{fmt.Sprintf("unexpected page: HTTP %d from %s", p.Status, p.URL)}
	}
	if strings.Contains(low, "page not found") {
		return Stop{"unexpected page: not found at " + p.URL}
	}
	switch kind {
	case KindList:
		if !strings.HasPrefix(p.URL, listURL) {
			return Stop{"unexpected page: " + p.URL + " is not the sermon list"}
		}
	case KindSermon, KindEpisode:
		if _, _, ok := CodeOf(p.URL); !ok {
			return Stop{"unexpected page: " + p.URL + " is not a sermon's page"}
		}
	case KindNotes:
		if !strings.HasPrefix(p.URL, "https://notes.subsplash.com/") {
			return Stop{"unexpected page: " + p.URL + " is not a notes page"}
		}
	}
	return nil
}

// PacedBrowser is a Browser that waits so that no two loads come less than the
// interval apart, and loads no address but the list and a sermon's own page. It
// is one tab: it makes one call at a time and a caller must not share it.
type PacedBrowser struct {
	b        Browser
	interval time.Duration
	now      func() time.Time
	sleep    func(time.Duration)
	last     time.Time
	loads    int
}

// Paced wraps b. An interval under MinPace is raised to it.
func Paced(b Browser, interval time.Duration, now func() time.Time, sleep func(time.Duration)) *PacedBrowser {
	if interval < MinPace {
		interval = MinPace
	}
	if now == nil {
		now = time.Now
	}
	if sleep == nil {
		sleep = time.Sleep
	}
	return &PacedBrowser{b: b, interval: interval, now: now, sleep: sleep}
}

// Loads is how many page loads have been made.
func (p *PacedBrowser) Loads() int { return p.loads }

func (p *PacedBrowser) wait() {
	if !p.last.IsZero() {
		if d := p.interval - p.now().Sub(p.last); d > 0 {
			p.sleep(d)
		}
	}
}

func (p *PacedBrowser) done() {
	p.last = p.now()
	p.loads++
}

func (p *PacedBrowser) Open(ctx context.Context, address string) (Page, error) {
	if !readable(address) {
		return Page{}, fmt.Errorf("%s is not a page this reads (the sermon list and a sermon's page only)", address)
	}
	p.wait()
	defer p.done()
	return p.b.Open(ctx, address)
}

func (p *PacedBrowser) OpenNotes(ctx context.Context) (Page, error) {
	p.wait()
	defer p.done()
	return p.b.OpenNotes(ctx)
}

func (p *PacedBrowser) More(ctx context.Context) (Page, error) {
	p.wait()
	defer p.done()
	return p.b.More(ctx)
}

func (p *PacedBrowser) ListPage(ctx context.Context, n int) (Page, error) {
	p.wait()
	defer p.done()
	return p.b.ListPage(ctx, n)
}

// Item is one entry of the list: a sermon or a Morning RePhil episode.
type Item struct {
	Title string
	Code  string
	Slug  string
	// Rephil is the episode's number from its title; -1 for an episode whose
	// title carries none; 0 for a sermon.
	Rephil int
}

// Address is the item's full sermon address, the form subsplash-links takes.
func (i Item) Address() string { return sermonBase + i.Code + "-" + i.Slug }

// Short is the item's share link: the short link its code makes.
func (i Item) Short() string { return shortBase + i.Code }

var (
	rephilRE    = regexp.MustCompile(`(?i)^morning\s+rephil\b`)
	rephilNumRE = regexp.MustCompile(`#\s*(\d+)`)
)

// RephilNumber is 0 for a title that is not a Morning RePhil episode's, its
// number when it prints one, and -1 when it prints none.
func RephilNumber(title string) int {
	if !rephilRE.MatchString(strings.TrimSpace(title)) {
		return 0
	}
	if m := rephilNumRE.FindStringSubmatch(title); m != nil {
		n, _ := strconv.Atoi(m[1])
		if n > 0 {
			return n
		}
	}
	return -1
}

// ParseList reads the sermon list: its items in the order shown (newest
// first) and how many pages the pager counts. A page with no item is no list.
func ParseList(p Page) (items []Item, pages int, err error) {
	seen := map[string]bool{}
	for _, l := range p.Links {
		code, slug, ok := CodeOf(l.Href)
		title := strings.TrimSpace(l.Text)
		if !ok || title == "" || seen[code] {
			continue
		}
		seen[code] = true
		items = append(items, Item{Title: title, Code: code, Slug: slug, Rephil: RephilNumber(title)})
	}
	if len(items) == 0 {
		return nil, 0, Stop{"unexpected page: no sermon in the list at " + p.URL}
	}
	pages = p.Pages
	if pages < 1 {
		pages = 1
	}
	return items, pages, nil
}

var metaRE = regexp.MustCompile(`^([A-Z][a-z]{2} \d{1,2}, \d{4})(?: • (.*))?$`)

func lines(text string) []string {
	var out []string
	for _, l := range strings.Split(text, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			out = append(out, l)
		}
	}
	return out
}

// Sermon is what a sermon's page says: its title, its Sunday, the preacher and
// the passage when it prints them, and whether it has notes.
type Sermon struct {
	Code      string
	Title     string
	Date      string // YYYY-MM-DD
	Speaker   string
	Reference string // as printed
	HasNotes  bool
}

// ParseSermon reads a sermon's page: the title is the line above the date line
// ("Nov 8, 2026 • Phil Morgan • Acts 1:8"), after which come a preacher and a
// passage, either of which may be left out.
func ParseSermon(p Page) (Sermon, error) {
	code, _, ok := CodeOf(p.URL)
	if !ok {
		return Sermon{}, Stop{"unexpected page: " + p.URL + " is not a sermon's page"}
	}
	ls := lines(p.Text)
	for i, l := range ls {
		m := metaRE.FindStringSubmatch(l)
		if m == nil || i == 0 {
			continue
		}
		t, err := time.Parse("Jan 2, 2006", m[1])
		if err != nil {
			continue
		}
		s := Sermon{Code: code, Title: ls[i-1], Date: t.Format(dateLayout), HasNotes: p.HasNotes}
		if m[2] != "" {
			for _, part := range strings.Split(m[2], " • ") {
				part = strings.TrimSpace(part)
				if part == "" {
					continue
				}
				if _, err := ReferencesToOSIS(part); err == nil && s.Reference == "" {
					s.Reference = part
				} else if s.Speaker == "" {
					s.Speaker = part
				}
			}
		}
		return s, nil
	}
	return Sermon{}, Stop{"unexpected page: no date line on the sermon page " + p.URL}
}

// NotesPage is a sermon's notes page: the title and passage of its first line,
// the date of its second, and the page's text, which is kept.
type NotesPage struct {
	Title     string
	Reference string
	Date      string // YYYY-MM-DD, empty when the page prints none
	Text      string
}

var (
	notesTitleRE = regexp.MustCompile(`^["“](.+?)["”](?:\s*[—–-]\s*(.+))?$`)
	notesPartRE  = regexp.MustCompile(`^Part of .*?\s+([A-Z][a-z]+ \d{1,2}, \d{4})$`)
)

// ParseNotes reads the page the Notes button opens: first a line
// `"Title" — Passage`, then `Part of <series>  <date>`.
func ParseNotes(p Page) (NotesPage, error) {
	ls := lines(p.Text)
	if len(ls) == 0 {
		return NotesPage{}, Stop{"unexpected page: nothing on the notes page " + p.URL}
	}
	m := notesTitleRE.FindStringSubmatch(ls[0])
	if m == nil {
		return NotesPage{}, Stop{"unexpected page: the notes page " + p.URL + " does not open with a quoted title"}
	}
	n := NotesPage{Title: strings.TrimSpace(m[1]), Reference: strings.TrimSpace(m[2]), Text: p.Text}
	for _, l := range ls[1:] {
		if pm := notesPartRE.FindStringSubmatch(l); pm != nil {
			if t, err := time.Parse("January 2, 2006", pm[1]); err == nil {
				n.Date = t.Format(dateLayout)
			}
			break
		}
	}
	return n, nil
}

// Episode is a Morning RePhil episode's page: its title, date, and any text
// the page carries between its buttons and the list of the other episodes.
type Episode struct {
	Code   string
	Title  string
	Number int // 0 when the title prints none
	Date   string
	Text   string
}

// ParseEpisode reads an episode's page.
func ParseEpisode(p Page) (Episode, error) {
	code, _, ok := CodeOf(p.URL)
	if !ok {
		return Episode{}, Stop{"unexpected page: " + p.URL + " is not an episode's page"}
	}
	ls := lines(p.Text)
	for i, l := range ls {
		m := metaRE.FindStringSubmatch(l)
		if m == nil || i == 0 {
			continue
		}
		t, err := time.Parse("Jan 2, 2006", m[1])
		if err != nil {
			continue
		}
		e := Episode{Code: code, Title: ls[i-1], Date: t.Format(dateLayout)}
		if n := RephilNumber(e.Title); n > 0 {
			e.Number = n
		}
		var text []string
		started := false
		for _, l := range ls[i+1:] {
			switch {
			case strings.HasPrefix(l, "More from "), strings.HasPrefix(l, "Related by "), strings.HasPrefix(l, "Media streaming powered"):
				return finishEpisode(e, text), nil
			case l == "Share" || l == "Download" || l == "Notes":
				started = true
			case started:
				text = append(text, l)
			}
		}
		return finishEpisode(e, text), nil
	}
	return Episode{}, Stop{"unexpected page: no date line on the episode page " + p.URL}
}

func finishEpisode(e Episode, text []string) Episode {
	e.Text = strings.Join(text, "\n")
	return e
}

var errNoBrowser = errors.New("no browser to read through: run `mw browser open https://subsplash.com/brookfieldfirst/media`, sign in, and give --cdp its address")

// SeriesItem is an episode of the series page: its number from its title
// (0 when it prints none), title, date and share code.
type SeriesItem struct {
	Number int
	Title  string
	Date   string
	Code   string
	Slug   string
}

var seriesLinkRE = regexp.MustCompile(`^https://brookfieldfirst\.org/media/([0-9a-z]+)/([a-z0-9-]+)$`)

// ParseSeries reads the episodes the series page shows so far, newest first.
// Each anchor's text is the title and, on the next line, the date.
func ParseSeries(p Page) ([]SeriesItem, error) {
	var items []SeriesItem
	seen := map[string]bool{}
	for _, l := range p.Links {
		m := seriesLinkRE.FindStringSubmatch(l.Href)
		if m == nil || seen[m[1]] {
			continue
		}
		ls := lines(l.Text)
		if len(ls) < 2 {
			return nil, Stop{"unexpected page: an episode of " + p.URL + " has no date under its title"}
		}
		// The line under the title is the date, and may go on with more
		// after a bullet (a preacher).
		dm := metaRE.FindStringSubmatch(strings.Join(strings.Fields(ls[1]), " "))
		var t time.Time
		var err error
		if dm != nil {
			t, err = time.Parse("Jan 2, 2006", dm[1])
		}
		if dm == nil || err != nil {
			return nil, Stop{"unexpected page: an episode of " + p.URL + " has the date '" + ls[1] + "'"}
		}
		seen[m[1]] = true
		it := SeriesItem{Title: ls[0], Date: t.Format(dateLayout), Code: m[1], Slug: m[2]}
		if n := RephilNumber(ls[0]); n > 0 {
			it.Number = n
		}
		items = append(items, it)
	}
	if len(items) == 0 {
		return nil, Stop{"unexpected page: no episode in the series at " + p.URL}
	}
	return items, nil
}

// ParseSeriesIndex reads a page of the church site's list of series: the
// address of each series it shows, in the order shown. A page with none is no
// index.
func ParseSeriesIndex(p Page) ([]string, error) {
	var out []string
	seen := map[string]bool{}
	for _, l := range p.Links {
		m := seriesPageRE.FindStringSubmatch(l.Href)
		if m == nil || seen[m[1]] {
			continue
		}
		seen[m[1]] = true
		out = append(out, l.Href)
	}
	if len(out) == 0 {
		return nil, Stop{"unexpected page: no series in the list at " + p.URL}
	}
	return out, nil
}
