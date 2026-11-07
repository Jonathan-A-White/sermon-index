# princeton-sermons-logos: what the import did not settle

<!-- toc: none -->

Written by `go run ./tools/logos-import --in <vault>/plans/sermons-2026-11-05/princeton-sermons-sermons.jsonl --title-form "by-preacher-ref" --id princeton-sermons-logos --out sources/princeton-sermons-logos`; do not edit by hand. Only ids, titles, references and addresses are listed.

## Counts

```
lines read:                          16
rows written:                        16
lines dropped:                       0
rows with a ref:                     16
  of them read from the label:       0
  of them read from the title:       16
  of them with cited references:     8
rows with no ref:                    0
series rows (ref over >3 chapters):  0
rows with their own date:            0
rows dated by the year alone:        0
references the book table refused:   0
Logos and the label disagree:        0
```

## Lines dropped (0)

A line with neither a ref nor a title, or with no id, no title, no address or an id already used, is not a row.

None.

## Titles that did not have the form (0)

--title-form "by-preacher-ref" reads the preacher, years, text or number printed in a title; these titles do not have it, so the row keeps the title and the preacher as the index gives them.

None.

## The title and Logos disagree (1)

The passage a title prints and the first passage Logos lists for the sermon differ; the row carries the title's, which is the book's own word for its text, and Logos's stays in the row among its cited passages.

- PRINCESERMON.1-170527: the title says John.2.13 ("John 2:13"), Logos's first cited reference says 1John.2.13

## References the book table refused (0)

A Logos reference to a book outside the sixty-six, or a chapter or verse past the end; it is left out of the row, and the row stays.

None.

## Series rows (0)

The ref spans more than three chapters (a whole book or a long text): the row is kept, found by its title and listed by its source, and matched by no passage.

None.

## Logos and the label disagree (0)

Both were read and differ; the row carries Logos's reference.

None.

## Rows with no ref (0)

Neither a Logos reference nor a label that could be read: the row is kept, found by its title and its source, and by no passage. Ids, in the file's order:

None.
