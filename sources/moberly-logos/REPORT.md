# moberly-logos: what the import did not settle

<!-- toc: none -->

Written by `go run ./tools/logos-import --in <vault>/plans/sermons-2026-11-05/sermons-on-the-beatitudes-with-others-mostly-preached-before-the-university-of-oxford-to-which-is-added-a-preface-relating-to-the-recent-volume-of-essays-and-reviews-sermons.jsonl --title-from-ref --id moberly-logos --out sources/moberly-logos`; do not edit by hand. Only ids, titles, references and addresses are listed.

## Counts

```
lines read:                          15
rows written:                        15
lines dropped:                       0
rows with a ref:                     15
  of them read from the label:       0
  of them with cited references:     5
rows with no ref:                    0
series rows (ref over >3 chapters):  0
rows with their own date:            0
rows dated by the year alone:        0
references the book table refused:   0
Logos and the label disagree:        2
titles that took their ref:          15
```

## Lines dropped (0)

A line with neither a ref nor a title, or with no id, no title, no address or an id already used, is not a row.

None.

## Bare sermon numbers with no ref (0)

--title-from-ref adds the text to a title that is only "Sermon N"; these have no ref to add, so the row keeps the title as it is.

None.

## References the book table refused (0)

A Logos reference to a book outside the sixty-six, or a chapter or verse past the end; it is left out of the row, and the row stays.

None.

## Series rows (0)

The ref spans more than three chapters (a whole book or a long text): the row is kept, found by its title and listed by its source, and matched by no passage.

None.

## Logos and the label disagree (2)

Both were read and differ; the row carries Logos's reference.

- SRMNSBTTD.1-469088: Logos says Rev.5.9; Rev.5.10, the label "REVELATION 5:9" says Rev.5.9
- SRMNSBTTD.1-489559: Logos says Rev.5.9; Rev.5.10, the label "REVELATION 5:9" says Rev.5.9

## Rows with no ref (0)

Neither a Logos reference nor a label that could be read: the row is kept, found by its title and its source, and by no passage. Ids, in the file's order:

None.
