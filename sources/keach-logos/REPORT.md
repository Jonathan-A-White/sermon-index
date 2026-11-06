# keach-logos: what the import did not settle

<!-- toc: none -->

Written by `go run ./tools/logos-import --in <vault>/plans/sermons-2026-11-05/the-display-of-glorious-grace-or-the-covenant-of-peace-opened-in-fourteen-sermons-sermons.jsonl --in <vault>/plans/sermons-2026-11-05/the-ax-laid-to-the-root-parts-i-ii-sermons.jsonl --id keach-logos --out sources/keach-logos`; do not edit by hand. Only ids, titles, references and addresses are listed.

## Counts

```
lines read:                          19
rows written:                        17
lines dropped:                       0
non-sermon lines dropped:            2
rows with a ref:                     17
  of them read from the label:       0
  of them with cited references:     5
rows with no ref:                    0
series rows (ref over >3 chapters):  0
rows with their own date:            0
rows dated by the year alone:        0
references the book table refused:   0
Logos and the label disagree:        0
contents entries split in parts:     14
```

## Rows before and after, by input

| input | lines | rows before | duplicates dropped | rows after |
|---|---|---|---|---|
| the-display-of-glorious-grace-or-the-covenant-of-peace-opened-in-fourteen-sermons | 15 | 14 | 0 | 14 |
| the-ax-laid-to-the-root-parts-i-ii | 4 | 3 | 0 | 3 |
| **all** | 19 | 17 | 0 | 17 |

## Lines dropped (0)

A line with neither a ref nor a title, or with no id, no title, no address or an id already used, is not a row.

None.

## Non-sermon lines dropped (2)

The archive's index lists them with the sermons, but they are a table, an index of Scripture, an introduction or a book's argument (prolegomena), not a sermon: not written, and not counted in "rows before".

- BPTSTCVNTTHLGY03.1-16782: "The Table of the Chief Heads Contained in This Book" (a table of contents)
- BPTSTCVNTTHLGYAX.2-1578: "The Epistle to the Reader" (front matter)

## References the book table refused (0)

A Logos reference to a book outside the sixty-six, or a chapter or verse past the end; it is left out of the row, and the row stays.

None.

## Series rows (0)

The ref spans more than three chapters (a whole book or a long text): the row is kept, found by its title and listed by its source, and matched by no passage.

None.

## Logos and the label disagree (0)

Both were read and differ; the row carries Logos's reference.

None.

## Contents entries split in parts (14)

The book's contents list splits the entry into parts, so the link opens the book where the entry begins (often a heading, with the sermon in the parts after it).

- BPTSTCVNTTHLGY03.1-26181
- BPTSTCVNTTHLGY03.1-57579
- BPTSTCVNTTHLGY03.1-106302
- BPTSTCVNTTHLGY03.1-147631
- BPTSTCVNTTHLGY03.1-178799
- BPTSTCVNTTHLGY03.1-209423
- BPTSTCVNTTHLGY03.1-237819
- BPTSTCVNTTHLGY03.1-264518
- BPTSTCVNTTHLGY03.1-290472
- BPTSTCVNTTHLGY03.1-312359
- BPTSTCVNTTHLGY03.1-338210
- BPTSTCVNTTHLGY03.1-361163
- BPTSTCVNTTHLGY03.1-386735
- BPTSTCVNTTHLGY03.1-419509

## Rows with no ref (0)

Neither a Logos reference nor a label that could be read: the row is kept, found by its title and its source, and by no passage. Ids, in the file's order:

None.
