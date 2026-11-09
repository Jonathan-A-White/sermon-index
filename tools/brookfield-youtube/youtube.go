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
)

// The YouTube Data API v3. Only two methods are called, both public-metadata
// reads (no OAuth, no video, audio or caption is fetched):
//
//	channels.list       1 unit  (part=contentDetails, forHandle=@BrookfieldFirst)
//	playlistItems.list  1 unit a page of up to 50 (part=snippet,contentDetails)
//
// The costs are those of Google's reference pages for the two methods, read
// on 2026-11-09 ("A call to this method has a quota cost of 1 unit").
const (
	apiBase     = "https://www.googleapis.com"
	callCost    = 1
	pageSize    = 50
	keyHeader   = "X-Goog-Api-Key"
	userAgent   = "sermon-index-brookfield-youtube (+https://github.com/Jonathan-A-White/sermon-index)"
	maxErrBytes = 2000
)

// Client reads the API. The key goes in a request header, never in the
// address, and any text of an error that holds it is made "[key]".
type Client struct {
	Base  string // the API's origin; apiBase when empty
	Key   string
	HTTP  *http.Client
	Units int // quota units spent so far
}

// Item is one entry of a playlist: a video as the uploads playlist lists it.
type Item struct {
	VideoID     string
	Kind        string // the resource kind, "youtube#video"
	Title       string
	Description string // read for a preacher and a passage; never written out
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

// get calls one method and decodes the JSON answer into v.
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
	c.Units += callCost // a request costs at least a unit, even a refused one
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
		return c.fail("%s: %s: %s", method, resp.Status, msg)
	}
	if err := json.Unmarshal(body, v); err != nil {
		return c.fail("%s: answer is not the JSON expected: %v", method, err)
	}
	return nil
}

// UploadsPlaylist resolves a handle ("@BrookfieldFirst") to the id of the
// channel's uploads playlist (channels.list, forHandle).
func (c *Client) UploadsPlaylist(ctx context.Context, handle string) (string, error) {
	var out struct {
		Items []struct {
			ContentDetails struct {
				RelatedPlaylists struct {
					Uploads string `json:"uploads"`
				} `json:"relatedPlaylists"`
			} `json:"contentDetails"`
		} `json:"items"`
	}
	q := url.Values{"part": {"contentDetails"}, "forHandle": {handle}}
	if err := c.get(ctx, "channels", q, &out); err != nil {
		return "", err
	}
	if len(out.Items) == 0 || out.Items[0].ContentDetails.RelatedPlaylists.Uploads == "" {
		return "", c.fail("channels.list: no channel with the handle %s", handle)
	}
	return out.Items[0].ContentDetails.RelatedPlaylists.Uploads, nil
}

// Uploads walks the playlist a page of 50 at a time, by nextPageToken, and
// returns every entry in the order the API gave them.
func (c *Client) Uploads(ctx context.Context, playlist string) ([]Item, error) {
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
						Kind    string `json:"kind"`
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
				Kind:        it.Snippet.ResourceID.Kind,
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
