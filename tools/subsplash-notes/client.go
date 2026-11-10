package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// maxReply is the most of a reply the client reads: Postern keeps 30 days of
// alerts of at most a few hundred bytes each.
const maxReply = 8 << 20

// Client asks Postern's alerts routes (docs/protocol.md section 26 there) as
// the sermon index's own family, every request signed by Key with a Postern2
// proof (docs/api.md, Authentication).
type Client struct {
	Base string // Postern's backend, scheme and host, e.g. https://postern.example.com
	Key  *Key
	HTTP *http.Client
}

// Refused is Postern answering outside 2xx: the route, the status and the
// reason code its body named (docs/api.md), empty when it named none.
type Refused struct {
	Route  string
	Status int
	Reason string
	Said   string
	Key    string // the caller's public key, for a no_licence refusal
}

func (e *Refused) Error() string {
	if e.Reason == "no_licence" {
		return fmt.Sprintf("Postern refused %s 401 no_licence: the key %s holds no licence; issue it one on Postern's Key screen", e.Route, e.Key)
	}
	s := fmt.Sprintf("Postern refused %s %d", e.Route, e.Status)
	if e.Reason != "" {
		s += " " + e.Reason
	}
	if said := strings.Join(strings.Fields(e.Said), " "); said != "" {
		if len(said) > 200 {
			said = said[:200] + "…"
		}
		s += ": " + said
	}
	return s
}

// Address asks for the family's alert address, made on the first ask and
// the same ever after: POST /api/alerts/address.
func (c *Client) Address(ctx context.Context) (string, error) {
	b, err := c.signed(ctx, http.MethodPost, "/api/alerts/address", nil)
	if err != nil {
		return "", err
	}
	var reply struct {
		Address string `json:"address"`
	}
	if err := json.Unmarshal(b, &reply); err != nil || reply.Address == "" {
		return "", errors.New("Postern answered POST /api/alerts/address without an address")
	}
	return reply.Address, nil
}

// Alerts is the body of GET /api/alerts?since=<since>: the family's alerts
// above since, as ReadAlerts reads them.
func (c *Client) Alerts(ctx context.Context, since int) ([]byte, error) {
	return c.signed(ctx, http.MethodGet, "/api/alerts?since="+strconv.Itoa(since), nil)
}

// Claim claims one alert, DELETE /api/alerts/{id}: it is gone for every key
// of the family. An alert already gone (404) is claimed.
func (c *Client) Claim(ctx context.Context, id string) error {
	_, err := c.signed(ctx, http.MethodDelete, "/api/alerts/"+url.PathEscape(id), nil)
	var r *Refused
	if errors.As(err, &r) && r.Status == http.StatusNotFound {
		return nil
	}
	return err
}

// signed makes one signed request and returns the reply's body. A nonce
// refusal is tried once more from a fresh challenge, as Postern asks: nothing
// was acted on.
func (c *Client) signed(ctx context.Context, method, path string, body []byte) ([]byte, error) {
	b, err := c.signedOnce(ctx, method, path, body)
	var r *Refused
	if errors.As(err, &r) && r.Status == http.StatusUnauthorized && r.Reason == "nonce" {
		b, err = c.signedOnce(ctx, method, path, body)
	}
	return b, err
}

func (c *Client) signedOnce(ctx context.Context, method, path string, body []byte) ([]byte, error) {
	nonce, err := c.challenge(ctx)
	if err != nil {
		return nil, err
	}
	u, err := url.Parse(strings.TrimRight(c.Base, "/") + path)
	if err != nil {
		return nil, fmt.Errorf("Postern's address: %w", err)
	}
	return c.do(ctx, method, u, body, c.Key.authorization(method, u.RequestURI(), body, nonce))
}

// authorization is the Postern2 header for one request: the key's signature
// over the v2 message, the method, the request target exactly as it goes on
// the request line, the body's sha256 and the nonce, joined by a line feed.
func (k Key) authorization(method, target string, body []byte, nonce string) string {
	sum := sha256.Sum256(body)
	msg := "postern-v2\n" + method + "\n" + target + "\n" + hex.EncodeToString(sum[:]) + "\n" + nonce
	return "Postern2 " + k.PublicHex() + ":" + nonce + ":" + k.sign([]byte(msg))
}

// challenge asks for a nonce to sign: GET /api/challenge, unsigned.
func (c *Client) challenge(ctx context.Context) (string, error) {
	u, err := url.Parse(strings.TrimRight(c.Base, "/") + "/api/challenge")
	if err != nil {
		return "", fmt.Errorf("Postern's address: %w", err)
	}
	b, err := c.do(ctx, http.MethodGet, u, nil, "")
	if err != nil {
		return "", err
	}
	var reply struct {
		Nonce string `json:"nonce"`
	}
	if err := json.Unmarshal(b, &reply); err != nil || reply.Nonce == "" {
		return "", errors.New("Postern's challenge carried no nonce")
	}
	return reply.Nonce, nil
}

func (c *Client) do(ctx context.Context, method string, u *url.URL, body []byte, auth string) ([]byte, error) {
	route := method + " " + u.Path
	var rd io.Reader
	if body != nil {
		rd = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), rd)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", route, err)
	}
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("asking Postern %s: %w", route, err)
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, maxReply))
	if err != nil {
		return nil, fmt.Errorf("reading Postern's reply to %s: %w", route, err)
	}
	if resp.StatusCode/100 != 2 {
		r := &Refused{Route: route, Status: resp.StatusCode}
		var said struct {
			Error  string `json:"error"`
			Reason string `json:"reason"`
		}
		if json.Unmarshal(b, &said) == nil {
			r.Reason, r.Said = said.Reason, said.Error
		}
		if c.Key != nil && c.Key.priv != nil {
			r.Key = c.Key.PublicHex()
		}
		return nil, r
	}
	return b, nil
}
