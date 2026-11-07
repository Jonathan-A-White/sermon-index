# burroughs-logos: what the import did not settle

<!-- toc: none -->

Written by `go run ./tools/logos-import --in <vault>/plans/sermons-2026-11-05/the-rare-jewel-of-christian-contentment-sermons.jsonl --in <vault>/plans/sermons-2026-11-05/the-saints-treasury-sermons.jsonl --in <vault>/plans/sermons-2026-11-05/the-saints-happiness-sermons.jsonl --in <vault>/plans/sermons-2026-11-05/the-difference-between-the-spots-of-the-godly-and-of-the-wicked-sermons.jsonl --id burroughs-logos --out sources/burroughs-logos`; do not edit by hand. Only ids, titles, references and addresses are listed.

## Counts

```
lines read:                          67
rows written:                        62
lines dropped:                       0
non-sermon lines dropped:            5
rows with a ref:                     57
  of them read from the label:       0
  of them with cited references:     60
rows with no ref:                    5
series rows (ref over >3 chapters):  0
rows with their own date:            4
  of them dates read from the title: 4
rows dated by the year alone:        0
references the book table refused:   0
Logos and the label disagree:        4
contents entries split in parts:     4
```

## Rows before and after, by input

| input | lines | rows before | duplicates dropped | rows after |
|---|---|---|---|---|
| the-rare-jewel-of-christian-contentment | 16 | 12 | 0 | 12 |
| the-saints-treasury | 6 | 5 | 0 | 5 |
| the-saints-happiness | 41 | 41 | 0 | 41 |
| the-difference-between-the-spots-of-the-godly-and-of-the-wicked | 4 | 4 | 0 | 4 |
| **all** | 67 | 62 | 0 | 62 |

## Lines dropped (0)

A line with neither a ref nor a title, or with no id, no title, no address or an id already used, is not a row.

None.

## Non-sermon lines dropped (5)

The archive's index lists them with the sermons, but they are a table, an index of Scripture, an introduction or a book's argument (prolegomena), not a sermon: not written, and not counted in "rows before".

- WRKSBURROUGHS01.1-4836: "The Contents of the Ensuing Treatise of Contentment" (a table of contents)
- WRKSBURROUGHS01.1-14304: "The Contents of the Last Sermon, on Exodus, 14:13" (a table of contents)
- WRKSBURROUGHS01.1-547273: "An Exact Alphabetical Table of the Foregoing Treatise of Contentment" (a table of contents)
- WRKSBURROUGHS01.1-563210: "An Exact Alphabetical Table of the Last Sermon on Exodus 13:14" (a table of contents)
- WRKSBURROUGHS02.1-290493: "The Contents of the Foregoing Sermons" (a table of contents)

## References the book table refused (0)

A Logos reference to a book outside the sixty-six, or a chapter or verse past the end; it is left out of the row, and the row stays.

None.

## Series rows (0)

The ref spans more than three chapters (a whole book or a long text): the row is kept, found by its title and listed by its source, and matched by no passage.

None.

## Logos and the label disagree (4)

Both were read and differ; the row carries Logos's reference.

- WRKSBURROUGHS01.1-478132: Logos says Exod.14; Exod.14.13, the label "Exodus 14" says Exod.14
- WRKSBURROUGHS04.1-620841: Logos says Ps.5.6, the label "Math . 5:6" says Matt.5.6
- WRKSBURROUGHS04.1-1285739: Logos says Matt.5; Matt.5.12, the label "Math . 5" says Matt.5
- WRKSBURROUGHS04.1-1369924: Logos says Matt.5.14; Matt.5.15, the label "Math . 5:14" says Matt.5.14

## Contents entries split in parts (4)

The book's contents list splits the entry into parts, so the link opens the book where the entry begins (often a heading, with the sermon in the parts after it).

- WRKSBURROUGHS01.1-354513
- WRKSBURROUGHS01.1-395931
- WRKSBURROUGHS01.1-431135
- WRKSBURROUGHS07.1-109402

## Rows with no ref (5)

Neither a Logos reference nor a label that could be read: the row is kept, found by its title and its source, and by no passage. Ids, in the file's order:

WRKSBURROUGHS01.1-15584, WRKSBURROUGHS04.1-46109, WRKSBURROUGHS07.1-34656, WRKSBURROUGHS07.1-74468, WRKSBURROUGHS07.1-109402
