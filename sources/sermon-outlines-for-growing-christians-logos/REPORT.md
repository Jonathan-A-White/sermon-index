# sermon-outlines-for-growing-christians-logos: what the import did not settle

<!-- toc: none -->

Written by `go run ./tools/logos-import --in <vault>/plans/sermons-2026-11-05/sermon-outlines-for-growing-christians-sermons.jsonl --title-form "ref-in-brackets" --id sermon-outlines-for-growing-christians-logos --out sources/sermon-outlines-for-growing-christians-logos`; do not edit by hand. Only ids, titles, references and addresses are listed.

## Counts

```
lines read:                          27
rows written:                        27
lines dropped:                       0
rows with a ref:                     27
  of them read from the label:       0
  of them read from the title:       1
  of them with cited references:     22
rows with no ref:                    0
series rows (ref over >3 chapters):  1
rows with their own date:            0
rows dated by the year alone:        0
references the book table refused:   1
Logos and the label disagree:        3
```

## Lines dropped (0)

A line with neither a ref nor a title, or with no id, no title, no address or an id already used, is not a row.

None.

## Titles that did not have the form (0)

--title-form "ref-in-brackets" reads the preacher, years, text or number printed in a title; these titles do not have it, so the row keeps the title and the preacher as the index gives them.

None.

## References the book table refused (1)

A Logos reference to a book outside the sixty-six, or a chapter or verse past the end; it is left out of the row, and the row stays.

- SOGROWCHR.1-80075: "bible.61.6.25.b": want book.chapter or book.chapter.verse

## Series rows (1)

The ref spans more than three chapters (a whole book or a long text): the row is kept, found by its title and listed by its source, and matched by no passage.

- SOGROWCHR.1-8708 Jonah.1-Jonah.4 (4 chapters)

## Logos and the label disagree (3)

Both were read and differ; the row carries Logos's reference.

- SOGROWCHR.1-39649: Logos says Luke.11.1-Luke.11.12; Luke.22.39-Luke.22.42, the label "Luke 11:1–12" says Luke.11.1-Luke.11.12
- SOGROWCHR.1-59429: Logos says Acts.4.36-Acts.4.37; Acts.9.26-Acts.9.30; Acts.15.1-Acts.15.41, the label "Acts 4:36–37" says Acts.4.36-Acts.4.37
- SOGROWCHR.1-72874: Logos says Matt.25.24-Matt.25.25; Phil.4.13, the label "Matthew 25:24–25" says Matt.25.24-Matt.25.25

## Rows with no ref (0)

Neither a Logos reference nor a label that could be read: the row is kept, found by its title and its source, and by no passage. Ids, in the file's order:

None.
