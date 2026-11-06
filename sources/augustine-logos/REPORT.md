# augustine-logos: what the import did not settle

<!-- toc: none -->

Written by `go run ./tools/logos-import --in <vault>/plans/sermons-2026-11-05/sermons-94a-147a-on-the-new-testament-sermons.jsonl --refs-from-title --id augustine-logos --out sources/augustine-logos`; do not edit by hand. Only ids, titles, references and addresses are listed.

## Counts

```
lines read:                          72
rows written:                        70
lines dropped:                       0
non-sermon lines dropped:            2
rows with a ref:                     60
  of them read from the label:       0
  of them read from the title:       60
  of them with cited references:     69
rows with no ref:                    10
series rows (ref over >3 chapters):  0
rows with their own date:            0
rows dated by the year alone:        0
references the book table refused:   12
Logos and the label disagree:        0
```

## Lines dropped (0)

A line with neither a ref nor a title, or with no id, no title, no address or an id already used, is not a row.

None.

## Non-sermon lines dropped (2)

The archive's index lists them with the sermons, but they are a table, an index of Scripture, an introduction or a book's argument (prolegomena), not a sermon: not written, and not counted in "rows before".

- AGSTNSRMNS04.1-1278985: "Chronological Table" (a chronological table)
- AGSTNSRMNS04.2-1288444: "New Testament" (an index of Scripture)

## References the book table refused (12)

A Logos reference to a book outside the sixty-six, or a chapter or verse past the end; it is left out of the row, and the row stays.

- AGSTNSRMNS04.1-308609: "bible.44.10.9": Logos book 44 is not one of the 66
- AGSTNSRMNS04.1-430260: "bible.44.29.27": Logos book 44 is not one of the 66
- AGSTNSRMNS04.1-472781: "bible.44.30.23": Logos book 44 is not one of the 66
- AGSTNSRMNS04.1-480398: "bible.43.1.11": Logos book 43 is not one of the 66
- AGSTNSRMNS04.1-618042: "bible.43.14.11": Logos book 43 is not one of the 66; "bible.44.2.5": Logos book 44 is not one of the 66
- AGSTNSRMNS04.1-716288: "bible.43.7.26": Logos book 43 is not one of the 66
- AGSTNSRMNS04.1-753749: "bible.43.7.26": Logos book 43 is not one of the 66
- AGSTNSRMNS04.1-797232: "bible.44.10.11": Logos book 44 is not one of the 66
- AGSTNSRMNS04.1-812898: "bible.43.1.9": Logos book 43 is not one of the 66
- AGSTNSRMNS04.1-847404: "bible.40.4.8-40.4.9": Logos book 40 is not one of the 66; "bible.40.4.10": Logos book 40 is not one of the 66; "bible.40.4.7": Logos book 40 is not one of the 66
- AGSTNSRMNS04.1-858691: "bible.44.36.6": Logos book 44 is not one of the 66
- AGSTNSRMNS04.1-1180424: "bible.44.9.13": Logos book 44 is not one of the 66

## Series rows (0)

The ref spans more than three chapters (a whole book or a long text): the row is kept, found by its title and listed by its source, and matched by no passage.

None.

## Logos and the label disagree (0)

Both were read and differ; the row carries Logos's reference.

None.

## Rows with no ref (10)

Neither a Logos reference nor a label that could be read: the row is kept, found by its title and its source, and by no passage. Ids, in the file's order:

AGSTNSRMNS04.1-270629, AGSTNSRMNS04.1-371611, AGSTNSRMNS04.1-411983, AGSTNSRMNS04.1-498862, AGSTNSRMNS04.1-563145, AGSTNSRMNS04.1-579952, AGSTNSRMNS04.1-618042, AGSTNSRMNS04.1-658555, AGSTNSRMNS04.1-847404, AGSTNSRMNS04.1-1081742
