# sermon-index

A sermon index: which sermons speak to a Bible passage or theme; a small service with Postgres on the Laptop, behind the VPS, called by Lampas.

The index points outward. A row is a sermon's title, its preaching text as a passage, and a link to where the sermon is (CCEL, Spurgeon Gems, a preacher's site); no sermon's words are kept. Rows come from **source directories** committed under `sources/`, one per sermon list or theme system, each with its licence quoted verbatim, so adding a preacher is a data change: a new directory, `sermons check`, `sermons load`. Passages are found by overlap first; themes (Thompson's 1934 topics, Torrey) come second.

This repository is one Go module, `github.com/Jonathan-A-White/sermon-index`, with one binary, `sermons`: it loads the source directories into Postgres and answers a read-only HTTP API, `/v1`, that Lampas calls ([docs/api.md](docs/api.md)).

<!-- toc -->

- [The record shape](#the-record-shape)
- [The sources](#the-sources)
- [Adding a source](#adding-a-source)
- [The commands](#the-commands)
- [Running it](#running-it)
- [On the Laptop](#on-the-laptop)
- [Running the tests](#running-the-tests)
- [Layout](#layout)
- [Licence](#licence)

<!-- /toc -->

## The record shape

A sermon source is `sources/<id>/source.toml` (name, kind, the licence verbatim, credit, homepage, terms url, date added, on by default) and `sources/<id>/rows.jsonl`, one sermon a line:

```json
{"id": "104", "title": "Love's Commendation", "ref": "Rom.5.8", "url": "https://ccel.org/ccel/spurgeon/sermons02/sermons02.l.html", "alt_url": "https://www.spurgeongems.org/sermon/chs104.pdf", "preacher": "C. H. Spurgeon"}
```

Optional fields: `ref` (the preaching text, OSIS), `alt_url`, `kind` (`link`, `hosted`, `exposition`), `preacher`, `date` (`1855`, `1855-03` or `1855-03-18`), `duration_s`, `cited` (other references), `notes`, `series` (a row over more than 3 chapters: kept and found by its title, matched by no passage), `expect_title` (what `--urls` must find in the page). A theme source has `topics.jsonl` (`{id, label, parts?, see?}`) and `links.jsonl` (`{topic, part?, ref, via?, from?}`) instead. [docs/sources.md](docs/sources.md) is the full format; `sources/_template/` is a filled-in example.

In the database a passage is an `int4range` of verse ordinals (`book*1000000 + chapter*1000 + verse`, Lampas's book order: Romans 5:8 is 45005008) with a GiST index, so "sermons on this verse, chapter or pericope" is `span && $1`. [docs/schema.md](docs/schema.md) draws the tables and says why each choice was made.

## The sources

| Source | What | Rows | Terms |
| --- | --- | --- | --- |
| `sources/spurgeon-ccel` | C. H. Spurgeon's sermons, 3,539 entries of CCEL's ThML edition (63 volumes) as one row a sermon number: title, preaching text, the references it cites, a date where its head gives one, the CCEL page and the Spurgeon Gems pdf. On by default. | 3,543 (3,467 with a preaching text); `rows.jsonl` is 935 KB | CCEL's, quoted verbatim in its `source.toml`: personal, educational or non-profit use. The words are public domain; the index keeps none of them. |
| `sources/themes-tcr1934` | Thompson Chain Topics, the subject index of Frank Charles Thompson's 1934 Bible, from CrossWire's SWORD module TCR: 1,728 topics with their numbered parts ("Love-Hatred (4) Of God"), See links between topics, and the passages each part names. On by default. | 1,728 topics, 2,229 parts, 27,076 links; `links.jsonl` is 1.4 MB | The module's: public domain; registered 1934 as A72501 and not renewed (the 1962 renewal catalogue was searched, see `licence_note`). The publisher's name for the printed book is a trademark and is never used for this source or any feature built on it. |
| `sources/themes-torrey` | R. A. Torrey's New Topical Textbook (1897), from CrossWire's SWORD module Torrey: 628 topics, their lines as parts, See links and passages. On by default. | 628 topics, 23,695 parts, 38,849 links; `links.jsonl` is 2.1 MB | The module's: public domain. |
| `sources/henry-sermons` | Matthew Henry's sermons, 52 rows made by hand from the Internet Archive's page scans (the 1830 Miscellaneous Works, 45 rows, and the 1847 Complete Works vol. 1, 7 rows): title, the preaching text read from the printed head (6 rows have none printed), the Archive's reader page, a date where printed; Philip Henry's sermons in the 1830 Appendix are not rows. On by default. | 52 rows, 46 with a preaching text | Public domain by date (Henry died 1714; printed 1830 and 1847); the Archive states no licence, its terms are quoted. |

`rows.jsonl` of `spurgeon-ccel` is written by a tool, not by hand: `go run ./tools/ccel-spurgeon --cache .ccel-cache --out sources/spurgeon-ccel --report tools/ccel-spurgeon/REPORT.md` fetches CCEL's 63 volumes into the (git-ignored) cache, one request at a time a second apart, and writes the same bytes again on a second run. [tools/ccel-spurgeon/REPORT.md](tools/ccel-spurgeon/REPORT.md) lists the numbers whose text is missing, doubled or left out.

`sources/henry-sermons` has no tool: its rows are read by hand from the Archive's OCR (kept in the git-ignored `.archive-cache/`, never committed), and `sources/henry-sermons/REPORT.md` lists what was read, the contents lines the OCR could not place, what was left out and, as candidates for a next source, three more preachers on the Archive (Whitefield, Edwards, Wesley).

The two theme sources are written the same way by `go run ./tools/sword-themes --module TCR --cache .sword-cache --out sources/themes-tcr1934 --report tools/sword-themes/REPORT.md` (and `--module Torrey ... --out sources/themes-torrey`), which fetches the SWORD module from CrossWire into the (git-ignored) cache and writes `topics.jsonl` and `links.jsonl`; `source.toml` is by hand. [tools/sword-themes/REPORT.md](tools/sword-themes/REPORT.md) lists the references and See terms the modules print in a form the tool cannot read. [docs/sources.md, "Theme sources"](docs/sources.md#theme-sources) says how a module's topics become parts, labels and links.

## Adding a source

A new preacher or collection is a data add: [docs/adding-a-source.md](docs/adding-a-source.md) is the recipe a Builder session follows (find the catalogue; read its terms and `robots.txt` first and quote them; link-out only; write `source.toml` and `rows.jsonl`, by a tool where the catalogue is machine-readable and by hand where it is a contents page; `sermons check` and `--urls`; load; the credit; the places outside this repository a new source touches), with Truth For Life as the worked example. [prompts/add-source.md](prompts/add-source.md) is the story a Mayor files for one, and `sermons propose sources/<id>` prints it filled from the source's `source.toml` and row count. [docs/candidates.md](docs/candidates.md) names the next three catalogues (Truth For Life, Ligonier, SermonAudio) with their terms quoted and a recommendation each.

## The commands

```sh
make build                                   # bin/sermons
export DATABASE_URL=postgres://user:password@127.0.0.1:5433/sermons

bin/sermons migrate                          # apply the embedded migrations (a second run is a no-op)
bin/sermons check sources/spurgeon-ccel      # one line per fault, file:line; exit 1 on any
bin/sermons check --urls 20 sources/spurgeon-ccel   # also HEAD a random 20 urls (network)
bin/sermons check --all sources              # every source directory (a warning, such as a row over 3 chapters not marked series, is printed and exits 0)
bin/sermons propose sources/henry-sermons    # the story for adding the source, filled from its source.toml and row count
bin/sermons load sources/spurgeon-ccel       # upsert the source in one transaction; prints the counts
bin/sermons load --all sources               # every source directory (not those beginning with '_')
```

`sermons load` prints, for example, `spurgeon-20: 20 rows, 19 passages, 0 removed`; loading the same files again changes nothing.

## Running it

`sermons serve` answers the API ([docs/api.md](docs/api.md): every route, its query, its JSON shape and an example the tests hold it to).

```sh
make build                                   # stamps /v1/health's version with git describe
export DATABASE_URL=postgres://user:password@127.0.0.1:5433/sermons
bin/sermons migrate && bin/sermons load --all sources
SERMONS_ADDR=127.0.0.1:8790 bin/sermons serve      # the default address; stops on Ctrl-C or SIGTERM

curl -s 'http://127.0.0.1:8790/v1/health'
curl -s 'http://127.0.0.1:8790/v1/sermons?ref=rom.5.8'           # the sermons on a verse, chapter or range
curl -s 'http://127.0.0.1:8790/v1/themes?ref=Rom.5'
curl -s 'http://127.0.0.1:8790/v1/search?q=spurgeon+grace'
```

`SERMONS_ORIGINS` is the origins a browser may call it from (default `https://lampas.allmymind.org,http://localhost:5173`). It refuses to start on a database with a migration not applied; with the database down it starts and `/v1/health` answers 503 until the database answers. Each request is one line on stderr (`GET /v1/sermons?ref=rom.5.8 200 3ms 412B`). Each client IP may make 60 requests a minute, 30 at once; behind a proxy on a private address, the proxy must send `X-Real-IP`.

## On the Laptop

The Laptop runs it as two systemd **user** units (no sudo): `sermon-index-pg` (Postgres 17 in Docker, `127.0.0.1:5433`) and `sermon-index` (`sermons serve` on `10.88.0.2:8790`, the WireGuard address the VPS's nginx will reach). [deploy/laptop/README.md](deploy/laptop/README.md) documents both unit files line by line.

```sh
scripts/install-laptop.sh      # build, env files, units, enable, migrate, load every source, smoke test
scripts/smoke.sh               # health 200 with at least 2 sources; Romans 5:8 starts with Love's Commendation
scripts/uninstall-laptop.sh    # the way back: stops and removes the units and the binary
```

The install is idempotent: a second run changes nothing and says "nothing changed". It generates the database password once (`openssl rand -hex 24`), never prints it, and keeps it in `~/.config/sermon-index/pg.env` and `env` (mode 0600). The uninstall leaves the database and those two files, and names them.

| What | Where |
| --- | --- |
| The service | `10.88.0.2:8790` (`SERMONS_ADDR` in `~/.config/sermon-index/env`) |
| Postgres | `127.0.0.1:5433`, loopback only, container `sermon-index-pg` |
| The binary | `~/.local/bin/sermons` |
| The data | `~/.local/state/sermon-index/pg` (owned by the container's `postgres` user, mode 0700: reading it takes `docker exec` or sudo) |
| The units | `~/.config/systemd/user/sermon-index-pg.service`, `sermon-index.service` |

**Back up** (the whole database; a few tens of megabytes; the sources are files in this repository, so the backup is a convenience, not the only copy):

```sh
docker exec sermon-index-pg pg_dump -U sermons sermons | gzip > ~/sermons-$(date +%F).sql.gz
```

**Restore, or move it to another host.** The sources are files, so the simpler move is nearly always: clone this repository on the new host, run `scripts/install-laptop.sh` there (set `SERMONS_ADDR` in `~/.config/sermon-index/env` to that host's address first, or edit it after and restart), and `sermons load --all sources/` fills the empty database. To carry the dump instead, install, then replace the fresh database with it:

```sh
systemctl --user stop sermon-index
docker exec sermon-index-pg psql -U sermons -d postgres -c 'DROP DATABASE sermons' -c 'CREATE DATABASE sermons'
gunzip -c sermons-2026-11-05.sql.gz | docker exec -i sermon-index-pg psql -U sermons -d sermons
systemctl --user start sermon-index
```

[docs/operations.md](docs/operations.md) has the logs, reloading after a new source, and what each failure looks like.

## Running the tests

```sh
make test      # go test ./...
make lint      # go vet ./... and gofmt
```

The tests need **no Docker and no installed Postgres**: they start a real Postgres 17 with [fergusstrange/embedded-postgres](https://github.com/fergusstrange/embedded-postgres) on a free port, in a temporary directory, and give each test its own database. **The first run downloads the Postgres binaries** (about 15 MB, from Maven Central) into `~/.cache/sermon-index/embedded-postgres/` (the user cache directory); later runs work offline. The tests never read `DATABASE_URL`, so they cannot touch a live database even when it is set.

## Layout

| Path | What it is |
| --- | --- |
| `cmd/sermons` | The binary: `migrate`, `check`, `propose`, `load`, `serve`. |
| `prompts` | `add-source.md`, the story for a new source, embedded in the binary for `sermons propose`. |
| `internal/api` | The HTTP API: routes, CORS, ETag and gzip, the rate limit, the request log. |
| `internal/passage` | OSIS references to verse-ordinal ranges; the 66-book table copied from Lampas. |
| `internal/migrate` | The embedded SQL migrations (`sql/NNNN_name.sql`) and the code that applies them. |
| `internal/store` | Opening the database; the API's queries; the tests' embedded Postgres. |
| `internal/sources` | Reading, checking and loading a source directory. |
| `internal/themes` | A theme source's records and their tables. |
| `tools/ccel-spurgeon` | The tool that writes `sources/spurgeon-ccel/rows.jsonl` from CCEL's ThML, and its `REPORT.md`. |
| `tools/sword-themes` | The tool that writes the two theme sources from CrossWire's SWORD modules, and its `REPORT.md`. |
| `sources/` | The source directories; `_template` and `_fixtures` (twenty Spurgeon sermons; a demo theme source) are never loaded by `--all`. |
| `deploy/laptop` | The two systemd user units for the Laptop and their line-by-line README. |
| `scripts` | `install-laptop.sh`, `uninstall-laptop.sh`, `smoke.sh`. |
| `tests` | The tests that drive those scripts with docker, systemctl, go and curl faked. |
| `docs/` | `api.md` (the HTTP API, its examples asserted by the tests), `schema.md` (the tables and why), `sources.md` (the source-directory format), `adding-a-source.md` (the recipe for a new source), `candidates.md` (the next three catalogues and their terms), `operations.md` (running it: logs, reloads, failures). |

## Licence

The code is MIT ([LICENSE](LICENSE)). Each source directory's rows carry their own terms, quoted in its `source.toml`.
