package api

import (
	"net/http"

	"github.com/Jonathan-A-White/sermon-index/internal/store"
)

type themesResp struct {
	Ref   string           `json:"ref"`
	Total int              `json:"total"`
	Rows  []store.ThemeRow `json:"rows"`
}

// themes is GET /v1/themes?ref=: the topics linked to passages overlapping
// ref, grouped by theme source, narrower first.
func (s *server) themes(w http.ResponseWriter, r *http.Request) {
	span, ok := parseRef(w, r)
	if !ok {
		return
	}
	rows, err := store.Themes(r.Context(), s.db, span)
	if err != nil {
		s.dbFailed(w, r, err)
		return
	}
	reply(w, r, themesResp{Ref: span.OSIS, Total: len(rows), Rows: rows})
}
