package api

import (
	"bytes"
	"context"
	"html/template"
	"net/http"
	"time"

	"github.com/Jonathan-A-White/sermon-index/internal/store"
)

// homeExample is the one example link on the bare address.
const homeExample = "/v1/sermons?ref=rom.5.8"

var homeRoutes = []struct{ Path, What string }{
	{"/v1/health", "is the service up, which build, and the counts"},
	{"/v1/sources", "the sources of sermons and of themes"},
	{"/v1/sermons", "sermons on a passage (?ref=), most relevant first"},
	{"/v1/themes", "topics that treat a passage (?ref=)"},
	{"/v1/search", "sermons by words (?q=)"},
}

var homeTmpl = template.Must(template.New("home").Parse(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Sermon index</title>
<style>body{font-family:system-ui,sans-serif;max-width:40em;margin:2em auto;padding:0 1em;line-height:1.5}code{background:#eee;padding:0 .25em}</style>
</head>
<body>
<h1>Sermon index</h1>
<p>A read-only index of sermons by the passages they preach and the themes they treat, for Lampas. It holds {{.Sources}} sources and {{.Sermons}} sermons (build {{.Version}}).</p>
<h2>Routes</h2>
<ul>
{{range .Routes}}<li><a href="{{.Path}}"><code>{{.Path}}</code></a>: {{.What}}</li>
{{end}}</ul>
<p>Try <a href="{{.Example}}"><code>{{.Example}}</code></a>.</p>
<p>Every answer is JSON. The routes, their queries and their shapes are in docs/api.md.</p>
</body>
</html>
`))

// home is GET /: a short page that says what the service is, with the
// counts of /v1/health, the routes and one example link; 503 when the
// database does not answer within two seconds.
func (s *server) home(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	t, err := store.Count(ctx, s.db)
	if err != nil {
		s.errLog.Printf("%s %s: database unreachable: %v", r.Method, r.URL.Path, err)
		fail(w, r, http.StatusServiceUnavailable, "unavailable", "the database is not answering; try again shortly")
		return
	}
	var buf bytes.Buffer
	err = homeTmpl.Execute(&buf, struct {
		store.Totals
		Version string
		Routes  []struct{ Path, What string }
		Example string
	}{t, s.version, homeRoutes, homeExample})
	if err != nil {
		fail(w, r, http.StatusInternalServerError, "internal", "rendering the page failed")
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", cacheControl)
	send(w, r, http.StatusOK, buf.Bytes())
}
