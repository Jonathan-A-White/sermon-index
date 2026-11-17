package main

// CDP is the Browser over Chromium's DevTools protocol, the one the browser
// `mw browser open` starts on the Laptop speaks on 127.0.0.1 (the endpoint
// `mw browser status` prints). It reads the pages of the tab he signed in
// on; it types no password, sets no cookie and keeps no session: the sign-in
// lives in that browser until `mw browser close` deletes it. Only the Go
// standard library is used, so the WebSocket is spoken here (RFC 6455, client
// side, text frames).

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// DefaultCDP is where `mw browser open` puts the DevTools port.
const DefaultCDP = "http://127.0.0.1:9222"

type wsConn struct {
	c net.Conn
	r *bufio.Reader
}

// dialWS opens a WebSocket to a ws:// address on this machine.
func dialWS(wsURL string) (*wsConn, error) {
	u, err := url.Parse(wsURL)
	if err != nil || u.Scheme != "ws" {
		return nil, fmt.Errorf("%s is not a ws:// address", wsURL)
	}
	c, err := net.DialTimeout("tcp", u.Host, 10*time.Second)
	if err != nil {
		return nil, err
	}
	key := make([]byte, 16)
	rand.Read(key)
	k := base64.StdEncoding.EncodeToString(key)
	fmt.Fprintf(c, "GET %s HTTP/1.1\r\nHost: %s\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: %s\r\nSec-WebSocket-Version: 13\r\n\r\n", u.RequestURI(), u.Host, k)
	br := bufio.NewReader(c)
	resp, err := http.ReadResponse(br, nil)
	if err != nil {
		c.Close()
		return nil, err
	}
	sum := sha1.Sum([]byte(k + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	if resp.StatusCode != 101 || resp.Header.Get("Sec-WebSocket-Accept") != base64.StdEncoding.EncodeToString(sum[:]) {
		c.Close()
		return nil, fmt.Errorf("the browser refused the WebSocket: %s", resp.Status)
	}
	return &wsConn{c: c, r: br}, nil
}

func (w *wsConn) Close() error { return w.c.Close() }

// send writes one masked text frame.
func (w *wsConn) send(payload []byte) error {
	hdr := []byte{0x81}
	switch n := len(payload); {
	case n < 126:
		hdr = append(hdr, 0x80|byte(n))
	case n < 1<<16:
		hdr = append(hdr, 0x80|126, byte(n>>8), byte(n))
	default:
		hdr = append(hdr, 0x80|127)
		hdr = binary.BigEndian.AppendUint64(hdr, uint64(n))
	}
	mask := make([]byte, 4)
	rand.Read(mask)
	hdr = append(hdr, mask...)
	body := make([]byte, len(payload))
	for i, b := range payload {
		body[i] = b ^ mask[i%4]
	}
	w.c.SetWriteDeadline(time.Now().Add(30 * time.Second))
	_, err := w.c.Write(append(hdr, body...))
	return err
}

// recv reads one whole message, answering a ping, until deadline.
func (w *wsConn) recv(deadline time.Time) ([]byte, error) {
	w.c.SetReadDeadline(deadline)
	var msg []byte
	for {
		var h [2]byte
		if _, err := io.ReadFull(w.r, h[:]); err != nil {
			return nil, err
		}
		fin, op := h[0]&0x80 != 0, h[0]&0x0f
		n := uint64(h[1] & 0x7f)
		switch n {
		case 126:
			var b [2]byte
			if _, err := io.ReadFull(w.r, b[:]); err != nil {
				return nil, err
			}
			n = uint64(binary.BigEndian.Uint16(b[:]))
		case 127:
			var b [8]byte
			if _, err := io.ReadFull(w.r, b[:]); err != nil {
				return nil, err
			}
			n = binary.BigEndian.Uint64(b[:])
		}
		if n > 64<<20 {
			return nil, errors.New("a WebSocket frame over 64 MiB")
		}
		var mask [4]byte
		masked := h[1]&0x80 != 0
		if masked {
			if _, err := io.ReadFull(w.r, mask[:]); err != nil {
				return nil, err
			}
		}
		p := make([]byte, n)
		if _, err := io.ReadFull(w.r, p); err != nil {
			return nil, err
		}
		if masked {
			for i := range p {
				p[i] ^= mask[i%4]
			}
		}
		switch op {
		case 0x8:
			return nil, io.EOF
		case 0x9: // ping: pong with the same bytes
			w.c.SetWriteDeadline(time.Now().Add(10 * time.Second))
			w.c.Write(append([]byte{0x8a, 0x80 | byte(len(p)), 0, 0, 0, 0}, p...))
			continue
		case 0xa:
			continue
		}
		msg = append(msg, p...)
		if fin {
			return msg, nil
		}
	}
}

// target is one tab of /json.
type target struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	URL   string `json:"url"`
	WS    string `json:"webSocketDebuggerUrl"`
	Title string `json:"title"`
}

// CDP drives the one signed-in tab.
type CDP struct {
	Endpoint string
	// Poll is how often a page that is loading is looked at (not a load).
	Poll time.Duration
	// Settle is the longest a page is waited for to stop changing; Grace is
	// how long of that a "Page not found" is given to go away.
	Settle time.Duration
	Grace  time.Duration

	http   *http.Client
	tab    target
	ws     *wsConn
	id     int
	status []seen
}

// seen is a response the browser reported while a page loaded.
type seen struct {
	URL    string
	Status int
	Doc    bool
}

// NewCDP connects to the browser at endpoint (a loopback address only) and
// takes its tab: the first tab on subsplash.com, else the first tab that is a
// page.
func NewCDP(endpoint string) (*CDP, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	if h := u.Hostname(); h != "127.0.0.1" && h != "localhost" && h != "::1" {
		return nil, fmt.Errorf("the browser's DevTools address %s is not on this machine: only 127.0.0.1 is used", endpoint)
	}
	c := &CDP{Endpoint: strings.TrimRight(endpoint, "/"), Poll: time.Second, Settle: 40 * time.Second, Grace: 12 * time.Second, http: &http.Client{Timeout: 15 * time.Second}}
	ts, err := c.targets()
	if err != nil {
		return nil, fmt.Errorf("%w (%v)", errNoBrowser, err)
	}
	var pick *target
	for i, t := range ts {
		if t.Type != "page" || t.WS == "" {
			continue
		}
		if strings.HasPrefix(t.URL, "https://subsplash.com/") && pick == nil {
			pick = &ts[i]
		}
	}
	if pick == nil {
		for i, t := range ts {
			if t.Type == "page" && t.WS != "" && !strings.HasPrefix(t.URL, "https://notes.subsplash.com/") {
				pick = &ts[i]
				break
			}
		}
	}
	if pick == nil {
		return nil, errors.New("the browser has no tab to read through")
	}
	c.tab = *pick
	if c.ws, err = dialWS(c.tab.WS); err != nil {
		return nil, err
	}
	if _, err := c.call("Network.enable", nil); err != nil {
		return nil, err
	}
	return c, nil
}

// Close lets go of the tab; the browser and its session stay as they were.
func (c *CDP) Close() error {
	if c.ws != nil {
		return c.ws.Close()
	}
	return nil
}

func (c *CDP) targets() ([]target, error) {
	resp, err := c.http.Get(c.Endpoint + "/json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var ts []target
	if err := json.NewDecoder(resp.Body).Decode(&ts); err != nil {
		return nil, err
	}
	return ts, nil
}

// call sends one command and waits for its reply, taking note of the
// responses the browser reports on the way.
func (c *CDP) call(method string, params any) (json.RawMessage, error) {
	return c.callOn(c.ws, method, params, true)
}

func (c *CDP) callOn(w *wsConn, method string, params any, track bool) (json.RawMessage, error) {
	c.id++
	id := c.id
	req, _ := json.Marshal(map[string]any{"id": id, "method": method, "params": params})
	if err := w.send(req); err != nil {
		return nil, err
	}
	deadline := time.Now().Add(60 * time.Second)
	for {
		b, err := w.recv(deadline)
		if err != nil {
			return nil, err
		}
		var m struct {
			ID     int             `json:"id"`
			Method string          `json:"method"`
			Result json.RawMessage `json:"result"`
			Error  *struct {
				Message string `json:"message"`
			} `json:"error"`
			Params struct {
				Type     string `json:"type"`
				Response struct {
					URL    string `json:"url"`
					Status int    `json:"status"`
				} `json:"response"`
			} `json:"params"`
		}
		if json.Unmarshal(b, &m) != nil {
			continue
		}
		if m.Method == "Network.responseReceived" && track {
			c.status = append(c.status, seen{m.Params.Response.URL, m.Params.Response.Status, m.Params.Type == "Document"})
			continue
		}
		if m.ID != id {
			continue
		}
		if m.Error != nil {
			return nil, fmt.Errorf("%s: %s", method, m.Error.Message)
		}
		return m.Result, nil
	}
}

// eval runs a JavaScript expression in a tab and returns its string value.
func (c *CDP) eval(w *wsConn, expr string) (string, error) {
	res, err := c.callOn(w, "Runtime.evaluate", map[string]any{"expression": expr, "returnByValue": true, "awaitPromise": false}, w == c.ws)
	if err != nil {
		return "", err
	}
	var r struct {
		Result struct {
			Value any `json:"value"`
		} `json:"result"`
		Exception *struct {
			Text string `json:"text"`
		} `json:"exceptionDetails"`
	}
	if err := json.Unmarshal(res, &r); err != nil {
		return "", err
	}
	if r.Exception != nil {
		return "", fmt.Errorf("the page raised: %s", r.Exception.Text)
	}
	switch v := r.Result.Value.(type) {
	case string:
		return v, nil
	case nil:
		return "", nil
	default:
		b, _ := json.Marshal(v)
		return string(b), nil
	}
}

const snapshotJS = `JSON.stringify({
 url: location.href, title: document.title, ready: document.readyState,
 text: document.body ? document.body.innerText : "",
 links: [...document.querySelectorAll('a')].map(a => ({t: (a.innerText||'').trim(), h: a.href})),
 notes: !!document.querySelector('[data-testid="Notes"]'),
 account: !!document.querySelector('#account-menu-button'),
 total: parseInt((document.querySelector('.sp-media-library') || {dataset: {}}).dataset.total) || 0,
 more: !!document.querySelector('#view-more-button') && document.querySelector('#view-more-button').offsetParent !== null,
 pages: Math.max(1, ...[...document.querySelectorAll('.kit-pagination__page')].map(e => parseInt(e.textContent.trim()) || 0))
})`

type snap struct {
	URL   string `json:"url"`
	Title string `json:"title"`
	Ready string `json:"ready"`
	Text  string `json:"text"`
	Links []struct {
		T string `json:"t"`
		H string `json:"h"`
	} `json:"links"`
	Notes   bool `json:"notes"`
	Account bool `json:"account"`
	Pages   int  `json:"pages"`
	Total   int  `json:"total"`
	More    bool `json:"more"`
}

func (c *CDP) snapshot(w *wsConn) (snap, error) {
	var s snap
	out, err := c.eval(w, snapshotJS)
	if err != nil {
		return s, err
	}
	return s, json.Unmarshal([]byte(out), &s)
}

var debug = os.Getenv("SUBSPLASH_NOTES_DEBUG") != ""

func firstLine(s string) string {
	if len(s) > 70 {
		s = s[:70]
	}
	return strings.ReplaceAll(s, "\n", " / ")
}

// settled waits for the page to be loaded and its text to stop changing (the
// pages are single-page apps that fill in after the load), looking at it
// every Poll for at most Settle. These looks load nothing. A sermon page
// (wantNotes) is not taken to have no Notes button until Grace has passed: the
// button can render after the text has held still.
func (c *CDP) settled(w *wsConn, notSame, urlPrefix string, wantNotes bool) (snap, error) {
	deadline := time.Now().Add(c.Settle)
	var prev snap
	stable := 0
	for {
		s, err := c.snapshot(w)
		if err != nil {
			return s, err
		}
		if debug {
			fmt.Fprintf(os.Stderr, "cdp: %s ready=%s links=%d text=%d more=%v total=%d %q\n", s.URL, s.Ready, len(s.Links), len(s.Text), s.More, s.Total, firstLine(s.Text))
		}
		// The app can show "Page not found" for a moment before its data
		// arrives: a page that says so is given a while to change.
		young := strings.Contains(strings.ToLower(s.Text), "page not found") && time.Until(deadline) > c.Settle-c.Grace
		if wantNotes && !s.Notes && time.Until(deadline) > c.Settle-c.Grace {
			young = true
		}
		if !young && s.Ready == "complete" && s.Text != "" && s.Text != notSame && strings.HasPrefix(s.URL, urlPrefix) && s.Text == prev.Text {
			stable++
			if stable >= 2 {
				return s, nil
			}
		} else {
			stable = 0
		}
		prev = s
		if time.Now().After(deadline) {
			return s, nil
		}
		time.Sleep(c.Poll)
	}
}

func isSermonAddress(address string) bool {
	_, _, ok := CodeOf(address)
	return ok
}

func (c *CDP) page(s snap, status int) Page {
	p := Page{URL: s.URL, Title: s.Title, Text: s.Text, HasNotes: s.Notes, Pages: s.Pages, Status: status, SignedIn: s.Account, Total: s.Total, More: s.More}
	for _, l := range s.Links {
		p.Links = append(p.Links, Link{Text: l.T, Href: l.H})
	}
	return p
}

// statusOf is the status of the load just made: a 429 or 403 from any
// Subsplash host wins, else the main document's.
func (c *CDP) statusOf(final string) int {
	doc := 0
	for _, s := range c.status {
		if (s.Status == 429 || s.Status == 403) && (strings.Contains(s.URL, "subsplash.") || strings.Contains(s.URL, "subspla.sh")) {
			return s.Status
		}
		if s.Doc && s.URL == final {
			doc = s.Status
		}
	}
	return doc
}

func (c *CDP) Open(_ context.Context, address string) (Page, error) {
	c.status = nil
	res, err := c.call("Page.navigate", map[string]any{"url": address})
	if err != nil {
		return Page{}, err
	}
	var nav struct {
		ErrorText string `json:"errorText"`
	}
	json.Unmarshal(res, &nav)
	if nav.ErrorText != "" {
		return Page{}, fmt.Errorf("the browser could not load %s: %s", address, nav.ErrorText)
	}
	// Until the tab is on the address, what it shows is the page before.
	s, err := c.settled(c.ws, "", address, isSermonAddress(address))
	if err != nil {
		return Page{}, err
	}
	return c.page(s, c.statusOf(s.URL)), nil
}

// OpenNotes presses the Notes button. The notes open in a new tab (or in
// this one); the page is read and the new tab closed, so one tab is open at
// a time.
func (c *CDP) OpenNotes(_ context.Context) (Page, error) {
	c.status = nil
	before, err := c.targets()
	if err != nil {
		return Page{}, err
	}
	known := map[string]bool{}
	for _, t := range before {
		known[t.ID] = true
	}
	here := c.tab.URL
	if s, err := c.snapshot(c.ws); err == nil {
		here = s.URL
	}
	// The button is a React Native element: only a real pointer press, as
	// from a hand, opens its notes.
	if err := c.press(`document.querySelector('[data-testid="Notes"]')`); err != nil {
		return Page{}, err
	}
	deadline := time.Now().Add(c.Settle)
	for time.Now().Before(deadline) {
		time.Sleep(c.Poll)
		ts, err := c.targets()
		if err != nil {
			return Page{}, err
		}
		for _, t := range ts {
			if t.Type == "page" && !known[t.ID] && t.WS != "" {
				w, err := dialWS(t.WS)
				if err != nil {
					return Page{}, err
				}
				s, err := c.settled(w, "", "https://notes.subsplash.com/", false)
				w.Close()
				if resp, herr := c.http.Get(c.Endpoint + "/json/close/" + t.ID); herr == nil {
					resp.Body.Close()
				}
				if err != nil {
					return Page{}, err
				}
				return c.page(s, c.statusOf(s.URL)), nil
			}
		}
		if s, err := c.snapshot(c.ws); err == nil && s.URL != here && strings.HasPrefix(s.URL, "https://notes.subsplash.com/") {
			s, err = c.settled(c.ws, "", "https://notes.subsplash.com/", false)
			if err != nil {
				return Page{}, err
			}
			return c.page(s, c.statusOf(s.URL)), nil
		}
	}
	return Page{URL: "(no notes page opened)"}, nil
}

// ListPage loads page n of the list: the address the pager gives it
// (https://subsplash.com/brookfieldfirst/media?page=n). The pager is never
// pressed, so the tab need not be on the list.
func (c *CDP) ListPage(ctx context.Context, n int) (Page, error) {
	return c.Open(ctx, fmt.Sprintf("%s?page=%d", listURL, n))
}

// press moves the pointer to the middle of the element the expression finds,
// presses and releases it, as a hand would.
func (c *CDP) press(element string) error {
	if _, err := c.eval(c.ws, `(() => { const e = `+element+`; if (e) e.scrollIntoView({block: "center", behavior: "instant"}); return "" })()`); err != nil {
		return err
	}
	time.Sleep(700 * time.Millisecond) // the scroll comes to rest
	out, err := c.eval(c.ws, `(() => { const e = `+element+`; if (!e) return ""; const r = e.getBoundingClientRect(); return JSON.stringify({x: r.left + r.width / 2, y: r.top + r.height / 2}) })()`)
	if err != nil {
		return err
	}
	if out == "" {
		return errors.New("the button is not on the page")
	}
	var pt struct{ X, Y float64 }
	if err := json.Unmarshal([]byte(strings.ToLower(out)), &pt); err != nil {
		return err
	}
	for _, typ := range []string{"mouseMoved", "mousePressed", "mouseReleased"} {
		if _, err := c.call("Input.dispatchMouseEvent", map[string]any{"type": typ, "x": pt.X, "y": pt.Y, "button": "left", "buttons": 1, "clickCount": 1}); err != nil {
			return err
		}
	}
	return nil
}

// More presses the series page's View More as a hand would and waits for the
// episodes it adds.
func (c *CDP) More(_ context.Context) (Page, error) {
	c.status = nil
	prev, err := c.snapshot(c.ws)
	if err != nil {
		return Page{}, err
	}
	if err := c.press(`document.querySelector('#view-more-button')`); err != nil {
		return Page{}, err
	}
	s, err := c.settled(c.ws, prev.Text, "", false)
	if err != nil {
		return Page{}, err
	}
	if s.Text == prev.Text {
		// A press that moved nothing: the button's own click handler.
		if _, err := c.eval(c.ws, `(() => { const e = document.querySelector('#view-more-button'); if (e) e.click(); return "" })()`); err != nil {
			return Page{}, err
		}
		if s, err = c.settled(c.ws, prev.Text, "", false); err != nil {
			return Page{}, err
		}
	}
	return c.page(s, c.statusOf(s.URL)), nil
}
