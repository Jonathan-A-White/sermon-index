# jd-greear-logos: what the import did not settle

<!-- toc: none -->

Written by `go run ./tools/logos-import --in <vault>/plans/sermons-2026-11-05/j-d-greear-sermons.jsonl --id jd-greear-logos --out sources/jd-greear-logos`; do not edit by hand. Only ids, titles, references and addresses are listed.

## Counts

```
lines read:                          248
rows written:                        248
lines dropped:                       0
rows with a ref:                     209
  of them read from the label:       0
  of them with cited references:     18
rows with no ref:                    39
series rows (ref over >3 chapters):  3
rows with their own date:            212
rows dated by the year alone:        0
references the book table refused:   0
Logos and the label disagree:        1
```

## Lines dropped (0)

A line with neither a ref nor a title, or with no id, no title, no address or an id already used, is not a row.

None.

## References the book table refused (0)

A Logos reference to a book outside the sixty-six, or a chapter or verse past the end; it is left out of the row, and the row stays.

None.

## Series rows (3)

The ref spans more than three chapters (a whole book or a long text): the row is kept, found by its title and listed by its source, and matched by no passage.

- CH52 2Sam.12-2Sam.18 (7 chapters)
- CH191 2Chr.28-2Chr.31 (4 chapters)
- CH214 Judg.17-Judg.21 (5 chapters)

## Logos and the label disagree (1)

Both were read and differ; the row carries Logos's reference.

- CH17: Logos says Eph.4.1-Eph.4.16; 1Pet.1.22-1Pet.2.10, the label "Ephesians 4:1–16" says Eph.4.1-Eph.4.16

## Rows with no ref (39)

Neither a Logos reference nor a label that could be read: the row is kept, found by its title and its source, and by no passage. Ids, in the file's order:

PT1.1, PT1.2, PT1.3, PT2.1, PT2.2, PT2.4, PT2.6, PT2.7, PT3.2, PT3.3, PT3.5, PT3.6, PT4.1, PT4.3, PT4.6, PT4.7, PT4.8, PT4.10, PT5.2, PT5.3, PT5.6, PT5.7, PT5.8, PT5.9, PT6.3, PT6.5, PT6.6, PT6.7, PT7.1, PT7.2, PT7.3, CH87, CH93, CH95, CH98, CH129, CH197, CH204, CH210
