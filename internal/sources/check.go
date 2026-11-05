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
	for _, f := range []struct{ name, value string }{{"homepage", s.Homepage}, {"terms_url", s.TermsURL}} {
		if f.value != "" {
			if msg := checkURL(f.value); msg != "" {
				add("%s '%s': %s", f.name, f.value, msg)
			}
		}
	}
	return fs
}

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
		}{{"preacher", r.Preacher}, {"notes", r.Notes}, {"date", r.Date}} {
			if f.value != nil && strings.TrimSpace(*f.value) == "" {
				add("%s is empty: leave it out instead", f.name)
			}
		}
		if r.Date != nil && *r.Date != "" && !checkDate(*r.Date) {
			add("date '%s': not YYYY, YYYY-MM or YYYY-MM-DD", *r.Date)
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
func CheckURLs(ctx context.Context, d *Dir, n int, client *http.Client, rnd *rand.Rand) []Fault {
	type target struct {
		line       int
		field, url string
		title      string
	}
	var ts []target
	for _, r := range d.Rows {
		ts = append(ts, target{r.Line, "url", r.URL, r.Title})
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
	for _, t := range ts {
		var msg string
		if t.field == "url" && titleHosts[hostOf(t.url)] {
			msg = checkTitle(ctx, client, t.url, t.title) // a GET: it needs the page, and says a non-200 too
		} else {
			msg = head(ctx, client, t.url)
		}
		if msg != "" {
			fs = append(fs, Fault{File: RowsFile, Line: t.line, Msg: fmt.Sprintf("%s '%s': %s", t.field, t.url, msg)})
		}
	}
	return fs
}

// titleHosts are the hosts that answer 200 to any address (CCEL does, to a
// section id that does not exist), so a 200 proves nothing: a row's url on
// such a host must also have the row's title in the page it answers with.
var titleHosts = map[string]bool{"ccel.org": true, "www.ccel.org": true}

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
