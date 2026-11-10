package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// KindForwardConfirm is the alert kind Postern's alerts mail gives Gmail's
// forwarding confirmation mail: its link is the one that confirms the
// forwarding address (Postern's docs/alerts-mail.md).
const KindForwardConfirm = "forward-confirm"

const (
	confirmHost   = "mail.google.com"
	confirmPrefix = "/mail/vf-"
)

// ForwardConfirmLink reads the body of GET /api/alerts and returns the link
// of the newest forward-confirm alert (the highest seq) and that alert's id.
// It only reads: the link is not visited and the alert is not claimed.
//
// The newest alert alone is judged. Its link must be one the receiver would
// have kept, https://mail.google.com/mail/vf-..., and a link on any other
// host or path is refused, with the alert named and the link not repeated;
// an older alert's good link is not offered in its place.
func ForwardConfirmLink(body []byte) (link, alertID string, err error) {
	var reply struct {
		Alerts []json.RawMessage `json:"alerts"`
	}
	if err := json.Unmarshal(body, &reply); err != nil {
		return "", "", fmt.Errorf("the alerts are not JSON of the form {\"alerts\": [...]}: %w", err)
	}
	var newest *struct {
		ID   string `json:"id"`
		Seq  int    `json:"seq"`
		Kind string `json:"kind"`
		Link string `json:"link"`
	}
	for _, raw := range reply.Alerts {
		var a struct {
			ID   string `json:"id"`
			Seq  int    `json:"seq"`
			Kind string `json:"kind"`
			Link string `json:"link"`
		}
		if json.Unmarshal(raw, &a) != nil || a.Kind != KindForwardConfirm {
			continue
		}
		if newest == nil || a.Seq > newest.Seq {
			newest = &a
		}
	}
	if newest == nil {
		return "", "", errors.New("no forward-confirm alert: Postern holds none for this family (Gmail's confirmation mail has not arrived, or was claimed, or is over 30 days old)")
	}
	if err := checkConfirmLink(newest.Link); err != nil {
		return "", "", fmt.Errorf("refused the link of forward-confirm alert %s: %w", newest.ID, err)
	}
	return newest.Link, newest.ID, nil
}

// checkConfirmLink accepts only https://mail.google.com/mail/vf-<token>,
// with an optional query: the host exactly, the default port, no user
// information, no whitespace or control character and no further path.
func checkConfirmLink(s string) error {
	if s == "" {
		return errors.New("it has no link")
	}
	for _, r := range s {
		if r <= ' ' || r == 0x7f {
			return errors.New("the link holds a space or control character")
		}
	}
	u, err := url.Parse(s)
	if err != nil {
		return errors.New("the link is not a URL")
	}
	switch {
	case u.Scheme != "https":
		return errors.New("the link is not https")
	case u.User != nil || u.Host != confirmHost || u.Port() != "":
		return fmt.Errorf("the link is not on https://%s", confirmHost)
	case !strings.HasPrefix(u.Path, confirmPrefix) || len(u.Path) == len(confirmPrefix) || strings.Contains(u.Path[len(confirmPrefix):], "/"):
		return fmt.Errorf("the link is not at https://%s%s<token>", confirmHost, confirmPrefix)
	}
	return nil
}
