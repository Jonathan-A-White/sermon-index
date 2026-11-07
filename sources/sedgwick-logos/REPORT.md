# sedgwick-logos: what the import did not settle

<!-- toc: none -->

Written by `go run ./tools/logos-import --in <vault>/plans/sermons-2026-11-05/a-thanksgiving-sermon-preached-before-the-honourable-house-of-commons-at-westminster-april-9-1644-sermons.jsonl --in <vault>/plans/sermons-2026-11-05/an-arke-against-a-deluge-sermons.jsonl --in <vault>/plans/sermons-2026-11-05/elisha-his-lamentation-upon-the-suddain-translation-of-elijah-sermons.jsonl --in <vault>/plans/sermons-2026-11-05/england-s-preservation-sermons.jsonl --in <vault>/plans/sermons-2026-11-05/haman-s-vanity-or-a-sermon-displaying-the-birthlesse-issues-of-church-destroying-adversaries-sermons.jsonl --in <vault>/plans/sermons-2026-11-05/the-nature-and-danger-of-heresies-sermons.jsonl --title-form "ref-in-brackets" --id sedgwick-logos --out sources/sedgwick-logos`; do not edit by hand. Only ids, titles, references and addresses are listed.

## Counts

```
lines read:                          7
rows written:                        7
lines dropped:                       0
rows with a ref:                     4
  of them read from the label:       0
  of them read from the title:       1
  of them with cited references:     6
rows with no ref:                    3
series rows (ref over >3 chapters):  0
rows with their own date:            0
rows dated by the year alone:        0
references the book table refused:   0
Logos and the label disagree:        0
contents entries split in parts:     3
```

## Rows before and after, by input

| input | lines | rows before | duplicates dropped | rows after |
|---|---|---|---|---|
| a-thanksgiving-sermon-preached-before-the-honourable-house-of-commons-at-westminster-april-9-1644 | 1 | 1 | 0 | 1 |
| an-arke-against-a-deluge | 1 | 1 | 0 | 1 |
| elisha-his-lamentation-upon-the-suddain-translation-of-elijah | 1 | 1 | 0 | 1 |
| england-s-preservation | 2 | 2 | 0 | 2 |
| haman-s-vanity-or-a-sermon-displaying-the-birthlesse-issues-of-church-destroying-adversaries | 1 | 1 | 0 | 1 |
| the-nature-and-danger-of-heresies | 1 | 1 | 0 | 1 |
| **all** | 7 | 7 | 0 | 7 |

## Lines dropped (0)

A line with neither a ref nor a title, or with no id, no title, no address or an id already used, is not a row.

None.

## Titles that did not have the form (6)

--title-form "ref-in-brackets" reads the preacher, years, text or number printed in a title; these titles do not have it, so the row keeps the title and the preacher as the index gives them.

- THNKSGVNGSRMNWS.1-3910: "A Sermon Preached at the Thanksgiving, before the Honourable House of Commons, Aprill 9. 1644"
- NARKADELUGE.1-2755: "A Sermon Preached before the Honorable House of Commons, at Their Extraordinary Day of Humiliation October 22. 1644"
- LSHHSTRNSLTNLJH.1-5309: "A Sermon Lately Preached in the Abbey of Westminster, July 4th. 1654"
- NGLNDSPRJDGMNTS.1-6622: "A Sermon Preached at the Late Fast, to the Commons House of Parliament"
- NGLNDSPRJDGMNTS.1-50961: "Second Part: And Sow Not among Thornes"
- HMNSVNTYCDVRSRS.1-5018: "A Sermon Preached before the Honourable House of Commons"

## References the book table refused (0)

A Logos reference to a book outside the sixty-six, or a chapter or verse past the end; it is left out of the row, and the row stays.

None.

## Series rows (0)

The ref spans more than three chapters (a whole book or a long text): the row is kept, found by its title and listed by its source, and matched by no passage.

None.

## Logos and the label disagree (0)

Both were read and differ; the row carries Logos's reference.

None.

## Contents entries split in parts (3)

The book's contents list splits the entry into parts, so the link opens the book where the entry begins (often a heading, with the sermon in the parts after it).

- NGLNDSPRJDGMNTS.1-6622
- NGLNDSPRJDGMNTS.1-50961
- NTRDNGRHRSS.1-4734

## Rows with no ref (3)

Neither a Logos reference nor a label that could be read: the row is kept, found by its title and its source, and by no passage. Ids, in the file's order:

LSHHSTRNSLTNLJH.1-5309, NGLNDSPRJDGMNTS.1-50961, HMNSVNTYCDVRSRS.1-5018
