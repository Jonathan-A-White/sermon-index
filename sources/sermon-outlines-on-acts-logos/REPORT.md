# sermon-outlines-on-acts-logos: what the import did not settle

<!-- toc: none -->

Written by `go run ./tools/logos-import --in <vault>/plans/sermons-2026-11-05/sermon-outlines-on-acts-sermons.jsonl --chapter-outlines --id sermon-outlines-on-acts-logos --out sources/sermon-outlines-on-acts-logos`; do not edit by hand. Only ids, titles, references and addresses are listed.

## Counts

```
lines read:                          86
rows written:                        38
lines dropped:                       0
rows with a ref:                     38
  of them read from the label:       0
  of them with cited references:     32
rows with no ref:                    0
series rows (ref over >3 chapters):  1
rows with their own date:            0
rows dated by the year alone:        0
references the book table refused:   1
Logos and the label disagree:        11
lines dropped as not a chapter outline: 48
titles made readable:                38
contents entries split in parts:     22
```

## Lines dropped (0)

A line with neither a ref nor a title, or with no id, no title, no address or an id already used, is not a row.

None.

## Chapter and sub-point lines dropped (48)

The book lists chapters, each an outline on a passage, and beside them the numbered sub-points of other chapters, a chapter or two with no passage and an afterword. The rule: a "Chapter N: Title" line with a passage is a row, one row for each outline; every other line is dropped here and not counted in "rows before".

- SOACTS.1-9251: "Chapter I: The New Testament" (a chapter with no passage, an introduction or an essay and not an outline)
- SOACTS.1-14217: "Chapter II: Acts of Apostles" (a chapter with no passage, an introduction or an essay and not an outline)
- SOACTS.2-88511: "1. Concerning the Fulfillment of the Old Institution (Matt. 5:17, 18)" (a numbered sub-point of a chapter, not an outline of its own)
- SOACTS.2-89564: "2. Concerning the Ultimate Rejection of the Jews (Matt. 8:10–12; Luke 13:24–30)" (a numbered sub-point of a chapter, not an outline of its own)
- SOACTS.2-90080: "3. Concerning the Cost of Discipleship (Matt. 10:32–42)" (a numbered sub-point of a chapter, not an outline of its own)
- SOACTS.2-90728: "4. Concerning the Destruction of the Cities of Galilee (Matt. 11:21–24; Luke 10:13–15)" (a numbered sub-point of a chapter, not an outline of its own)
- SOACTS.2-91308: "5. Concerning the Establishment of the Church (Matt. 16:13–20)" (a numbered sub-point of a chapter, not an outline of its own)
- SOACTS.2-91538: "6. Concerning His Sufferings and Death, and Resurrection on the Third Day (Matt. 20:17–19; 17:22, 23; Mark 10:33, 34; Luke 18:31–33; 9:22; John 2:19–21; 6:70)" (a numbered sub-point of a chapter, not an outline of its own)
- SOACTS.2-91789: "7. Concerning His Ascension (John 7:33, 34; 20:17)" (a numbered sub-point of a chapter, not an outline of its own)
- SOACTS.2-91870: "8. Concerning the Destruction of Jerusalem and of the Temple (Matt. 24:1, 2; Mark 13:2; Luke 13:34, 35; 19:41–44; 21:6)" (a numbered sub-point of a chapter, not an outline of its own)
- SOACTS.2-92223: "9. Concerning His Second Coming" (a numbered sub-point of a chapter, not an outline of its own)
- SOACTS.2-92668: "10. Concerning the General Resurrection of the Dead (John 5:25–29)" (a numbered sub-point of a chapter, not an outline of its own)
- SOACTS.2-92846: "11. Concerning the General Judgment (Matt. 25:31–46)" (a numbered sub-point of a chapter, not an outline of its own)
- SOACTS.2-113276: "1. The Parable of the Unjust Steward (Luke 16:1–13)" (a numbered sub-point of a chapter, not an outline of its own)
- SOACTS.2-114479: "2. The Example of the Church at Jerusalem (Acts 2:44, 45; 4:32–37)" (a numbered sub-point of a chapter, not an outline of its own)
- SOACTS.2-115549: "3. The New Testament Plan of Christian Giving (1 Cor. 16:1, 2)" (a numbered sub-point of a chapter, not an outline of its own)
- SOACTS.2-118348: "4. The Manner of Christian Giving" (a numbered sub-point of a chapter, not an outline of its own)
- SOACTS.2-118790: "5. The Bible Condemnation of Covetousness (Luke 12:15; 1 Cor. 6:9, 10; Eph. 5:3–5)" (a numbered sub-point of a chapter, not an outline of its own)
- SOACTS.2-122902: "1. Ordination of Officers" (a numbered sub-point of a chapter, not an outline of its own)
- SOACTS.2-125724: "2. Qualifications of Deacons (Acts 6:1–7; 1 Tim. 3:8–13)" (a numbered sub-point of a chapter, not an outline of its own)
- SOACTS.2-126387: "3. The Duties of Deacons" (a numbered sub-point of a chapter, not an outline of its own)
- SOACTS.2-126822: "4. Deaconesses (Rom. 16:1, 2; Acts 18:24–26; 1 Tim. 5:9–15)" (a numbered sub-point of a chapter, not an outline of its own)
- SOACTS.2-127054: "5. Relationship between the Deacons and the Congregation" (a numbered sub-point of a chapter, not an outline of its own)
- SOACTS.2-222682: "1. Concerning the New Birth (John 3:1–17)" (a numbered sub-point of a chapter, not an outline of its own)
- SOACTS.2-225652: "2. Concerning the Bread of Life (John 6)" (a numbered sub-point of a chapter, not an outline of its own)
- SOACTS.2-226671: "3. Concerning Spiritual Works (John 14:12)" (a numbered sub-point of a chapter, not an outline of its own)
- SOACTS.2-227831: "4. Concerning the Kingdom (John 18:36)" (a numbered sub-point of a chapter, not an outline of its own)
- SOACTS.2-229484: "5. Concerning Salvation (John 3:17; 12:47; Acts 4:12)" (a numbered sub-point of a chapter, not an outline of its own)
- SOACTS.2-246306: "1. What Is Faith?" (a numbered sub-point of a chapter, not an outline of its own)
- SOACTS.2-247413: "2. How Is Faith Acquired?" (a numbered sub-point of a chapter, not an outline of its own)
- SOACTS.2-248342: "3. What Must Men Believe to Be Saved? (John 20:30, 31; Acts 8:37)" (a numbered sub-point of a chapter, not an outline of its own)
- SOACTS.2-248645: "4. The Scope of Faith" (a numbered sub-point of a chapter, not an outline of its own)
- SOACTS.2-248932: "5. What Is the Object of Faith?" (a numbered sub-point of a chapter, not an outline of its own)
- SOACTS.1-275413: "Chapter XLI: The Sabbath or the Lord’s Day—Which? (Concluded)" (a chapter with no passage, an introduction or an essay and not an outline)
- SOACTS.2-284842: "2. How Often Should the Institution Be Observed?" (a numbered sub-point of a chapter, not an outline of its own)
- SOACTS.2-287359: "3. Who Should Participate in the Observance of the Institution?" (a numbered sub-point of a chapter, not an outline of its own)
- SOACTS.2-288291: "4. What Is the Purpose of the Institution? (1 Cor. 11:26)" (a numbered sub-point of a chapter, not an outline of its own)
- SOACTS.2-291308: "1. By Whom Selected?" (a numbered sub-point of a chapter, not an outline of its own)
- SOACTS.2-292447: "2. How Selected?" (a numbered sub-point of a chapter, not an outline of its own)
- SOACTS.2-293785: "3. Number of Elders" (a numbered sub-point of a chapter, not an outline of its own)
- SOACTS.2-294388: "4. Term of Office" (a numbered sub-point of a chapter, not an outline of its own)
- SOACTS.2-294752: "5. Qualifications (1 Tim. 3:1–7; Tit. 1:5–9)" (a numbered sub-point of a chapter, not an outline of its own)
- SOACTS.2-297005: "6. Titles" (a numbered sub-point of a chapter, not an outline of its own)
- SOACTS.2-298405: "7. Duties" (a numbered sub-point of a chapter, not an outline of its own)
- SOACTS.2-300048: "8. Relationship between the Eldership and the Congregation" (a numbered sub-point of a chapter, not an outline of its own)
- SOACTS.1-350557: "Chapter XLIX: The Divine Missionary Policy" (a chapter with no passage, an introduction or an essay and not an outline)
- SOACTS.1-359888: "Chapter L: Apostolic Evangelism" (a chapter with no passage, an introduction or an essay and not an outline)
- SOACTS.1-372551: "Afterword" (not a chapter of the book)

## References the book table refused (1)

A Logos reference to a book outside the sixty-six, or a chapter or verse past the end; it is left out of the row, and the row stays.

- SOACTS.1-174987: "bible.66.16.25": ref 'Rom.16.25': verse past the chapter; "bible.66.16.26": ref 'Rom.16.26': verse past the chapter

## Series rows (1)

The ref spans more than three chapters (a whole book or a long text): the row is kept, found by its title and listed by its source, and matched by no passage.

- SOACTS.1-102303 Acts.1-Acts.7 (7 chapters)

## Logos and the label disagree (11)

Both were read and differ; the row carries Logos's reference.

- SOACTS.1-33015: Logos says Matt.3.11; Matt.3.12; Acts.1.4; Acts.1.5; Acts.11.1-Acts.11.4; Acts.2.38-Acts.2.42; Acts.8.36-Acts.8.39; Acts.10.44-Acts.10.48; Acts.19.1-Acts.19.5, the label "Matt. 3:11" says Matt.3.11
- SOACTS.1-129186: Logos says Acts.7.51-Acts.7.53; Acts.8.26-Acts.8.40, the label "Acts 7:51–53" says Acts.7.51-Acts.7.53
- SOACTS.1-136277: Logos says Acts.8.5-Acts.8.25; Acts.19.1-Acts.19.12, the label "Acts 8:5–25" says Acts.8.5-Acts.8.25
- SOACTS.1-171217: Logos says Acts.9.1-Acts.9.22; Acts.22; Acts.26, the label "Acts 9:1–22" says Acts.9.1-Acts.9.22
- SOACTS.1-174987: Logos says Acts.10; Acts.11.1-Acts.11.18, the label "Acts 10" says Acts.10
- SOACTS.1-182989: Logos says Acts.10; Acts.11.1-Acts.11.18, the label "Acts 10" says Acts.10
- SOACTS.1-191395: Logos says Acts.10; Acts.11.1-Acts.11.20, the label "Acts 10" says Acts.10
- SOACTS.1-258455: Logos says Acts.17.30; Acts.17.31, the label "Acts 17:30" says Acts.17.30
- SOACTS.1-301604: Logos says Acts.24; Acts.21; Acts.22; Acts.23; Acts.24.10-Acts.24.21; Acts.24.22; Acts.24.23, the label "Acts 24" says Acts.24
- SOACTS.1-328746: Logos says Acts.2.36; Acts.7.51-Acts.7.53; Acts.8.1-Acts.8.4; Acts.12.1-Acts.12.5; Acts.28.22, the label "Acts 2:36" says Acts.2.36
- SOACTS.1-338489: Logos says Acts.2.36; Acts.7.51-Acts.7.53; Acts.8.1-Acts.8.4; Acts.12.1-Acts.12.5; Acts.28.22, the label "Acts 2:36" says Acts.2.36

## Contents entries split in parts (22)

The book's contents list splits the entry into parts, so the link opens the book where the entry begins (often a heading, with the sermon in the parts after it).

- SOACTS.1-33015
- SOACTS.1-43987
- SOACTS.1-49453
- SOACTS.1-55147
- SOACTS.1-66627
- SOACTS.1-70821
- SOACTS.1-102303
- SOACTS.1-142154
- SOACTS.1-149462
- SOACTS.1-155273
- SOACTS.1-171217
- SOACTS.1-174987
- SOACTS.1-182989
- SOACTS.1-191395
- SOACTS.1-210422
- SOACTS.1-249951
- SOACTS.1-263508
- SOACTS.1-301604
- SOACTS.1-309288
- SOACTS.1-320411
- SOACTS.1-328746
- SOACTS.1-338489

## Rows with no ref (0)

Neither a Logos reference nor a label that could be read: the row is kept, found by its title and its source, and by no passage. Ids, in the file's order:

None.
