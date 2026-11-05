# Source directories

A source is a directory under `sources/`, committed to this repository: a sermon list (Spurgeon from CCEL, Matthew Henry by hand, a modern preacher's catalogue) or a theme system (Thompson's 1934 topics, Torrey). The files are the truth; `sermons load` puts them in the database and `sermons check` says what is wrong with them, by file and line. Lampas bundles the same files for its offline copy, so the format is a contract with Lampas too. `sources/_template/` is a filled-in example with every field explained.

```
sources/
  _template/          the template (never loaded: its name begins with '_')
  _fixtures/          sources the tests load (never loaded by --all)
    spurgeon-20/
  spurgeon-ccel/      a sermon source
    source.toml
    rows.jsonl
  themes-tcr1934/     a theme source
    source.toml
    topics.jsonl
    links.jsonl
```

`sermons load --all sources/` and `sermons check --all sources/` take every directory under `sources/` whose name does not begin with `_` or `.`.

<!-- toc -->

- [source.toml](#sourcetoml)
- [rows.jsonl (a sermon source)](#rowsjsonl-a-sermon-source)
- [topics.jsonl and links.jsonl (a theme source)](#topicsjsonl-and-linksjsonl-a-theme-source)
- [References](#references)
- [sermons check](#sermons-check)
- [sermons load](#sermons-load)
- [The fixture and the template](#the-fixture-and-the-template)

<!-- /toc -->

## source.toml

TOML. Every key is required except `licence_note`; an unknown key is a fault, so a misspelt key is never silently lost.

| Key | What it is |
| --- | --- |
| `id` | The source's id: lower-case letters, digits and `-`, and the same as the directory's name (a directory whose name begins with `_` is exempt). It is the first half of every row's key, and the id the API and Lampas use. Never change it once loaded: a new id is a new source. |
| `name` | The name a reader sees: "C. H. Spurgeon's sermons (CCEL)". |
| `kind` | `"sermons"` (the directory has `rows.jsonl`) or `"themes"` (it has `topics.jsonl` and `links.jsonl`). |
| `licence` | The terms of the site or edition, **quoted verbatim** from its terms page, never summarised; not empty. Say in a comment where and when it was read. A TOML multi-line string (`"""`) holds a long text; a line ending in `\` joins the next. |
| `licence_note` | Optional. Our reading of the licence in our own words: why the rows may be indexed or linked, and anything to watch ("the index keeps only titles, references and links"). |
| `credit` | Whom the app credits: the author and the edition or site. |
| `homepage` | The source's home page, https. |
| `terms_url` | The page `licence` was quoted from, https. |
| `added` | The day the source was added, a TOML date (`2026-11-05`, unquoted). |
| `on_by_default` | `true` or `false`: whether a reader who has never chosen sees the source. Required, so the choice is always made on purpose. |

## rows.jsonl (a sermon source)

One JSON object a line, one sermon each. A blank line is skipped. An unknown field is a fault. An optional field is left out when it has no value, never given as `""` (`sermons check` refuses `"preacher": ""`).

```json
{"id": "104", "title": "Love's Commendation", "ref": "Rom.5.8", "url": "https://ccel.org/ccel/spurgeon/sermons02/sermons02.l.html", "alt_url": "https://www.spurgeongems.org/sermon/chs104.pdf", "preacher": "C. H. Spurgeon"}
```

| Field | | What it is |
| --- | --- | --- |
| `id` | required | The row's id within the source: letters, digits, `.`, `_`, `-`. The preacher's own number where there is one (`"104"`), else a stable slug. Unique in the file. |
| `title` | required | The title as the source prints it. |
| `ref` | optional | The preaching text, an OSIS reference (below). Left out only when the sermon has no single text (an address); such a row is still loaded and found by its source and by search, but not by passage. |
| `url` | required | https; the page a reader is sent to. |
| `alt_url` | optional | https; a second form (a pdf, an audio file). |
| `kind` | optional | `"link"` (the default: the sermon is elsewhere), `"hosted"` (a file this service hosts), `"exposition"` (an exposition of a passage rather than a sermon on a text). |
| `preacher` | optional | The preacher's name as the app shows it. |
| `date` | optional | When it was preached, ISO 8601 at the precision known: `"1855"`, `"1855-03"`, `"1855-03-18"`. |
| `duration_s` | optional | Length in seconds (audio, video); a positive whole number. |
| `cited` | optional | Other references the sermon cites, `["John.3.16", "Rom.5.1-Rom.5.11"]`; indexed with the role `cited`, below the preaching text. |
| `notes` | optional | Anything a later reader of the file should know: where the `ref` came from when the source did not tag it, why a field is missing. |

## topics.jsonl and links.jsonl (a theme source)

A theme source has no `rows.jsonl`. Its `topics.jsonl` has one topic a line:

```json
{"id": "love-hatred", "label": "Love-Hatred", "parts": [{"n": 4, "label": "Of God"}], "see": ["grace-divine", "themes-torrey:love-of-god"]}
```

| Field | | What it is |
| --- | --- | --- |
| `id` | required | The topic's id within the source: letters, digits, `.`, `_`, `-`; unique in the file. |
| `label` | required | The topic's name as the source prints it. |
| `parts` | optional | The topic's numbered parts, `{"n": 4, "label": "Of God"}`; `n` a positive whole number, unique within the topic. |
| `see` | optional | Related topics: a bare id is a topic of this file (checked); `"<source>:<id>"` names another source's topic (not checked). |

Its `links.jsonl` has one link from a topic to a passage a line:

```json
{"topic": "love-hatred", "part": 4, "ref": "Rom.5.8"}
{"topic": "grace-divine", "ref": "Eph.1.7", "via": "sermon", "from": "spurgeon-ccel:330"}
```

| Field | | What it is |
| --- | --- | --- |
| `topic` | required | A topic id of this source's `topics.jsonl`. |
| `part` | optional | A part number of that topic. |
| `ref` | required | The passage, an OSIS reference (below). |
| `via` | optional | Where the link came from: `"source"` (the default: the theme system itself), `"bible"` (a theme noticed in reading), `"sermon"` (a theme a sermon carries). |
| `from` | with `via: "sermon"` | The sermon, as `"<source>:<id>"`. |

A link given twice (the same topic, part, reference, via and from) is a fault.

## References

A reference is OSIS: `Book.Chapter.Verse` (`Rom.5.8`), `Book.Chapter` (a whole chapter, `Ps.23`), `Book` (a whole book, `Jude`), or a range of two of those joined by `-` with the end written in full (`Rom.5.1-Rom.5.11`, `Rom.5.12-Rom.6.4`, `Rom.5-Rom.6`). The book is the OSIS name (`Gen`, `Exod`, `Ps`, `Song`, `Matt`, `1Cor`, `Phlm`, `Rev`) or Lampas's code (`gen`, `exo`, `psa`, `sng`, `mat`, `1co`, `phm`, `rev`), in any case; the database keeps the OSIS form. Chapters and verses are numbered as Lampas numbers them, which is the KJV's numbering with the exceptions in [schema.md, "The book table"](schema.md#the-book-table). `Rom.5.1-11` (a short end) is refused with the whole form suggested.

## sermons check

```
sermons check sources/spurgeon-ccel
sermons check --urls 20 sources/spurgeon-ccel
sermons check --all sources
```

Prints one line per fault, with the file and line, and exits 1 on any (0 and `ok: N source(s)` on none):

```
rows.jsonl:7: ref 'Rom.5.99': verse past the chapter
```

It checks:

- `source.toml`: present, TOML, every required key given and not empty (the licence above all), no unknown key, `kind` known, `id` well formed and the directory's name, `homepage` and `terms_url` https and well formed.
- The files of the kind are present and not empty, and the other kind's are absent.
- Every line is one JSON object with no unknown field.
- `rows.jsonl`: `id`, `title` and `url` given; ids well formed and unique (`id '104' repeats line 6`); `ref` and every `cited` reference parse (`unknown book 'Tob'`, `chapter past the book`, `verse past the chapter`, `verse 0 is not a verse ...`, `the range ends before it starts`); `url` and `alt_url` https and well formed; `kind`, `date` and `duration_s` well formed; no optional field given empty.
- `topics.jsonl`: ids well formed and unique, a label, parts numbered and unique, every bare `see` id a topic of the file.
- `links.jsonl`: the topic is in `topics.jsonl`, the part is one of the topic's, the reference parses, `via` is known, a link via a sermon names it in `from`, no link repeats.
- `--urls N` (off by default; uses the network): sends a HEAD to a random N of the source's `url` and `alt_url` values and reports any that does not answer 200 after redirects (`rows.jsonl:12: url 'https://...': 404 Not Found`). It is run only when the rest passes. A 200 does not prove the page is the sermon: CCEL answers 200 to a made-up section id, so a CCEL row's title must be checked by the tool that writes it.

## sermons load

```
sermons load sources/spurgeon-ccel
sermons load --all sources
```

Needs `DATABASE_URL` and a migrated database (`sermons migrate`). Checks the directory first and loads nothing if it has a fault. Then, in one transaction per source: upserts the source and every row by its key, deletes the source's rows that are no longer in the files, and prints the counts:

```
spurgeon-20: 20 rows, 19 passages, 0 removed
demo-themes: 2 topics, 1 parts, 3 links, 0 removed
```

`passages` counts the preaching texts and cited references; `removed` counts what was in the database and is no longer in the files (a sermon source's rows; a theme source's topics and links). Loading the same files again changes nothing and prints the same counts.

## The fixture and the template

`sources/_fixtures/spurgeon-20/` is the twenty Spurgeon sermons the 2026-11-05 probe tried (Lampas `docs/sources/sermon-index.md`, section 6), with their CCEL page urls (the section ids read from CCEL's ThML) and Spurgeon Gems pdfs. 19 have a preaching text; 385 has none. Sermon 12's text (Psalm 127:2) is from its printed head, which CCEL's ThML does not tag; 62 is an exposition with no Spurgeon Gems pdf. The tests load it.

`sources/_template/` is a source directory with every field filled in and explained in `source.toml`'s comments; copy it to start a new source. `sermons check` passes on both (a test keeps it so).
