# bunyan-logos: what the import did not settle

<!-- toc: none -->

Written by `go run ./tools/logos-import --in <vault>/plans/sermons-2026-11-05/bunyan-s-last-sermon-sermons.jsonl --id bunyan-logos --out sources/bunyan-logos`; do not edit by hand. Only ids, titles, references and addresses are listed.

## Counts

```
lines read:                          1
rows written:                        1
lines dropped:                       0
rows with a ref:                     0
  of them read from the label:       0
  of them with cited references:     1
rows with no ref:                    1
series rows (ref over >3 chapters):  0
rows with their own date:            0
rows dated by the year alone:        0
references the book table refused:   0
Logos and the label disagree:        0
contents entries split in parts:     1
```

## Lines dropped (0)

A line with neither a ref nor a title, or with no id, no title, no address or an id already used, is not a row.

None.

## References the book table refused (0)

A Logos reference to a book outside the sixty-six, or a chapter or verse past the end; it is left out of the row, and the row stays.

None.

## Series rows (0)

The ref spans more than three chapters (a whole book or a long text): the row is kept, found by its title and listed by its source, and matched by no passage.

None.

## Logos and the label disagree (0)

Both were read and differ; the row carries Logos's reference.

None.

## Contents entries split in parts (1)

The book's contents list splits the entry into parts, so the link opens the book where the entry begins (often a heading, with the sermon in the parts after it).

- WJB_BYNLST.1-1758

## Rows with no ref (1)

Neither a Logos reference nor a label that could be read: the row is kept, found by its title and its source, and by no passage. Ids, in the file's order:

WJB_BYNLST.1-1758

## No row has a Scripture ref

None of the 1 rows has a Scripture ref (the archive's index names no text for any of them; any dates and series it has are kept, and the 1 that cite passages cite them without taking one as the text): none can show on a verse, and all are found by title, preacher and source. A row the index dates by no day is listed here.

### Rows with no date of their own (1)

WJB_BYNLST.1-1758
