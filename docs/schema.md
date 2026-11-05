# The schema (v1)

The sermon index's database, as `internal/migrate/sql/0001_init.sql` makes it, and why each choice was made. The API (story A2) and Lampas's sermon epic build on this shape; the questions it leaves open are named at the end.

<!-- toc -->

- [The tables](#the-tables)
- [Why](#why)
- [The book table](#the-book-table)
- [Left open, by name](#left-open-by-name)

<!-- /toc -->

## The tables

```
sources                               one directory under sources/
  id  text PK  ─────────────┐         'spurgeon-ccel'  (= the directory's name)
  name, kind ('sermons'|'themes'), licence (verbatim, not empty), licence_note?,
  credit, homepage, terms_url (https), added (date), on_by_default, loaded_at
  UNIQUE (id, kind)  ◄──────┼─────────────── the kind-checked foreign keys below
                            │
theme_sources  (a VIEW: the sources whose kind is 'themes')
                            │
sermons                     │         one line of rows.jsonl
  PK (source_id, id) ───────┼──┐      ('spurgeon-ccel', '104')
  source_id, source_kind='sermons' ──► sources (id, kind)  ON DELETE CASCADE
  title, preacher?, url (https), alt_url?, kind ('link'|'hosted'|'exposition'),
  date? ('1855' | '1855-03' | '1855-03-18'), duration_s?, notes?
                            │  │
sermon_passages             │  │      the sermon's preaching text and cited refs
  PK (source_id, sermon_id, role, ref_osis)
  (source_id, sermon_id) ──────┘ ──►  sermons  ON DELETE CASCADE
  ref_osis ('Rom.5.1-Rom.5.11'), role ('text'|'cited'),
  span int4range  [first ordinal, last ordinal + 1)      GiST index on span
                            │
themes                      │         one line of topics.jsonl
  PK (source_id, id)        │
  source_id, source_kind='themes' ───► sources (id, kind)  ON DELETE CASCADE
  label, see text[]  (a bare id: this source's topic; 'src:id': another's)
                            │
theme_parts                           a topic's numbered parts, "(4) Of God"
  PK (theme_source_id, theme_id, n) ─► themes  ON DELETE CASCADE
  label
                            │
theme_links                           one line of links.jsonl
  (source_id, theme_id) ─────────────► themes  ON DELETE CASCADE
  (source_id, theme_id, part_n) ─────► theme_parts  (MATCH SIMPLE: no part, no check)
  part_n?, ref_osis, span int4range (GiST index), via ('source'|'bible'|'sermon'),
  from_ref? (required when via = 'sermon': '<source>:<sermon id>')
  UNIQUE NULLS NOT DISTINCT (source_id, theme_id, part_n, ref_osis, via, from_ref)

schema_migrations (version PK, name, applied_at)   made by `sermons migrate`
```

`?` marks a column that may be NULL. Every other column is NOT NULL.

## Why

**The files are the truth; the database is their index.** Every row comes from a file under `sources/` and `sermons load` makes the database match the files. Nothing is typed into the database by hand, so it can be dropped and rebuilt from the repository at any time (`sermons migrate && sermons load --all sources/`). This is what makes "a new preacher is a data add" true: a new directory, `sermons check`, `sermons load`.

**Text keys from the files, no surrogate ids.** A sermon's key is `(source_id, id)`: the source's directory name and the row's own id (Spurgeon's number, `'104'`). A serial id would be a second name for the same row that exists only in one database: it would differ between the Laptop and a rebuild, Lampas could not use it offline (Lampas bundles the same files, not the database), and an upsert would need it looked up first. With the files' own keys, a reload is `INSERT ... ON CONFLICT (key) DO UPDATE`, the API's ids are the files' ids, and Lampas's offline copy names the same rows by the same keys. A key is never reused for another sermon: a source that renumbers is a new source.

**One sources table for both kinds; `theme_sources` is a view.** A licence and a credit are the same thing for a sermon list and a theme system, and the API's `/v1/sources` lists both, so they share one table with a `kind`. `theme_sources` is a view over it, for queries that want only theme systems.

**Kind-checked foreign keys.** `sermons` and `themes` carry a constant `source_kind` column with a foreign key to `sources (id, kind)`. The database itself then refuses a sermon in a theme source or a topic in a sermon source; the check in the loader is not the only guard.

**The rules live in the database too.** CHECK constraints refuse an empty licence, a non-https url, an unknown kind or role, an empty or unbounded span and a badly shaped date, so a bad row cannot arrive by a path that skips `sermons check` (a hand `psql`, a future tool). `sermons check` says the same things earlier and by file and line.

**NULL for absent, never `''`.** An optional column is NULL when the file leaves the field out, and a CHECK refuses `''`, so "no preacher" has one spelling. `sermons check` refuses `"preacher": ""` in a file ("leave it out instead") for the same reason.

**The verse ordinal: `book * 1000000 + chapter * 1000 + verse`.** Books are numbered 1 to 66 in Lampas's order (Genesis 1, Matthew 40, Romans 45, Revelation 66), chapters and verses as Lampas numbers them. Romans 5:8 is 45005008. An ordinal is an integer that sorts in Bible order, so a passage is a range of integers and passage overlap is integer range overlap. It is readable by eye (45 005 008), it fits an `int4` (the largest is Revelation 22:21, 66022021), and a gap between chapters (45005022 to 45005999 has no verse) costs nothing. The factors leave room: no chapter has 1000 verses and no book has 1000 chapters.

**A passage is an `int4range`, `[first, last + 1)`.** Postgres keeps an integer range in this canonical form (lower bound included, upper excluded), so Romans 5:1-11 is `[45005001,45005012)` and a single verse is a range of one. The question every passage query asks, "does this sermon's text overlap the verse, chapter or pericope I am reading?", is `span && $1`, and the GiST index on `span` answers it without a scan. Containment (`<@`, `@>`) gives "within this chapter" and "covers this verse". The alternative, one row per verse, multiplies rows by the width of each passage (a whole-chapter text is 30 rows) and makes "narrower first" a GROUP BY; a range keeps one row per passage and its width is `upper(span) - lower(span)` (within a chapter the number of verses; a range that crosses chapters is about 1000 a chapter wider, which still sorts it after every one-chapter range).

**The ordinal is not stored on its own; `ref_osis` is kept beside the span.** `ref_osis` is the canonical OSIS form of the reference as written (`Rom.5.1-Rom.5.11`; a Lampas code such as `rom.5.8` in a file is stored as `Rom.5.8`), so the API can show the reference without turning numbers back into names, and the passage's key is readable. The span is derived from it by `internal/passage` at load time.

**A sermon's preaching text and its cited references are both passages, told apart by `role`.** `'text'` is the text the sermon is preached on (Spurgeon's `scripCom type="Sermon"`); `'cited'` is a reference it quotes. The API answers by the text first (A2: "preaching text first"); the theme tier (A5) matches only the text. A sermon with no single text (an address, a series introduction) has no `'text'` passage and is still listed by its source and found by search.

**The date is text, ISO 8601 at the precision known.** `'1855'`, `'1855-03'` or `'1855-03-18'`: many sermons are known only to the year (Matthew Henry's) or the month, and a `date` column would force a made-up day. Text in this shape sorts in time order. `sermons check` also refuses a day that does not exist (`1855-02-30`).

**`theme_links` has no primary key, but a unique natural key.** A link's identity is its topic, its part, its reference, where it came from (`via`) and which sermon (`from_ref`); the part and the sermon may be absent. A primary key cannot hold a NULL, so the key is `UNIQUE NULLS NOT DISTINCT` (Postgres 15 or later: a link with no part is a key like any other, and a reload's `ON CONFLICT ON CONSTRAINT theme_links_key` finds it).

**A part is named by its number, not its label.** Thompson's parts are numbered within a topic ("Love-Hatred (4) Of God"); the number is stable where the label's wording may be corrected. A link to a part is checked by a foreign key; a link to the whole topic has `part_n` NULL and is not.

**Loading.** `sermons load <dir>` checks the directory first and refuses it whole if it has a fault (nothing is written). Then, in one transaction: the source is upserted; each row is upserted by its key (the update is skipped when nothing changed, so a reload of unchanged files writes no row versions); rows of the source that are no longer in the files are deleted (their passages, parts and links go with them by `ON DELETE CASCADE`); and the passages are made exactly those the files give. A second load of the same files changes nothing and prints the same counts. `--all` loads each directory in its own transaction, after checking them all.

**Migrations.** `internal/migrate/sql/NNNN_name.sql`, embedded in the binary, numbered 1, 2, 3 with no gaps, each applied once in its own transaction and recorded in `schema_migrations`; a run holds a transaction-scoped advisory lock, so two at once apply each once. An applied migration is never edited: a change is a new file. `sermons load` refuses a database with a migration not yet applied.

**Tests run on a real Postgres 17, not a fake.** `internal/store/testdb.go` starts `fergusstrange/embedded-postgres` once per test binary, on a free port, in a temporary directory, and gives each test its own database. Production is Postgres 17 too (the epic's call), so the GiST index, `NULLS NOT DISTINCT` and range operators the tests use are the ones production has. The tests never read `DATABASE_URL`.

## The book table

`internal/passage/books.go` is the 66 books in Lampas's order with their Lampas code (`rom`), OSIS name (`Rom`), English name and the number of each chapter's last verse. It was generated once from the Lampas repository at commit `931be110`: `public/data/index.json` for the books, their order, codes and names, and each `public/data/<code>/<chapter>.json` for the last verse number. It is committed and not regenerated by the build; a change to Lampas's versification is a deliberate change here, by hand, with this section updated.

One trap, found while building it: **`index.json`'s `verses` is a count of the verses Lampas's text has, not the last verse's number.** Five chapters lack a verse in Lampas's text (Nehemiah 7:68, Luke 17:36, Acts 8:37, 15:34, 24:7), so Acts 8 has 39 verses numbered up to 40. The table holds the last verse number (40), so a reference to Acts 8:40 is accepted, and so is Acts 8:37 (a verse Lampas's text lacks but the numbering keeps).

What the table assumes, and what the service does today when a reference does not fit it:

| Case | What Lampas's numbering is | What the service does |
| --- | --- | --- |
| The Apocrypha (Tobit, Sirach, 1-2 Maccabees, Wisdom, ...), Psalm 151 as a book, and any other book outside the 66 | Not in the table | **Refuses**: `unknown book 'Tob'` |
| Psalm 151 (`Ps.151`) | Psalms has 150 chapters | **Refuses**: `chapter past the book` |
| Greek (Septuagint/Vulgate) Psalm numbering (Greek Ps 22 is Hebrew and English Ps 23) | Hebrew/English numbering (Psalm 23 is "The Lord is my shepherd") | **Ignores**: a reference in Greek numbering is read as English numbering and lands on the wrong psalm without a fault. A source in Greek numbering must be mapped by its tool before its rows are written. |
| Psalm superscriptions (titles) | Not numbered: verse 1 is the first verse after the title, as in the KJV (Psalm 3 has 8 verses, Psalm 51 has 19) | **Refuses** verse 0 (`verse 0 is not a verse (a superscription is not numbered)`). A reference in Hebrew numbering, where the title is verse 1, is **ignored**: it lands one or two verses late without a fault, or is refused as `verse past the chapter` at a psalm's end (Hebrew Ps 51:21). |
| Old Testament chapter divisions | English (KJV) divisions: Malachi has 4 chapters, Joel 3, Nehemiah 4:1 is the KJV's | References in Hebrew chapter division (Malachi 3:19-24, Joel 4) are **refused** where the chapter does not exist (`Mal.3.19`: verse past the chapter) and **ignored** (read as English) where it does. |
| Romans 14 and 16 (the doxology) | Robinson-Pierpont (Byzantine) numbering: the doxology is Romans 14:24-26, so Romans 14 has 26 verses and Romans 16 has 24 | A KJV reference to Romans 16:25-27 is **refused** (`verse past the chapter`). A reference to Romans 14:24-26 is accepted and means the doxology. |
| Verses Lampas's text omits (Neh 7:68, Luke 17:36, Acts 8:37, 15:34, 24:7) | The numbering keeps them; the text lacks them | **Accepted** as verses: the ordinal exists, Lampas will have no text to show for it. |
| Every other chapter and verse | The same as the KJV's (the table's 31,097 verses are the KJV's 31,102 less the five omitted verses; Romans 14 and 16 trade three) | Accepted. |

No mapping between versifications is done anywhere in this service today: a reference is either in Lampas's numbering or refused. Mapping (KJV Romans 16:25-27, Greek Psalms, Hebrew Malachi) is open, for Lampas's versification story.

## Left open, by name

These are not decided by v1. Each is a question for the story named, or for the Governor.

1. **Removing a whole source.** `load` removes rows no longer in a source's files, but a source whose directory is deleted stays in the database. A `sermons unload <id>` (one `DELETE FROM sources`, the rest cascades) is not written. (Whoever first retires a source.)
2. **Search.** There is no text index yet. A2's `/v1/search?q=` needs one: a migration `0002` adding a generated `tsvector` over title, preacher and notes with a GIN index is the expected shape. (A2.)
3. **Ordering within an answer.** The schema gives what A2's "preaching text first, narrower first" needs (`role`, `upper(span) - lower(span)`); the order of equal widths (by source, by date, by id) is A2's to choose.
4. **The theme tier's 300-verse cap.** A topic's width (the verses its links cover) is not stored; A5 may compute it in the query or add a materialised column in a migration. (A5.)
5. **A link's `from` and `see` across sources** (`'spurgeon-ccel:330'`) are text, not foreign keys: a source may be loaded before the one it names. Whether `sermons check --all` should verify them is open.
6. **Hosted sermons.** `kind = 'hosted'` exists, but nothing says where a hosted file lives or how it is served. (The later hosting story.)
7. **Versification mapping** (the table above). (Lampas's versification story.)
8. **Book names in the API.** The parser accepts OSIS names and Lampas codes in any case and stores OSIS. Whether the API answers in OSIS (`Rom.5.8`) or in Lampas's codes (`rom.5.8`) is A2's call.
9. **The test helper is linked into the binary.** `internal/store/testdb.go` is in package `store`, so `sermons` carries embedded-postgres's code (the binary is about 17 MB). Moving it to its own package (`internal/store/storetest`) would drop that; it changes nothing the tests do.
