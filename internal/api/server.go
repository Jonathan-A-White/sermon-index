// Package api is the sermon index's read-only HTTP API, /v1, the contract
// Lampas codes against. docs/api.md documents every route, its query, its
// shape and an example response; the tests read those examples and assert
// the server answers them byte for byte.
//
// Every route is a GET (HEAD too); every answer is JSON, indented by two
// spaces, except the bare address / (home.go), a short HTML page. A 200 carries an ETag (a hash of its body) and is cached for five
// minutes; an error is {error, detail} and is not cached. Around the routes:
// CORS for an allow-list of origins (cors.go), a token bucket per client IP
// (ratelimit.go), gzip when the client accepts it (cache.go), and one log
// line a request.
package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DefaultOrigins is SERMONS_ORIGINS when it is not set: Lampas in
// production and Lampas's dev server.
const DefaultOrigins = "https://lampas.allmymind.org,http://localhost:5173"

// Config is a server's settings.
type Config struct {
	DB      *pgxpool.Pool
	Version string           // /v1/health's version: the build's git describe
	Origins []string         // the origins CORS allows (ParseOrigins)
	Log     io.Writer        // one line a request; nil writes none
	Now     func() time.Time // the rate limiter's clock; nil is time.Now
}

// ParseOrigins reads a comma-separated list of origins, dropping blanks
// and a trailing '/'.
func ParseOrigins(s string) []string {
	var out []string
	for _, o := range strings.Split(s, ",") {
		o = strings.TrimSuffix(strings.TrimSpace(o), "/")
		if o != "" {
			out = append(out, o)
		}
	}
	return out
}

type server struct {
	db      *pgxpool.Pool
	version string
	errLog  *log.Logger
}

// New is the API's handler: the routes inside the request log, CORS and
// the rate limiter.
func New(c Config) http.Handler {
	now := c.Now
	if now == nil {
		now = time.Now
	}
	logw := c.Log
	if logw == nil {
		logw = io.Discard
	}
	s := &server{db: c.DB, version: c.Version, errLog: log.New(logw, "", 0)}

	mux := http.NewServeMux()
	// A "GET" pattern answers HEAD too.
	mux.HandleFunc("GET /{$}", s.home)
	mux.HandleFunc("GET /v1/health", s.health)
	mux.HandleFunc("GET /v1/sources", s.sources)
	mux.HandleFunc("GET /v1/sermons", s.sermons)
	mux.HandleFunc("GET /v1/themes", s.themes)
	mux.HandleFunc("GET /v1/search", s.search)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fail(w, r, http.StatusNotFound, "not_found", "no such route: "+r.URL.Path+" (the routes are in docs/api.md)")
	})

	var h http.Handler = mux
	h = onlyReads(h)
	h = newLimiter(now).wrap(h)
	h = cors(c.Origins, h)
	h = logRequests(logw, h)
	return h
}

// onlyReads answers a method other than GET, HEAD and OPTIONS 405; an
// OPTIONS that cors did not answer as a preflight is 204 with Allow.
func onlyReads(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead:
			next.ServeHTTP(w, r)
		case http.MethodOptions:
			w.Header().Set("Allow", "GET, HEAD, OPTIONS")
			w.WriteHeader(http.StatusNoContent)
		default:
			w.Header().Set("Allow", "GET, HEAD, OPTIONS")
			fail(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "the API is read-only: GET and HEAD only, not "+r.Method)
		}
	})
}

// dbFailed answers a failed query: 503 when the database does not answer
// at all, else 500. The error itself goes to the log, not the client.
func (s *server) dbFailed(w http.ResponseWriter, r *http.Request, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if errors.Is(err, context.Canceled) && r.Context().Err() != nil {
		return // the client went away
	}
	if perr := s.db.Ping(ctx); perr != nil {
		s.errLog.Printf("%s %s: database unreachable: %v", r.Method, r.URL.Path, err)
		fail(w, r, http.StatusServiceUnavailable, "unavailable", "the database is not answering; try again shortly")
		return
	}
	s.errLog.Printf("%s %s: %v", r.Method, r.URL.Path, err)
	fail(w, r, http.StatusInternalServerError, "internal", "the query failed; it is logged")
}

// logWriter counts what a handler wrote, for the log line.
type logWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (l *logWriter) WriteHeader(code int) {
	if l.status == 0 {
		l.status = code
	}
	l.ResponseWriter.WriteHeader(code)
}

func (l *logWriter) Write(p []byte) (int, error) {
	if l.status == 0 {
		l.status = http.StatusOK
	}
	n, err := l.ResponseWriter.Write(p)
	l.bytes += n
	return n, err
}

// logRequests writes one line a request to w when it ends: method, path
// with its query, status, milliseconds and the body's bytes as sent.
func logRequests(w io.Writer, next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		start := time.Now()
		lw := &logWriter{ResponseWriter: rw}
		next.ServeHTTP(lw, r)
		if lw.status == 0 {
			lw.status = http.StatusOK
		}
		target := r.URL.Path
		if r.URL.RawQuery != "" {
			target += "?" + r.URL.RawQuery
		}
		fmt.Fprintf(w, "%s %s %d %dms %dB\n", r.Method, target, lw.status, time.Since(start).Milliseconds(), lw.bytes)
	})
}
