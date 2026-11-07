# sermon-outlines-on-romans-logos: what the import did not settle

<!-- toc: none -->

Written by `go run ./tools/logos-import --in <vault>/plans/sermons-2026-11-05/sermon-outlines-on-romans-sermons.jsonl --title-form "ref-in-brackets" --id sermon-outlines-on-romans-logos --out sources/sermon-outlines-on-romans-logos`; do not edit by hand. Only ids, titles, references and addresses are listed.

## Counts

```
lines read:                          28
rows written:                        28
lines dropped:                       0
rows with a ref:                     28
  of them read from the label:       0
  of them with cited references:     23
rows with no ref:                    0
series rows (ref over >3 chapters):  0
rows with their own date:            0
rows dated by the year alone:        0
references the book table refused:   1
Logos and the label disagree:        3
```

## Lines dropped (0)

A line with neither a ref nor a title, or with no id, no title, no address or an id already used, is not a row.

None.

## Titles that did not have the form (1)

--title-form "ref-in-brackets" reads the preacher, years, text or number printed in a title; these titles do not have it, so the row keeps the title and the preacher as the index gives them.

- SOROM.1-81744: "Pears or Prickle? (Romans 14:13–19 and Galatians 5:19–23)"

## References the book table refused (1)

A Logos reference to a book outside the sixty-six, or a chapter or verse past the end; it is left out of the row, and the row stays.

- SOROM.1-6791: "bible.66.2.17-66.2.21.a": want book.chapter or book.chapter.verse; "bible.66.2.21.b-66.2.24": want book.chapter or book.chapter.verse

## Series rows (0)

The ref spans more than three chapters (a whole book or a long text): the row is kept, found by its title and listed by its source, and matched by no passage.

None.

## Logos and the label disagree (3)

Both were read and differ; the row carries Logos's reference.

- SOROM.1-45842: Logos says Rom.10.4; Rom.10.10-Rom.10.15, the label "Romans 10:4" says Rom.10.4
- SOROM.1-54915: Logos says Rom.12.1-Rom.12.2; Rom.12.9-Rom.12.21, the label "Romans 12:1–2" says Rom.12.1-Rom.12.2
- SOROM.1-81744: Logos says Rom.14.13-Rom.14.19; Gal.5.19-Gal.5.23, the label "Romans 14:13–19" says Rom.14.13-Rom.14.19

## Rows with no ref (0)

Neither a Logos reference nor a label that could be read: the row is kept, found by its title and its source, and by no passage. Ids, in the file's order:

None.
