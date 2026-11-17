package api

// GET /v1/text?id=<sermon id>[&source=<source id>]: the words of a sermon
// whose source is marked `text = "public-domain"` in its source.toml.
//
// The index keeps facts only (docs/sources.md), never words, so the words
// are read from the sermon's own page (its row's url) on the first request
// and kept on disk, under Config.TextDir, never in the repository: one fetch
// a sermon, ever. Only the sermon's body is served, not the site's
// navigation, the head block of the sermon (its number, date and place) or
// any editor's note. A reader exists for each host (readers below); a host
// with none is not served.
//
// The fetches are polite: one at a time to a host, at least TextDelay apart
// (CCEL's robots.txt asks Crawl-delay: 10), with a User-Agent that says who
// asks. A request that would wait longer than textMaxWait for its turn is
// answered 503 busy with Retry-After.

import (
	"context"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Jonathan-A-White/sermon-index/internal/store"
)

// DefaultTextDelay is the pause between two fetches from one host: CCEL's
// Crawl-delay (https://ccel.org/robots.txt, read 2026-11-16).
const DefaultTextDelay = 10 * time.Second

// textMaxWait is the longest a request waits for its turn at a host: the
// server's write timeout is 30 seconds.
const textMaxWait = 20 * time.Second

// textMaxPage is the biggest page read: a sermon page is about 60 KB.
const textMaxPage = 4 << 20

const textUserAgent = "sermon-index (+https://github.com/Jonathan-A-White/sermon-index)"

type textResp struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	Source     string   `json:"source"`
	URL        string   `json:"url"`
	Paragraphs []string `json:"paragraphs"`
}

// readers are the pages' readers by host: each takes a saved page and gives
// the sermon's body as paragraphs.
var readers = map[string]func(page []byte) ([]string, error){
	"ccel.org":     ccelParagraphs,
	"www.ccel.org": ccelParagraphs,
}

// text is GET /v1/text.
func (s *server) text(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	id := strings.TrimSpace(q.Get("id"))
	if id == "" {
		fail(w, r, http.StatusBadRequest, "bad_id", "id is required: a sermon's id, such as 311 (and source, when two sources have that id)")
		return
	}
	source := strings.TrimSpace(q.Get("source"))
	found, err := store.TextSermons(r.Context(), s.db, id, source)
	if err != nil {
		s.dbFailed(w, r, err)
		return
	}
	if len(found) == 0 {
		detail := "no sermon has the id '" + id + "'"
		if source != "" {
			detail += " in the source '" + source + "'"
		}
		fail(w, r, http.StatusNotFound, "not_found", detail)
		return
	}
	var served []store.TextSermon
	for _, f := range found {
		if f.Text {
			served = append(served, f)
		}
	}
	switch {
	case len(served) == 0:
		fail(w, r, http.StatusNotFound, "no_text", "the words of sermon '"+id+"' are not served: its source ("+found[0].Source+
			") is not marked as public-domain text; the index keeps only a link to it")
		return
	case len(served) > 1:
		var ids []string
		for _, f := range served {
			ids = append(ids, f.Source)
		}
		fail(w, r, http.StatusBadRequest, "ambiguous_id", "more than one source has a sermon '"+id+"' with text ("+strings.Join(ids, ", ")+"): add source=")
		return
	}
	m := served[0]
	u, err := url.Parse(m.URL)
	read := readers[strings.ToLower(u.Hostname())]
	if err != nil || read == nil {
		fail(w, r, http.StatusNotFound, "no_text", "the words of sermon '"+id+"' are not served: there is no reader for its page's host")
		return
	}
	page, err := s.pages.get(r.Context(), m.Source, m.ID, m.URL, read)
	switch {
	case errors.Is(err, errBusy):
		w.Header().Set("Retry-After", strconv.Itoa(int(s.pages.delay/time.Second)+1))
		fail(w, r, http.StatusServiceUnavailable, "busy", "the sermon's page is fetched slowly, one at a time; try again shortly")
		return
	case r.Context().Err() != nil:
		return // the client went away
	case errors.Is(err, errNoText):
		s.errLog.Printf("%s %s: %v", r.Method, r.URL.Path, err)
		fail(w, r, http.StatusBadGateway, "no_text_in_page", "the sermon's page has no sermon text on it")
		return
	case err != nil:
		s.errLog.Printf("%s %s: %v", r.Method, r.URL.Path, err)
		fail(w, r, http.StatusBadGateway, "fetch_failed", "the sermon's page could not be fetched; try again later")
		return
	}
	reply(w, r, textResp{ID: m.ID, Title: m.Title, Source: m.Source, URL: m.URL, Paragraphs: page})
}

var (
	errBusy   = errors.New("the wait for a turn is too long")
	errNoText = errors.New("no sermon text in the page")
)

// pageCache fetches sermon pages slowly and keeps them on disk.
type pageCache struct {
	dir   string
	fetch func(ctx context.Context, url string) ([]byte, error)
	delay time.Duration

	mu       sync.Mutex
	next     map[string]time.Time // the earliest the next fetch from a host may start
	inflight map[string]*pageCall // by cache file
}

type pageCall struct {
	done chan struct{}
	page []string
	err  error
}

func newPageCache(dir string, fetch func(context.Context, string) ([]byte, error), delay time.Duration) *pageCache {
	if dir == "" {
		base, err := os.UserCacheDir()
		if err != nil {
			base = os.TempDir()
		}
		dir = filepath.Join(base, "sermon-index", "text")
	}
	if fetch == nil {
		fetch = httpFetch
	}
	if delay <= 0 {
		delay = DefaultTextDelay
	}
	return &pageCache{dir: dir, fetch: fetch, delay: delay, next: map[string]time.Time{}, inflight: map[string]*pageCall{}}
}

// file is where a sermon's page is kept: a file a sermon, in a directory a
// source. A sermon id is letters, digits, '.', '_' and '-' (the schema's
// check), and the suffix keeps '..' from being a directory.
func (c *pageCache) file(source, id string) string {
	return filepath.Join(c.dir, source, id+".html")
}

// get is the sermon's paragraphs: from the disk, else fetched (waiting for
// the host's turn), read, and kept. A page that does not read is not kept.
// Callers for one sermon share one fetch.
func (c *pageCache) get(ctx context.Context, source, id, pageURL string, read func([]byte) ([]string, error)) ([]string, error) {
	file := c.file(source, id)
	if b, err := os.ReadFile(file); err == nil {
		if ps, err := read(b); err == nil {
			return ps, nil
		}
	}
	c.mu.Lock()
	if call, ok := c.inflight[file]; ok {
		c.mu.Unlock()
		select {
		case <-call.done:
			return call.page, call.err
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	call := &pageCall{done: make(chan struct{})}
	c.inflight[file] = call
	c.mu.Unlock()
	// The fetch is not tied to this caller's request: others wait on it, and
	// a page fetched for a reader who left is kept for the next.
	fctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), textMaxWait+30*time.Second)
	call.page, call.err = c.fetchAndKeep(fctx, file, pageURL, read)
	cancel()
	c.mu.Lock()
	delete(c.inflight, file)
	c.mu.Unlock()
	close(call.done)
	return call.page, call.err
}

func (c *pageCache) fetchAndKeep(ctx context.Context, file, pageURL string, read func([]byte) ([]string, error)) ([]string, error) {
	u, err := url.Parse(pageURL)
	if err != nil {
		return nil, err
	}
	host := strings.ToLower(u.Hostname())
	// Take the next turn at the host.
	c.mu.Lock()
	now := time.Now()
	start := c.next[host]
	if start.Before(now) {
		start = now
	}
	if start.Sub(now) > textMaxWait {
		c.mu.Unlock()
		return nil, errBusy
	}
	c.next[host] = start.Add(c.delay)
	c.mu.Unlock()
	if wait := time.Until(start); wait > 0 {
		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	b, err := c.fetch(ctx, pageURL)
	if err != nil {
		return nil, fmt.Errorf("fetching %s: %w", pageURL, err)
	}
	ps, err := read(b)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", pageURL, err)
	}
	if err := writeFileAtomic(file, b); err != nil {
		return nil, fmt.Errorf("keeping %s: %w", pageURL, err)
	}
	return ps, nil
}

func writeFileAtomic(file string, b []byte) error {
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(file), ".page-*")
	if err != nil {
		return err
	}
	_, err = tmp.Write(b)
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Rename(tmp.Name(), file)
	}
	if err != nil {
		os.Remove(tmp.Name())
	}
	return err
}

var textClient = &http.Client{Timeout: 20 * time.Second}

// httpFetch GETs a page over https as a polite client.
func httpFetch(ctx context.Context, pageURL string) ([]byte, error) {
	if !strings.HasPrefix(pageURL, "https://") {
		return nil, fmt.Errorf("not an https address: %s", pageURL)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", textUserAgent)
	res, err := textClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s answered %s", pageURL, res.Status)
	}
	return io.ReadAll(io.LimitReader(res.Body, textMaxPage))
}

var (
	ccelEnd   = regexp.MustCompile(`id="book_navbar_bottom"`)
	ccelNote  = regexp.MustCompile(`(?is)<(div|span|a|sup|p)\b[^>]*\bclass="[^"]*\b(?:notes?|footnotes?|fn[a-z]*)\b[^"]*"[^>]*>`)
	ccelBlock = regexp.MustCompile(`(?is)<(?:p|h[2-6])\b[^>]*>(.*?)</(?:p|h[2-6])>`)
	htmlTag   = regexp.MustCompile(`(?s)<[^>]*>`)
	ccelHead  = regexp.MustCompile(`(?i)^(?:(?:an?|the) (?:sermon|exposition)\b.{0,80}|\(?nos?\.? ?\d.{0,40}|(?:delivered|intended for|preached|published|written)\b.*|(?:the )?rev\..*|by .{0,60}|at [^.]{0,60}\.?)$`)
)

// ccelParagraphs reads a CCEL section page: the paragraphs of the sermon
// inside the page's book-content block, after the head block (its number,
// date, preacher and place), without markup or notes. CCEL answers 200 to a
// section it does not have, so a page with no book-content or no words is
// errNoText.
func ccelParagraphs(page []byte) ([]string, error) {
	s := string(page)
	i := strings.Index(s, `class="book-content"`)
	if i < 0 {
		return nil, errNoText
	}
	if j := strings.Index(s[i:], ">"); j >= 0 {
		s = s[i+j+1:]
	}
	if loc := ccelEnd.FindStringIndex(s); loc != nil {
		s = s[:loc[0]]
	}
	if k := strings.LastIndex(s, "</div>"); k >= 0 {
		s = s[:k]
	}
	s = dropNotes(s)
	var out []string
	inHead := true
	for _, m := range ccelBlock.FindAllStringSubmatch(s, -1) {
		p := strings.Join(strings.Fields(strings.ReplaceAll(html.UnescapeString(htmlTag.ReplaceAllString(m[1], "")), " ", " ")), " ")
		if p == "" {
			continue
		}
		if inHead && ccelHead.MatchString(p) {
			continue
		}
		inHead = false
		out = append(out, p)
	}
	if len(out) == 0 {
		return nil, errNoText
	}
	return out, nil
}

// dropNotes removes each element whose class says it is a note or a
// footnote (an editor's note, a footnote's mark), with what is inside it.
func dropNotes(s string) string {
	for {
		loc := ccelNote.FindStringSubmatchIndex(s)
		if loc == nil {
			return s
		}
		name := strings.ToLower(s[loc[2]:loc[3]])
		tags := regexp.MustCompile(`(?i)<(/?)` + name + `\b`)
		depth, end := 1, len(s)
		for _, m := range tags.FindAllStringSubmatchIndex(s[loc[1]:], -1) {
			if m[2] == m[3] {
				depth++
			} else if depth--; depth == 0 {
				if gt := strings.Index(s[loc[1]+m[1]:], ">"); gt >= 0 {
					end = loc[1] + m[1] + gt + 1
				}
				break
			}
		}
		s = s[:loc[0]] + s[end:]
	}
}
