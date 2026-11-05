// Package prompts holds the text of the stories a Mayor files for this rig,
// so the `sermons` binary can fill them (`sermons propose`).
package prompts

import _ "embed"

// AddSource is add-source.md: the story for a new preacher or collection,
// a Go text/template whose fields internal/sources.Propose fills.
//
//go:embed add-source.md
var AddSource string
