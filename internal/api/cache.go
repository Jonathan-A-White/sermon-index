package api

// Writing an answer: JSON indented by two spaces; a 200 with an ETag (a
// hash of the body), Cache-Control and If-None-Match's 304; gzip when the
// client accepts it; an error as {error, detail}, never cached.

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

// cacheControl is every 200's Cache-Control: the index changes only when
// a source is loaded, so five minutes stale is harmless.
const cacheControl = "public, max-age=300"

// apiError is every error's body.
type apiError struct {
	Error  string `json:"error"`
	Detail string `json:"detail"`
}

// fail answers an error: status, {error: code, detail}, not cached.
func fail(w http.ResponseWriter, r *http.Request, status int, code, detail string) {
	w.Header().Set("Cache-Control", "no-store")
	write(w, r, status, apiError{Error: code, Detail: detail})
}

// reply answers 200 with v: an ETag of the body, Cache-Control, and 304
// when If-None-Match names the ETag.
func reply(w http.ResponseWriter, r *http.Request, v any) {
	b, err := encode(v)
	if err != nil {
		fail(w, r, http.StatusInternalServerError, "internal", "encoding the answer failed")
		return
	}
	sum := sha256.Sum256(b)
	etag := `W/"` + hex.EncodeToString(sum[:16]) + `"`
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", cacheControl)
	if matches(r.Header.Get("If-None-Match"), etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	send(w, r, http.StatusOK, b)
}

func write(w http.ResponseWriter, r *http.Request, status int, v any) {
	b, err := encode(v)
	if err != nil {
		status, b = http.StatusInternalServerError, []byte("{\n  \"error\": \"internal\",\n  \"detail\": \"encoding the answer failed\"\n}\n")
	}
	send(w, r, status, b)
}

// encode is v as JSON, two-space indented, '<', '>' and '&' as themselves,
// ending in a newline: the bytes docs/api.md shows.
func encode(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	err := enc.Encode(v)
	return buf.Bytes(), err
}

// send writes the body, gzipped when the client accepts gzip; a HEAD gets
// the headers only.
func send(w http.ResponseWriter, r *http.Request, status int, b []byte) {
	h := w.Header()
	if h.Get("Content-Type") == "" {
		h.Set("Content-Type", "application/json")
	}
	if acceptsGzip(r.Header.Get("Accept-Encoding")) {
		var z bytes.Buffer
		zw, _ := gzip.NewWriterLevel(&z, gzip.BestSpeed)
		zw.Write(b)
		zw.Close()
		b = z.Bytes()
		h.Set("Content-Encoding", "gzip")
	}
	h.Set("Content-Length", strconv.Itoa(len(b)))
	w.WriteHeader(status)
	if r.Method != http.MethodHead {
		w.Write(b)
	}
}

// matches says whether an If-None-Match header names etag, by the weak
// comparison (W/ ignored), or is "*".
func matches(header, etag string) bool {
	if header == "" {
		return false
	}
	want := strings.TrimPrefix(etag, "W/")
	for _, t := range strings.Split(header, ",") {
		t = strings.TrimSpace(t)
		if t == "*" || strings.TrimPrefix(t, "W/") == want {
			return true
		}
	}
	return false
}

// acceptsGzip says whether an Accept-Encoding header accepts gzip: named,
// with no q or a q above 0.
func acceptsGzip(header string) bool {
	for _, part := range strings.Split(header, ",") {
		coding, params, _ := strings.Cut(strings.TrimSpace(part), ";")
		if !strings.EqualFold(strings.TrimSpace(coding), "gzip") {
			continue
		}
		params = strings.TrimSpace(params)
		if q, ok := strings.CutPrefix(params, "q="); ok {
			f, err := strconv.ParseFloat(q, 64)
			return err == nil && f > 0
		}
		return true
	}
	return false
}
