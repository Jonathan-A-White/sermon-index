package sources

import (
	"context"
	"fmt"
	"html"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/Jonathan-A-White/sermon-index/internal/passage"
	"github.com/Jonathan-A-White/sermon-index/internal/themes"
)

var (
	sourceID = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)
	rowID    = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
)

// Check reads and checks a source directory and returns every fault, in
// file and line order. No fault means `sermons load` will take it.
func Check(dir string) []Fault {
	_, fs := checked(dir)
	return fs
}

// WideChapters is the most chapters a row's preaching text may span before
// `sermons check` warns: more is a series, or a whole book, and would match
// every chapter in it.
const WideChapters = 3

// CheckReport is Check and the warnings besides: a warning is a row that is
// valid and loads, but that the author should look at. There are none for a
// directory with a fault in it, but the rows that parse still warn.
func CheckReport(dir string) (faults, warnings []Fault) {
	d, faults := checked(dir)
	return faults, warn(d)
}

// CheckAllReport is CheckAll and the warnings besides, their files named
// from root as the faults' are.
func CheckAllReport(root string) (dirs []string, faults, warnings []Fault) {
	dirs, err := sourceDirs(root)
	if err != nil {
		return nil, []Fault{{File: root, Msg: err.Error()}}, nil
	}
	for _, dir := range dirs {
		fs, ws := CheckReport(dir)
		for _, f := range fs {
			f.File = filepath.Base(dir) + "/" + f.File
			faults = append(faults, f)
		}
		for _, w := range ws {
			w.File = filepath.Base(dir) + "/" + w.File
			warnings = append(warnings, w)
		}
	}
	return dirs, faults, warnings
}

// warn lists the warnings of a checked directory, in line order: a row
// whose preaching text spans more than WideChapters chapters and is not
// marked `series`.
func warn(d *Dir) []Fault {
	var ws []Fault
	for _, r := range d.Rows {
		if r.text == nil || r.IsSeries() {
			continue
		}
		if n := r.text.Chapters(); n > WideChapters {
			ws = append(ws, Fault{File: RowsFile, Line: r.Line, Msg: fmt.Sprintf(
				"warning: ref '%s' spans %d chapters: a series over it would match every chapter: give the row a `series: true` and the narrowest text, or split it",
				r.text.OSIS, n)})
		}
	}
	return ws
}

// checked reads and checks a directory; on no faults the Dir's references
// are parsed, ready to load.
func checked(dir string) (*Dir, []Fault) {
	d, faults := Read(dir)
	faults = append(faults, checkSource(d)...)
	switch d.Source.Kind {
	case KindSermons:
		faults = append(faults, checkRows(d)...)
	case KindThemes:
		faults = append(faults, checkThemes(d)...)
	}
	sort.SliceStable(faults, func(i, j int) bool {
		if faults[i].File != faults[j].File {
			return fileOrder(faults[i].File) < fileOrder(faults[j].File)
		}
		return faults[i].Line < faults[j].Line
	})
	return d, faults
}

func fileOrder(name string) int {
	for i, f := range []string{SourceFile, RowsFile, TopicsFile, LinksFile} {
		if f == name {
			return i
		}
	}
	return 99
}

func checkSource(d *Dir) []Fault {
	var fs []Fault
	add := func(format string, args ...any) {
		fs = append(fs, Fault{File: SourceFile, Msg: fmt.Sprintf(format, args...)})
	}
	s := d.Source
	if s.ID != "" {
		if !sourceID.MatchString(s.ID) {
			add("id '%s': lower-case letters, digits and '-' only", s.ID)
		}
		// A directory whose name begins with '_' (the template, a fixture's
		// parent) is never loaded by --all, so its id need not match.
		path := d.Path
		if abs, err := filepath.Abs(path); err == nil {
			path = abs // so that `sermons check .` knows the directory's name
		}
		base := filepath.Base(path)
		if base != s.ID && !strings.HasPrefix(base, "_") {
			add("id '%s' is not the directory's name '%s'", s.ID, base)
		}
	}
	if s.Kind != "" && s.Kind != KindSermons && s.Kind != KindThemes {
		add("kind '%s': not sermons or themes", s.Kind)
	}
	if s.Language != "" && !slices.Contains(Languages, s.Language) {
		add("language '%s': not one of %s", s.Language, strings.Join(Languages, ", "))
	}
	for _, f := range []struct{ name, value string }{{"homepage", s.Homepage}, {"terms_url", s.TermsURL}} {
		if f.value != "" {
			if msg := checkURL(f.value); msg != "" {
				add("%s '%s': %s", f.name, f.value, msg)
			}
		}
	}
	hosts := make([]string, 0, len(s.HostRules))
	for h := range s.HostRules {
		hosts = append(hosts, h)
	}
	sort.Strings(hosts)
	for _, h := range hosts {
		rule := s.HostRules[h]
		switch {
		case !hostName.MatchString(h):
			add("host_rules '%s': a host name only, lower case, with no scheme, path or port", h)
		case rule.TitleInPage == nil && rule.DelayMS == nil:
			add("host_rules '%s': no rule given (title_in_page or delay_ms)", h)
		case rule.DelayMS != nil && *rule.DelayMS < 0:
			add("host_rules '%s': delay_ms %d: not 0 or more", h, *rule.DelayMS)
		}
	}
	return fs
}

// hostName is a lower-case DNS name: labels of letters, digits and '-'.
var hostName = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)*$`)

// checkURL returns why a URL is refused, or "".
func checkURL(s string) string {
	u, err := url.Parse(s)
	if err != nil || u.Host == "" || u.Hostname() == "" || strings.ContainsAny(s, " \t\n") || u.Opaque != "" {
		if err == nil && u.Scheme != "" && u.Scheme != "https" {
			return "not https"
		}
		return "not a well-formed URL"
	}
	if u.Scheme != "https" {
		return "not https"
	}
	return ""
}

var dateShape = regexp.MustCompile(`^[0-9]{4}(-[0-9]{2}(-[0-9]{2})?)?$`)

// checkDate: ISO 8601 at the precision known, and a real date.
func checkDate(s string) bool {
	if !dateShape.MatchString(s) {
		return false
	}
	layout := map[int]string{4: "2006", 7: "2006-01", 10: "2006-01-02"}[len(s)]
	_, err := time.Parse(layout, s)
	return err == nil
}

func checkRows(d *Dir) []Fault {
	var fs []Fault
	seen := map[string]int{}
	for i := range d.Rows {
		r := &d.Rows[i]
		add := func(format string, args ...any) {
			fs = append(fs, Fault{File: RowsFile, Line: r.Line, Msg: fmt.Sprintf(format, args...)})
		}
		switch {
		case r.ID == "":
			add("id is missing")
		case !rowID.MatchString(r.ID):
			add("id '%s': only letters, digits, '.', '_' and '-'", r.ID)
		default:
			if prev, ok := seen[r.ID]; ok {
				add("id '%s' repeats line %d", r.ID, prev)
			} else {
				seen[r.ID] = r.Line
			}
		}
		if strings.TrimSpace(r.Title) == "" {
			add("title is missing")
		}
		if r.Ref != nil {
			if *r.Ref == "" {
				add("ref is empty: leave it out instead")
			} else if rg, err := passage.Parse(*r.Ref); err != nil {
				add("%v", err)
			} else {
				r.text = &rg
			}
		}
		r.cited = r.cited[:0]
		for _, c := range r.Cited {
			if rg, err := passage.Parse(c); err != nil {
				add("cited %v", err)
			} else {
				r.cited = append(r.cited, rg)
			}
		}
		if r.URL == "" {
			add("url is missing")
		} else if msg := checkURL(r.URL); msg != "" {
			add("url '%s': %s", r.URL, msg)
		}
		if r.AltURL != nil {
			if *r.AltURL == "" {
				add("alt_url is empty: leave it out instead")
			} else if msg := checkURL(*r.AltURL); msg != "" {
				add("alt_url '%s': %s", *r.AltURL, msg)
			}
		}
		if r.Kind != nil {
			switch *r.Kind {
			case RowLink, RowHosted, RowExposition:
			default:
				add("kind '%s': not link, hosted or exposition", *r.Kind)
			}
		}
		for _, f := range []struct {
			name  string
			value *string
		}{{"preacher", r.Preacher}, {"notes", r.Notes}, {"date", r.Date}, {"expect_title", r.ExpectTitle}} {
			if f.value != nil && strings.TrimSpace(*f.value) == "" {
				add("%s is empty: leave it out instead", f.name)
			}
		}
		if r.Date != nil && *r.Date != "" && !checkDate(*r.Date) {
			add("date '%s': not YYYY, YYYY-MM or YYYY-MM-DD", *r.Date)
		}
		if r.Series != nil && !*r.Series {
			add("series is false: leave it out instead")
		}
		if r.DurationS != nil && *r.DurationS <= 0 {
			add("duration_s %d: not a positive number of seconds", *r.DurationS)
		}
	}
	return fs
}

func checkThemes(d *Dir) []Fault {
	var fs []Fault
	topics := map[string]*themes.Topic{}
	parts := map[string]map[int]bool{}
	for i := range d.Topics {
		t := &d.Topics[i]
		add := func(format string, args ...any) {
			fs = append(fs, Fault{File: TopicsFile, Line: t.Line, Msg: fmt.Sprintf(format, args...)})
		}
		switch {
		case t.ID == "":
			add("id is missing")
		case !rowID.MatchString(t.ID):
			add("id '%s': only letters, digits, '.', '_' and '-'", t.ID)
		default:
			if prev, ok := topics[t.ID]; ok {
				add("id '%s' repeats line %d", t.ID, prev.Line)
			} else {
				topics[t.ID] = t
			}
		}
		if strings.TrimSpace(t.Label) == "" {
			add("label is missing")
		}
		ns := map[int]bool{}
		for _, p := range t.Parts {
			switch {
			case p.N <= 0:
				add("part %d: not a positive number", p.N)
			case ns[p.N]:
				add("part %d repeats", p.N)
			}
			ns[p.N] = true
			if strings.TrimSpace(p.Label) == "" {
				add("part %d: label is missing", p.N)
			}
		}
		if _, ok := parts[t.ID]; !ok {
			parts[t.ID] = ns
		}
	}
	// A bare see id is a topic of this source; '<source>:<id>' names
	// another source's and is not checked here.
	for _, t := range d.Topics {
		for _, s := range t.See {
			if !strings.Contains(s, ":") && topics[s] == nil {
				fs = append(fs, Fault{File: TopicsFile, Line: t.Line, Msg: fmt.Sprintf("see '%s': not in topics.jsonl", s)})
			}
		}
	}

	seen := map[string]int{}
	for i := range d.Links {
		l := &d.Links[i]
		add := func(format string, args ...any) {
			fs = append(fs, Fault{File: LinksFile, Line: l.Line, Msg: fmt.Sprintf(format, args...)})
		}
		switch {
		case l.Topic == "":
			add("topic is missing")
		case topics[l.Topic] == nil:
			add("topic '%s': not in topics.jsonl", l.Topic)
		case l.Part != nil && !parts[l.Topic][*l.Part]:
			add("part %d: topic '%s' has no such part", *l.Part, l.Topic)
		}
		ok := true
		if l.Ref == "" {
			add("ref is missing")
			ok = false
		} else if rg, err := passage.Parse(l.Ref); err != nil {
			add("%v", err)
			ok = false
		} else {
			l.Span = rg
		}
		switch via := l.ViaOrDefault(); via {
		case themes.ViaSource, themes.ViaBible:
		case themes.ViaSermon:
			if l.From == nil {
				add("from is missing: a link via a sermon names it as <source>:<id>")
			}
		default:
			add("via '%s': not source, bible or sermon", via)
		}
		if l.From != nil {
			if f := *l.From; f == "" {
				add("from is empty: leave it out instead")
			} else if l.ViaOrDefault() == themes.ViaSermon && !strings.Contains(f, ":") {
				add("from '%s': not <source>:<id>", f)
			}
		}
		if ok {
			part, from := "", ""
			if l.Part != nil {
				part = fmt.Sprint(*l.Part)
			}
			if l.From != nil {
				from = *l.From
			}
			key := strings.Join([]string{l.Topic, part, l.Span.OSIS, l.ViaOrDefault(), from}, "\x00")
			if prev, dup := seen[key]; dup {
				add("repeats line %d", prev)
			} else {
				seen[key] = l.Line
			}
		}
	}
	return fs
}

// CheckURLs sends a HEAD to a random n of the source's URLs (url and
// alt_url of its rows) and returns a fault for each that does not answer
// 200 OK after redirects. It uses the network; `sermons check` runs it only
// when asked (--urls N).
//
// A host with a title_in_page rule (the source's host_rules, over
// DefaultHostRules) and a row with an expect_title are asked with GET, and
// the title, or the expect_title, must also be in the page. A host with a
// delay_ms rule is waited for between requests.
func CheckURLs(ctx context.Context, d *Dir, n int, client *http.Client, rnd *rand.Rand) []Fault {
	type target struct {
		line       int
		field, url string
		title      string // required in the page ("" to ask only for a 200)
	}
	rules := hostRules(d.Source)
	var ts []target
	for _, r := range d.Rows {
		title := ""
		switch {
		case r.ExpectTitle != nil:
			title = *r.ExpectTitle
		case rules[hostOf(r.URL)].titleInPage():
			title = r.Title
		}
		ts = append(ts, target{r.Line, "url", r.URL, title})
		if r.AltURL != nil {
			ts = append(ts, target{r.Line, "alt_url", *r.AltURL, ""})
		}
	}
	rnd.Shuffle(len(ts), func(i, j int) { ts[i], ts[j] = ts[j], ts[i] })
	if n < len(ts) {
		ts = ts[:n]
	}
	sort.Slice(ts, func(i, j int) bool { return ts[i].line < ts[j].line })

	var fs []Fault
	last := map[string]time.Time{} // when each host was last asked
	for _, t := range ts {
		host := hostOf(t.url)
		if delay := rules[host].delay(); delay > 0 {
			if at, ok := last[host]; ok {
				if wait := time.Until(at.Add(delay)); wait > 0 {
					select {
					case <-time.After(wait):
					case <-ctx.Done():
					}
				}
			}
		}
		var msg string
		if t.title != "" {
			msg = checkTitle(ctx, client, t.url, t.title) // a GET: it needs the page, and says a non-200 too
		} else {
			msg = head(ctx, client, t.url)
		}
		last[host] = time.Now()
		if msg != "" {
			fs = append(fs, Fault{File: RowsFile, Line: t.line, Msg: fmt.Sprintf("%s '%s': %s", t.field, t.url, msg)})
		}
	}
	return fs
}

// DefaultHostRules are the rules every source has, which its own host_rules
// override: CCEL answers 200 to a section id that does not exist, so a 200
// proves nothing and a row's url on CCEL must have the row's title in the
// page it answers with.
var DefaultHostRules = map[string]HostRule{
	"ccel.org":     {TitleInPage: ptr(true)},
	"www.ccel.org": {TitleInPage: ptr(true)},
}

func ptr[T any](v T) *T { return &v }

// hostRules is the default rules with the source's laid over them, a key at
// a time (a source's delay_ms leaves the default title_in_page).
func hostRules(s Source) map[string]hostRule {
	out := map[string]hostRule{}
	for _, m := range []map[string]HostRule{DefaultHostRules, s.HostRules} {
		for h, r := range m {
			cur := out[h]
			if r.TitleInPage != nil {
				cur.title = *r.TitleInPage
			}
			if r.DelayMS != nil {
				cur.delayMS = *r.DelayMS
			}
			out[h] = cur
		}
	}
	return out
}

// hostRule is a HostRule with its defaults filled in.
type hostRule struct {
	title   bool
	delayMS int
}

func (r hostRule) titleInPage() bool    { return r.title }
func (r hostRule) delay() time.Duration { return time.Duration(r.delayMS) * time.Millisecond }

func hostOf(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return strings.ToLower(u.Hostname())
}

// checkTitle fetches the page (GET) and returns "" when the title is in it,
// else what is wrong.
func checkTitle(ctx context.Context, client *http.Client, u, title string) string {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err.Error()
	}
	req.Header.Set("User-Agent", "sermon-index-check (+https://github.com/Jonathan-A-White/sermon-index)")
	resp, err := client.Do(req)
	if err != nil {
		return err.Error()
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return resp.Status
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return err.Error()
	}
	if !titleInPage(title, string(body)) {
		return fmt.Sprintf("%s but the title '%s' is not in the page", resp.Status, title)
	}
	return ""
}

var (
	tagRE   = regexp.MustCompile(`<[^>]*>`)
	quoteFx = strings.NewReplacer("\u2018", "'", "\u2019", "'", "\u201c", `"`, "\u201d", `"`, "\u00a0", " ")
)

// titleInPage reports whether a title is in an HTML page's text, the markup
// stripped, entities read, curly quotes made straight, spaces collapsed and
// case ignored.
func titleInPage(title, page string) bool {
	norm := func(s string) string {
		s = quoteFx.Replace(html.UnescapeString(s))
		return strings.ToLower(strings.Join(strings.Fields(s), " "))
	}
	return strings.Contains(norm(tagRE.ReplaceAllString(page, "")), norm(title))
}

// head returns "" when the URL answers 200, else the status or the error.
func head(ctx context.Context, client *http.Client, u string) string {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, u, nil)
	if err != nil {
		return err.Error()
	}
	req.Header.Set("User-Agent", "sermon-index-check (+https://github.com/Jonathan-A-White/sermon-index)")
	resp, err := client.Do(req)
	if err != nil {
		return err.Error()
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return resp.Status
	}
	return ""
}

// CheckAll checks every source directory under root (those whose names do
// not begin with '_' or '.'); a fault's file is named from root
// ("spurgeon-ccel/rows.jsonl").
func CheckAll(root string) ([]string, []Fault) {
	dirs, err := sourceDirs(root)
	if err != nil {
		return nil, []Fault{{File: root, Msg: err.Error()}}
	}
	var all []Fault
	for _, dir := range dirs {
		for _, f := range Check(dir) {
			f.File = filepath.Base(dir) + "/" + f.File
			all = append(all, f)
		}
	}
	return dirs, all
}

// sourceDirs lists the directories under root that --all loads.
func sourceDirs(root string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var dirs []string
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), "_") || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		dirs = append(dirs, filepath.Join(root, e.Name()))
	}
	return dirs, nil
}
