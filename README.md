# sermon-index

A sermon index: which sermons speak to a Bible passage or theme; a small service with Postgres on the Laptop, behind the VPS, called by Lampas.

The index points outward. A row is a sermon's title, its preaching text as a passage, and a link to where the sermon is (CCEL, Spurgeon Gems, a preacher's site); no sermon's words are kept. Rows come from **source directories** committed under `sources/`, one per sermon list or theme system, each with its licence quoted verbatim, so adding a preacher is a data change: a new directory, `sermons check`, `sermons load`. Passages are found by overlap first; themes (Thompson's 1934 topics, Torrey) come second.

This repository is one Go module, `github.com/Jonathan-A-White/sermon-index`, with one binary, `sermons`. The HTTP API (`sermons serve`) is the next story.

<!-- toc -->

- [The record shape](#the-record-shape)
- [The commands](#the-commands)
- [Running the tests](#running-the-tests)
- [Layout](#layout)
- [Licence](#licence)

<!-- /toc -->

## The record shape

A sermon source is `sources/<id>/source.toml` (name, kind, the licence verbatim, credit, homepage, terms url, date added, on by default) and `sources/<id>/rows.jsonl`, one sermon a line:

```json
{"id": "104", "title": "Love's Commendation", "ref": "Rom.5.8", "url": "https://ccel.org/ccel/spurgeon/sermons02/sermons02.l.html", "alt_url": "https://www.spurgeongems.org/sermon/chs104.pdf", "preacher": "C. H. Spurgeon"}
```

Optional fields: `ref` (the preaching text, OSIS), `alt_url`, `kind` (`link`, `hosted`, `exposition`), `preacher`, `date` (`1855`, `1855-03` or `1855-03-18`), `duration_s`, `cited` (other references), `notes`. A theme source has `topics.jsonl` (`{id, label, parts?, see?}`) and `links.jsonl` (`{topic, part?, ref, via?, from?}`) instead. [docs/sources.md](docs/sources.md) is the full format; `sources/_template/` is a filled-in example.

In the database a passage is an `int4range` of verse ordinals (`book*1000000 + chapter*1000 + verse`, Lampas's book order: Romans 5:8 is 45005008) with a GiST index, so "sermons on this verse, chapter or pericope" is `span && $1`. [docs/schema.md](docs/schema.md) draws the tables and says why each choice was made.

## The commands

```sh
make build                                   # bin/sermons
export DATABASE_URL=postgres://user:password@127.0.0.1:5433/sermons

bin/sermons migrate                          # apply the embedded migrations (a second run is a no-op)
bin/sermons check sources/spurgeon-ccel      # one line per fault, file:line; exit 1 on any
bin/sermons check --urls 20 sources/spurgeon-ccel   # also HEAD a random 20 urls (network)
bin/sermons check --all sources              # every source directory
bin/sermons load sources/spurgeon-ccel       # upsert the source in one transaction; prints the counts
bin/sermons load --all sources               # every source directory (not those beginning with '_')
```

`sermons load` prints, for example, `spurgeon-20: 20 rows, 19 passages, 0 removed`; loading the same files again changes nothing.

## Running the tests

```sh
make test      # go test ./...
make lint      # go vet ./... and gofmt
```

The tests need **no Docker and no installed Postgres**: they start a real Postgres 17 with [fergusstrange/embedded-postgres](https://github.com/fergusstrange/embedded-postgres) on a free port, in a temporary directory, and give each test its own database. **The first run downloads the Postgres binaries** (about 15 MB, from Maven Central) into `~/.cache/sermon-index/embedded-postgres/` (the user cache directory); later runs work offline. The tests never read `DATABASE_URL`, so they cannot touch a live database even when it is set.

## Layout

| Path | What it is |
| --- | --- |
| `cmd/sermons` | The binary: `migrate`, `check`, `load`. |
| `internal/passage` | OSIS references to verse-ordinal ranges; the 66-book table copied from Lampas. |
| `internal/migrate` | The embedded SQL migrations (`sql/NNNN_name.sql`) and the code that applies them. |
| `internal/store` | Opening the database; the tests' embedded Postgres. |
| `internal/sources` | Reading, checking and loading a source directory. |
| `internal/themes` | A theme source's records and their tables. |
| `sources/` | The source directories; `_template` and `_fixtures` are never loaded by `--all`. |
| `docs/` | `schema.md` (the tables and why), `sources.md` (the source-directory format). |

## Licence

The code is MIT ([LICENSE](LICENSE)). Each source directory's rows carry their own terms, quoted in its `source.toml`.
