package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	userAgent    = "sermon-index-biblethinker (+https://github.com/Jonathan-A-White/sermon-index)"
	maxPages     = 60  // no series has more pages than this; a loop of next links stops here
	restPageSize = 100 // posts a REST page, the most WordPress gives
)

// Fetcher reads the site: one request at a time, a pause before each request
// after the first, a User-Agent that names the project, and robots.txt
// obeyed. What it fetched is kept in Cache (never committed) and not fetched
// again.
type Fetcher struct {
	Base  string // the site's origin; siteRoot when empty
	HTTP  *http.Client
	Cache string        // directory for the pages fetched
	Delay time.Duration // the pause between two requests to the site
	Retry time.Duration // the pause before asking again after a server error, times the attempt; 10 s when 0
	Log   io.Writer     // one line a request; nil for none

	robots *Robots
	last   time.Time
	n      int // requests made
}

func (f *Fetcher) base() string {
	if f.Base != "" {
		return strings.TrimRight(f.Base, "/")
	}
	return siteRoot
}

// Requests is how many requests the fetcher made to the site.
func (f *Fetcher) Requests() int { return f.n }

// get returns the body of GET Base+path from the cache or the site.
func (f *Fetcher) get(ctx context.Context, path string) ([]byte, error) {
	if f.robots != nil {
		p := path
		if i := strings.Index(p, "?"); i >= 0 {
			p = p[:i]
		}
		if !f.robots.Allowed(p) {
			return nil, fmt.Errorf("robots.txt disallows %s: not fetched", p)
		}
	}
	name := filepath.Join(f.Cache, cacheName(path))
	if b, err := os.ReadFile(name); err == nil {
		return b, nil
	}
	if f.n > 0 && f.Delay > 0 {
		if wait := f.Delay - time.Since(f.last); wait > 0 {
			select {
			case <-time.After(wait):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}
	client := f.HTTP
	if client == nil {
		client = &http.Client{Timeout: 2 * time.Minute}
	}
	retry := f.Retry
	if retry == 0 {
		retry = 10 * time.Second
	}
	var body []byte
	for attempt := 1; ; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, f.base()+path, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", userAgent)
		f.n++
		f.last = time.Now()
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		body, err = io.ReadAll(io.LimitReader(resp.Body, 32<<20))
		resp.Body.Close()
		if f.Log != nil {
			fmt.Fprintf(f.Log, "GET %s %d\n", path, resp.StatusCode)
		}
		if err != nil {
			return nil, err
		}
		if resp.StatusCode >= 500 && attempt < 4 {
			// a server error is the site's, not an answer: wait and ask again, politely
			select {
			case <-time.After(time.Duration(attempt) * retry):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
			continue
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("GET %s: %s", path, resp.Status)
		}
		break
	}
	if err := os.MkdirAll(f.Cache, 0o755); err != nil {
		return nil, err
	}
	return body, os.WriteFile(name, body, 0o644)
}

func cacheName(path string) string {
	n := strings.Trim(path, "/")
	n = strings.NewReplacer("/", "_", "?", "_", "&", "_", "=", "_", ",", "_").Replace(n)
	if n == "" {
		n = "index"
	}
	return n
}

// LoadRobots reads robots.txt and keeps its rules for every later request.
func (f *Fetcher) LoadRobots(ctx context.Context) error {
	f.robots = nil
	b, err := f.get(ctx, "/robots.txt")
	if err != nil {
		return err
	}
	f.robots = ParseRobots(string(b))
	return nil
}

// RobotsText is the robots.txt as read (from the cache or the site).
func (f *Fetcher) RobotsText(ctx context.Context) (string, error) {
	r := f.robots
	f.robots = nil // robots.txt itself is not behind its own rules
	b, err := f.get(ctx, "/robots.txt")
	f.robots = r
	return string(b), err
}

// SeriesData is what the site gave for one series.
type SeriesData struct {
	Pages    []string       // the archive pages read, in order
	Episodes []Episode      // every article of them
	Dates    map[int]string // post id to the REST API's date
	Count    int            // the series category's post count, as the REST API says
}

// FetchSeries reads a series: its archive page and every later page (the
// rel="next" links), then the category's count and its posts' dates from the
// WordPress REST API (id and date only).
func (f *Fetcher) FetchSeries(ctx context.Context, s Series) (*SeriesData, error) {
	out := &SeriesData{Dates: map[int]string{}}
	path := "/" + s.Slug + "/"
	for len(out.Pages) < maxPages && path != "" {
		page, err := f.get(ctx, path)
		if err != nil {
			return nil, err
		}
		out.Pages = append(out.Pages, f.base()+path)
		eps, next, err := ParseArchive(page)
		if err != nil {
			return nil, fmt.Errorf("%s: %v", path, err)
		}
		out.Episodes = append(out.Episodes, eps...)
		path = ""
		if next != "" {
			u, err := url.Parse(next)
			if err != nil {
				return nil, fmt.Errorf("%s: next page %q: %v", s.Slug, next, err)
			}
			path = u.Path
		}
	}
	if path != "" {
		return nil, fmt.Errorf("%s: more than %d archive pages", s.Slug, maxPages)
	}

	b, err := f.get(ctx, "/wp-json/wp/v2/categories?slug="+url.QueryEscape(s.Slug)+"&_fields=id,count,slug")
	if err != nil {
		return nil, err
	}
	var cats []struct {
		ID    int    `json:"id"`
		Count int    `json:"count"`
		Slug  string `json:"slug"`
	}
	if err := json.Unmarshal(b, &cats); err != nil {
		return nil, fmt.Errorf("%s: category: %v", s.Slug, err)
	}
	if len(cats) != 1 || cats[0].Slug != s.Slug {
		return nil, fmt.Errorf("%s: the REST API has %d categories of that slug", s.Slug, len(cats))
	}
	out.Count = cats[0].Count
	for page := 1; ; page++ {
		b, err := f.get(ctx, "/wp-json/wp/v2/posts?categories="+strconv.Itoa(cats[0].ID)+"&per_page="+strconv.Itoa(restPageSize)+"&page="+strconv.Itoa(page)+"&_fields=id,date")
		if err != nil {
			return nil, err
		}
		var posts []struct {
			ID   int    `json:"id"`
			Date string `json:"date"`
		}
		if err := json.Unmarshal(b, &posts); err != nil {
			return nil, fmt.Errorf("%s: posts: %v", s.Slug, err)
		}
		for _, p := range posts {
			if len(p.Date) >= 10 {
				out.Dates[p.ID] = p.Date[:10] // the day: the site stamps every post at midnight
			}
		}
		if len(posts) < restPageSize {
			break
		}
	}
	return out, nil
}

// FetchBody reads one post's page (from the cache or the site) and returns the
// text of its own words (BodyText). Only the passage in it is used; nothing of
// the text is kept in a row.
func (f *Fetcher) FetchBody(ctx context.Context, e Episode) (string, error) {
	u, err := url.Parse(e.URL)
	if err != nil {
		return "", err
	}
	page, err := f.get(ctx, u.Path)
	if err != nil {
		return "", err
	}
	return BodyText(page), nil
}

// Robots is the rules of robots.txt's "User-agent: *" group.
type Robots struct{ disallow []string }

// ParseRobots reads the Disallow lines of the "*" group; an empty Disallow
// forbids nothing.
func ParseRobots(text string) *Robots {
	r := &Robots{}
	inStar, afterAgent := false, false
	for _, line := range strings.Split(text, "\n") {
		if i := strings.Index(line, "#"); i >= 0 {
			line = line[:i]
		}
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		k, v = strings.ToLower(strings.TrimSpace(k)), strings.TrimSpace(v)
		switch k {
		case "user-agent":
			if !afterAgent {
				inStar = false
			}
			afterAgent = true
			if v == "*" {
				inStar = true
			}
		case "disallow":
			afterAgent = false
			if inStar && v != "" {
				r.disallow = append(r.disallow, v)
			}
		default:
			afterAgent = false
		}
	}
	return r
}

// Allowed reports whether robots.txt lets a robot fetch the path.
func (r *Robots) Allowed(path string) bool {
	for _, d := range r.disallow {
		if strings.HasPrefix(path, d) {
			return false
		}
	}
	return true
}
