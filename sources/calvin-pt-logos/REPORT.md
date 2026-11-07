# calvin-pt-logos: what the import did not settle

<!-- toc: none -->

Written by `go run ./tools/logos-import --in <vault>/plans/sermons-2026-11-05/serm-es-sobre-elei-o-e-reprova-o-sermons.jsonl --title-form "pt-ref" --id calvin-pt-logos --out sources/calvin-pt-logos`; do not edit by hand. Only ids, titles, references and addresses are listed.

## Counts

```
lines read:                          15
rows written:                        13
lines dropped:                       0
non-sermon lines dropped:            2
rows with a ref:                     13
  of them read from the label:       0
  of them read from the title:       13
  of them with cited references:     13
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

## Non-sermon lines dropped (2)

The archive's index lists them with the sermons, but they are a table, an index of Scripture, an introduction or a book's argument (prolegomena), not a sermon: not written, and not counted in "rows before".

- SRMSSBRLRPRV.1-4439: "Prefácio" (front matter)
- SRMSSBRLRPRV.1-520149: "Resposta a Certas Calúnias e Blasfêmias" (an appendix, a reply to slanders, not a sermon)

## Titles that did not have the form (0)

--title-form "pt-ref" reads the preacher, years, text or number printed in a title; these titles do not have it, so the row keeps the title and the preacher as the index gives them.

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
