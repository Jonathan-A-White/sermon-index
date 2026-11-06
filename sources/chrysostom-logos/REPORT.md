# chrysostom-logos: what the import did not settle

<!-- toc: none -->

Written by `go run ./tools/logos-import --in <vault>/plans/sermons-2026-11-05/nicene-and-post-nicene-fathers-1-10-saint-chrysostom-homilies-on-the-gospel-of-saint-matthew-sermons.jsonl --in <vault>/plans/sermons-2026-11-05/nicene-and-post-nicene-fathers-1-11-saint-chrysostom-homilies-on-the-acts-of-the-apostles-and-the-epistle-to-the-romans-sermons.jsonl --in <vault>/plans/sermons-2026-11-05/nicene-and-post-nicene-fathers-1-12-saint-chrysostom-homilies-on-the-epistles-of-paul-to-the-corinthians-sermons.jsonl --in <vault>/plans/sermons-2026-11-05/nicene-and-post-nicene-fathers-1-13-saint-chrysostom-homilies-on-galatians-ephesians-philippians-colossians-thessalonians-timothy-titus-and-philemon-sermons.jsonl --in <vault>/plans/sermons-2026-11-05/nicene-and-post-nicene-fathers-1-14-saint-chrysostom-homilies-on-the-gospel-of-st-john-and-epistle-to-the-hebrews-sermons.jsonl --preacher "John Chrysostom" --id chrysostom-logos --out sources/chrysostom-logos`; do not edit by hand. Only ids, titles, references and addresses are listed.

## Counts

```
lines read:                          491
rows written:                        483
lines dropped:                       0
non-sermon lines dropped:            8
rows with a ref:                     479
  of them read from the label:       0
  of them with cited references:     338
rows with no ref:                    4
series rows (ref over >3 chapters):  0
rows with their own date:            0
rows dated by the year alone:        0
references the book table refused:   16
Logos and the label disagree:        107
```

## Rows before and after, by input

| input | lines | rows before | duplicates dropped | rows after |
|---|---|---|---|---|
| nicene-and-post-nicene-fathers-1-10-saint-chrysostom-homilies-on-the-gospel-of-saint-matthew | 92 | 90 | 0 | 90 |
| nicene-and-post-nicene-fathers-1-11-saint-chrysostom-homilies-on-the-acts-of-the-apostles-and-the-epistle-to-the-romans | 88 | 87 | 0 | 87 |
| nicene-and-post-nicene-fathers-1-12-saint-chrysostom-homilies-on-the-epistles-of-paul-to-the-corinthians | 74 | 74 | 0 | 74 |
| nicene-and-post-nicene-fathers-1-13-saint-chrysostom-homilies-on-galatians-ephesians-philippians-colossians-thessalonians-timothy-titus-and-philemon | 112 | 110 | 0 | 110 |
| nicene-and-post-nicene-fathers-1-14-saint-chrysostom-homilies-on-the-gospel-of-st-john-and-epistle-to-the-hebrews | 125 | 122 | 0 | 122 |
| **all** | 491 | 483 | 0 | 483 |

## Lines dropped (0)

A line with neither a ref nor a title, or with no id, no title, no address or an id already used, is not a row.

None.

## Non-sermon lines dropped (8)

The archive's index lists them with the sermons, but they are a table, an index of Scripture, an introduction or a book's argument (prolegomena), not a sermon: not written, and not counted in "rows before".

- 6.60.10.3-214862: "I. The Place of Chrysostom in the History of Exegesis" (prolegomena)
- 6.60.10.3-224294: "II. Extent and Character of Chrysostom’s Exegetical Labors" (prolegomena)
- 6.60.11.3-2125319: "The Argument" (the argument of the work)
- 6.60.13.1-249403: "St. Chrysostom as a Homilist" (prolegomena)
- 6.60.13.4-1098707: "Introductory Discourse" (prolegomena)
- 6.60.14.3-1909115: "Argument, and Summary of the Epistle" (the argument of the work)
- 6.60.14.2-2760243: "Homilies on the Gospel of John: Index of Texts" (an index)
- 6.60.14.2-2826404: "Homilies on the Epistle to the Hebrews: Index of Texts" (an index)

## References the book table refused (16)

A Logos reference to a book outside the sixty-six, or a chapter or verse past the end; it is left out of the row, and the row stays.

- 6.60.11.3-2879551: "bible.47.1.15": Logos book 47 is not one of the 66; "bible.47.1.16": Logos book 47 is not one of the 66
- 6.60.11.3-3054772: "bible.66.16.25-66.16.27": ref 'Rom.16.25-Rom.16.27': verse past the chapter; "Rom. 16:25–27": no book
- 6.60.12.3-363224: "bible.44.16.12": Logos book 44 is not one of the 66
- 6.60.12.3-507781: "bible.44.15.16": Logos book 44 is not one of the 66
- 6.60.12.3-533621: "bible.44.9.15": Logos book 44 is not one of the 66
- 6.60.12.3-622537: "bible.47.1.6": Logos book 47 is not one of the 66; "bible.47.1.10": Logos book 47 is not one of the 66
- 6.60.12.3-752303: "bible.44.27.5": Logos book 44 is not one of the 66
- 6.60.12.3-965559: "bible.44.12.13": Logos book 44 is not one of the 66
- 6.60.12.3-1052922: "bible.44.18.16": Logos book 44 is not one of the 66; "bible.44.18.17": Logos book 44 is not one of the 66
- 6.60.12.3-1824532: "bible.44.34.23": Logos book 44 is not one of the 66
- 6.60.12.3-2118659: "bible.43.2.15": Logos book 43 is not one of the 66
- 6.60.13.4-931367: "bible.44.25.1": Logos book 44 is not one of the 66; "bible.44.40.23": Logos book 44 is not one of the 66
- 6.60.13.4-1158077: "bible.44.2.10": Logos book 44 is not one of the 66
- 6.60.14.3-682396: "bible.44.5.6": Logos book 44 is not one of the 66; "bible.44.16.12": Logos book 44 is not one of the 66
- 6.60.14.3-1056630: "bible.43.2.24": Logos book 43 is not one of the 66
- 6.60.14.3-1414289: "bible.44.18.30": Logos book 44 is not one of the 66

## Series rows (0)

The ref spans more than three chapters (a whole book or a long text): the row is kept, found by its title and listed by its source, and matched by no passage.

None.

## Logos and the label disagree (107)

Both were read and differ; the row carries Logos's reference.

- 6.60.11.3-763153: Logos says Acts.1.1; Acts.1.2; Acts.1.1-Acts.1.2, the label "Acts 1:1" says Acts.1.1
- 6.60.11.3-863563: Logos says Acts.2.1; Acts.2.2; Acts.2.1-Acts.2.2, the label "Acts 2:1" says Acts.2.1
- 6.60.11.3-1073984: Logos says Acts.4.36; Acts.4.37; Acts.4.36-Acts.4.37, the label "Acts 4:36" says Acts.4.36
- 6.60.11.3-1094675: Logos says Acts.5.17; Acts.5.18; Acts.5.17-Acts.5.18, the label "Acts 5:17" says Acts.5.17
- 6.60.11.3-1176596: Logos says Acts.7.6; Acts.7.7; Acts.7.6-Acts.7.7, the label "Acts 7:6" says Acts.7.6
- 6.60.11.3-1257968: Logos says Acts.8.26; Acts.8.27; Acts.8.26-Acts.8.27, the label "Acts 8:26" says Acts.8.26
- 6.60.11.3-1288302: Logos says Acts.9.10; Acts.9.12, the label "Acts 9:10" says Acts.9.10
- 6.60.11.3-1310001: Logos says Acts.9.26; Acts.9.27; Acts.9.26-Acts.9.27, the label "Acts 9:26" says Acts.9.26
- 6.60.11.3-1358706: Logos says Acts.10.23; Acts.10.24; Acts.10.23-Acts.10.24, the label "Acts 10:23" says Acts.10.23
- 6.60.11.3-1385647: Logos says Acts.10.44; Acts.10.46, the label "Acts 10:44" says Acts.10.44
- 6.60.11.3-1436096: Logos says Acts.12.1; Acts.12.2; Acts.12.1-Acts.12.2, the label "Acts 12:1" says Acts.12.1
- 6.60.11.3-1459581: Logos says Acts.12.18; Acts.12.19; Acts.12.18-Acts.12.19, the label "Acts 12:18" says Acts.12.18
- 6.60.11.3-1479134: Logos says Acts.13.4; Acts.13.5; Acts.13.4-Acts.13.5, the label "Acts 13:4" says Acts.13.4
- 6.60.11.3-1495381: Logos says Acts.13.16; Acts.13.17; Acts.13.16-Acts.13.17, the label "Acts 13:16" says Acts.13.16
- 6.60.11.3-1548972: Logos says Acts.14.14; Acts.14.15; Acts.14.14-Acts.14.15, the label "Acts 14:14" says Acts.14.14
- 6.60.11.3-1591889: Logos says Acts.15.13; Acts.15.15, the label "Acts 15:13" says Acts.15.13
- 6.60.11.3-1644497: Logos says Acts.16.13; Acts.16.14; Acts.16.13-Acts.16.14, the label "Acts 16:13" says Acts.16.13
- 6.60.11.3-1662468: Logos says Acts.16.25; Acts.16.26; Acts.16.25-Acts.16.26, the label "Acts 16:25" says Acts.16.25
- 6.60.11.3-1682049: Logos says Acts.17.1; Acts.17.2; Acts.17.3; Acts.17.1-Acts.17.3, the label "Acts 17:1" says Acts.17.1
- 6.60.11.3-1701215: Logos says Acts.17.16; Acts.17.17; Acts.17.16-Acts.17.17, the label "Acts 17:16" says Acts.17.16
- 6.60.11.3-1729399: Logos says Acts.17.32-Acts.17.34; Acts.18.1, the label "Acts 17:32–34" says Acts.17.32-Acts.17.34
- 6.60.11.3-1772120: Logos says Acts.19.8; Acts.19.9; Acts.19.8-Acts.19.9, the label "Acts 19:8" says Acts.19.8
- 6.60.11.3-1802137: Logos says Acts.19.21; Acts.19.23, the label "Acts 19:21" says Acts.19.21
- 6.60.11.3-1891360: Logos says Acts.21.18; Acts.21.19; Acts.21.18-Acts.21.19, the label "Acts 21:18" says Acts.21.18
- 6.60.11.3-1909284: Logos says Acts.21.39; Acts.21.40; Acts.21.39-Acts.21.40, the label "Acts 21:39" says Acts.21.39
- 6.60.11.3-1971700: Logos says Acts.23.31; Acts.23.32; Acts.23.33; Acts.23.31-Acts.23.33, the label "Acts 23:31" says Acts.23.31
- 6.60.11.3-1997454: Logos says Acts.24.22; Acts.24.23; Acts.24.22-Acts.24.23, the label "Acts 24:22" says Acts.24.22
- 6.60.13.4-267181: Logos says Gal.1.1-Gal.1.3; Gal.1, the label "Galatians 1:1–3" says Gal.1.1-Gal.1.3
- 6.60.13.4-329308: Logos says Gal.2.1-Gal.2.2; Gal.2, the label "Galatians 2:1–2" says Gal.2.1-Gal.2.2
- 6.60.13.4-374209: Logos says Gal.3.1; Gal.3, the label "Galatians 3:1" says Gal.3.1
- 6.60.13.4-403847: Logos says Gal.4.1-Gal.4.3; Gal.4, the label "Galatians 4:1–3" says Gal.4.1-Gal.4.3
- 6.60.13.4-427808: Logos says Gal.5.1; Gal.5, the label "Galatians 5:1" says Gal.5.1
- 6.60.13.4-460454: Logos says Gal.6.1; Gal.6, the label "Galatians 6:1" says Gal.6.1
- 6.60.13.4-580342: Logos says Eph.2.11; Eph.2.12, the label "Ephesians 2:11" says Eph.2.11
- 6.60.13.4-646046: Logos says Eph.4.1; Eph.4.2, the label "Ephesians 4:1" says Eph.4.1
- 6.60.13.4-847687: Logos says Eph.4.31; Eph.4.32, the label "Ephesians 4:31" says Eph.4.31
- 6.60.13.4-862912: Logos says Eph.4.32; Eph.5.1; Eph.5.2, the label "Ephesians 4:32" says Eph.4.32
- 6.60.13.4-880326: Logos says Eph.5.5; Eph.5.6, the label "Ephesians 5:5" says Eph.5.5
- 6.60.13.4-902681: Logos says Eph.5.15; Eph.5.16; Eph.5.17, the label "Ephesians 5:15" says Eph.5.15
- 6.60.13.4-1110459: Logos says Phil.1.1; Phil.1.2, the label "Philippians 1:1" says Phil.1.1
- 6.60.13.4-1384928: Logos says Phil.3.13; Phil.3.14, the label "Philippians 3:13" says Phil.3.13
- 6.60.13.4-1468867: Logos says Col.1.1; Col.1.2, the label "Colossians 1:1" says Col.1.1
- 6.60.13.4-1499705: Logos says Col.1.9; Col.1.10, the label "Colossians 1:9" says Col.1.9
- 6.60.13.4-1598527: Logos says Col.2.6; Col.2.7, the label "Colossians 2:6" says Col.2.6
- 6.60.13.4-1675764: Logos says Col.3.16; Col.3.17, the label "Colossians 3:16" says Col.3.16
- 6.60.13.4-1719908: Logos says Col.4.5; Col.4.6, the label "Colossians 4:5" says Col.4.5
- 6.60.13.4-1743816: Logos says Col.4.12; Col.4.13, the label "Colossians 4:12" says Col.4.12
- 6.60.13.4-1900927: Logos says 1Thess.4.9; 1Thess.4.10, the label "1 Thessalonians 4:9" says 1Thess.4.9
- 6.60.13.4-1960839: Logos says 1Thess.5.1; 1Thess.5.2, the label "1 Thessalonians 5:1" says 1Thess.5.1
- 6.60.13.4-1989634: Logos says 1Thess.5.12; 1Thess.5.13, the label "1 Thessalonians 5:12" says 1Thess.5.12
- 6.60.13.4-2047133: Logos says 2Thess.1.1; 2Thess.1.2, the label "2 Thessalonians 1:1" says 2Thess.1.1
- 6.60.13.4-2071676: Logos says 2Thess.1.9; 2Thess.1.10, the label "2 Thessalonians 1:9" says 2Thess.1.9
- 6.60.13.4-2161293: Logos says 1Tim.1.1; 1Tim.1.2, the label "1 Timothy 1:1" says 1Tim.1.1
- 6.60.13.4-2217078: Logos says 1Tim.1.15; 1Tim.1.16, the label "1 Timothy 1:15" says 1Tim.1.15
- 6.60.13.4-2235702: Logos says 1Tim.1.18; 1Tim.1.19, the label "1 Timothy 1:18" says 1Tim.1.18
- 6.60.13.4-2481478: Logos says 2Tim.1.1; 2Tim.1.2, the label "2 Timothy 1:1" says 2Tim.1.1
- 6.60.13.4-2583840: Logos says 2Tim.2.20; 2Tim.2.21, the label "2 Timothy 2:20" says 2Tim.2.20
- 6.60.13.4-2652335: Logos says 2Tim.3.16; 2Tim.3.17, the label "2 Timothy 3:16" says 2Tim.3.16
- 6.60.13.4-2718401: Logos says Titus.1.5; Titus.1.6, the label "Titus 1:5" says Titus.1.5
- 6.60.14.3-499018: Logos says John.1.28; John.1.29, the label "John 1:28" says John.1.28
- 6.60.14.3-541277: Logos says John.1.41; John.1.42, the label "John 1:41" says John.1.41
- 6.60.14.3-553917: Logos says John.1.43; John.1.44, the label "John 1:43" says John.1.43
- 6.60.14.3-567670: Logos says John.1.49; John.1.50, the label "John 1:49" says John.1.49
- 6.60.14.3-667582: Logos says John.3.12; John.3.13, the label "John 3:12" says John.3.12
- 6.60.14.3-730377: Logos says John.3.35; John.3.36, the label "John 3:35" says John.3.35
- 6.60.14.3-757449: Logos says John.4.13; John.4.14, the label "John 4:13" says John.4.13
- 6.60.14.3-775614: Logos says John.4.21; John.4.22, the label "John 4:21" says John.4.21
- 6.60.14.3-791698: Logos says John.4.28; John.4.29, the label "John 4:28" says John.4.28
- 6.60.14.3-826146: Logos says John.4.54; John.5.1, the label "John 4:54" says John.4.54
- 6.60.14.3-839262: Logos says John.5.6; John.5.7, the label "John 5:6" says John.5.6
- 6.60.14.3-882309: Logos says John.5.23; John.5.24, the label "John 5:23" says John.5.23
- 6.60.14.3-914953: Logos says John.5.31; John.5.32, the label "John 5:31" says John.5.31
- 6.60.14.3-935061: Logos says John.5.39; John.5.40, the label "John 5:39" says John.5.39
- 6.60.14.3-949867: Logos says John.6.1; John.6.4, the label "John 6:1" says John.6.1
- 6.60.14.3-981218: Logos says John.6.26; John.6.27, the label "John 6:26" says John.6.26
- 6.60.14.3-1014123: Logos says John.6.41; John.6.42, the label "John 6:41" says John.6.41
- 6.60.14.3-1032008: Logos says John.6.53; John.6.54, the label "John 6:53" says John.6.53
- 6.60.14.3-1056630: Logos says John.7.1; John.7.2, the label "John 7:1" says John.7.1
- 6.60.14.3-1070236: Logos says John.7.9; John.7.10, the label "John 7:9" says John.7.9
- 6.60.14.3-1105459: Logos says John.7.37; John.7.38, the label "John 7:37" says John.7.37
- 6.60.14.3-1120638: Logos says John.7.45; John.7.46, the label "John 7:45" says John.7.45
- 6.60.14.3-1153342: Logos says John.8.31; John.8.32, the label "John 8:31" says John.8.31
- 6.60.14.3-1173089: Logos says John.8.48; John.8.49, the label "John 8:48" says John.8.48
- 6.60.14.3-1186864: Logos says John.9.1; John.9.2, the label "John 9:1" says John.9.1
- 6.60.14.3-1204740: Logos says John.9.6; John.9.7, the label "John 9:6" says John.9.6
- 6.60.14.3-1219968: Logos says John.9.17; John.9.18, the label "John 9:17" says John.9.17
- 6.60.14.3-1265596: Logos says John.10.14; John.10.15, the label "John 10:14" says John.10.14
- 6.60.14.3-1316668: Logos says John.11.1; John.11.2, the label "John 11:1" says John.11.1
- 6.60.14.3-1339688: Logos says John.11.30; John.11.31, the label "John 11:30" says John.11.30
- 6.60.14.3-1357986: Logos says John.11.41; John.11.42, the label "John 11:41" says John.11.41
- 6.60.14.3-1381761: Logos says John.11.49; John.11.50, the label "John 11:49" says John.11.49
- 6.60.14.3-1414289: Logos says John.12.25; John.12.26, the label "John 12:25" says John.12.25
- 6.60.14.3-1440504: Logos says John.12.42; John.12.43, the label "John 12:42" says John.12.42
- 6.60.14.3-1519856: Logos says John.14.8; John.14.9, the label "John 14:8" says John.14.8
- 6.60.14.3-1556771: Logos says John.14.31; John.15.1, the label "John 14:31" says John.14.31
- 6.60.14.3-1571451: Logos says John.15.11; John.15.12, the label "John 15:11" says John.15.11
- 6.60.14.3-1616237: Logos says John.16.16; John.16.17, the label "John 16:16" says John.16.16
- 6.60.14.3-1765413: Logos says John.20.10; John.20.11, the label "John 20:10" says John.20.10
- 6.60.14.3-1784662: Logos says John.20.24; John.20.25, the label "John 20:24" says John.20.24
- 6.60.14.3-1920636: Logos says Heb.1.1; Heb.1.2, the label "Hebrews 1:1" says Heb.1.1
- 6.60.14.3-2029159: Logos says Heb.2.16; Heb.2.17, the label "Hebrews 2:16" says Heb.2.16
- 6.60.14.3-2153911: Logos says Heb.6.7; Heb.6.8, the label "Hebrews 6:7" says Heb.6.7
- 6.60.14.3-2244716: Logos says Heb.8.1; Heb.8.2, the label "Hebrews 8:1" says Heb.8.1
- 6.60.14.3-2355894: Logos says Heb.10.26; Heb.10.27, the label "Hebrews 10:26" says Heb.10.26
- 6.60.14.3-2393387: Logos says Heb.11.3; Heb.11.4, the label "Hebrews 11:3" says Heb.11.3
- 6.60.14.3-2519529: Logos says Heb.11.37; Heb.11.38, the label "Hebrews 11:37" says Heb.11.37
- 6.60.14.3-2628679: Logos says Heb.12.28; Heb.12.29, the label "Hebrews 12:28" says Heb.12.28

## Rows with no ref (4)

Neither a Logos reference nor a label that could be read: the row is kept, found by its title and its source, and by no passage. Ids, in the file's order:

6.60.10.3-235067, 6.60.11.3-3054772, 6.60.13.4-2034764, 6.60.14.3-220151
