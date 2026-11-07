# sermon-outlines-for-special-days-logos: what the import did not settle

<!-- toc: none -->

Written by `go run ./tools/logos-import --in <vault>/plans/sermons-2026-11-05/sermon-outlines-for-special-days-sermons.jsonl --title-form "ref-in-brackets" --id sermon-outlines-for-special-days-logos --out sources/sermon-outlines-for-special-days-logos`; do not edit by hand. Only ids, titles, references and addresses are listed.

## Counts

```
lines read:                          28
rows written:                        28
lines dropped:                       0
rows with a ref:                     28
  of them read from the label:       0
  of them read from the title:       1
  of them with cited references:     24
rows with no ref:                    0
series rows (ref over >3 chapters):  0
rows with their own date:            0
rows dated by the year alone:        0
references the book table refused:   0
Logos and the label disagree:        2
```

## Lines dropped (0)

A line with neither a ref nor a title, or with no id, no title, no address or an id already used, is not a row.

None.

## Titles that did not have the form (3)

--title-form "ref-in-brackets" reads the preacher, years, text or number printed in a title; these titles do not have it, so the row keeps the title and the preacher as the index gives them.

- SOSPECDAYS.1-5233: "Valentine’s Day: Love Letters Written in the Sand (John 8:3 ff.)"
- SOSPECDAYS.1-19883: "Pentecost: Wind that Fills the Sails (Acts 2:1–4; 37–41)"
- SOSPECDAYS.1-23109: "Mother’s Day: The Power of Example (Ruth 1:16; 19–22)"

## References the book table refused (0)

A Logos reference to a book outside the sixty-six, or a chapter or verse past the end; it is left out of the row, and the row stays.

None.

## Series rows (0)

The ref spans more than three chapters (a whole book or a long text): the row is kept, found by its title and listed by its source, and matched by no passage.

None.

## Logos and the label disagree (2)

Both were read and differ; the row carries Logos's reference.

- SOSPECDAYS.1-19883: Logos says Acts.2.1-Acts.2.4; Acts.2.37-Acts.2.41, the label "Acts 2:1–4" says Acts.2.1-Acts.2.4
- SOSPECDAYS.1-23109: Logos says Ruth.1.16; Ruth.1.19-Ruth.1.22, the label "Ruth 1:16" says Ruth.1.16

## Rows with no ref (0)

Neither a Logos reference nor a label that could be read: the row is kept, found by its title and its source, and by no passage. Ids, in the file's order:

None.
