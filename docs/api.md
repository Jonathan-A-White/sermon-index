# The API (v1)

`sermons serve` answers a small, read-only HTTP API: the contract Lampas's sermon client codes against. Every route is under `/v1`; a change that would break a client is a new version (`/v2`), never an edit to these shapes. A new field may be added to a row; a client must ignore fields it does not know.

**The examples below are tests.** Each `Example:` line names a request, and the `json` block under it is the exact body the server answers, byte for byte, over the two test fixtures (`sources/_fixtures/spurgeon-20` and `sources/_fixtures/themes-demo`). `TestDocExamples` in `internal/api/server_test.go` reads this file, sends each request and compares; a 200 example is required for every route. So the doc is written by hand and the tests hold it to the server: change a shape and this file must change with it.

<!-- toc -->

- [Running it](#running-it)
- [What every answer has in common](#what-every-answer-has-in-common)
- [GET /v1/health](#get-v1health)
- [GET /v1/sources](#get-v1sources)
- [GET /v1/sermons](#get-v1sermons)
  - [The theme tier](#the-theme-tier)
- [GET /v1/themes](#get-v1themes)
- [GET /v1/search](#get-v1search)
- [Errors](#errors)
- [Not done yet](#not-done-yet)

<!-- /toc -->

## Running it

```sh
export DATABASE_URL=postgres://user:password@127.0.0.1:5433/sermons
bin/sermons migrate                 # serve refuses a database with a migration not applied
SERMONS_ADDR=127.0.0.1:8790 bin/sermons serve
curl -s 'http://127.0.0.1:8790/v1/sermons?ref=rom.5.8'
```

| Variable | Default | What it is |
| --- | --- | --- |
| `DATABASE_URL` | (required) | The Postgres URL. If the database is down when `serve` starts, it still starts, logs a warning, and answers 503 until the database answers. |
| `SERMONS_ADDR` | `127.0.0.1:8790` | The address to listen on (`10.88.0.2:8790` on the Laptop, behind the VPS). |
| `SERMONS_ORIGINS` | `https://lampas.allmymind.org,http://localhost:5173` | The origins a browser may read the API from (CORS), comma-separated. |

`serve` stops cleanly on SIGINT or SIGTERM, letting requests in flight finish (up to 10 seconds).

## What every answer has in common

- **JSON, indented by two spaces, ending in a newline**, `Content-Type: application/json`. A field with no value is `null`, never left out, so every row of a route has the same keys; a list with nothing in it is `[]`, never `null`.
- **References are OSIS, in Lampas's numbering.** A `ref` in a query may be an OSIS name or a Lampas code in any case (`rom.5.8`, `Rom.5.8`, `ROM.5`), a verse, a chapter, a book or a range written in full (`Rom.5.1-Rom.5.11`, `rom.3-rom.5`); answers give the canonical OSIS form (`Rom.5.8`). [sources.md, "References"](sources.md#references) is the grammar and [schema.md, "The book table"](schema.md#the-book-table) the numbering.
- **Caching.** A 200 carries `ETag: W/"<32 hex>"` (the first half of the SHA-256 of the uncompressed body) and `Cache-Control: public, max-age=300`. A request whose `If-None-Match` names the ETag (or is `*`) is answered `304 Not Modified` with no body. An error carries `Cache-Control: no-store` and no ETag.
- **gzip** when the request's `Accept-Encoding` accepts it (`gzip`, not `gzip;q=0`); the ETag is the same either way. Every answer has `Vary: Origin, Accept-Encoding`.
- **CORS.** A request whose `Origin` is in `SERMONS_ORIGINS` is answered with `Access-Control-Allow-Origin: <that origin>` and `Access-Control-Expose-Headers: ETag, Retry-After`; its preflight (`OPTIONS` with `Access-Control-Request-Method`) is answered 204 with `Access-Control-Allow-Methods: GET, HEAD` and `Access-Control-Allow-Headers: If-None-Match`. Any other origin gets no CORS header. Methods: `GET` and `HEAD` (the headers alone); any other is 405 (`OPTIONS` that is not a preflight is 204 with `Allow`).
- **A rate limit per client IP**: a token bucket of 60 requests a minute, up to 30 at once (constants in `internal/api/ratelimit.go`, not settings). Past it: `429` with `Retry-After: <seconds>`. The client IP is the peer's address; when the peer is loopback or a private address (the VPS's nginx over WireGuard), `X-Real-IP`, else the last address in `X-Forwarded-For`, is believed instead, so **the front door must set `X-Real-IP`** or every reader shares the proxy's bucket.
- **A log line a request** on stderr when it ends: method, path with its query, status, milliseconds and the bytes of body sent (after gzip), as `GET /v1/sermons?ref=rom.5.8 200 3ms 412B`. A database failure logs a second line with the error, which the client is not told.

## GET /v1/health

Is the service up, which build is it, and does the database answer. `version` is the build's `git describe --tags --always --dirty`, set by `make build` (`-ldflags -X main.version=...`; `dev` for a `go run` or a plain `go build`); `sources` counts sources of both kinds and `sermons` the sermons. When the database does not answer within two seconds: `503` and the error `unavailable`.

Example: `GET /v1/health`

```json
{
  "status": "ok",
  "version": "v0.2.0",
  "sources": 2,
  "sermons": 20
}
```

## GET /v1/sources

Every source, sermon sources first, then by id: `{rows: [...]}`, each row `{id, name, kind, licence, credit, homepage, on_by_default, rows}`. `kind` is `sermons` or `themes`; `licence` is the source's terms quoted verbatim; `rows` counts a sermon source's sermons or a theme source's topics. `on_by_default` is whether a reader who has never chosen sees the source. Lampas's credits and its Settings switches per source read this.

Example: `GET /v1/sources`

```json
{
  "rows": [
    {
      "id": "spurgeon-20",
      "name": "C. H. Spurgeon's sermons (twenty, a fixture)",
      "kind": "sermons",
      "licence": "CCEL.org website and special contents copyright 1993-2020 Harry Plantinga. Most of the editions at the Christian Classics Ethereal library are based on books that are public domain in the United States. However, they may have copyrighted introductions, cover art, and other special contents. A few books are under another publisher's copyright and are used by permission; these are noted on the book information page. These books may be used for personal, educational, or non-profit purposes. Contact us for permission to republish CCEL works or to use them commercially. Copyright laws in other countries vary; it may be that some books that are in the public domain in the United States but are still under copyright in other countries. Check your local copyright law before copying.",
      "credit": "Charles Haddon Spurgeon; the Christian Classics Ethereal Library (ccel.org); Spurgeon Gems (spurgeongems.org)",
      "homepage": "https://ccel.org/ccel/spurgeon",
      "on_by_default": false,
      "rows": 20
    },
    {
      "id": "themes-demo",
      "name": "Demo themes (a fixture)",
      "kind": "themes",
      "licence": "Written for this repository's tests; MIT, as the code is.",
      "credit": "The sermon index's tests",
      "homepage": "https://github.com/Jonathan-A-White/sermon-index",
      "on_by_default": false,
      "rows": 4
    }
  ]
}
```

## GET /v1/sermons

The sermons whose passages overlap a reference.

| Query | | What it is |
| --- | --- | --- |
| `ref` | required | The passage: a verse, chapter, book or range (above). A bad one is 400 `bad_ref` with the parser's message. |
| `source` | optional | Comma-separated source ids: only those sources. An id that is not a sermon source is ignored (it matches nothing), so a client's stale list does not fail. |
| `tier` | optional | Which tiers to answer: `passage` (the default: the answer below), `theme` (or `passage,theme`: the passage rows, then the theme tier, [below](#the-theme-tier)) or `theme-only`. Anything else is 400 `bad_tier`. |
| `theme` | optional | With a theme tier: only these topics, comma-separated `<theme source>:<topic>` (`themes-torrey:atonement-the`). Not of that shape is 400 `bad_theme`; a topic that does not link the ref, or is over the cap, matches nothing. Ignored for `tier=passage`. |
| `limit` | optional | Rows in the page, 1 to 200; default 50. Outside that, 400 `bad_limit`. |
| `offset` | optional | Rows to skip, 0 or more; default 0. Otherwise 400 `bad_offset`. With two tiers, `limit` and `offset` apply to each tier's page. |

The answer is `{ref, total, rows}`: `ref` the reference read, in OSIS; `total` the number of sermons that overlap it (all pages); `rows` this page. **One row a sermon**, with the one passage of it that answered:

| Field | What it is |
| --- | --- |
| `source`, `id` | The sermon's key: its source's id and its id within the source (Spurgeon's number). |
| `title` | As the source prints it. |
| `preacher`, `date` | Or `null`. `date` is ISO 8601 at the precision known: `1855`, `1855-03` or `1855-03-18`. |
| `url`, `alt_url` | Where the sermon is (https); `alt_url` a second form (a pdf) or `null`. |
| `kind` | `link`, `hosted` or `exposition`. |
| `ref`, `role` | The passage that answered: the preaching text (`role: "text"`) when it overlaps, else the narrowest reference the sermon cites (`role: "cited"`) that does. |
| `width` | That passage's length in verses (`Rom.5.12-Rom.6.4` is 14). |

**The order**: preaching texts before cited references; then narrower before wider, by the passage's span of verse ordinals, so a passage within one chapter comes before any that crosses a chapter boundary (a three-verse passage across Romans 5 and 6 comes after the whole of Romans 5; within one chapter it is the number of verses); then by source id; then by id, a number as a number (`9` before `10`).

Example: `GET /v1/sermons?ref=rom.5.8`

```json
{
  "ref": "Rom.5.8",
  "total": 1,
  "rows": [
    {
      "source": "spurgeon-20",
      "id": "104",
      "title": "Love's Commendation",
      "preacher": "C. H. Spurgeon",
      "date": null,
      "url": "https://ccel.org/ccel/spurgeon/sermons02/sermons02.l.html",
      "alt_url": "https://www.spurgeongems.org/sermon/chs104.pdf",
      "kind": "link",
      "ref": "Rom.5.8",
      "role": "text",
      "width": 1
    }
  ]
}
```

A page: Romans 5 has six preaching texts in the fixture; the second and third.

Example: `GET /v1/sermons?ref=Rom.5&limit=2&offset=1`

```json
{
  "ref": "Rom.5",
  "total": 6,
  "rows": [
    {
      "source": "spurgeon-20",
      "id": "104",
      "title": "Love's Commendation",
      "preacher": "C. H. Spurgeon",
      "date": null,
      "url": "https://ccel.org/ccel/spurgeon/sermons02/sermons02.l.html",
      "alt_url": "https://www.spurgeongems.org/sermon/chs104.pdf",
      "kind": "link",
      "ref": "Rom.5.8",
      "role": "text",
      "width": 1
    },
    {
      "source": "spurgeon-20",
      "id": "330",
      "title": "Reigning Grace",
      "preacher": "C. H. Spurgeon",
      "date": null,
      "url": "https://ccel.org/ccel/spurgeon/sermons06/sermons06.xliv.html",
      "alt_url": "https://www.spurgeongems.org/sermon/chs330.pdf",
      "kind": "link",
      "ref": "Rom.5.21",
      "role": "text",
      "width": 1
    }
  ]
}
```

Example (400): `GET /v1/sermons?ref=Rom.5.99`

```json
{
  "error": "bad_ref",
  "detail": "ref 'Rom.5.99': verse past the chapter"
}
```

### The theme tier

`tier=theme` adds a second tier below the passage rows: **sermons whose preaching text shares a topic with the passage** ("if a Bible passage you are reading covers a theme, it would note sermons on the same theme"). The answer is nested, `{ref, passage: {total, rows}, theme: {total, rows}}`; with `tier=theme-only` it is the same with `"passage": null` (so every answer has the same three keys). `passage` is exactly the flat answer's `total` and `rows`.

A sermon is in the theme tier when all of these hold:

- **its preaching text** (`role: "text"`, never a cited reference) overlaps a link of a topic, and **that topic also has a link overlapping `ref`**. The match is on the preaching text only: a sermon that merely cites a verse of the topic is not matched.
- **the topic covers at most 300 verses** (all its links together, a verse counted once; `THEME_MAX_VERSES` in `internal/store/queries.go`, `store.ThemeMaxVerses`). The probe that set it (Lampas `docs/sources/sermon-index.md`, section 4) found Nave's whole-subject topics, `JESUS, THE CHRIST` over 4,800 verses and `GOD` 4,000, put 2,735 sermons on Romans 5, and topics of 300 verses or fewer still 799; a topic's verse count is `verses` on `/v1/themes`.
- **it is not already a passage row.** A sermon with any passage (text or cited) overlapping `ref` is in the passage tier and is never repeated in the theme tier, so the two tiers' rows are different sermons.

**One row a sermon**: a sermon that meets several topics is listed once, with its **narrowest topic** (fewest verses; then theme source id, then topic id). Each row is the sermon's row (as above; `ref` is the preaching text that met the topic, `role` is always `text`, `width` its verses) and a `theme` naming how it came in:

| Field | What it is |
| --- | --- |
| `theme.source`, `theme.topic`, `theme.label` | The theme source's id, the topic's id and its name: "Love of God, The". |
| `theme.part`, `theme.part_label` | The numbered part of the topic the sermon's text met (`4`, `"Of God"`), or both `null` for a link to the whole topic. Of the topic's links the text meets, the narrowest, then the lowest part. |

**The order**: the topic's size ascending (a six-verse topic's sermons before a 140-verse topic's: the narrower the topic, the closer the sermon), then the sermon's source id, then its id as a number. A topic can still merge two senses (Thompson's `Oil` has a part "Figuratively used" beside the oil itself); the row names the topic and part so the reader can judge.

Example: `GET /v1/sermons?ref=rom.5.8&tier=theme`

```json
{
  "ref": "Rom.5.8",
  "passage": {
    "total": 1,
    "rows": [
      {
        "source": "spurgeon-20",
        "id": "104",
        "title": "Love's Commendation",
        "preacher": "C. H. Spurgeon",
        "date": null,
        "url": "https://ccel.org/ccel/spurgeon/sermons02/sermons02.l.html",
        "alt_url": "https://www.spurgeongems.org/sermon/chs104.pdf",
        "kind": "link",
        "ref": "Rom.5.8",
        "role": "text",
        "width": 1
      }
    ]
  },
  "theme": {
    "total": 1,
    "rows": [
      {
        "source": "spurgeon-20",
        "id": "1191",
        "title": "For Whom Did Christ Die?",
        "preacher": "C. H. Spurgeon",
        "date": null,
        "url": "https://ccel.org/ccel/spurgeon/sermons20/sermons20.viii_1.html",
        "alt_url": "https://www.spurgeongems.org/sermon/chs1191.pdf",
        "kind": "link",
        "ref": "Rom.5.6",
        "role": "text",
        "width": 1,
        "theme": {
          "source": "themes-demo",
          "topic": "atonement",
          "label": "Atonement",
          "part": null,
          "part_label": null
        }
      }
    ]
  }
}
```

Over the fixtures Romans 5:8 has one passage row (104, preached on the verse) and one theme row: 1191 (Romans 5:6) meets the demo topic Atonement, which links Romans 5:6-11. 104 meets Love and Atonement as well and is not repeated.

Example: `GET /v1/sermons?ref=rom.5.8&tier=theme-only&theme=themes-demo:atonement`

```json
{
  "ref": "Rom.5.8",
  "passage": null,
  "theme": {
    "total": 1,
    "rows": [
      {
        "source": "spurgeon-20",
        "id": "1191",
        "title": "For Whom Did Christ Die?",
        "preacher": "C. H. Spurgeon",
        "date": null,
        "url": "https://ccel.org/ccel/spurgeon/sermons20/sermons20.viii_1.html",
        "alt_url": "https://www.spurgeongems.org/sermon/chs1191.pdf",
        "kind": "link",
        "ref": "Rom.5.6",
        "role": "text",
        "width": 1,
        "theme": {
          "source": "themes-demo",
          "topic": "atonement",
          "label": "Atonement",
          "part": null,
          "part_label": null
        }
      }
    ]
  }
}
```

Example (400): `GET /v1/sermons?ref=Rom.5&tier=story`

```json
{
  "error": "bad_tier",
  "detail": "tier must be passage (the default), theme (the passage rows, then the theme tier), theme-only, or passage,theme"
}
```

## GET /v1/themes

The topics of every theme source linked to a passage overlapping `ref` (required, as above): `{ref, total, rows}`, all of them (no paging). One row per topic, or per numbered part of a topic, with the narrowest of its links that overlaps:

| Field | What it is |
| --- | --- |
| `source` | The theme source's id. |
| `topic`, `label` | The topic's id within the source and its name. |
| `part`, `part_label` | The numbered part the link names (`4`, `"Of God"`), or both `null` for a link to the whole topic. |
| `ref` | The link's passage (OSIS). |
| `see` | Related topics: a bare id is a topic of the same source, `"<source>:<id>"` another source's; `[]` when none. |
| `sermons` | How many sermons are on the **whole topic** (any part) by their preaching text, from every sermon source: so a client can say "Atonement · 158 sermons". A sermon with several texts counts once. |
| `verses` | How many verses the topic's links cover together. **A topic of more than 300 verses is not used by the theme tier** (above), so its `sermons` is a count of sermons the tier would not list: a client that links "N sermons" to the tier shows it only when `verses` is 300 or fewer. `0` for a topic loaded before migration `0003` and not yet reloaded (`sermons load`). |

**The order**: grouped by theme source (by id); within a source, narrower links first (as for sermons), then by label and part.

Example: `GET /v1/themes?ref=rom.5.8`

```json
{
  "ref": "Rom.5.8",
  "total": 2,
  "rows": [
    {
      "source": "themes-demo",
      "topic": "love",
      "label": "Love",
      "part": 1,
      "part_label": "Of God",
      "ref": "Rom.5.8",
      "see": [
        "grace"
      ],
      "sermons": 1,
      "verses": 3
    },
    {
      "source": "themes-demo",
      "topic": "atonement",
      "label": "Atonement",
      "part": null,
      "part_label": null,
      "ref": "Rom.5.6-Rom.5.11",
      "see": [],
      "sermons": 2,
      "verses": 7
    }
  ]
}
```

## GET /v1/search

Full-text search over the sermons' titles and preachers and the themes' labels: Postgres `websearch_to_tsquery('english', q)` against generated `tsvector` columns with GIN indexes (migration `0002_search`). The query is the syntax a search box expects: words are all required (`spurgeon grace`), `"a phrase"` in quotes, `or` between alternatives, `-word` to exclude. English stemming applies (`graces` finds "Grace"); stop words alone (`the`) find nothing. A sermon's title weighs more than its preacher.

`q` is required: empty (or only spaces) is 400 `bad_query`, as is one over 200 characters. The answer is `{q, rows}`, the best 50 by rank, then sermons before themes, then by source and id:

| Field | What it is |
| --- | --- |
| `kind` | `sermon` or `theme`. |
| `source`, `id` | The row's key (a theme's `id` is its topic id). |
| `title` | A sermon's title or a theme's label. |
| `ref` | A sermon's preaching text, or `null` (a sermon with none; every theme). |
| `url` | A sermon's url; `null` for a theme. |
| `rank` | Postgres `ts_rank`, to four places: higher is better; comparable only within one answer. |

Example: `GET /v1/search?q=spurgeon+grace`

```json
{
  "q": "spurgeon grace",
  "rows": [
    {
      "kind": "sermon",
      "source": "spurgeon-20",
      "id": "37",
      "title": "Law and Grace",
      "ref": "Rom.5.20",
      "url": "https://ccel.org/ccel/spurgeon/sermons01/sermons01.xxxvi.html",
      "rank": 0.6157
    },
    {
      "kind": "sermon",
      "source": "spurgeon-20",
      "id": "126",
      "title": "Justification by Grace",
      "ref": "Rom.3.24",
      "url": "https://ccel.org/ccel/spurgeon/sermons03/sermons03.xx.html",
      "rank": 0.6157
    },
    {
      "kind": "sermon",
      "source": "spurgeon-20",
      "id": "330",
      "title": "Reigning Grace",
      "ref": "Rom.5.21",
      "url": "https://ccel.org/ccel/spurgeon/sermons06/sermons06.xliv.html",
      "rank": 0.6157
    },
    {
      "kind": "sermon",
      "source": "spurgeon-20",
      "id": "385",
      "title": "Exposition of the Doctrines of Grace",
      "ref": null,
      "url": "https://ccel.org/ccel/spurgeon/sermons07/sermons07.viii_1.html",
      "rank": 0.6157
    }
  ]
}
```

Example (400): `GET /v1/search?q=`

```json
{
  "error": "bad_query",
  "detail": "q is required: words to search for, such as spurgeon grace"
}
```

## Errors

Every error is `{error, detail}`: `error` a stable code a client may branch on, `detail` a sentence for a person.

| Status | `error` | When |
| --- | --- | --- |
| 400 | `bad_ref` | `ref` is missing or does not parse; `detail` is the parser's message (`ref 'Rom.5.99': verse past the chapter`). |
| 400 | `bad_tier`, `bad_theme` | `tier` is not `passage`, `theme`, `passage,theme` or `theme-only`; `theme` is not `<theme source>:<topic>`. |
| 400 | `bad_limit`, `bad_offset` | `limit` is not 1 to 200; `offset` is not 0 or more. |
| 400 | `bad_query` | `q` is empty or over 200 characters. |
| 404 | `not_found` | No such route. |
| 405 | `method_not_allowed` | A method other than GET, HEAD or OPTIONS; `Allow: GET, HEAD, OPTIONS`. |
| 429 | `rate_limited` | Past the rate limit; `Retry-After` says when to try again. |
| 500 | `internal` | A query failed while the database answers; the error is logged. |
| 503 | `unavailable` | The database does not answer. |

## Not done yet

The API reads; it does not yet do these, each its own story:

- **Topics left out of the theme tier by hand.** A topic that merges two senses (Thompson's `Oil` and its paired topics such as `Faith-Unbelief`, Torrey's `Man`) is used whole; an `exclude` list on a theme source's `source.toml` is proposed, not built (the story mw-hxztgfo.5 sweep).
- **Writes** of any kind: the sources are files, loaded by `sermons load`; there is no route that changes the index.
- **Authentication**: the API is free and public; it has no keys and no accounts.
- **A hosted sermon's file**: `kind: "hosted"` exists in the schema, but no route serves a file.
- **Per-reader state** (which sources a reader has switched on, what they have read): Lampas keeps that on the device.
