# maclaren-logos: what the import did not settle

<!-- toc: none -->

Written by `go run ./tools/logos-import --in <vault>/plans/sermons-2026-11-05/the-beatitudes-and-other-sermons-sermons.jsonl --infer-text --id maclaren-logos --out sources/maclaren-logos`; do not edit by hand. Only ids, titles, references and addresses are listed.

## Counts

```
lines read:                          30
rows written:                        30
lines dropped:                       0
rows with a ref:                     20
  of them read from the label:       0
  of them with cited references:     10
  of them an inferred text (cited):  20
rows with no ref:                    10
series rows (ref over >3 chapters):  0
rows with their own date:            0
rows dated by the year alone:        0
references the book table refused:   0
Logos and the label disagree:        0
titles made readable:                30
contents entries split in parts:     30
```

## Lines dropped (0)

A line with neither a ref nor a title, or with no id, no title, no address or an id already used, is not a row.

None.

## References the book table refused (0)

A Logos reference to a book outside the sixty-six, or a chapter or verse past the end; it is left out of the row, and the row stays.

None.

## Rows given an inferred text (20)

The archive's index names no text for these. Each takes the first passage it cites as its text (the only passage it cites, a passage its title names, or the passage the sermons beside it are on); the row's notes say it is inferred, and the passage is the row's ref and no longer among its cited ones. A sermon that cites several passages with none of these has no text.

- BTTDSTHRSRMNS.1-4253: Matt.5.3 (it is the only passage the sermon cites)
- BTTDSTHRSRMNS.1-19681: Matt.5.4 (it is the only passage the sermon cites)
- BTTDSTHRSRMNS.1-33462: Matt.5.5 (it is the only passage the sermon cites)
- BTTDSTHRSRMNS.1-48736: Matt.5.6 (it is the only passage the sermon cites)
- BTTDSTHRSRMNS.1-63600: Matt.5.7 (it is the only passage the sermon cites)
- BTTDSTHRSRMNS.1-79870: Matt.5.8 (it is the only passage the sermon cites)
- BTTDSTHRSRMNS.1-94085: Matt.5.9 (it is the only passage the sermon cites)
- BTTDSTHRSRMNS.1-110326: John.3.2 (it is the only passage the sermon cites)
- BTTDSTHRSRMNS.1-128457: Ezek.12.27 (it is the only passage the sermon cites)
- BTTDSTHRSRMNS.1-159681: 1Cor.7.22 (it is the only passage the sermon cites)
- BTTDSTHRSRMNS.1-207397: Gen.17.18 (it is the only passage the sermon cites)
- BTTDSTHRSRMNS.1-233432: Acts.3.26 (it is the only passage the sermon cites)
- BTTDSTHRSRMNS.1-248914: Acts.3.16 (it is the only passage the sermon cites)
- BTTDSTHRSRMNS.1-264361: 1Pet.1.7 (it is the only passage the sermon cites)
- BTTDSTHRSRMNS.1-281860: Acts.16.10 (it is the only passage the sermon cites)
- BTTDSTHRSRMNS.1-295942: Judg.5.31 (it is the only passage the sermon cites)
- BTTDSTHRSRMNS.1-326026: 2Sam.14.14 (it is the only passage the sermon cites)
- BTTDSTHRSRMNS.1-375576: Dan.12.13 (it is the only passage the sermon cites)
- BTTDSTHRSRMNS.1-416150: Prov.27.3 (it is the only passage the sermon cites)
- BTTDSTHRSRMNS.1-432427: Luke.23.9 (it is the only passage the sermon cites)

## Series rows (0)

The ref spans more than three chapters (a whole book or a long text): the row is kept, found by its title and listed by its source, and matched by no passage.

None.

## Logos and the label disagree (0)

Both were read and differ; the row carries Logos's reference.

None.

## Contents entries split in parts (30)

The book's contents list splits the entry into parts, so the link opens the book where the entry begins (often a heading, with the sermon in the parts after it).

- BTTDSTHRSRMNS.1-4253
- BTTDSTHRSRMNS.1-19681
- BTTDSTHRSRMNS.1-33462
- BTTDSTHRSRMNS.1-48736
- BTTDSTHRSRMNS.1-63600
- BTTDSTHRSRMNS.1-79870
- BTTDSTHRSRMNS.1-94085
- BTTDSTHRSRMNS.1-110326
- BTTDSTHRSRMNS.1-128457
- BTTDSTHRSRMNS.1-144838
- BTTDSTHRSRMNS.1-159681
- BTTDSTHRSRMNS.1-175586
- BTTDSTHRSRMNS.1-190482
- BTTDSTHRSRMNS.1-207397
- BTTDSTHRSRMNS.1-216939
- BTTDSTHRSRMNS.1-233432
- BTTDSTHRSRMNS.1-248914
- BTTDSTHRSRMNS.1-264361
- BTTDSTHRSRMNS.1-281860
- BTTDSTHRSRMNS.1-295942
- BTTDSTHRSRMNS.1-308860
- BTTDSTHRSRMNS.1-326026
- BTTDSTHRSRMNS.1-343171
- BTTDSTHRSRMNS.1-360606
- BTTDSTHRSRMNS.1-375576
- BTTDSTHRSRMNS.1-391620
- BTTDSTHRSRMNS.1-403351
- BTTDSTHRSRMNS.1-416150
- BTTDSTHRSRMNS.1-432427
- BTTDSTHRSRMNS.1-448234

## Rows with no ref (10)

Neither a Logos reference nor a label that could be read: the row is kept, found by its title and its source, and by no passage. Ids, in the file's order:

BTTDSTHRSRMNS.1-144838, BTTDSTHRSRMNS.1-175586, BTTDSTHRSRMNS.1-190482, BTTDSTHRSRMNS.1-216939, BTTDSTHRSRMNS.1-308860, BTTDSTHRSRMNS.1-343171, BTTDSTHRSRMNS.1-360606, BTTDSTHRSRMNS.1-391620, BTTDSTHRSRMNS.1-403351, BTTDSTHRSRMNS.1-448234
