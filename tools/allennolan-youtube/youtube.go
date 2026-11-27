package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// The YouTube Data API v3. Only three methods are called, all public-metadata
// reads (no OAuth, no video, audio or caption is fetched):
//
//	channels.list       1 unit  (part=snippet, id=<channel id>)
//	playlists.list      1 unit a page of up to 50 (part=snippet,contentDetails, channelId=<channel id>)
//	playlistItems.list  1 unit a page of up to 50 (part=snippet,contentDetails)
//
// The costs are those of Google's reference pages for the methods ("A call to
// this method has a quota cost of 1 unit"), as tools/brookfield-youtube read
// them on 2026-11-09.
const (
	apiBase     = "https://www.googleapis.com"
	callCost    = 1
	pageSize    = 50
	keyHeader   = "X-Goog-Api-Key"
	userAgent   = "sermon-index-allennolan-youtube (+https://github.com/Jonathan-A-White/sermon-index)"
	maxErrBytes = 2000
	requestGap  = time.Second // the pause between two requests to the API
)

// Client reads the API, one request at a time with a pause between them. The
// key goes in a request header, never in the address, and any text of an error
// that holds it is made "[key]".
type Client struct {
	Base  string // the API's origin; apiBase when empty
	Key   string
	HTTP  *http.Client
	Gap   time.Duration // the pause between two requests; requestGap when 0
	Units int           // quota units spent so far
	last  time.Time     // when the last request ended
}

// Channel is the channel as channels.list gives it.
type Channel struct {
	ID     string
	Title  string
	Handle string // its custom URL, "@allennolan"
}

// Playlist is one playlist of the channel: its id, its title and the number of
// videos it holds as the API counts them.
type Playlist struct {
	ID    string
	Title string
	Count int
}

// Item is one entry of a playlist: a video as the playlist lists it.
type Item struct {
	VideoID     string
	Title       string
	Description string // read for a passage; never written out
	Published   string // RFC 3339, when the video was published
}

func (c *Client) base() string {
	if c.Base != "" {
		return strings.TrimRight(c.Base, "/")
	}
	return apiBase
}

// redact makes every occurrence of the key in s harmless.
func (c *Client) redact(s string) string {
	if c.Key == "" {
		return s
	}
	s = strings.ReplaceAll(s, c.Key, "[key]")
	return strings.ReplaceAll(s, url.QueryEscape(c.Key), "[key]")
}

func (c *Client) fail(format string, a ...any) error {
	return errors.New(c.redact(fmt.Sprintf(format, a...)))
}

// pause waits until Gap has passed since the last request ended; the first
// request does not wait.
func (c *Client) pause(ctx context.Context) error {
	if c.last.IsZero() {
		return nil
	}
	gap := c.Gap
	if gap == 0 {
		gap = requestGap
	}
	select {
	case <-time.After(gap - time.Since(c.last)):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// get calls one method, after the pause, and decodes the JSON answer into v.
func (c *Client) get(ctx context.Context, method string, q url.Values, v any) error {
	u := c.base() + "/youtube/v3/" + method + "?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return c.fail("%s: %v", method, err)
	}
	req.Header.Set(keyHeader, c.Key)
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")
	hc := c.HTTP
	if hc == nil {
		hc = http.DefaultClient
	}
	if err := c.pause(ctx); err != nil {
		return c.fail("%s: %v", method, err)
	}
	c.Units += callCost // a request costs at least a unit, even a refused one
	defer func() { c.last = time.Now() }()
	resp, err := hc.Do(req)
	if err != nil {
		return c.fail("%s: %v", method, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return c.fail("%s: %v", method, err)
	}
	if resp.StatusCode != http.StatusOK {
		var e struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		msg := ""
		if json.Unmarshal(body, &e) == nil {
			msg = e.Error.Message
		}
		if len(msg) > maxErrBytes {
			msg = msg[:maxErrBytes]
		}
		return c.fail("%s: %s: %s", method, resp.Status, msg)
	}
	if err := json.Unmarshal(body, v); err != nil {
		return c.fail("%s: answer is not the JSON expected: %v", method, err)
	}
	return nil
}

// Channel reads the channel by its id (channels.list), to say whose it is.
func (c *Client) Channel(ctx context.Context, id string) (Channel, error) {
	var out struct {
		Items []struct {
			ID      string `json:"id"`
			Snippet struct {
				Title     string `json:"title"`
				CustomURL string `json:"customUrl"`
			} `json:"snippet"`
		} `json:"items"`
	}
	q := url.Values{"part": {"snippet"}, "id": {id}}
	if err := c.get(ctx, "channels", q, &out); err != nil {
		return Channel{}, err
	}
	if len(out.Items) == 0 {
		return Channel{}, c.fail("channels.list: no channel with the id %s", id)
	}
	it := out.Items[0]
	return Channel{ID: it.ID, Title: it.Snippet.Title, Handle: it.Snippet.CustomURL}, nil
}

// Playlists lists every playlist of the channel (playlists.list), a page of 50
// at a time by nextPageToken, in the order the API gave them.
func (c *Client) Playlists(ctx context.Context, channelID string) ([]Playlist, error) {
	var pls []Playlist
	token := ""
	for {
		q := url.Values{
			"part":       {"snippet,contentDetails"},
			"maxResults": {fmt.Sprint(pageSize)},
			"channelId":  {channelID},
		}
		if token != "" {
			q.Set("pageToken", token)
		}
		var page struct {
			NextPageToken string `json:"nextPageToken"`
			Items         []struct {
				ID      string `json:"id"`
				Snippet struct {
					Title string `json:"title"`
				} `json:"snippet"`
				ContentDetails struct {
					ItemCount int `json:"itemCount"`
				} `json:"contentDetails"`
			} `json:"items"`
		}
		if err := c.get(ctx, "playlists", q, &page); err != nil {
			return nil, err
		}
		for _, it := range page.Items {
			pls = append(pls, Playlist{ID: it.ID, Title: it.Snippet.Title, Count: it.ContentDetails.ItemCount})
		}
		if page.NextPageToken == "" {
			return pls, nil
		}
		token = page.NextPageToken
	}
}

// PlaylistItems walks one playlist a page of 50 at a time, by nextPageToken,
// and returns every entry in the order the API gave them.
func (c *Client) PlaylistItems(ctx context.Context, playlist string) ([]Item, error) {
	var items []Item
	token := ""
	for {
		q := url.Values{
			"part":       {"snippet,contentDetails"},
			"maxResults": {fmt.Sprint(pageSize)},
			"playlistId": {playlist},
		}
		if token != "" {
			q.Set("pageToken", token)
		}
		var page struct {
			NextPageToken string `json:"nextPageToken"`
			Items         []struct {
				Snippet struct {
					PublishedAt string `json:"publishedAt"`
					Title       string `json:"title"`
					Description string `json:"description"`
					ResourceID  struct {
						VideoID string `json:"videoId"`
					} `json:"resourceId"`
				} `json:"snippet"`
				ContentDetails struct {
					VideoID          string `json:"videoId"`
					VideoPublishedAt string `json:"videoPublishedAt"`
				} `json:"contentDetails"`
			} `json:"items"`
		}
		if err := c.get(ctx, "playlistItems", q, &page); err != nil {
			return nil, err
		}
		for _, it := range page.Items {
			id := it.ContentDetails.VideoID
			if id == "" {
				id = it.Snippet.ResourceID.VideoID
			}
			pub := it.ContentDetails.VideoPublishedAt
			if pub == "" {
				pub = it.Snippet.PublishedAt
			}
			items = append(items, Item{
				VideoID:     id,
				Title:       it.Snippet.Title,
				Description: it.Snippet.Description,
				Published:   pub,
			})
		}
		if page.NextPageToken == "" {
			return items, nil
		}
		token = page.NextPageToken
	}
}
