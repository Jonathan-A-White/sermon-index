# evangelistic-sermon-outlines-logos: what the import did not settle

<!-- toc: none -->

Written by `go run ./tools/logos-import --in <vault>/plans/sermons-2026-11-05/evangelistic-sermon-outlines-sermons.jsonl --skipped "Sermon Outlines: Soul Winning! Church Building! (LLS:SOSOULWIN): none of its 24 titles carries a Scripture ref and Logos gives none, so the index found 0 of 24 passages; not imported, for a later decision. Titles, in the book's order || Preaching with Purpose || “Behold, Thy King Cometh unto Thee” || Christ Calling Men || Cheerios of Jesus—“Be of Good Cheer” ’ || Is the Church of the First Century Adequate for Today? || Authority || The Upper Room || The Place of Baptism in Salvation || “This Man Receiveth Sinners” || The Prodigal Son || The Ascension of Christ || Myth, Man, or Messiah || “Follow Me” || The New Birth || Some Things Jesus Would Not Do || Christ, a Fact || The Birth of the Church || “But Barnabas” || Repentance || Asleep in the Church || Why Jesus Came || “The Faith … Once Delivered unto the Saints” || The Great Invitation || What I Learned in Thirty-Eight and a Half Years" --id evangelistic-sermon-outlines-logos --out sources/evangelistic-sermon-outlines-logos`; do not edit by hand. Only ids, titles, references and addresses are listed.

## Counts

```
lines read:                          29
rows written:                        29
lines dropped:                       0
rows with a ref:                     29
  of them read from the label:       0
  of them with cited references:     18
rows with no ref:                    0
series rows (ref over >3 chapters):  0
rows with their own date:            0
rows dated by the year alone:        0
references the book table refused:   0
Logos and the label disagree:        10
```

## Skipped (1)

Books of the Governor's order this source leaves out, with the titles the Mayor's index found in each (named by hand with --skipped; the tool did not read them). A decision for later.

- Sermon Outlines: Soul Winning! Church Building! (LLS:SOSOULWIN): none of its 24 titles carries a Scripture ref and Logos gives none, so the index found 0 of 24 passages; not imported, for a later decision. Titles, in the book's order
  - Preaching with Purpose
  - “Behold, Thy King Cometh unto Thee”
  - Christ Calling Men
  - Cheerios of Jesus—“Be of Good Cheer” ’
  - Is the Church of the First Century Adequate for Today?
  - Authority
  - The Upper Room
  - The Place of Baptism in Salvation
  - “This Man Receiveth Sinners”
  - The Prodigal Son
  - The Ascension of Christ
  - Myth, Man, or Messiah
  - “Follow Me”
  - The New Birth
  - Some Things Jesus Would Not Do
  - Christ, a Fact
  - The Birth of the Church
  - “But Barnabas”
  - Repentance
  - Asleep in the Church
  - Why Jesus Came
  - “The Faith … Once Delivered unto the Saints”
  - The Great Invitation
  - What I Learned in Thirty-Eight and a Half Years

## Lines dropped (0)

A line with neither a ref nor a title, or with no id, no title, no address or an id already used, is not a row.

None.

## References the book table refused (0)

A Logos reference to a book outside the sixty-six, or a chapter or verse past the end; it is left out of the row, and the row stays.

None.

## Series rows (0)

The ref spans more than three chapters (a whole book or a long text): the row is kept, found by its title and listed by its source, and matched by no passage.

None.

## Logos and the label disagree (10)

Both were read and differ; the row carries Logos's reference.

- SOEVANG.1-1054: Logos says John.3.14; John.3.15; John.8.28; John.12.32, the label "John 3:14" says John.3.14
- SOEVANG.1-5923: Logos says 1Pet.4.16; Acts.4.12, the label "1 Peter 4:16" says 1Pet.4.16
- SOEVANG.1-23006: Logos says Dan.12.3; Jas.5.20, the label "Daniel 12:3" says Dan.12.3
- SOEVANG.1-30893: Logos says John.14.8; John.14.9, the label "John 14:8" says John.14.8
- SOEVANG.1-40561: Logos says Luke.18.18; Luke.18.22, the label "Luke 18:18" says Luke.18.18
- SOEVANG.1-42571: Logos says Mark.10.21; Mark.10.22, the label "Mark 10:21" says Mark.10.21
- SOEVANG.1-50273: Logos says Heb.2.14; Heb.2.15, the label "Hebrews 2:14" says Heb.2.14
- SOEVANG.1-54394: Logos says 2Cor.5.11-2Cor.5.21; Gal.3.27, the label "2 Corinthians 5:11–21" says 2Cor.5.11-2Cor.5.21
- SOEVANG.1-60213: Logos says Acts.8.21; Acts.8.22, the label "Acts 8:21" says Acts.8.21
- SOEVANG.1-61523: Logos says John.6.60-John.6.71; Phil.2.16, the label "John 6:60–71" says John.6.60-John.6.71

## Rows with no ref (0)

Neither a Logos reference nor a label that could be read: the row is kept, found by its title and its source, and by no passage. Ids, in the file's order:

None.
