# edwards-works-2-logos: what the import did not settle

<!-- toc: none -->

Written by `go run ./tools/logos-import --in <vault>/plans/sermons-2026-11-05/the-works-of-jonathan-edwards-volume-2-sermons.jsonl --id edwards-works-2-logos --out sources/edwards-works-2-logos`; do not edit by hand. Only ids, titles, references and addresses are listed.

## Counts

```
lines read:                          63
rows written:                        63
lines dropped:                       0
rows with a ref:                     63
  of them read from the label:       0
  of them with cited references:     54
rows with no ref:                    0
series rows (ref over >3 chapters):  0
rows with their own date:            0
rows dated by the year alone:        0
references the book table refused:   0
Logos and the label disagree:        13
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

## Logos and the label disagree (13)

Both were read and differ; the row carries Logos's reference.

- WRKJONEDWRDS2.3-171527: Logos says 1Cor.1.29; 1Cor.1.30; 1Cor.1.31, the label "1 Cor. 1:29" says 1Cor.1.29
- WRKJONEDWRDS2.3-291068: Logos says Isa.62.4; Isa.62.5, the label "Isaiah 62:4" says Isa.62.4
- WRKJONEDWRDS2.3-904680: Logos says 1Cor.16.1; 1Cor.16.2, the label "1 Cor. 16:1" says 1Cor.16.1
- WRKJONEDWRDS2.3-930651: Logos says 1Cor.16.1; 1Cor.16.2, the label "1 Cor. 16:1" says 1Cor.16.1
- WRKJONEDWRDS2.3-967579: Logos says 1Cor.16.1; 1Cor.16.2, the label "1 Cor. 16:1" says 1Cor.16.1
- WRKJONEDWRDS2.2-1562574: Logos says Ps.139.23; Ps.139.24, the label "Psalm 139:23" says Ps.139.23
- WRKJONEDWRDS2.2-1662255: Logos says Ezek.23.37; Ezek.23.38; Ezek.23.39, the label "Ezek. 23:37" says Ezek.23.37
- WRKJONEDWRDS2.2-1885248: Logos says 1Cor.15.25; 1Cor.15.26, the label "1 Corinthians 15:25" says 1Cor.15.25
- WRKJONEDWRDS2.2-2127497: Logos says Heb.11.13; Heb.11.14, the label "Heb. 11:13" says Heb.11.13
- WRKJONEDWRDS2.2-6905928: Logos says Acts.16.29; Acts.16.30, the label "Acts 16:29" says Acts.16.29
- WRKJONEDWRDS2.2-7430601: Logos says Rom.2.8; Rom.2.9, the label "Romans 2:8" says Rom.2.8
- WRKJONEDWRDS2.2-7774186: Logos says Matt.11.16; Matt.11.17; Matt.11.18; Matt.11.19, the label "Matthew 11:16" says Matt.11.16
- WRKJONEDWRDS2.2-8137001: Logos says John.13.15; John.13.16, the label "John 13:15" says John.13.15

## Rows with no ref (0)

Neither a Logos reference nor a label that could be read: the row is kept, found by its title and its source, and by no passage. Ids, in the file's order:

None.
