package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/Jonathan-A-White/sermon-index/internal/passage"
	"github.com/Jonathan-A-White/sermon-index/internal/store"
)

// /v1/sermons's page: 50 rows unless limit says otherwise, never more than
// 200.
const (
	defaultLimit = 50
	maxLimit     = 200
)

type sermonsResp struct {
	Ref   string            `json:"ref"`
	Total int               `json:"total"`
	Rows  []store.SermonRow `json:"rows"`
}

// tierResp is the answer when a tier is asked for: the passage tier (null
// for theme-only) and the theme tier. Each tier has its own total and page.
type tierResp struct {
	Ref     string                          `json:"ref"`
	Passage *tierPage[store.SermonRow]      `json:"passage"`
	Theme   *tierPage[store.ThemeSermonRow] `json:"theme"`
}

type tierPage[R any] struct {
	Total int `json:"total"`
	Rows  []R `json:"rows"`
}

const refRequired = "ref is required: an OSIS reference such as Rom.5.8, Rom.5 or Rom.5.1-Rom.5.11"

// parseRef reads the ref query parameter; on a fault it answers 400 and
// reports false.
func parseRef(w http.ResponseWriter, r *http.Request) (passage.Range, bool) {
	raw := strings.TrimSpace(r.URL.Query().Get("ref"))
	if raw == "" {
		fail(w, r, http.StatusBadRequest, "bad_ref", refRequired)
		return passage.Range{}, false
	}
	span, err := passage.Parse(raw)
	if err != nil {
		var perr *passage.Error
		if !errors.As(err, &perr) {
			perr = &passage.Error{Ref: raw, Reason: err.Error()}
		}
		fail(w, r, http.StatusBadRequest, "bad_ref", perr.Error())
		return passage.Range{}, false
	}
	return span, true
}

const badTier = "tier must be passage (the default), theme (the passage rows, then the theme tier), theme-only, or passage,theme"

// parseTier reads the tier parameter: whether to answer the passage tier,
// the theme tier, and in the nested shape. With no tier parameter it is the
// flat passage answer; a bad one answers 400 and reports false.
func parseTier(w http.ResponseWriter, r *http.Request) (passageTier, themeTier, nested, ok bool) {
	q := r.URL.Query()
	if !q.Has("tier") {
		return true, false, false, true
	}
	switch strings.ReplaceAll(strings.TrimSpace(q.Get("tier")), " ", "") {
	case "passage":
		return true, false, false, true
	case "theme", "passage,theme":
		return true, true, true, true
	case "theme-only":
		return false, true, true, true
	}
	fail(w, r, http.StatusBadRequest, "bad_tier", badTier)
	return false, false, false, false
}

// parseThemes reads theme=<source:topic>[,<source:topic>...].
func parseThemes(w http.ResponseWriter, r *http.Request) ([]string, bool) {
	var out []string
	for _, id := range strings.Split(r.URL.Query().Get("theme"), ",") {
		if id = strings.TrimSpace(id); id == "" {
			continue
		}
		src, topic, found := strings.Cut(id, ":")
		if !found || src == "" || topic == "" {
			fail(w, r, http.StatusBadRequest, "bad_theme", "theme must be <theme source>:<topic>, such as themes-torrey:atonement-the")
			return nil, false
		}
		out = append(out, id)
	}
	return out, true
}

const badDirect = "direct must be only (the theme tier's directly linked sermons alone), or left out"

// parseDirect reads direct=only: the theme tier's direct rows alone. Left
// out it is false; any other value answers 400 and reports false.
func parseDirect(w http.ResponseWriter, r *http.Request) (directOnly, ok bool) {
	q := r.URL.Query()
	if !q.Has("direct") {
		return false, true
	}
	if strings.TrimSpace(q.Get("direct")) == "only" {
		return true, true
	}
	fail(w, r, http.StatusBadRequest, "bad_direct", badDirect)
	return false, false
}

// sermons is GET /v1/sermons?ref=&source=&tier=&theme=&direct=&limit=&offset=.
func (s *server) sermons(w http.ResponseWriter, r *http.Request) {
	span, ok := parseRef(w, r)
	if !ok {
		return
	}
	wantPassage, wantTheme, nested, ok := parseTier(w, r)
	if !ok {
		return
	}
	themeIDs, ok := parseThemes(w, r)
	if !ok {
		return
	}
	directOnly, ok := parseDirect(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	sq := store.SermonQuery{Span: span, Limit: defaultLimit}
	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > maxLimit {
			fail(w, r, http.StatusBadRequest, "bad_limit", "limit must be a whole number from 1 to "+strconv.Itoa(maxLimit))
			return
		}
		sq.Limit = n
	}
	if v := q.Get("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			fail(w, r, http.StatusBadRequest, "bad_offset", "offset must be a whole number, 0 or more")
			return
		}
		sq.Offset = n
	}
	if v := q.Get("source"); v != "" {
		for _, id := range strings.Split(v, ",") {
			if id = strings.TrimSpace(id); id != "" {
				sq.Sources = append(sq.Sources, id)
			}
		}
	}
	resp := tierResp{Ref: span.OSIS}
	if wantPassage {
		total, rows, err := store.Sermons(r.Context(), s.db, sq)
		if err != nil {
			s.dbFailed(w, r, err)
			return
		}
		if !nested {
			reply(w, r, sermonsResp{Ref: span.OSIS, Total: total, Rows: rows})
			return
		}
		resp.Passage = &tierPage[store.SermonRow]{Total: total, Rows: rows}
	}
	if wantTheme {
		total, rows, err := store.ThemeSermons(r.Context(), s.db, store.ThemeQuery{
			Span: span, Sources: sq.Sources, Themes: themeIDs, DirectOnly: directOnly, PassageTier: wantPassage, Limit: sq.Limit, Offset: sq.Offset})
		if err != nil {
			s.dbFailed(w, r, err)
			return
		}
		resp.Theme = &tierPage[store.ThemeSermonRow]{Total: total, Rows: rows}
	}
	reply(w, r, resp)
}
