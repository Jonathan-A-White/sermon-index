# sermon-outlines-on-the-gospels-logos: what the import did not settle

<!-- toc: none -->

Written by `go run ./tools/logos-import --in <vault>/plans/sermons-2026-11-05/sermon-outlines-on-the-gospels-sermons.jsonl --title-form "ref-in-brackets" --id sermon-outlines-on-the-gospels-logos --out sources/sermon-outlines-on-the-gospels-logos`; do not edit by hand. Only ids, titles, references and addresses are listed.

## Counts

```
lines read:                          26
rows written:                        26
lines dropped:                       0
rows with a ref:                     26
  of them read from the label:       0
  of them read from the title:       2
  of them with cited references:     26
rows with no ref:                    0
series rows (ref over >3 chapters):  0
rows with their own date:            0
rows dated by the year alone:        0
references the book table refused:   3
Logos and the label disagree:        1
```

## Lines dropped (0)

A line with neither a ref nor a title, or with no id, no title, no address or an id already used, is not a row.

None.

## Titles that did not have the form (0)

--title-form "ref-in-brackets" reads the preacher, years, text or number printed in a title; these titles do not have it, so the row keeps the title and the preacher as the index gives them.

None.

## References the book table refused (3)

A Logos reference to a book outside the sixty-six, or a chapter or verse past the end; it is left out of the row, and the row stays.

- SOGOSP.1-7483: "bible.64.10.10.b": want book.chapter or book.chapter.verse
- SOGOSP.1-67708: "bible.63.10.34.a": want book.chapter or book.chapter.verse; "bible.63.10.34.b-63.10.35": want book.chapter or book.chapter.verse
- SOGOSP.1-89302: "bible.62.14.33.a": want book.chapter or book.chapter.verse; "bible.62.14.33.b-62.14.35": want book.chapter or book.chapter.verse

## Series rows (0)

The ref spans more than three chapters (a whole book or a long text): the row is kept, found by its title and listed by its source, and matched by no passage.

None.

## Logos and the label disagree (1)

Both were read and differ; the row carries Logos's reference.

- SOGOSP.1-34640: Logos says Matt.11.7; Matt.12.20, the label "Matthew 11:7" says Matt.11.7

## Rows with no ref (0)

Neither a Logos reference nor a label that could be read: the row is kept, found by its title and its source, and by no passage. Ids, in the file's order:

None.
