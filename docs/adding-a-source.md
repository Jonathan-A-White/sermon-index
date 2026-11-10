# Adding a source

Adding a preacher or a collection is a **data add**: a new directory under `sources/`, checked by `sermons check`, put in the database by `sermons load`. No code changes. This is the recipe a Builder session follows, in order, for a story written from [prompts/add-source.md](../prompts/add-source.md). [sources.md](sources.md) is the format of the files; this is the way to get to them.

**The Governor's rule, which every step serves: an index of facts only.** A row keeps a title, a passage and an address, and links out; no sentence of a sermon is written anywhere in this repository. Every term the source's site states is quoted, never summarised. A source that needs a key or money is the Governor's to decide, not the Builder's. A source whose terms forbid an index of facts is not added.

<!-- toc -->

- [Before any crawl or download from a web host](#before-any-crawl-or-download-from-a-web-host)
- [1. Find the catalogue, and read its terms first](#1-find-the-catalogue-and-read-its-terms-first)
- [2. Decide: link-out or not at all](#2-decide-link-out-or-not-at-all)
- [3. Write source.toml and rows.jsonl](#3-write-sourcetoml-and-rowsjsonl)
- [4. Check, and check the addresses](#4-check-and-check-the-addresses)
- [5. Load and query](#5-load-and-query)
- [6. Credit, and what Lampas will carry](#6-credit-and-what-lampas-will-carry)
- [Outside this repo](#outside-this-repo)
- [A worked example: Truth For Life](#a-worked-example-truth-for-life)
- [A Logos archive: data only](#a-logos-archive-data-only)
  - [The book-archive form (a book of sermons, not an archive with an index section)](#the-book-archive-form-a-book-of-sermons-not-an-archive-with-an-index-section)
- [The shapes already here](#the-shapes-already-here)

<!-- /toc -->

## Before any crawl or download from a web host

This comes first, before step 1 and before any tool is written, for any new source that is fetched from a web host (a crawl of a site, a download of a file or an archive).

1. **Read the host's terms of use and its `robots.txt`** (the way step 1(b) says: those two pages first, a named User-Agent, one request at a time).
2. **Quote the clause that allows, or forbids, automated download** word for word on the story's bead before it is released, with the address and the day it was read. Where the terms are silent, say "not addressed"; silence is not a permission. A `robots.txt` that allows a path does not outweigh terms that forbid it.
3. **A host whose terms forbid automated download is not crawled** without the Governor's or the owner's word. Stop, comment on the bead with the quoted clause and hand back; do not write a tool, fetch the catalogue, or write rows from memory. A link a person found is another matter.
4. **Check the terms; prefer an official API.** Where the host offers an official API (a documented, keyed interface that its terms allow a program to read), a source is read through it and not by crawling the pages: the key is the host's way of saying yes, and the answer is already structured. The key stays in the vault and in an environment variable, never in a file, a log or a report. And **read what the API's terms let a client keep**: YouTube's Developer Policies (III.E.4) let a client store public API data for no more than 30 calendar days before it is deleted or refreshed, so a source from that API is a list to be regenerated at least that often, and says so in its `licence_note`.

**The worked example: Subsplash (mw-qtu3zvz.13).** A story to crawl Brookfield First's site, which Subsplash hosts, was filed and released with no one having read the host's terms. The Builder read them at `https://subsplash.com/legal/terms-and-conditions` and found that clauses **A.5** and **A.6** forbid "automated means, including spiders, robots, crawlers, data mining tools" to download data from the Services and the Site, although `robots.txt` allows the crawl (and the lists page shows only 12 items by GET; the rest is a POST). It stopped with no commits, and the Governor was given a decision card. Had the clauses been quoted on the bead before release, the story would not have been released. (The wording is as the Builder of mw-qtu3zvz.13 recorded it on that bead; the terms page is drawn by script, so quote it afresh from a copy that has the text if you rely on it.)

**What was done instead (mw-qtu3zvz.13, second version): YouTube's official Data API.** The Governor chose the church's YouTube channel, `@BrookfieldFirst`, read by `tools/brookfield-youtube` with a free Google API key (`YOUTUBE_API_KEY="$(mw secrets get youtube_api_key)" go run ./tools/brookfield-youtube`): `channels.list` (`forHandle`) for the uploads playlist, then `playlistItems.list`, 50 a page, 1 quota unit a call (12 units for 522 videos; the allowance is 10,000 a day). It writes `sources/brookfield-first` (a row for every video that is not a *Morning RePhil*: the title as YouTube prints it, the date the title prints or else the publish day, the watch link, and a preacher and a passage only where the title or the description names one) and `data/morning-rephil.jsonl` (the devotional list). Nothing is downloaded. Its `licence` is YouTube's Developer Policy on storing API data, quoted, and its `licence_note` carries the 30-day rule above.

**The sweep of the tools (2026-11-09, mw-qtu3zvz.13): which tool reads a third-party site, and the terms line that allows it.** Of the four tools that fetch from a host, none reads a site whose terms *forbid* automated reading, but only one has a line that allows it:

| Tool | Reads | What the host's terms say, as the source recorded them |
| --- | --- | --- |
| `tools/brookfield-youtube` | YouTube Data API v3, with a key | The API is the host's own offer to a program; its Developer Policies are quoted in `sources/brookfield-first/source.toml` (including the 30-day limit on keeping the data). |
| `tools/ccel-spurgeon` | `ccel.org/ccel/spurgeon/sermons01.xml` to `63` (63 files, a second apart) | CCEL's copyright page (`sources/spurgeon-ccel/source.toml`, read 2026-11-05): "These books may be used for personal, educational, or non-profit purposes." It says nothing of automated reading: **not addressed**. |
| `tools/sword-themes` | CrossWire's SWORD mirror, two module zips and two `.conf` files | The modules are published for download, with `DistributionLicense=Public Domain` (quoted in each `source.toml`); the site's own terms were not read. |
| `scripts/fetch-henry-ocr.sh` (not under `tools/`) | `archive.org/download/<item>/<item>_djvu.xml`, two files | The Archive's terms, quoted in `sources/henry-sermons/source.toml`: use must be noninfringing; automated reading is **not addressed**. |
| `tools/logos-import` | nothing: it reads a file the Mayor made | (no host) |

Silence is not a permission (rule 2 above): the CCEL and Archive lines are a reason to read their terms again for any larger crawl, not a licence for one.

## 1. Find the catalogue, and read its terms first

**(a) Find where the sermons are indexed by passage.** In the order of how easy a machine finds the passage:

| The catalogue is | The passage is | Rows are written |
| --- | --- | --- |
| A file or an API that tags each sermon with a reference (CCEL's ThML `scripCom type="Sermon" osisRef`) | read straight, as OSIS | by a tool under `tools/<source>` |
| A page per passage or per book (a "Browse by Scripture"), a sitemap | printed as a label ("Romans 5:1–11") that the tool parses to OSIS | by a tool, with a REPORT of the labels it could not parse |
| A contents page, a book's table of contents, a scan's OCR | printed in a heading the Builder reads | by hand, as `henry-sermons` |
| A video list read through an official API (a YouTube channel) | none unless a title or description names it (`tools/brookfield-youtube` reads "Genesis 18:16-33" and "(Pastor Phil Morgan)" out of them) | by a tool; most rows have no passage, and `REPORT.md` says how many |

**(b) Read the terms and the `robots.txt` BEFORE anything else.** Fetch only those two (and the terms page's own links to a copyright or permissions policy) until you have, with a User-Agent that names the project (`sermon-index-check (+https://github.com/Jonathan-A-White/sermon-index)`), one request at a time, a second or more apart. Then:

- **Quote what the pages say**, word for word, in your notes and later in `source.toml`'s `licence`. Never paraphrase a term, and never read a silence as a permission: where the terms say nothing about an index of titles and passages, say "not addressed" and take it to the Mayor.
- **A page drawn by script gives a script no text** (archive.org's terms page is one). Quote it from a copy that has the text (the Wayback Machine's `id_` form of the address) and say so in `source.toml`.
- **A `robots.txt` is a rule about robots, and also a signal.** Read it for the paths the tool may fetch (`Disallow` of query forms such as `?page=`; a `Crawl-delay`) and for any line about AI or search use (`Content-Signal: search=yes, ai-train=yes, ai-input=yes`). A `User-agent: *` of `Disallow: /` is a no: a link to a page a human found is another matter, an index built by script is not. A terms page that forbids "any robot, scraper, crawler, or other automated means" for any purpose (Ligonier's does) is a no for a tool; a source that can only be had by hand needs the owner's written permission, which is the Governor's to ask for.
- **Say which pages you read, and which you could not** (a terms page behind a login, a terms page that says "contact us"): the story stops at the terms it could not read.

## 2. Decide: link-out or not at all

**(c) The rule: nothing but title, passage and address is kept.** A row holds the sermon's title as the site prints it, the passage it is preached on, and the address a reader is sent to (`url`; `alt_url` for a second form such as an audio file), and optionally the preacher, the date, the length. It never holds a sentence of the sermon, a summary, a transcript, a description, or the site's markup.

Then decide, and write the decision down as the first thing in `licence_note`:

| What the terms say | What you do |
| --- | --- |
| Free to read and to link; an index of titles, passages and addresses is not forbidden | Go on. `licence_note` says the index keeps only titles, passages and links, and that the site's terms bind the pages we link to. |
| Public domain by date (the preacher died long ago, the edition was printed before 1929) | Go on, quote the site's own terms for the edition too, and say how the date was arrived at, as `spurgeon-ccel` and `henry-sermons` do. |
| Forbids copying, an index of facts, or automated access, or is silent and the site is a business | **Stop.** Comment on the bead with the quoted words and hand back. Do not add the source and do not write rows from memory. |
| Needs a key, an account, a licence fee, or money of any kind | **Stop and hand back to the Governor**: say how the key is obtained, whether it is free, who it is issued to (an account of a broadcaster is not a key for reading), and what the terms of the key say. A key or money is his; the Builder never creates an account or accepts terms for him. |
| Cloudflare-style challenge, a login wall | Not readable by a script: say so; do not work around it. |

**The one exception the Governor has made (mw-qtu3zvz.49, 2026-11-10, his answer "B: Expand the links").** Subsplash's terms forbid automated download (the A.5 and A.6 clauses quoted above), and `subsplash-terms-forbid-automated-download` stands for everything else. For the links he shares by hand, `tools/subsplash-links` may send **one HTTP HEAD per short link he shared**, to `brookfieldfirst.subspla.sh` only, with the redirect not followed and no body read, **1 second apart**, and the answer **cached** (`data/subsplash-links-cache.jsonl`) so a link is never asked twice. The address it learns goes in the row's `alt_url`, never in `url`. It is no leave to crawl, list or read a page of Subsplash, and no other host or purpose is covered. `tools/brookfield-youtube` keeps each row's `alt_url` when it regenerates `rows.jsonl` (matched on the row id), so the monthly refresh does not wipe them.

## 3. Write source.toml and rows.jsonl

**(d) `source.toml`.** Copy `sources/_template/` to `sources/<id>/` and fill every key ([sources.md](sources.md#sourcetoml)). `licence` is the terms **quoted verbatim**, with a comment above it saying where and on what day it was read. `licence_note` is your reading in your own words. `on_by_default` is `false` unless the Governor has said otherwise: `true` is for a source free of licence doubt that the app vouches for. A source whose titles are not English says so with `language = "pt"` (a two-letter code from `en`, `pt`, `la`, `de`, `nl`, `fr`, `el`, `he`; left out, the source is English), as `sources/calvin-pt-logos` does; the key is stored and checked, nothing in the site shows or filters by it yet. A host that answers 200 to an address it does not have, or asks for a crawl delay, gets a `[host_rules."<host>"]` table ([sources.md](sources.md#host-rules)).

**(e) `rows.jsonl`.** One sermon a line ([sources.md](sources.md#rowsjsonl-a-sermon-source)):

- `id`: the catalogue's own number, or a stable slug (the last path of its address); it never changes once loaded.
- `title`: as printed. `ref`: the preaching text as an OSIS reference; leave it out, with `notes` saying why ("no text printed"), where the catalogue gives none. A row the Builder is unsure of carries `notes` "unsure: <why>".
- `url`: https, a page a reader can open. A page that is not the sermon is worse than no row: `sermons check --urls` finds one.
- **A row over more than 3 chapters is a series.** A series over Acts 1:1–Jude 1:25 would match every chapter of the New Testament, and a reader on Romans 5:8 would be shown an overview of it. `sermons check` warns of any row whose `ref` spans more than 3 chapters; answer each warning by giving the row `"series": true` and its narrowest text (it is kept, found by its title, listed by its source, and matched by no passage), or by splitting it into a row per text.
- `expect_title`: where the page prints another title than the catalogue's, the words `--urls` must find in the page.

**By a tool or by hand.** Where the catalogue is machine-readable, write `tools/<source>` as `tools/ccel-spurgeon` and `tools/sword-themes` are: it fetches into a git-ignored cache (a file already there is not fetched again), a User-Agent naming the project, one request at a time and a second apart, keeps only ids, titles, references and addresses (no word of a sermon), writes the same bytes on a second run, has unit tests on a small committed fixture with no network, and writes a `REPORT.md` of every label it could not read, so a person sees what is left. Where the catalogue is a contents page, read it and write the rows by hand, as `sources/henry-sermons` was written, with a `REPORT.md` of what you could not place. Either way, count: rows, rows with a preaching text, `series` rows, and say them in `source.toml`'s comments and the closing comment.

## 4. Check, and check the addresses

**(f)**

```
sermons check sources/<id>                 # faults exit 1; a warning is printed and exits 0
sermons check --urls 20 sources/<id>       # network: HEAD a random 20 urls
```

Fix every fault. Read every warning: a wide `ref` is answered as in step 3. Then `--urls 20` (more for a small source): every sampled address must answer 200. Where the host answers 200 to anything, its `host_rules` table says `title_in_page = true` and `--urls` fetches the page and requires the title in it; a row's `expect_title` does the same for one row. Run `--urls` again, and once more with a larger N before you hand back; a 200 is not proof the page is the sermon unless the host's rule says so.

## 5. Load and query

```
DATABASE_URL=... sermons migrate
DATABASE_URL=... sermons load sources/<id>
```

It prints `<id>: N rows, M passages, 0 removed` (a `series` row is a row and no passage). Query one passage that is in the source (`GET /v1/sermons?ref=rom.5.8&source=<id>`) and put its answer in the closing comment. Add a test that loads the real file on the embedded Postgres and asserts the counts, as `TestLoadHenrySermons` does; a test must not depend on what `sources/` holds as a whole. On the Laptop, a new source is loaded into the live database by `sermons load --all sources/` ([operations.md](operations.md)).

Then `sermons propose sources/<id>` prints the story text for the source (`prompts/add-source.md` filled from its `source.toml` and its row count), which a Builder who has written a source can hand the Mayor as the next step: more rows, or the next source of the same kind.

## 6. Credit, and what Lampas will carry

In the closing comment, write the lines Lampas will carry, so the Mayor can place them without reading the diff:

- **The credit**: the preacher and the site, as `credit` in `source.toml` ("Alistair Begg, Truth For Life (truthforlife.org)"), and the licence line in a few words ("free to link; index of titles, passages and addresses; terms quoted in the source").
- **The What's new line**: one plain sentence the Governor would understand, no file names ("New: sermons by Alistair Begg, found by the passage you are reading").
- This repository's own: a row in README's source table and a line in CHANGELOG.

## Outside this repo

A new source touches these places that this repository does not hold. Name each in the closing comment, with what has to be done; the Mayor files them. (Lampas's sermon stories, plans/sermons-2026-11-05/lampas.json B1 to B4, were not built when this was written, so the paths are those the plan names.)

| Where | What changes for a new source |
| --- | --- |
| The live database on the Laptop | `sermons load --all sources/` (or `sermons load sources/<id>`) after the merge. The API then answers the source with no restart and no app release (`/v1/sources` lists it). |
| Lampas `data/sermons/sources.json` | The bundle list: the repository, the commit (full sha) and the source ids to bundle. A new source id is added and the commit raised to one that holds the directory. |
| Lampas `public/sermons/index.json` and `public/sermons/<id>/<book>.json` | Written by `npm run sermons:build` from that list (committed as `public/data` is), with the source's licence, credit and counts; a `series` row has no passage and is left out of the book files; `expect_title` and `host_rules` are for this repo's checker and are ignored. |
| Lampas credits, four entries | `ATTRIBUTION.md` (the source, its licence with a link, `Changes:`), `src/preface.ts`, `src/tutor/credits.ts` (`CREDITS`), and a rule in `tests/support/credits.ts` for a bundled directory that has none; `tests/unit/credits.test.ts` fails until each exists. |
| Lampas Settings | `Settings > Study resources > Sermons` has one On | Off row per source (`src/settings/definitions/sermons.ts`), its start taken from `on_by_default`; `src/tutor/knows.ts` `BEARS_ON` classifies the new row, or `tests/unit/tutor-knows.test.ts` fails. |
| Lampas CHANGELOG | The What's new line (step 6). |
| This repo's README source table and CHANGELOG | The row and the line, in the same commit. |

A source that is not free of licence doubt is loaded into the service (the API is read-only and free) but is not put in Lampas's bundle, which ships offline and free; that choice is the Governor's.

## A worked example: Truth For Life

Truth For Life (Alistair Begg, `www.truthforlife.org`) is the first modern preacher the recipe is tried on. All of this was read on 2026-11-05; of Truth For Life only its `robots.txt` and its Policies page were fetched, and no row has been written.

**(a) The catalogue.** A page per passage: `https://www.truthforlife.org/bible/Romans 5:8/` (a space and a colon in the address; send it encoded as `Romans%205:8`) listed, in the 2026-11-05 probe (Lampas's `docs/sources/sermon-index.md`, section 3), 12 sermons, each with its text as free text (`Romans 5:8`, `Romans 5:1–11`, `Acts 1:1 – Jude 1:25`), a date, a duration, whether it has a transcript, and an address of the form `/resources/sermon/<slug>/` (one is `/resources/sermon/how-deep-fathers-love/`). "Browse by Scripture" is in its menu, and a sitemap is named in `robots.txt`. So the catalogue is the second kind in the table: a page per passage with printed labels, read by a tool (`tools/truthforlife`) that walks the passage pages, parses each label to OSIS with `passage.Parse` after normalising the en dash and the book name, lists every label it cannot parse in a REPORT, and dedups sermons that appear under several passages by their slug.

**(b) `robots.txt`**, `https://www.truthforlife.org/robots.txt`, 796 bytes, quoted:

```
User-agent: *
Content-Signal: search=yes, ai-train=yes, ai-input=yes
Disallow: *?month=
Disallow: *?year=
Disallow: *?topic=
Disallow: *?scripture=
Disallow: *?type=
Disallow: *?sort=
Disallow: *?chapter=
Disallow: *?per_page=
...
Disallow: *?page=
...
Sitemap: https://www.truthforlife.org/sitemap.xml
```

(and `Crawl-delay: 1` for one named bot, `Disallow: /` for YandexBot; neither is `*`). So a passage page, which has no query, is allowed to be read; the tool must never add `?page=`, `?chapter=` or the other disallowed forms, and should keep to the one-request-a-second it keeps anyway.

**The terms** are not at `/terms`; the page is **Policies**, `https://www.truthforlife.org/about/policies/`, read in full. What it says that bears on an index of facts, quoted:

- Audio and video: "You may duplicate and distribute copies of Alistair’s sermons without prior permission as long as you: Do not alter or edit the content in any way; Do not charge for the material (even to recover your fee) ...". "If you would like to include a link to an audio or video clip on your site, you may embed a link to the file on our site. You must include a link back to our site at www.truthforlife.org."
- "Your use of the Audio and Video Content constitutes to an agreement by you and Truth For Life to be bound by these policies." and "We reserve the right to revoke or modify these permissions at any time."
- Web content: "You may post written content from www.truthforlife.org on your website as long as you: Do not edit the content in any way; Include a link to the original article and our homepage; Provide the appropriate credit for the content: “Copyright Truth For Life. Used with Permission. www.truthforlife.org.”"
- "Truth For Life and Alistair Begg reserve all copyright protections under applicable law."

**What a row would hold**, from what the page showed and nothing more: `id` the slug (`how-deep-fathers-love`), `title` as the page prints it, `ref` the label parsed to OSIS (`Rom.5.8`; `Rom.5.1-Rom.5.11`), `url` `https://www.truthforlife.org/resources/sermon/<slug>/`, `preacher` `Alistair Begg`, `date` and `duration_s` where the page gives them, `kind` `link`. The row for the sermon labelled `Acts 1:1 – Jude 1:25` is `"series": true` with `ref` `Acts.1.1` (its narrowest text, the first), or it would match every chapter of the New Testament; `sermons check` warns until it is. Its host would get a `[host_rules."www.truthforlife.org"]` of `delay_ms = 1000` (and `title_in_page = true` only if a probe finds the site answers 200 to an address it does not have; it was not tested, because that would fetch a page that is not a terms or a robots page). A transcript, a description, the audio file: none is kept, and `alt_url` is not used.

**Where it stops: the terms it could not read.** The Policies page speaks of what may be done with the sermons (play, copy unaltered and free, link to a file with a link back, quote under fair use) and of the written content of the site. **It does not say whether a small index of titles, passages and addresses may be kept, nor whether a script may read the passage pages**; the `robots.txt` allows the latter, the terms say nothing. The syndication terms (`/syndicate/instructions/`, "the Content Use Terms and Conditions" an account must agree to) were seen as search excerpts only and not read. By the table in step 2, silence on an index of facts with a business behind it is a stop: the Builder who takes this story writes the quoted words on the bead and asks the Governor, with the recommendation that a link-only index (the title the site prints, the passage, the address, with the credit "Truth For Life. Used with Permission. www.truthforlife.org" where the app shows it) is what the Policies page's own words invite, and that the Governor, or someone for him, could ask the ministry to say so in writing.

## A Logos archive: data only

The Governor's Logos sermon archives (a resource in his library that is a collection of one preacher's sermons) are the fourth shape, and the one where a later archive needs no code. The Mayor makes an **index file** of the archive in the vault (`plans/sermons-2026-11-05/<slug>-sermons.jsonl`, with a `<slug>-report.md`): a line a sermon with `id` (Logos's article id), `title`, `preacher`, `date`, `year`, `ref` and `refs` (the label as Logos prints it), `bibleRefs` (Logos's data-reference), `series`, `part` and `url` (the `ref.ly` link). The files are only on the Laptop, so a story that imports one runs there. Then:

```
go run ./tools/logos-import --in <vault>/plans/sermons-2026-11-05/<slug>-sermons.jsonl --id <id> --out sources/<id>
cp sources/greg-laurie-logos/source.toml sources/<id>/source.toml   # then edit id, name, credit, the comment, the counts
go run ./cmd/sermons check sources/<id>
```

The tool writes `rows.jsonl` and `REPORT.md` (the same bytes for the same input) and prints the counts. Per line:

- `ref` is the first of `bibleRefs` as OSIS; the rest are `cited`. **Logos's book numbers are its own**: `bible.<book>.<chapter>[.<verse>]`, a whole chapter written `book.chapter` (`bible.1.1` is Genesis **1**, not Genesis 1:1), a range `bible.19.51.1-19.51.19` or `bible.1.1-1.3`. Books 1 to 39 are the Old Testament in the usual order and 61 (Matthew) to 87 (Revelation) the New; 40 to 60 are not among the sixty-six and are refused. The table is `logosBook` in `tools/logos-import/osis.go`, tested for all 66; the import also reads the label and says in the report where Logos and the label disagree (one row of 1,458 so far).
- A line with no Logos reference takes `ref` from the label (`Jonah`, `Romans 5:1-11`), noted in `notes`. A reference the book table refuses (a verse past the chapter) is left out and listed; the row stays.
- A text over more than 3 chapters (a whole book) is a `series` row, as the checker asks.
- `kind` is `link`, `url` the `ref.ly` link, `notes` the series and part (`Series: X, part 2.`), `date` the sermon's own date, else the year the archive files it under.
- **Dropped, and listed in `REPORT.md` with the reason:** a line with neither a ref nor a title, and one with no id, no title, no address or an id already used.

`source.toml` is by hand and follows the three already here: `on_by_default = false`; `licence_note` begins with the Governor's line ("From the Governor's own Logos library at his word ..."); no Logos text is quoted anywhere; the `licence` is his own words, since there is no terms page to quote. Then load, add the source to a test beside `TestLoadLogosArchives` (`internal/sources/load_test.go`) with its counts, and write the README row and the CHANGELOG line.

### The book-archive form (a book of sermons, not an archive with an index section)

Most of his sermon *books* have no index section, and the Mayor's parser falls back to the book's contents list (`bin/logos-sermons-parse.mjs`, `fallbackRecord`). The index file's lines then carry no date and no series, and add `cited` (the passages the entry cites, as `bible.*`), `offset`, `textFromArticle`, `multipart` and `length`; the id is an offset, `2~14522`, and the `url` is `https://ref.ly/logosres/LLS:<resource>?off=<offset>` (**unverified**: no one has tapped one yet). The same tool reads it; what differs:

- **One source from several books**: give `--in` more than once (the rows follow the files in order). Row ids are `<resource>.<article id with the ~ made a ->`, `MYSRMNOTES01.2-14522`, because the format refuses a `~` and two books can share an offset. `spurgeon-sermon-notes-logos` is two books, `wesley-logos` four, `moody-logos` four.
- **`--dedupe-titles`** drops a line whose title is the title of a row of an *earlier* `--in` file (lower case; a leading Roman numeral or "Sermon N" taken off; apostrophes and marks ignored): Wesley's *Works* repeat his *Sermons on Several Occasions*, so that book goes first. Titles that repeat inside one file are all kept. `REPORT.md` has the counts before and after for each file and each row dropped (117 of 140 Works entries for `wesley-logos`).
- **`cited`** is the file's `cited` list after the rest of `bibleRefs`, minus the `ref`; a trailing `+` or `++` ("and the verses after") is read as the verse. A line with no ref of its own keeps its cited passages as `cited` (they load as cited passages) and **none is taken to be the preaching text**; it has no passage match, like any row with no ref.
- **A title "Chapter I: Gen. 19:15"** is titled by its passage with the book written out ("Genesis 19:15"; `Phil.` is Philippians or Philemon by the row's own reference), or by the heading after the colon ("Chapter I: “Love that Passeth Knowledge”"); the "Chapter I." goes into the notes. A title with no "Chapter N: " in front is left as it is.
- **`multipart`**: the contents list splits the entry in parts, and the entry itself may be only a heading (Moody's "The Overcoming Life", 20 characters); the row is kept, the link opens the book at that offset, and the ids are listed in `REPORT.md`.

Batch 3 (Augustine, Chrysostom, Piper, Bunyan, Charles Wesley, Jimmy Evans) needed three more things, all in the same tool, none on by default so an earlier source writes the same bytes:

- **Non-sermon lines are dropped**, always, and counted in `REPORT.md` ("Non-sermon lines dropped"): a title that is, after a leading Roman numeral or "Sermon N" is off, "Chronological Table", "New Testament", "Argument" or "The Argument" (or one that begins so), "Index of Texts" at the end (or "Index of ..." at the start), "Introductory Discourse", "Prolegomena", or one of the two Schaff essays and "St. Chrysostom as a Homilist" of the Chrysostom volumes. The list is `NonSermon` in `tools/logos-import/clean.go`; it goes by the title alone, so a new archive's introduction under another name needs a line there. They are not counted in a file's "rows before".
- **A reference is cleaned before it is read**: a comma after the book ("Matthew, 5:20") and an OCR `l` for `1` in a number ("1 Kings, 18:2l" is 18:21) (`CleanLabel`).
- **`--refs-from-title`** reads the passage from the title where the line has no Logos reference and no label that reads: Augustine's titles ("Sermon 95: On the Words of the Gospel, Mark 8:1–9, Where ...", "Luke’s Gospel, 13:6–17", "John, Chapter 9") carry the passage. The first passage is the `ref`, the others `cited`; the row's notes say so and `REPORT.md` counts them. Use it only where titles name the text: Skip Heitzig's "Exodus 3-4" would be read too, and the tool would not stop it from reading a title that names a passage it is not on.
- **`--preacher "<name>"`** sets the preacher of every row: the Chrysostom volumes name their editor, Philip Schaff, on every line.
- **A source with no Scripture ref at all** (Jimmy Evans: dates and series, no passage) is written like any other; `REPORT.md` then says that none of its rows has a ref and lists the rows the index dates by the year alone.

Batch 4a (Shedd, Burroughs, Keach, Maclaren, Kuyper, Broadus) needed two more, again inert on an earlier source:

- **Front matter and tables of contents are dropped** with the other non-sermon lines (`NonSermon`): "Preface", "Author’s Preface", "Translator’s Preface", "To the Reader", "The Epistle to the Reader", "Foreword", "Contents", a title that begins "The Contents of ...", "An Exact Alphabetical Table ..." or "The Table of the Chief Heads ...". A row that is only "Sermon N" (no title of its own) is dropped too when the same file has a row "Sermon N: ..." (the number is compared, so "Sermon III" is covered by "Sermon 3: Faith"); alone, a bare "Sermon I" is a sermon and stays (Burroughs's *Rare Jewel* and Keach's rows are titled so). Each drop is named in `REPORT.md`.
- **A date in brackets in the title** is the row's date when the index gives none: Burroughs's "The First Sermon (July the 6th, 1645)" is 1645-07-06 (`TitleDate`: "Month the 6th/23d, YYYY" or "Month 4, YYYY", a day the month has); the title is left as it is, the notes say where the date came from and `REPORT.md` counts them. A date outside brackets is not read.

The "Chapter N: " the contents list puts in front of a title (Shedd's first book, Maclaren, Broadus) is taken off and kept in the notes, as for the sermon books above.

Batch 4b (The World's Great Sermons, Princeton Sermons, McPherson's Divine Healing Sermons, A. H. Strong) reads more out of a title, with flags that are off unless given, so an earlier source regenerates byte for byte (checked by regenerating all nineteen):

- **`--title-form preacher-years`** (the World's Great Sermons, ten volumes as one source, `--in` ten times): "Basil (329–379): The Creation of the World" is the preacher Basil, the years in the notes ("Preacher’s years: 329–379.", or "Preacher born 1851." for "Wagner (Born in 1851): ..."), and the title "The Creation of the World"; Volume 2 prints "The Heavenly Footman—Bunyan (1628–1688)" and is read the same. The index names the editor, Grenville Kleiser, as the preacher of every line, so he is kept in the notes as "Book credit: Grenville Kleiser." and is no one's preacher. The preacher is as printed (most are surnames: Hooker, Edwards, Wesley). The sermon's own date is not the preacher's years and is never set from them.
- **`--title-form by-preacher-ref`** (Princeton): "Christ’s Desire for His People—By William Henry Green (John 17:24)" is the title, the preacher and the `ref` John 17:24 ("2 Cor." is written out; a stray comma, "William, Paxton", is mended and "Jr." is kept). The index's name for the book, Princeton Theological Seminary, is the book credit in the notes. The ref is the title's even where Logos's first cited passage differs ("Religion in College": John 2:13 in the title, 1 John 2:13 in Logos); `REPORT.md` lists each such row and Logos's passage stays among the cited ones.
- **`--title-form numbered`** (Strong): "XXIII: The Claims of ..." loses its number, kept in the notes as "Number XXIII of the collection."; **`--note "<sentence>"`** adds a sentence to every row (Strong's says the kind of piece, address, essay or sermon, is not told by the index). Essays and addresses are kept.
- A title that does not have the named form is kept as it is and listed in `REPORT.md` ("Titles that did not have the form"; none in these four).
- **`--not-imported "<book>: <why>"`** (more than once) names a book of the order that gives no rows, in the report's "Books not imported". Scofield's *Things New and Old* (lesson commentary: its lines are the same sub-headings repeated under every lesson) and McPherson's *This Is That* (an autobiography; her sermons are only inside its Part III) are named so in `mcpherson-logos/REPORT.md`.
- `NonSermon` also drops a "General Index" and a heading "Addresses to Successive Graduating Classes ..." (Strong's last line, over addresses the index does not list one by one).

Batch 5a (Sedgwick, Moberly, Eyton, Chapin, Watson) adds two more flags, also off unless given (checked by regenerating all twenty-three earlier sources):

- **`--title-form ref-in-brackets`** (Sedgwick, six books as one source, `--in` six times): "The Nature and Danger of Heresies (Reyel. 12:15, 16.)" is the title "The Nature and Danger of Heresies" and the `ref` Revelation 12:15-16. The old spelling is written out ("Reyel." is Revelation, "Jerem ." Jeremiah) and verses that follow one another in a chapter ("12:15, 16") are one range, where Logos lists them as two refs; the title's ref replaces Logos's only when it holds every one of them. A bracket that is not a passage (a date) is not read, and a title with no bracket is kept as it is and listed in `REPORT.md` ("Titles that did not have the form": six of Sedgwick's seven).
- **`--title-from-ref`** (Moberly): a title that is only "Sermon I" becomes "Sermon I: Matthew 5:3" ("Sermon IX: Matthew 23:29–32"), with a note that the index titles it by the number alone. A bare "Sermon N" with no ref is kept and listed ("Bare sermon numbers with no ref").
- Eyton, Chapin and Watson need no rule: Eyton's and Chapin's index names no preaching text (the Beatitude each preaches on is among the cited passages, none taken as the text), Watson's has one for every sermon.

Batch 5b (Pope, Smale, W. I. Evans, Calvin in Portuguese) adds one more flag, off unless given (checked by regenerating all twenty-eight earlier sources):

- **`--title-form pt-ref`** (Calvin's *Sermões sobre Eleição e Reprovação*): "I. Primeiro Sermão: Gênesis 25.12–22" keeps its whole title, in Portuguese, and the passage after the colon is the `ref` (`Gen.25.12-Gen.25.22`): the book is read by its Portuguese name (accents ignored; `tools/logos-import/pt.go` has all sixty-six), a full stop parts chapter from verse and a comma lists verses ("25.21,22" is Genesis 25:21-22, one range). The index gives Calvin's rows no reference of their own, only the verses among the cited ones; those inside the text's range are left out of `cited`. The book is Portuguese, so its `source.toml` says `language = "pt"` and its name says so too.
- `NonSermon` also drops a Portuguese "Prefácio" (and "Prefácio à Edição Brasileira") and Calvin's appendix "Resposta a Certas Calúnias e Blasfêmias", matched with the accents folded away.
- Smale's and W. I. Evans's "Chapter N: " is taken off by the rule that was already there (batch 4a); Pope needs none. None of the three has a preaching text but Evans's fourth chapter (1 Corinthians 14, from Logos's own reference).

Batch 6a (six of the Sermon Outlines books: Evangelistic, Growing Christians, Seekers, Special Days, 1 Corinthians, Acts) adds two flags and one correction, all off unless given (checked by regenerating all thirty-two earlier sources: the same bytes):

- **`--chapter-outlines`** (Sermon Outlines on Acts): the book's index mixes chapters, each an outline on a passage, with the numbered sub-points of other chapters ("3. The Duties of Deacons"), two introductory chapters with no passage, a "(Concluded)" chapter, two essay chapters and an afterword. The rule: a "Chapter N: Title" line with a passage (a Logos reference or a label) is a row, one row for each outline (38 of the 86 lines); every other line is dropped and listed in `REPORT.md` under "Chapter and sub-point lines dropped" with its reason (42 sub-points, 5 chapters with no passage, the afterword). "Chapter N: " is taken off the title by the rule of batch 4a and noted.
- **`--skipped "<book>: <why> || <title> || <title>"`** names a book of the order the source leaves out and the titles the Mayor's index found in it, in a "Skipped" section of `REPORT.md`; the tool does not read the book. `evangelistic-sermon-outlines-logos` uses it for *Sermon Outlines: Soul Winning! Church Building!* (LLS:SOSOULWIN), whose 24 titles carry no passage (the index found 0 of 24), for a later decision.
- The four books whose titles print the passage ("Winning over Worry (Matthew 6:25–34)") use the existing `--title-form ref-in-brackets`; Evangelistic's titles print none and Acts's chapters none, so they use no form. Three Special Days titles print a form the book table does not read ("John 8:3 ff.", "Acts 2:1–4; 37–41", "Ruth 1:16; 19–22"): they keep their whole title and Logos's reference, and `REPORT.md` lists them.
- Correction: `--title-form ref-in-brackets` no longer says "The ref is read from the title" (in the row's notes, and in the count) when the title prints exactly the one reference Logos already has; that happened to none of Sedgwick's rows, so no earlier source changed.

Batch 6b (Sermon Outlines on Christian Living, Ephesians, Galatians-Ephesians-Philippians-Colossians, Romans and the Gospels, and Sermon Starters, Volume 1) uses `--title-form ref-in-brackets` for all six and adds two small rules, both checked by regenerating all thirty-eight earlier sources (the same bytes):

- A part-of-verse letter after a verse in the passage a title prints ("Faithfulness in Hard Places (2 Peter 2:1–10a)") is dropped, so the title's reading is the whole verse (2 Peter 2:1-10). Logos writes the same passage with a letter (`bible.82.2.1-82.2.10.a`) that the book table refuses, and the index's own label is "2 Peter 2:1–10a", so the row would have had no passage; now a Logos reference and a label that both fail to read let the title's passage be the ref. Logos's refused reference is still listed in `REPORT.md`. Cited passages with a letter are still refused and listed.
- `NonSermon` also drops a "Scripture Index", "Subject Index" or "Topical Index" (the last line of Sermon Starters, Volume 1: 201 lines, 200 rows, every one with a passage).
- *Sermon Outlines on the Galatians, Ephesians, Philippians, Colossians* has one row for each outline (30), each title printing its own letter and passage ("Paul and the Gospel (Galatians 1:1–2:21)", over two chapters), not four book names. One title in *on Romans* prints two passages ("Pears or Prickle? (Romans 14:13–19 and Galatians 5:19–23)"): the form does not read it, so the row keeps the whole title and Logos's reference, and `REPORT.md` lists it.

Batch 6c (Sermon Starters, Volumes 2 to 5) uses `--title-form ref-in-brackets` and one new rule, checked by regenerating all earlier Logos sources (the same bytes):

- `NonSermon` drops a numbered point of an outline: a title that begins "First", "Second", "Third", "Fourth", "Fifth" or "Sixth" and a dash ("First—The Purpose in the Message"; volume 4 has four of them, in a row, with no passage). The dash is the mark: "First Things First", "Judgment First" and "Fourth of July" are kept. Each volume's last line is its "Scripture Index", dropped by the rule of batch 6b. The counts: volume 2, 199 lines and 198 rows; 3, 200 and 199; 4, 203 and 198; 5, 200 and 199, every row with a passage and none with a bracket left in the title.

`--infer-text` (mw-qtu3zvz.35; off unless given, checked by regenerating all earlier sources: the same bytes) gives a row the index names no text for the first passage it cites as its text, when (a) it is the only passage the sermon cites, (b) its title names it, or (c) at least two sermons next to it in the book (an unbroken run) are on it. The row's `ref` is that passage, it is no longer among the `cited` ones, and the row's notes say "The ref is inferred, not the index's" and why (a row has no other field for it; Lampas can read the note). Logos's own reference and a label that reads always win. A sermon that cites several passages with none of these has no text, so Maclaren's Chapters I to VII (Matthew 5:3 to 5:9) are texts and his Chapter X (two passages) is not. It is used for `burroughs-logos`, `maclaren-logos` and `broadus-logos`; `REPORT.md` has an "inferred text" count and a "Rows given an inferred text" list.

What the import does not do: read a passage from a **title** unless `--refs-from-title` is given (Skip Heitzig's "Exodus 3-4" rows carry no Logos reference, so they have no passage; about 117 rows), and tell a sermon from a heading of the archive beyond the `NonSermon` list (the book and series headings are rows).

## The shapes already here

| Source | Catalogue | Written by | Look at |
| --- | --- | --- | --- |
| `spurgeon-ccel` | CCEL's ThML: the preaching text tagged | `tools/ccel-spurgeon` | the tool, its REPORT, and `host_rules` (CCEL's built-in title rule) |
| `brookfield-first` | A YouTube channel, by the official Data API (keyed) | `tools/brookfield-youtube` | the tool, its `REPORT.md`, and the 30-day rule in its `licence_note` |
| `henry-sermons` | Scans' OCR, a contents page | by hand | its REPORT: rows the OCR could not place, the unsure ones |
| `greg-laurie-logos`, `jd-greear-logos`, `skip-heitzig-logos`, `spurgeon-sermon-notes-logos`, `wesley-logos`, `whitefield-logos`, `moody-logos`, `augustine-logos`, `chrysostom-logos`, `piper-logos`, `bunyan-logos`, `charles-wesley-logos`, `jimmy-evans-logos`, `shedd-logos`, `burroughs-logos`, `keach-logos`, `maclaren-logos`, `kuyper-logos`, `broadus-logos`, `worlds-great-sermons-logos`, `princeton-sermons-logos`, `mcpherson-logos`, `strong-logos`, `sedgwick-logos`, `moberly-logos`, `eyton-logos`, `chapin-logos`, `watson-logos`, `pope-logos`, `smale-logos`, `wi-evans-logos`, `calvin-pt-logos`, `evangelistic-sermon-outlines-logos`, `sermon-outlines-for-growing-christians-logos`, `sermon-outlines-for-seekers-logos`, `sermon-outlines-for-special-days-logos`, `sermon-outlines-on-1-corinthians-logos`, `sermon-outlines-on-acts-logos`, `sermon-outlines-on-christian-living-logos`, `sermon-outlines-on-ephesians-logos`, `sermon-outlines-on-galatians-ephesians-philippians-colossians-logos`, `sermon-outlines-on-romans-logos`, `sermon-outlines-on-the-gospels-logos`, `sermon-starters-volume-1-logos`, `sermon-starters-volume-2-logos`, `sermon-starters-volume-3-logos`, `sermon-starters-volume-4-logos`, `sermon-starters-volume-5-logos` | The Governor's Logos archives and sermon books: an index file in the vault | `tools/logos-import` | [A Logos archive](#a-logos-archive-data-only), and each source's REPORT |
| `themes-tcr1934`, `themes-torrey` | CrossWire SWORD modules | `tools/sword-themes` | [sources.md, "Theme sources"](sources.md#theme-sources); a theme source is a different recipe (topics and links) and is not covered here |

The next three candidates, with their terms quoted, are in the closing comment of the story that wrote this page (mw-hxztgfo.8) and in [candidates.md](candidates.md).
