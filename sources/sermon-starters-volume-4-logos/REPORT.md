# sermon-starters-volume-4-logos: what the import did not settle

<!-- toc: none -->

Written by `go run ./tools/logos-import --in <vault>/plans/sermons-2026-11-05/sermon-starters-volume-4-sermons.jsonl --title-form "ref-in-brackets" --id sermon-starters-volume-4-logos --out sources/sermon-starters-volume-4-logos`; do not edit by hand. Only ids, titles, references and addresses are listed.

## Counts

```
lines read:                          203
rows written:                        198
lines dropped:                       0
non-sermon lines dropped:            5
rows with a ref:                     198
  of them read from the label:       0
  of them with cited references:     16
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

## Non-sermon lines dropped (5)

The archive's index lists them with the sermons, but they are a table, an index of Scripture, an introduction or a book's argument (prolegomena), not a sermon: not written, and not counted in "rows before".

- SERMONSTARTERS04.3-283057: "First—The Purpose in the Message" (a numbered point of an outline, not an outline)
- SERMONSTARTERS04.3-283799: "Second—The Precept in the Message" (a numbered point of an outline, not an outline)
- SERMONSTARTERS04.3-284308: "Third—The Person Ordering the Message" (a numbered point of an outline, not an outline)
- SERMONSTARTERS04.3-284574: "Fourth—The Particulars of the Message" (a numbered point of an outline, not an outline)
- SERMONSTARTERS04.1-568465: "Scripture Index" (an index)

## Titles that did not have the form (0)

--title-form "ref-in-brackets" reads the preacher, years, text or number printed in a title; these titles do not have it, so the row keeps the title and the preacher as the index gives them.

None.

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
