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
  themes-tcr1934/     a theme source (Thompson's 1934 topics)
    source.toml
    topics.jsonl
    links.jsonl
  themes-torrey/      a theme source (Torrey's New Topical Textbook)
```

`sermons load --all sources/` and `sermons check --all sources/` take every directory under `sources/` whose name does not begin with `_` or `.`.

<!-- toc -->

- [source.toml](#sourcetoml)
- [rows.jsonl (a sermon source)](#rowsjsonl-a-sermon-source)
- [topics.jsonl and links.jsonl (a theme source)](#topicsjsonl-and-linksjsonl-a-theme-source)
- [References](#references)
- [sermons check](#sermons-check)
- [sermons load](#sermons-load)
- [Theme sources](#theme-sources)
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
- `--urls N` (off by default; uses the network): sends a HEAD to a random N of the source's `url` and `alt_url` values and reports any that does not answer 200 after redirects (`rows.jsonl:12: url 'https://...': 404 Not Found`). It is run only when the rest passes. A 200 does not prove the page is the sermon: CCEL answers 200 to a made-up section id, so for a `url` on `ccel.org` the check fetches the page (a GET, not a HEAD) and reports `200 OK but the title '...' is not in the page` unless the row's title is in the page's text (markup stripped, entities read, curly quotes made straight, case ignored). The rule is per host (`titleHosts` in `internal/sources/check.go`); `alt_url` is only asked with HEAD.

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

## Theme sources

`sources/themes-tcr1934` and `sources/themes-torrey` are written by `tools/sword-themes` from two CrossWire SWORD modules, `TCR` and `Torrey` (RawLD dictionaries: a topic a key, ThML in the entry). The tool fetches `packages/rawzip/<Module>.zip` and `raw/mods.d/<module>.conf` from `https://www.crosswire.org/ftpmirror/pub/sword` into a git-ignored cache (a file already there is not fetched again), writes `topics.jsonl` and `links.jsonl`, and writes the same bytes on a second run. `source.toml` is written by hand: the module's `DistributionLicense` and `About` lines are the `licence`, quoted verbatim. [tools/sword-themes/REPORT.md](../tools/sword-themes/REPORT.md) is what the tool could not read.

```
go run ./tools/sword-themes --module TCR    --cache .sword-cache --out sources/themes-tcr1934 --report tools/sword-themes/REPORT.md
go run ./tools/sword-themes --module Torrey --cache .sword-cache --out sources/themes-torrey   --report tools/sword-themes/REPORT.md
```

**Never name a feature with the trademark.** "Thompson Chain-Reference" is a trademark of the publisher of the printed Bible (B. B. Kirkbride's, now Zondervan's). The source's `name` is the module's own, "Thompson Chain Topics (1934 subject index)"; the credit goes to Frank Charles Thompson and to CrossWire's module (keyed and proofed by Paul Houghton). The licence field quotes the module's header, which uses the trademark, because it must be verbatim; nothing else here, and no feature, screen or API name built on this source, uses it as a name: say "Thompson's topics" or "the 1934 subject index".

**What a module's text becomes.**

| In the module | In the source |
| --- | --- |
| A key (`LOVE-HATRED`) | A topic: `label` is the key in title case ("Love-Hatred"; small words such as *of* and *the* lower case unless they begin the text or follow a comma, semicolon, hyphen or bracket: "Sinner's Friend, Christ the", "Atonement, The"), `id` is the key's slug (`love-hatred`; a second key with the same slug takes `-2`). |
| Thompson: `<b>(4) Of God</b>` or `<b>(4)</b> Of God` | A part: `{"n": 4, "label": "Of God"}`. An all-capitals label is made title case. A number with no label at all is "Part N". |
| Thompson: `<b>(C) LOVE, SPIRITUAL</b>` | A section; not a part. Its name is used only below. |
| Torrey: a line beginning `- ` | A part, numbered as the lines come (1, 2, ...). |
| Torrey: a line beginning `. ` under it, with references | A part of its own, its label the line above's, `: `, then its own ("Described as: Sovereign"). A line that wraps (a line with none of these before a reference) goes on the label above it. |
| `<scripRef>Ro 5:8; 1Jo 4:9,16; Ps 19:8-10</scripRef>` | One link a comma element: `Rom.5.8`, `1John.4.9`, `1John.4.16`, `Ps.19.8-Ps.19.10`, to the part above it, or to no part when it comes before any. A book's spelling is read through a table of about 75 (`Ge`, `Le`, `Jud` is Judges, `Joe`, `Mr`, `Php`, `Re` ...); a later item without a book takes the earlier one's; a number alone is a chapter, or a verse in a book of one chapter; a range may end in another chapter (`Isa 52:13-53:12`) or another book. The same topic, part and reference twice is written once. |
| Thompson: `--SEE X, <term>KEY</term>` (and `& <term>KEY</term>`); Torrey: `. See <term>KEY</term>` | A `see` entry of the topic. Thompson's terms name a topic by its whole key, by one half of a pair (`POVERTY` for `POVERTY-WEALTH`) or by the word before the comma (`GRACE` for `GRACE, DIVINE`); the topic itself, a term that matches no topic or more than one, and an empty term are listed in REPORT.md and not kept. |

**Thompson's lettered sections repeat numbers.** A topic such as `LOVE-HATRED` has sections (A) Love, Brotherly, (B) Love, Family, (C) Love, Spiritual and (D) Hatred, each numbering its parts again from (1), and a part's number is unique in a topic. The first part with a number keeps it ("(4) Of God", Romans 5:8, is part 4); a later part with the same number takes the next number past the topic's printed ones, and its label begins with its section's name ("Hatred: Of the Saints"). 116 topics are renumbered; REPORT.md lists them. Only the parts' labels carry the section; a reader never sees the number alone.

**Psalm numbering: the modules and Lampas agree, on the KJV's.** Both modules number the Psalms as the KJV does, where a superscription is not numbered and verse 1 is the first verse after it; so does Lampas ([schema.md, "The book table"](schema.md#the-book-table)). The check of this, on 2026-11-05: no Psalm reference in either module was refused by the numbering (every `Ps` item parses; none is in REPORT.md), and the spot checks fit the KJV: Psalm 51:5 ("shapen in iniquity") is under Heredity, 51:7 (hyssop) under Hyssop and Snow, 51:12 (restore the joy) under Restoration, where Hebrew numbering would put them at 51:7, 51:9 and 51:14. So the loader keeps the module's numbers as they are, and no verse is shifted. Where a Hebrew-numbered source would shift a verse is the Psalms whose Hebrew text counts the title as one or two verses. Counting the verse ends (sof pasuq) of each psalm in Lampas's Hebrew text against its KJV-numbered verses, the 62 psalms whose counts differ are (KJV verses→Hebrew verses): 3 (8→9), 4 (8→9), 5 (12→13), 6 (10→11), 7 (17→18), 8 (9→10), 9 (20→21), 12 (8→9), 18 (50→51), 19 (14→15), 20 (9→10), 21 (13→14), 22 (31→32), 30 (12→13), 31 (24→25), 34 (22→23), 36 (12→13), 38 (22→23), 39 (13→14), 40 (17→18), 41 (13→14), 42 (11→12), 44 (26→27), 45 (17→18), 46 (11→12), 47 (9→10), 48 (14→15), 49 (20→21), 51 (19→21), 52 (9→11), 53 (6→7), 54 (7→9), 55 (23→24), 56 (13→14), 57 (11→12), 58 (11→12), 59 (17→18), 60 (12→14), 61 (8→9), 62 (12→13), 63 (11→12), 64 (10→11), 65 (13→14), 67 (7→8), 68 (35→36), 69 (36→37), 70 (5→6), 75 (10→11), 76 (12→13), 77 (20→21), 80 (19→20), 81 (16→17), 83 (18→19), 84 (12→13), 85 (13→14), 88 (18→19), 89 (52→53), 92 (15→16), 102 (28→29), 108 (13→14), 140 (13→14), 142 (7→8). (A count can miss a psalm whose title is counted but whose last verse the KJV splits in two, as Psalm 13; a source in Hebrew numbering must be mapped, psalm by psalm, before its rows are written.) For the Lampas versification story. Romans 16:25-27 is the other known difference: 16 references in the two modules to the doxology as the KJV numbers it are refused (`verse past the chapter`: Lampas has Romans 16 end at verse 24 and the doxology at 14:24-26) and listed in REPORT.md, not mapped.

**What is not read.** Thompson's 20 items in 12 groups and Torrey's 14 items in 13 groups that the reader leaves out are in REPORT.md with their topic and group: a book missing in the print (`86:13; 116:3; 139:8; Pr 5:5 ...` begins with no book), Romans 16:25-27, a chapter a book does not have (`Ob 3:2`, `Phm 3:2`), a range that ends before it starts (`Ge 36:31-29`), a stray `(marg)`. Each item is left out alone; the rest of its group is kept. None is guessed.

**The renewal check of Thompson's 1934 index.** `themes-tcr1934/source.toml`'s `licence_note` says what was read and found: no renewal entry for registration A72501 in the Catalog of Copyright Entries, Third Series, Part 1 (Books and Pamphlets), the four volumes from January 1961 to December 1962, searched in the Internet Archive's OCR text. If a later reading finds a renewal, the source is removed, not edited.

## The fixture and the template

`sources/_fixtures/spurgeon-20/` is the twenty Spurgeon sermons the 2026-11-05 probe tried (Lampas `docs/sources/sermon-index.md`, section 6), with their CCEL page urls (the section ids read from CCEL's ThML) and Spurgeon Gems pdfs. 19 have a preaching text; 385 has none. Sermon 12's text (Psalm 127:2) is from its printed head, which CCEL's ThML does not tag; 62 is an exposition with no Spurgeon Gems pdf. The tests load it.

`sources/_fixtures/themes-demo/` is a theme source of four invented topics on Romans 5 (Love with two parts, Grace, Atonement, Adam and Christ): not Thompson's or Torrey's index, only something for the API's tests and [api.md](api.md)'s examples to answer from.

`sources/_template/` is a source directory with every field filled in and explained in `source.toml`'s comments; copy it to start a new source. `sermons check` passes on both (a test keeps it so).
