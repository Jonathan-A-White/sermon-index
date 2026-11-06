# shedd-logos: what the import did not settle

<!-- toc: none -->

Written by `go run ./tools/logos-import --in <vault>/plans/sermons-2026-11-05/sermons-to-the-natural-man-sermons.jsonl --in <vault>/plans/sermons-2026-11-05/sermons-to-the-spiritual-man-sermons.jsonl --id shedd-logos --out sources/shedd-logos`; do not edit by hand. Only ids, titles, references and addresses are listed.

## Counts

```
lines read:                          46
rows written:                        46
lines dropped:                       0
rows with a ref:                     46
  of them read from the label:       0
  of them with cited references:     29
rows with no ref:                    0
series rows (ref over >3 chapters):  0
rows with their own date:            0
rows dated by the year alone:        0
references the book table refused:   0
Logos and the label disagree:        4
titles made readable:                20
```

## Rows before and after, by input

| input | lines | rows before | duplicates dropped | rows after |
|---|---|---|---|---|
| sermons-to-the-natural-man | 20 | 20 | 0 | 20 |
| sermons-to-the-spiritual-man | 26 | 26 | 0 | 26 |
| **all** | 46 | 46 | 0 | 46 |

## Lines dropped (0)

A line with neither a ref nor a title, or with no id, no title, no address or an id already used, is not a row.

None.

## References the book table refused (0)

A Logos reference to a book outside the sixty-six, or a chapter or verse past the end; it is left out of the row, and the row stays.

None.

## Series rows (0)

The ref spans more than three chapters (a whole book or a long text): the row is kept, found by its title and listed by its source, and matched by no passage.

None.

## Logos and the label disagree (4)

Both were read and differ; the row carries Logos's reference.

- SRMNATMAN.1-468419: Logos says Prov.9.10; Luke.12.4; Luke.12.5, the label "Proverbs 9:10" says Prov.9.10
- SRMNATMAN.1-594717: Logos says John.6.28; John.6.29, the label "John 6:28" says John.6.28
- SRMSPRTMAN.1-72445: Logos says Matt.19.16; Matt.19.17, the label "Matthew 19:16" says Matt.19.16
- SRMSPRTMAN.1-721536: Logos says Isa.55.10; Isa.55.11, the label "Isaiah 55:10" says Isa.55.10

## Rows with no ref (0)

Neither a Logos reference nor a label that could be read: the row is kept, found by its title and its source, and by no passage. Ids, in the file's order:

None.
