# wi-evans-logos: what the import did not settle

<!-- toc: none -->

Written by `go run ./tools/logos-import --in <vault>/plans/sermons-2026-11-05/this-river-must-flow-sermons.jsonl --id wi-evans-logos --out sources/wi-evans-logos`; do not edit by hand. Only ids, titles, references and addresses are listed.

## Counts

```
lines read:                          10
rows written:                        10
lines dropped:                       0
rows with a ref:                     1
  of them read from the label:       0
  of them with cited references:     8
rows with no ref:                    9
series rows (ref over >3 chapters):  0
rows with their own date:            0
rows dated by the year alone:        0
references the book table refused:   0
Logos and the label disagree:        0
titles made readable:                10
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

- THSRVRMSTFLW.1-13372

## Rows with no ref (9)

Neither a Logos reference nor a label that could be read: the row is kept, found by its title and its source, and by no passage. Ids, in the file's order:

THSRVRMSTFLW.1-506, THSRVRMSTFLW.1-13372, THSRVRMSTFLW.1-27261, THSRVRMSTFLW.1-43383, THSRVRMSTFLW.1-54722, THSRVRMSTFLW.1-69518, THSRVRMSTFLW.1-83354, THSRVRMSTFLW.1-91823, THSRVRMSTFLW.1-106740
