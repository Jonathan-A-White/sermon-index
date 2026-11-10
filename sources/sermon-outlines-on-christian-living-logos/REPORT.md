# sermon-outlines-on-christian-living-logos: what the import did not settle

<!-- toc: none -->

Written by `go run ./tools/logos-import --in <vault>/plans/sermons-2026-11-05/sermon-outlines-on-christian-living-sermons.jsonl --title-form "ref-in-brackets" --id sermon-outlines-on-christian-living-logos --out sources/sermon-outlines-on-christian-living-logos`; do not edit by hand. Only ids, titles, references and addresses are listed.

## Counts

```
lines read:                          30
rows written:                        30
lines dropped:                       0
rows with a ref:                     30
  of them read from the label:       1
  of them read from the title:       3
  of them with cited references:     24
rows with no ref:                    0
series rows (ref over >3 chapters):  0
rows with their own date:            0
rows dated by the year alone:        0
references the book table refused:   1
Logos and the label disagree:        6
```

## Lines dropped (0)

A line with neither a ref nor a title, or with no id, no title, no address or an id already used, is not a row.

None.

## Titles that did not have the form (0)

--title-form "ref-in-brackets" reads the preacher, years, text or number printed in a title; these titles do not have it, so the row keeps the title and the preacher as the index gives them.

None.

## References the book table refused (1)

A Logos reference to a book outside the sixty-six, or a chapter or verse past the end; it is left out of the row, and the row stays.

- SOCHRLIV.1-25496: "bible.82.2.1-82.2.10.a": want book.chapter or book.chapter.verse

## Series rows (0)

The ref spans more than three chapters (a whole book or a long text): the row is kept, found by its title and listed by its source, and matched by no passage.

None.

## Logos and the label disagree (6)

Both were read and differ; the row carries Logos's reference.

- SOCHRLIV.1-7592: Logos says Phil.1.2; Phil.1.12-Phil.1.30; Gal.2.11-Gal.2.21, the label "Philippians 1:2" says Phil.1.2
- SOCHRLIV.1-9555: Logos says Luke.17.20; Luke.17.21; Matt.25.14-Matt.25.30, the label "Luke 17:20" says Luke.17.20
- SOCHRLIV.1-11916: Logos says 1Cor.12.31; 1Cor.13.13; Jas.2.8, the label "1 Corinthians 12:31" says 1Cor.12.31
- SOCHRLIV.1-23196: Logos says Eph.4.31; Eph.4.32; Matt.18.15-Matt.18.35, the label "Ephesians 4:31" says Eph.4.31
- SOCHRLIV.1-40452: Logos says Matt.6.5-Matt.6.15; Matt.7.7-Matt.7.12, the label "Matthew 6:5–15" says Matt.6.5-Matt.6.15
- SOCHRLIV.1-46751: Logos says Acts.2.36-Acts.2.47; Acts.4.32-Acts.4.35, the label "Acts 2:36–47" says Acts.2.36-Acts.2.47

## Rows with no ref (0)

Neither a Logos reference nor a label that could be read: the row is kept, found by its title and its source, and by no passage. Ids, in the file's order:

None.
