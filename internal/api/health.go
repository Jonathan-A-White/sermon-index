package api

import (
	"context"
	"net/http"
	"time"

	"github.com/Jonathan-A-White/sermon-index/internal/store"
)

type healthResp struct {
	Status  string `json:"status"`
	Version string `json:"version"`
	store.Totals
}

// health is GET /v1/health: {status: "ok", version, sources, sermons}, or
// 503 when the database does not answer within two seconds.
func (s *server) health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	t, err := store.Count(ctx, s.db)
	if err != nil {
		s.errLog.Printf("%s %s: database unreachable: %v", r.Method, r.URL.Path, err)
		fail(w, r, http.StatusServiceUnavailable, "unavailable", "the database is not answering; try again shortly")
		return
	}
	reply(w, r, healthResp{Status: "ok", Version: s.version, Totals: t})
}
