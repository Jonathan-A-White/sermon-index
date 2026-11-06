# spurgeon-sermon-notes-logos: what the import did not settle

<!-- toc: none -->

Written by `go run ./tools/logos-import --in <vault>/plans/sermons-2026-11-05/my-sermon-notes-volumes-1-2-genesis-to-malachi-sermons.jsonl --in <vault>/plans/sermons-2026-11-05/my-sermon-notes-volumes-3-4-matthew-to-revelation-sermons.jsonl --id spurgeon-sermon-notes-logos --out sources/spurgeon-sermon-notes-logos`; do not edit by hand. Only ids, titles, references and addresses are listed.

## Counts

```
lines read:                          264
rows written:                        264
lines dropped:                       0
rows with a ref:                     264
  of them read from the label:       0
  of them with cited references:     185
rows with no ref:                    0
series rows (ref over >3 chapters):  0
rows with their own date:            0
rows dated by the year alone:        0
references the book table refused:   0
Logos and the label disagree:        24
titles made readable:                264
```

## Rows before and after, by input

| input | lines | rows before | duplicates dropped | rows after |
|---|---|---|---|---|
| my-sermon-notes-volumes-1-2-genesis-to-malachi | 129 | 129 | 0 | 129 |
| my-sermon-notes-volumes-3-4-matthew-to-revelation | 135 | 135 | 0 | 135 |
| **all** | 264 | 264 | 0 | 264 |

## Lines dropped (0)

A line with neither a ref nor a title, or with no id, no title, no address or an id already used, is not a row.

None.

## References the book table refused (0)

A Logos reference to a book outside the sixty-six, or a chapter or verse past the end; it is left out of the row, and the row stays.

None.

## Series rows (0)

The ref spans more than three chapters (a whole book or a long text): the row is kept, found by its title and listed by its source, and matched by no passage.

None.

## Logos and the label disagree (24)

Both were read and differ; the row carries Logos's reference.

- MYSRMNOTES01.2-124881: Logos says 1Sam.18.3; 1Sam.20.17, the label "1 Samuel 18:3" says 1Sam.18.3
- MYSRMNOTES01.2-145495: Logos says 1Kgs.2.28; 1Kgs.2.30, the label "1 Kings 2:28" says 1Kgs.2.28
- MYSRMNOTES01.2-197468: Logos says 2Kgs.17.25; 2Kgs.17.33; 2Kgs.17.34, the label "2 Kings 17:25" says 2Kgs.17.25
- MYSRMNOTES01.2-339008: Logos says Ps.115.17; Ps.115.18, the label "Psalm 115:17" says Ps.115.17
- MYSRMNOTES01.2-504542: Logos says Isa.49.20; Isa.49.21, the label "Isaiah 49:20" says Isa.49.20
- MYSRMNOTES01.2-557165: Logos says Jer.3.12; Jer.3.14; Jer.3.22, the label "Jeremiah 3:12" says Jer.3.12
- MYSRMNOTES01.2-611611: Logos says Ezek.36.30; Ezek.36.31, the label "Ezekiel 36:30" says Ezek.36.30
- MYSRMNOTES01.2-637909: Logos says Hos.2.6; Hos.2.7, the label "Hosea 2:6" says Hos.2.6
- MYSRMNOTES01.2-651141: Logos says Hos.2.23; Rom.9.25; Rom.9.26, the label "Hosea 2:23" says Hos.2.23
- MYSRMNOTES01.2-702408: Logos says Jonah.3.4; Matt.12.41, the label "Jonah 3:4" says Jonah.3.4
- MYSRMNOTES01.2-746423: Logos says Hab.2.4; Rom.1.17; Gal.3.11; Heb.10.38, the label "Habakkuk 2:4" says Hab.2.4
- MYSRMNOTES01.2-762434: Logos says Hag.2.13; Hag.2.14, the label "Haggai 2:13" says Hag.2.13
- MYSRMNOTES01.2-775911: Logos says Zech.7.5; Zech.7.6, the label "Zechariah 7:5" says Zech.7.5
- MYSRMNOTES01.2-782710: Logos says Zech.9.11; Zech.9.12, the label "Zechariah 9:11" says Zech.9.11
- MYSRMNOTES03.2-33563: Logos says Matt.8.7; Luke.8.7, the label "Matthew 8:7" says Matt.8.7
- MYSRMNOTES03.2-87441: Logos says Matt.20.3; Matt.20.4, the label "Matthew 20:3" says Matt.20.3
- MYSRMNOTES03.2-111394: Logos says Matt.28.9; Matt.28.10, the label "Matthew 28:9" says Matt.28.9
- MYSRMNOTES03.2-124266: Logos says Mark.5.6; Luke.15.20, the label "Mark 5:6" says Mark.5.6
- MYSRMNOTES03.2-148634: Logos says Mark.10.49; Mark.10.50, the label "Mark 10:49" says Mark.10.49
- MYSRMNOTES03.2-215385: Logos says Luke.22.19; Luke.22.20, the label "Luke 22:19" says Luke.22.19
- MYSRMNOTES03.2-312450: Logos says John.10.22; John.10.23, the label "John 10:22" says John.10.22
- MYSRMNOTES03.2-429082: Logos says Rom.6.11; Rom.6.12, the label "Romans 6:11" says Rom.6.11
- MYSRMNOTES03.2-765563: Logos says John.3.3, the label "1 John 3:3" says 1John.3.3
- MYSRMNOTES03.2-777406: Logos says 1John.3.20; 1John.3.21, the label "1 John 3:20" says 1John.3.20

## Rows with no ref (0)

Neither a Logos reference nor a label that could be read: the row is kept, found by its title and its source, and by no passage. Ids, in the file's order:

None.
