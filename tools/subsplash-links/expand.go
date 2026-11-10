package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// userAgent names no person and no machine.
const userAgent = "link-reader/1.0 (reads where a shared short link points; one HEAD each)"

// minInterval is the least the tool waits between two requests.
const minInterval = time.Second

// Expander reads where a short link on brookfieldfirst.subspla.sh points,
// with one HTTP HEAD each, the redirect not followed and no body read, at
// most one request a second. Every answer that is a redirect is kept in the
// Cache, so a link is never requested twice.
type Expander struct {
	Client    *http.Client
	Cache     *Cache
	Interval  time.Duration
	UserAgent string
	Now       func() time.Time
	Sleep     func(time.Duration)

	last    time.Time
	stopped error
}

// noRedirect makes the client return the redirect itself instead of
// following it.
func noRedirect(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }

// NewExpander is an Expander over the cache c (which may be nil: no cache is
// then kept).
func NewExpander(c *Cache) *Expander {
	if c == nil {
		c = &Cache{m: map[string]string{}}
	}
	return &Expander{
		Client:    &http.Client{Timeout: 30 * time.Second, CheckRedirect: noRedirect},
		Cache:     c,
		Interval:  minInterval,
		UserAgent: userAgent,
		Now:       time.Now,
		Sleep:     time.Sleep,
	}
}

// interval is the wait between requests, never under a second.
func (e *Expander) interval() time.Duration {
	if e.Interval < minInterval {
		return minInterval
	}
	return e.Interval
}

// Expand is the full address the short link with this code points to, with
// its query dropped. A cached code makes no request. Otherwise one HEAD goes
// to https://brookfieldfirst.subspla.sh/<code>; a redirect is cached (and
// the cache saved) whatever it points at, and any other answer is an error
// and is not cached. After a 403 or 429 no further request is made.
func (e *Expander) Expand(code string) (string, error) {
	if !codeRE.MatchString(code) {
		return "", fmt.Errorf("'%s' is not a short code: not requested", code)
	}
	if addr, ok := e.Cache.Get(code); ok {
		return addr, nil
	}
	if e.stopped != nil {
		return "", fmt.Errorf("stopped asking: %w", e.stopped)
	}
	if !e.last.IsZero() {
		if wait := e.interval() - e.Now().Sub(e.last); wait > 0 {
			e.Sleep(wait)
		}
	}
	e.last = e.Now()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodHead, "https://"+shortHost+"/"+code, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", e.UserAgent)
	resp, err := e.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("HEAD %s/%s: %w", shortHost, code, err)
	}
	resp.Body.Close() // a HEAD has no body, and none is read
	switch resp.StatusCode {
	case http.StatusForbidden, http.StatusTooManyRequests:
		e.stopped = fmt.Errorf("%s answered %s to %s", shortHost, resp.Status, code)
		return "", e.stopped
	case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther, http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
	default:
		return "", fmt.Errorf("HEAD %s/%s answered %s, not a redirect", shortHost, code, resp.Status)
	}
	loc, err := resp.Location()
	if err != nil {
		return "", fmt.Errorf("HEAD %s/%s: %s without a usable Location: %w", shortHost, code, resp.Status, err)
	}
	if loc.Scheme != "https" || loc.Host != "subsplash.com" {
		return "", fmt.Errorf("HEAD %s/%s points to %s, not to subsplash.com", shortHost, code, describe(loc))
	}
	dest := &url.URL{Scheme: "https", Host: loc.Host, Path: loc.Path}
	e.Cache.Put(code, dest.String())
	if err := e.Cache.Save(); err != nil {
		return dest.String(), fmt.Errorf("the cache is not saved: %w", err)
	}
	return dest.String(), nil
}

// describe is a URL for a report: its host and path, nothing after.
func describe(u *url.URL) string {
	s := u.Host + u.Path
	if s == "" {
		return "an address with no host"
	}
	return strings.TrimSpace(s)
}
