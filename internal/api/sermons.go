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

// sermons is GET /v1/sermons?ref=&source=&limit=&offset=.
func (s *server) sermons(w http.ResponseWriter, r *http.Request) {
	span, ok := parseRef(w, r)
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
	total, rows, err := store.Sermons(r.Context(), s.db, sq)
	if err != nil {
		s.dbFailed(w, r, err)
		return
	}
	reply(w, r, sermonsResp{Ref: span.OSIS, Total: total, Rows: rows})
}
