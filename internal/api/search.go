package api

import (
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/Jonathan-A-White/sermon-index/internal/store"
)

// maxQuery is the longest q /v1/search reads, in characters.
const maxQuery = 200

type searchResp struct {
	Q    string      `json:"q"`
	Rows []store.Hit `json:"rows"`
}

// search is GET /v1/search?q=: full-text search over sermons and themes.
func (s *server) search(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		fail(w, r, http.StatusBadRequest, "bad_query", "q is required: words to search for, such as spurgeon grace")
		return
	}
	if utf8.RuneCountInString(q) > maxQuery {
		fail(w, r, http.StatusBadRequest, "bad_query", "q is longer than "+strconv.Itoa(maxQuery)+" characters")
		return
	}
	rows, err := store.Search(r.Context(), s.db, q)
	if err != nil {
		s.dbFailed(w, r, err)
		return
	}
	reply(w, r, searchResp{Q: q, Rows: rows})
}
