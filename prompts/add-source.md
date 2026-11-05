{{- /*
The story a Mayor files to add a preacher or a collection to the sermon index.

Fill it by hand (replace each {{.Field}} below) or let `sermons propose
sources/<id>` print it filled from a source directory and its row count.
The fields a Mayor chooses are
  .Preacher       the preacher or the collection ("Alistair Begg")
  .CatalogueURL   the page or API that lists the sermons by passage
  .TermsURL       the page whose terms were read (quoted verbatim in source.toml)
  .Size           S (up to 300 rows), M (up to 2,000) or L (more)
`sermons propose` also fills .ID, .Name, .Credit, .RowsText, .WithTextText,
.Licence (quoted) and .LicenceNote. The text of this comment is not printed.
The rest of this file is the story; docs/adding-a-source.md is the recipe
its Builder follows, and the acceptance criteria below are those of the
epic's A3 (a source written by a tool) and A6 (by hand) stories.
*/ -}}
Title: sermon-index: add {{.Name}} as a source ({{.RowsText}}): the catalogue read, its terms quoted, rows written, `sermons check` and `--urls` passing

Path: rig sermon-index; formula tdd-feature
Preacher: {{.Preacher}}
Catalogue: {{.CatalogueURL}}
Terms: {{.TermsURL}}
Size: {{.Size}}

Description:
Add {{.Preacher}} to the sermon index as a source directory, sources/{{.ID}}/: a data add, no change of code. Read first: docs/adding-a-source.md (the recipe: follow its steps in order), docs/sources.md (the format), and sources/spurgeon-ccel (written by a tool, tools/ccel-spurgeon) and sources/henry-sermons (written by hand) as the two shapes.

The catalogue: {{.CatalogueURL}}. The terms: {{.TermsURL}}. Read the terms and the site's robots.txt BEFORE anything else and quote what they say; fetch nothing else until you have. The Governor's rule: an index of facts only (a title, a passage and an address, never a sentence of a sermon), link-out, every term quoted, and a key or money is his: if the catalogue needs a key or a payment, or forbids an index of facts, stop and say so on the bead; do not add the source.

What the source has now: {{.RowsText}} ({{.WithTextText}}), credited to {{.Credit}}.

The terms as quoted in source.toml (read at {{.TermsURL}}):
{{.LicenceQuoted}}
{{- if .LicenceNote}}

Our reading of them (licence_note): {{.LicenceNote}}
{{- end}}

Acceptance criteria:
1. sources/{{.ID}}/rows.jsonl: one row per sermon of {{.Preacher}} that {{.CatalogueURL}} lists: id the catalogue's own number or a stable slug, title as printed, ref the preaching text as an OSIS reference (absent, with notes 'no text printed', where the catalogue gives none), url the sermon's page at the catalogue (https, one address a reader can open), preacher, date where given, kind link; nothing of a sermon's words anywhere. A row whose ref spans more than 3 chapters carries `"series": true` and its narrowest text, or is split; a row the Builder is unsure of carries notes 'unsure: <why>'. Written by a tool, tools/{{.ID}}, where the catalogue is machine-readable (as tools/ccel-spurgeon: fetches into a git-ignored cache, a User-Agent naming the project, one request at a time and a second apart, a second run writes the same bytes, unit tests on a committed fixture with no network, a REPORT.md of what it could not read), or by hand where it is a contents page (as henry-sermons).
2. The counts, printed by the tool or counted, in the closing comment and in source.toml's comments: how many rows, how many with a preaching text, how many `series` rows; every row the catalogue lists that the file does not hold, named with why.
3. source.toml: id {{.ID}} (the directory's name), name, kind sermons, licence the terms quoted verbatim from {{.TermsURL}} (a comment says where and on what day it was read; a page drawn by script is quoted from the Wayback copy and says so), licence_note our reading in our own words (why an index of titles, passages and addresses may be kept, and anything to watch), credit, homepage, terms_url, added, on_by_default false unless the Governor has said otherwise; a `host_rules` entry for the catalogue's host where it answers 200 to an address it does not have, or asks for a crawl delay.
4. `sermons check sources/{{.ID}}` passes (a warning is read and answered, not ignored); `sermons check --urls 20 sources/{{.ID}}` reports every sampled url 200 with the title in the page where the host's rule says so; `sermons load sources/{{.ID}}` loads it and a query for one passage in it answers the sermon (in a test on the embedded Postgres that loads the real file, or by hand with the output in the comment); `sermons propose sources/{{.ID}}` prints this story from the file.
5. README's source table and CHANGELOG; the size of rows.jsonl is said.
6. Sweep: name every place outside this repo that a new source touches (Lampas's bundle list data/sermons/sources.json and its credits: docs/adding-a-source.md, "Outside this repo") and the credit line and the What's new line Lampas will carry, in the closing comment.
