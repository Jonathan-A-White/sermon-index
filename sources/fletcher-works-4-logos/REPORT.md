# fletcher-works-4-logos: what the import did not settle

<!-- toc: none -->

Written by `go run ./tools/logos-import --in <vault>/plans/sermons-2026-11-05/the-works-of-the-reverend-john-fletcher-volume-4-sermons.jsonl --id fletcher-works-4-logos --out sources/fletcher-works-4-logos`; do not edit by hand. Only ids, titles, references and addresses are listed.

## Counts

```
lines read:                          41
rows written:                        41
lines dropped:                       0
rows with a ref:                     41
  of them read from the label:       41
  of them with cited references:     30
rows with no ref:                    0
series rows (ref over >3 chapters):  0
rows with their own date:            0
rows dated by the year alone:        0
references the book table refused:   1
Logos and the label disagree:        0
```

## Lines dropped (0)

A line with neither a ref nor a title, or with no id, no title, no address or an id already used, is not a row.

None.

## References the book table refused (1)

A Logos reference to a book outside the sixty-six, or a chapter or verse past the end; it is left out of the row, and the row stays.

- WORKSFLETCH04.2-651612: "bible.66.16.25": ref 'Rom.16.25': verse past the chapter

## Series rows (0)

The ref spans more than three chapters (a whole book or a long text): the row is kept, found by its title and listed by its source, and matched by no passage.

None.

## Logos and the label disagree (0)

Both were read and differ; the row carries Logos's reference.

None.

## Rows with no ref (0)

Neither a Logos reference nor a label that could be read: the row is kept, found by its title and its source, and by no passage. Ids, in the file's order:

None.
