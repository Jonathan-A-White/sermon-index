# sermon-outlines-for-seekers-logos: what the import did not settle

<!-- toc: none -->

Written by `go run ./tools/logos-import --in <vault>/plans/sermons-2026-11-05/sermon-outlines-for-seekers-sermons.jsonl --title-form "ref-in-brackets" --id sermon-outlines-for-seekers-logos --out sources/sermon-outlines-for-seekers-logos`; do not edit by hand. Only ids, titles, references and addresses are listed.

## Counts

```
lines read:                          26
rows written:                        26
lines dropped:                       0
rows with a ref:                     26
  of them read from the label:       0
  of them with cited references:     16
rows with no ref:                    0
series rows (ref over >3 chapters):  0
rows with their own date:            0
rows dated by the year alone:        0
references the book table refused:   5
Logos and the label disagree:        1
```

## Lines dropped (0)

A line with neither a ref nor a title, or with no id, no title, no address or an id already used, is not a row.

None.

## Titles that did not have the form (0)

--title-form "ref-in-brackets" reads the preacher, years, text or number printed in a title; these titles do not have it, so the row keeps the title and the preacher as the index gives them.

None.

## References the book table refused (5)

A Logos reference to a book outside the sixty-six, or a chapter or verse past the end; it is left out of the row, and the row stays.

- SOSEEKERS.1-6897: "bible.61.6.32.b": want book.chapter or book.chapter.verse
- SOSEEKERS.1-32455: "bible.66.5.2.b-66.5.4": want book.chapter or book.chapter.verse
- SOSEEKERS.1-55515: "bible.68.1.4.a": want book.chapter or book.chapter.verse; "bible.68.1.4.b": want book.chapter or book.chapter.verse
- SOSEEKERS.1-58527: "bible.80.1.19-80.1.21.a": want book.chapter or book.chapter.verse; "bible.80.1.21.b": want book.chapter or book.chapter.verse
- SOSEEKERS.1-61866: "bible.81.1.3.a": want book.chapter or book.chapter.verse; "bible.81.1.3.b": want book.chapter or book.chapter.verse

## Series rows (0)

The ref spans more than three chapters (a whole book or a long text): the row is kept, found by its title and listed by its source, and matched by no passage.

None.

## Logos and the label disagree (1)

Both were read and differ; the row carries Logos's reference.

- SOSEEKERS.1-12814: Logos says Matt.7.1-Matt.7.5; Matt.5.38-Matt.5.42; Matt.5.23-Matt.5.26; Matt.5.43-Matt.5.48; Matt.7.12, the label "Matthew 7:1–5" says Matt.7.1-Matt.7.5

## Rows with no ref (0)

Neither a Logos reference nor a label that could be read: the row is kept, found by its title and its source, and by no passage. Ids, in the file's order:

None.
