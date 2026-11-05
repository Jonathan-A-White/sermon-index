package api

import "net/http"

// cors lets the origins given read the API from a browser: a request from
// one of them gets Access-Control-Allow-Origin naming it (and its ETag
// readable), and its preflight is answered here, for GET and HEAD only. A
// request from any other origin gets no CORS header, so a browser keeps
// the answer from that page. Every answer varies by Origin and by
// Accept-Encoding (gzip).
func cors(origins []string, next http.Handler) http.Handler {
	allowed := make(map[string]bool, len(origins))
	for _, o := range origins {
		allowed[o] = true
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Vary", "Origin, Accept-Encoding")
		origin := r.Header.Get("Origin")
		if origin == "" || !allowed[origin] {
			next.ServeHTTP(w, r)
			return
		}
		h.Set("Access-Control-Allow-Origin", origin)
		h.Set("Access-Control-Expose-Headers", "ETag, Retry-After")
		if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
			h.Set("Access-Control-Allow-Methods", "GET, HEAD")
			h.Set("Access-Control-Allow-Headers", "If-None-Match")
			h.Set("Access-Control-Max-Age", "600")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
