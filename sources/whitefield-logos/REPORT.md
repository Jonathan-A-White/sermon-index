# whitefield-logos: what the import did not settle

<!-- toc: none -->

Written by `go run ./tools/logos-import --in <vault>/plans/sermons-2026-11-05/selected-sermons-of-george-whitefield-sermons.jsonl --id whitefield-logos --out sources/whitefield-logos`; do not edit by hand. Only ids, titles, references and addresses are listed.

## Counts

```
lines read:                          59
rows written:                        59
lines dropped:                       0
rows with a ref:                     47
  of them read from the label:       0
  of them with cited references:     30
rows with no ref:                    12
series rows (ref over >3 chapters):  0
rows with their own date:            0
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

## Series rows (0)

The ref spans more than three chapters (a whole book or a long text): the row is kept, found by its title and listed by its source, and matched by no passage.

None.

## Logos and the label disagree (1)

Both were read and differ; the row carries Logos's reference.

- 36.0.12.1-802127: Logos says Matt.25.13; Matt.25.1-Matt.25.13, the label "Matthew 25:13" says Matt.25.13

## Rows with no ref (12)

Neither a Logos reference nor a label that could be read: the row is kept, found by its title and its source, and by no passage. Ids, in the file's order:

36.0.12.1-143501, 36.0.12.1-205010, 36.0.12.1-343941, 36.0.12.1-1415469, 36.0.12.1-1450347, 36.0.12.1-1469574, 36.0.12.1-1518046, 36.0.12.1-1545938, 36.0.12.1-1573212, 36.0.12.1-1600092, 36.0.12.1-1674209, 36.0.12.1-1697785
