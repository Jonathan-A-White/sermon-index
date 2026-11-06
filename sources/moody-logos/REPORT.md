# moody-logos: what the import did not settle

<!-- toc: none -->

Written by `go run ./tools/logos-import --in <vault>/plans/sermons-2026-11-05/men-of-the-bible-sermons.jsonl --in <vault>/plans/sermons-2026-11-05/the-overcoming-life-and-other-sermons-sermons.jsonl --in <vault>/plans/sermons-2026-11-05/the-way-to-god-and-how-to-find-it-sermons.jsonl --in <vault>/plans/sermons-2026-11-05/twelve-select-sermons-sermons.jsonl --id moody-logos --out sources/moody-logos`; do not edit by hand. Only ids, titles, references and addresses are listed.

## Counts

```
lines read:                          32
rows written:                        32
lines dropped:                       0
rows with a ref:                     8
  of them read from the label:       0
  of them with cited references:     17
rows with no ref:                    24
series rows (ref over >3 chapters):  0
rows with their own date:            0
rows dated by the year alone:        0
references the book table refused:   0
Logos and the label disagree:        4
titles made readable:                9
contents entries split in parts:     10
```

## Rows before and after, by input

| input | lines | rows before | duplicates dropped | rows after |
|---|---|---|---|---|
| men-of-the-bible | 7 | 7 | 0 | 7 |
| the-overcoming-life-and-other-sermons | 7 | 7 | 0 | 7 |
| the-way-to-god-and-how-to-find-it | 9 | 9 | 0 | 9 |
| twelve-select-sermons | 9 | 9 | 0 | 9 |
| **all** | 32 | 32 | 0 | 32 |

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

- WAYGODMOODY.1-1639: Logos says Eph.3.19, the label "Ephesians" says Eph
- WAYGODMOODY.1-29140: Logos says John.3.3, the label "John" says John
- WAYGODMOODY.1-100436: Logos says Matt.16.1; John.6.69, the label "Matthew" says Matt
- WAYGODMOODY.1-166656: Logos says Col.3.11, the label "Colossians" says Col

## Contents entries split in parts (10)

The book's contents list splits the entry into parts, so the link opens the book where the entry begins (often a heading, with the sermon in the parts after it).

- MENBBLMOODY.1-401
- MENBBLMOODY.1-125566
- OVRCMNGLFMOODY.1-485
- OVRCMNGLFMOODY.1-52133
- OVRCMNGLFMOODY.1-93341
- OVRCMNGLFMOODY.1-135330
- OVRCMNGLFMOODY.1-156862
- 12SELSERMMOODY.1-163809
- 12SELSERMMOODY.1-215700
- 12SELSERMMOODY.1-260215

## Rows with no ref (24)

Neither a Logos reference nor a label that could be read: the row is kept, found by its title and its source, and by no passage. Ids, in the file's order:

MENBBLMOODY.1-401, MENBBLMOODY.1-27930, MENBBLMOODY.1-42590, MENBBLMOODY.1-72239, MENBBLMOODY.1-98793, MENBBLMOODY.1-125566, MENBBLMOODY.1-155943, OVRCMNGLFMOODY.1-485, OVRCMNGLFMOODY.1-52133, OVRCMNGLFMOODY.1-71840, OVRCMNGLFMOODY.1-93341, OVRCMNGLFMOODY.1-118282, OVRCMNGLFMOODY.1-135330, OVRCMNGLFMOODY.1-156862, WAYGODMOODY.1-62478, WAYGODMOODY.1-83618, WAYGODMOODY.1-112517, WAYGODMOODY.1-189373, 12SELSERMMOODY.1-64431, 12SELSERMMOODY.1-90685, 12SELSERMMOODY.1-122889, 12SELSERMMOODY.1-163809, 12SELSERMMOODY.1-215700, 12SELSERMMOODY.1-260215
