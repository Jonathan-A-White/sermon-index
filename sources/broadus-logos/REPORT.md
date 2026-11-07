# broadus-logos: what the import did not settle

<!-- toc: none -->

Written by `go run ./tools/logos-import --in <vault>/plans/sermons-2026-11-05/sermons-and-addresses-sermons.jsonl --infer-text --id broadus-logos --out sources/broadus-logos`; do not edit by hand. Only ids, titles, references and addresses are listed.

## Counts

```
lines read:                          21
rows written:                        21
lines dropped:                       0
rows with a ref:                     13
  of them read from the label:       0
  of them with cited references:     2
  of them an inferred text (cited):  13
rows with no ref:                    8
series rows (ref over >3 chapters):  0
rows with their own date:            0
rows dated by the year alone:        0
references the book table refused:   0
Logos and the label disagree:        0
titles made readable:                21
contents entries split in parts:     1
```

## Lines dropped (0)

A line with neither a ref nor a title, or with no id, no title, no address or an id already used, is not a row.

None.

## References the book table refused (0)

A Logos reference to a book outside the sixty-six, or a chapter or verse past the end; it is left out of the row, and the row stays.

None.

## Rows given an inferred text (13)

The archive's index names no text for these. Each takes the first passage it cites as its text (the only passage it cites, a passage its title names, or the passage the sermons beside it are on); the row's notes say it is inferred, and the passage is the row's ref and no longer among its cited ones. A sermon that cites several passages with none of these has no text.

- JABSERMADDR.1-8364: John.4.24 (it is the only passage the sermon cites)
- JABSERMADDR.1-48146: John.4.32-John.4.38 (it is the only passage the sermon cites)
- JABSERMADDR.1-77481: 1Thess.5.18 (it is the only passage the sermon cites)
- JABSERMADDR.1-95620: Matt.7.7 (it is the only passage the sermon cites)
- JABSERMADDR.1-115747: Heb.7.25 (it is the only passage the sermon cites)
- JABSERMADDR.1-139716: Rom.5.1 (it is the only passage the sermon cites)
- JABSERMADDR.1-179591: Rom.9.3 (it is the only passage the sermon cites)
- JABSERMADDR.1-200965: Acts.1.14 (it is the only passage the sermon cites)
- JABSERMADDR.1-223894: Eph.3.8 (it is the only passage the sermon cites)
- JABSERMADDR.1-248105: 2Tim.3.15 (it is the only passage the sermon cites)
- JABSERMADDR.1-316930: 2Tim.2.14 (it is the only passage the sermon cites)
- JABSERMADDR.1-563313: Rom.14.7-Rom.14.8 (it is the only passage the sermon cites)
- JABSERMADDR.1-596175: Prov.22.6 (it is the only passage the sermon cites)

## Series rows (0)

The ref spans more than three chapters (a whole book or a long text): the row is kept, found by its title and listed by its source, and matched by no passage.

None.

## Logos and the label disagree (0)

Both were read and differ; the row carries Logos's reference.

None.

## Contents entries split in parts (1)

The book's contents list splits the entry into parts, so the link opens the book where the entry begins (often a heading, with the sermon in the parts after it).

- JABSERMADDR.1-397727

## Rows with no ref (8)

Neither a Logos reference nor a label that could be read: the row is kept, found by its title and its source, and by no passage. Ids, in the file's order:

JABSERMADDR.1-158993, JABSERMADDR.1-266843, JABSERMADDR.1-346070, JABSERMADDR.1-397727, JABSERMADDR.1-428569, JABSERMADDR.1-485485, JABSERMADDR.1-558782, JABSERMADDR.1-588779
