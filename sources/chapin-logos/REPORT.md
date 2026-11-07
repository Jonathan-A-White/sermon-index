# chapin-logos: what the import did not settle

<!-- toc: none -->

Written by `go run ./tools/logos-import --in <vault>/plans/sermons-2026-11-05/discourses-on-the-beatitudes-sermons.jsonl --id chapin-logos --out sources/chapin-logos`; do not edit by hand. Only ids, titles, references and addresses are listed.

## Counts

```
lines read:                          8
rows written:                        8
lines dropped:                       0
rows with a ref:                     0
  of them read from the label:       0
  of them with cited references:     8
rows with no ref:                    8
series rows (ref over >3 chapters):  0
rows with their own date:            0
rows dated by the year alone:        0
references the book table refused:   0
Logos and the label disagree:        0
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

## Rows with no ref (8)

Neither a Logos reference nor a label that could be read: the row is kept, found by its title and its source, and by no passage. Ids, in the file's order:

DSCRSSBTTDS.1-2048, DSCRSSBTTDS.1-20293, DSCRSSBTTDS.1-38554, DSCRSSBTTDS.1-57395, DSCRSSBTTDS.1-75111, DSCRSSBTTDS.1-93350, DSCRSSBTTDS.1-111928, DSCRSSBTTDS.1-131074

## No row has a Scripture ref

None of the 8 rows has a Scripture ref (the archive's index names no text for any of them; any dates and series it has are kept, and the 8 that cite passages cite them without taking one as the text): none can show on a verse, and all are found by title, preacher and source. A row the index dates by no day is listed here.

### Rows with no date of their own (8)

DSCRSSBTTDS.1-2048, DSCRSSBTTDS.1-20293, DSCRSSBTTDS.1-38554, DSCRSSBTTDS.1-57395, DSCRSSBTTDS.1-75111, DSCRSSBTTDS.1-93350, DSCRSSBTTDS.1-111928, DSCRSSBTTDS.1-131074
