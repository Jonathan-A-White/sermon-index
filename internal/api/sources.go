package api

import (
	"net/http"

	"github.com/Jonathan-A-White/sermon-index/internal/store"
)

type sourcesResp struct {
	Rows []store.Source `json:"rows"`
}

// sources is GET /v1/sources: every source, sermon sources first.
func (s *server) sources(w http.ResponseWriter, r *http.Request) {
	rows, err := store.Sources(r.Context(), s.db)
	if err != nil {
		s.dbFailed(w, r, err)
		return
	}
	reply(w, r, sourcesResp{Rows: rows})
}
