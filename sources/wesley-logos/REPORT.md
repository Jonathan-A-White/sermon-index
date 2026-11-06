# wesley-logos: what the import did not settle

<!-- toc: none -->

Written by `go run ./tools/logos-import --in <vault>/plans/sermons-2026-11-05/sermons-on-several-occasions-sermons.jsonl --in <vault>/plans/sermons-2026-11-05/the-works-of-john-wesley-volume-5-sermons.jsonl --in <vault>/plans/sermons-2026-11-05/the-works-of-john-wesley-volume-6-sermons.jsonl --in <vault>/plans/sermons-2026-11-05/the-works-of-john-wesley-volume-7-sermons.jsonl --dedupe-titles --id wesley-logos --out sources/wesley-logos`; do not edit by hand. Only ids, titles, references and addresses are listed.

## Counts

```
lines read:                          281
rows written:                        164
lines dropped:                       0
rows with a ref:                     137
  of them read from the label:       0
  of them with cited references:     122
rows with no ref:                    27
series rows (ref over >3 chapters):  0
rows with their own date:            0
rows dated by the year alone:        0
references the book table refused:   2
Logos and the label disagree:        3
duplicates dropped (earlier input):  117
```

## Rows before and after, by input

A line whose title (lower case, a leading Roman numeral or "Sermon N" taken off, marks ignored) is the title of a row of an earlier input is dropped as a repeat. "Before" is the rows each input gives with no title taken out, "after" the rows written.

| input | lines | rows before | duplicates dropped | rows after |
|---|---|---|---|---|
| sermons-on-several-occasions | 141 | 141 | 0 | 141 |
| the-works-of-john-wesley-volume-5 | 39 | 39 | 22 | 17 |
| the-works-of-john-wesley-volume-6 | 46 | 46 | 44 | 2 |
| the-works-of-john-wesley-volume-7 | 55 | 55 | 51 | 4 |
| **all** | 281 | 281 | 117 | 164 |

## Lines dropped (0)

A line with neither a ref nor a title, or with no id, no title, no address or an id already used, is not a row.

None.

## Duplicates dropped (117)

The title is the title of a row of an earlier input (the first input is the one kept).

- WORKSWESLEY05.2-138047: "Sermon I: Salvation by Faith" repeats 36.0.11.2-45431
- WORKSWESLEY05.2-160727: "Sermon II: The Almost Christian" repeats 36.0.11.2-67994
- WORKSWESLEY05.2-179509: "Sermon III: Awake, Thou that Sleepest" repeats 36.0.11.2-86826
- WORKSWESLEY05.2-206474: "Sermon IV: Scriptural Christianity" repeats 36.0.11.2-113755
- WORKSWESLEY05.2-244131: "Sermon V: Justification by Faith" repeats 36.0.11.2-151344
- WORKSWESLEY05.2-272050: "Sermon VI: The Righteousness of Faith" repeats 36.0.11.2-179370
- WORKSWESLEY05.2-299486: "Sermon VII: The Way to the Kingdom" repeats 36.0.11.2-206797
- WORKSWESLEY05.2-323908: "Sermon VIII: The First Fruits of the Spirit" repeats 36.0.11.2-231177
- WORKSWESLEY05.2-350143: "Sermon IX: The Spirit of Bondage and of Adoption" repeats 36.0.11.2-257386
- WORKSWESLEY05.2-436681: "Sermon XII: The Witness of Our Own Spirit" repeats 36.0.11.2-343946
- WORKSWESLEY05.2-460638: "Sermon XIII: On Sin in Believers" repeats 36.0.11.2-367637
- WORKSWESLEY05.2-488695: "Sermon XIV: The Repentance of Believers" repeats 36.0.11.2-395874
- WORKSWESLEY05.2-522034: "Sermon XV: The Great Assize" repeats 36.0.11.2-429197
- WORKSWESLEY05.2-555346: "Sermon XVI: The Means of Grace" repeats 36.0.11.2-462696
- WORKSWESLEY05.2-594442: "Sermon XVII: The Circumcision of the Heart" repeats 36.0.11.2-501747
- WORKSWESLEY05.2-619380: "Sermon XVIII: The Marks of the New Birth" repeats 36.0.11.2-526569
- WORKSWESLEY05.2-645441: "Sermon XIX: The Great Privilege of Those that Are Born of God" repeats 36.0.11.2-552634
- WORKSWESLEY05.2-669282: "Sermon XX: The Lord Our Righteousness" repeats 36.0.11.2-576477
- WORKSWESLEY05.2-1134862: "Sermon XXXIV: The Original, Nature, Property, and Use of the Law" repeats 36.0.11.2-1041725
- WORKSWESLEY05.2-1213651: "Sermon XXXVII: The Nature of Enthusiasm" repeats 36.0.11.2-1120448
- WORKSWESLEY05.2-1242203: "Sermon XXXVIII: A Caution against Bigotry" repeats 36.0.11.2-1149015
- WORKSWESLEY05.2-1273459: "Sermon XXXIX: Catholic Spirit" repeats 36.0.11.2-1180245
- WORKSWESLEY06.2-56290: "Sermon XLI: Wandering Thoughts" repeats 36.0.11.2-1262088
- WORKSWESLEY06.2-78220: "Sermon XLII: Satan’s Devices" repeats 36.0.11.2-1284016
- WORKSWESLEY06.2-104824: "Sermon XLIII: The Scripture Way of Salvation" repeats 36.0.11.2-1310572
- WORKSWESLEY06.2-130704: "Sermon XLIV: Original Sin" repeats 36.0.11.2-1336404
- WORKSWESLEY06.2-156896: "Sermon XLV: The New Birth" repeats 36.0.11.2-1362359
- WORKSWESLEY06.2-185633: "Sermon XLVI: The Wilderness State" repeats 36.0.11.2-1391035
- WORKSWESLEY06.2-219585: "Sermon XLVII: Heaviness through Manifold Temptations" repeats 36.0.11.2-1424908
- WORKSWESLEY06.2-247608: "Sermon XLVIII: Self-Denial" repeats 36.0.11.2-1452936
- WORKSWESLEY06.2-273534: "Sermon XLIX: The Cure of Evil-Speaking" repeats 36.0.11.2-1478861
- WORKSWESLEY06.2-297514: "Sermon L: The Use of Money" repeats 36.0.11.2-1502764
- WORKSWESLEY06.2-325936: "Sermon LI: The Good Steward" repeats 36.0.11.2-1531118
- WORKSWESLEY06.2-439837: "Sermon LIV: On Eternity" repeats 36.0.11.2-1640946
- WORKSWESLEY06.2-462091: "Sermon LV: On the Trinity" repeats 36.0.11.2-1662914
- WORKSWESLEY06.2-478368: "Sermon LVI: God’s Approbation of His Works" repeats 36.0.11.2-1678787
- WORKSWESLEY06.2-498451: "Sermon LVII: On the Fall of Man" repeats 36.0.11.2-1698880
- WORKSWESLEY06.2-521420: "Sermon LVIII: On Predestination" repeats 36.0.11.2-1721890
- WORKSWESLEY06.2-535274: "Sermon LIX: God’s Love to Fallen Man" repeats 36.0.11.2-1735602
- WORKSWESLEY06.2-558614: "Sermon LX: The General Deliverance" repeats 36.0.11.2-1758858
- WORKSWESLEY06.2-586897: "Sermon LXI: The Mystery of Iniquity" repeats 36.0.11.2-1787138
- WORKSWESLEY06.2-620738: "Sermon LXII: The End of Christ’s Coming" repeats 36.0.11.2-1821536
- WORKSWESLEY06.2-644189: "Sermon LXIII: The General Spread of the Gospel" repeats 36.0.11.2-1845016
- WORKSWESLEY06.2-670798: "Sermon LXIV: The New Creation" repeats 36.0.11.2-1871603
- WORKSWESLEY06.2-688434: "Sermon LXV: The Duty of Reproving Our Neighbour" repeats 36.0.11.2-1889245
- WORKSWESLEY06.2-706416: "Sermon LXVI: The Signs of the Times" repeats 36.0.11.2-1907292
- WORKSWESLEY06.2-727742: "Sermon LXVII: On Divine Providence" repeats 36.0.11.2-1928653
- WORKSWESLEY06.2-756328: "Sermon LXVIII: The Wisdom of God’s Counsels" repeats 36.0.11.2-1957192
- WORKSWESLEY06.2-784500: "Sermon LXIX: The Imperfection of Human Knowledge" repeats 36.0.11.2-1985357
- WORKSWESLEY06.2-816697: "Sermon LXX: The Case of Reason Impartially Considered" repeats 36.0.11.2-2017280
- WORKSWESLEY06.2-840667: "Sermon LXXI: Of Good Angels" repeats 36.0.11.2-2041161
- WORKSWESLEY06.2-861632: "Sermon LXXII: Of Evil Angels" repeats 36.0.11.2-2062126
- WORKSWESLEY06.2-886864: "Sermon LXXIII: Of Hell" repeats 36.0.11.2-2087347
- WORKSWESLEY06.2-911644: "Sermon LXXIV: Of the Church" repeats 36.0.11.2-2112130
- WORKSWESLEY06.2-933099: "Sermon LXXV: On Schism" repeats 36.0.11.2-2133734
- WORKSWESLEY06.2-955684: "Sermon LXXVI: On Perfection" repeats 36.0.11.2-2156316
- WORKSWESLEY06.2-987795: "Sermon LXXVII: Spiritual Worship" repeats 36.0.11.2-2188514
- WORKSWESLEY06.2-1011540: "Sermon LXXVIII: Spiritual Idolatry" repeats 36.0.11.2-2212379
- WORKSWESLEY06.2-1032659: "Sermon LXXIX: On Dissipation" repeats 36.0.11.2-2233491
- WORKSWESLEY06.2-1050834: "Sermon LXXX: On Friendship with the World" repeats 36.0.11.2-2251817
- WORKSWESLEY06.2-1078440: "Sermon LXXXI: In What Sense We Are to Leave the World" repeats 36.0.11.2-2279440
- WORKSWESLEY06.2-1105708: "Sermon LXXXII: On Temptation" repeats 36.0.11.2-2306701
- WORKSWESLEY06.2-1127449: "Sermon LXXXIII: On Patience" repeats 36.0.11.2-2328489
- WORKSWESLEY06.2-1146769: "Sermon LXXXIV: The Important Question" repeats 36.0.11.2-2347885
- WORKSWESLEY06.2-1175705: "Sermon LXXXV: On Working out Our Own Salvation" repeats 36.0.11.2-2377003
- WORKSWESLEY06.2-1194507: "Sermon LXXXVI: A Call to Backsliders" repeats 36.0.11.2-2395822
- WORKSWESLEY07.2-14130: "Sermon LXXXVII: The Danger of Riches" repeats 36.0.11.2-2427719
- WORKSWESLEY07.2-46396: "Sermon LXXXVIII: On Dress" repeats 36.0.11.2-2460132
- WORKSWESLEY07.2-71579: "Sermon LXXXIX: The More Excellent Way" repeats 36.0.11.2-2485463
- WORKSWESLEY07.2-98262: "Sermon XC: An Israelite Indeed" repeats 36.0.11.2-2512080
- WORKSWESLEY07.2-116656: "Sermon XCI: On Charity" repeats 36.0.11.2-2530475
- WORKSWESLEY07.2-144085: "Sermon XCII: On Zeal" repeats 36.0.11.2-2557979
- WORKSWESLEY07.2-167377: "Sermon XCIII: On Redeeming the Time" repeats 36.0.11.2-2581266
- WORKSWESLEY07.2-187630: "Sermon XCIV: On Family Religion" repeats 36.0.11.2-2601517
- WORKSWESLEY07.2-211814: "Sermon XCV: On the Education of Children" repeats 36.0.11.2-2625264
- WORKSWESLEY07.2-241314: "Sermon XCVI: On Obedience to Parents" repeats 36.0.11.2-2654694
- WORKSWESLEY07.2-264378: "Sermon XCVII: On Obedience to Pastors" repeats 36.0.11.2-2677903
- WORKSWESLEY07.2-284905: "Sermon XCVIII: On Visiting the Sick" repeats 36.0.11.2-2698402
- WORKSWESLEY07.2-334852: "Sermon C: On Pleasing All Men" repeats 36.0.11.2-2748865
- WORKSWESLEY07.2-353346: "Sermon CI: The Duty of Constant Communion" repeats 36.0.11.2-2767350
- WORKSWESLEY07.2-377844: "Sermon CII: Of Former Times" repeats 36.0.11.2-2791568
- WORKSWESLEY07.2-399772: "Sermon CIII: What Is Man?" repeats 36.0.11.2-2813847
- WORKSWESLEY07.2-416638: "Sermon CIV: On Attending the Church Service" repeats 36.0.11.2-2830526
- WORKSWESLEY07.2-444087: "Sermon CV: On Conscience" repeats 36.0.11.2-2858030
- WORKSWESLEY07.2-465153: "Sermon CVI: On Faith" repeats 36.0.11.2-2879032
- WORKSWESLEY07.2-482416: "Sermon CVII: On God’s Vineyard" repeats 36.0.11.2-2896268
- WORKSWESLEY07.2-509117: "Sermon CVIII: On Riches" repeats 36.0.11.2-2922977
- WORKSWESLEY07.2-528803: "Sermon CIX: What Is Man?" repeats 36.0.11.2-2813847
- WORKSWESLEY07.2-542700: "Sermon CX: On the Discoveries of Faith" repeats 36.0.11.2-2956345
- WORKSWESLEY07.2-559809: "Sermon CXI: On the Omnipresence of God" repeats 36.0.11.2-2973424
- WORKSWESLEY07.2-574273: "Sermon CXII: The Rich Man and Lazarus" repeats 36.0.11.2-2987869
- WORKSWESLEY07.2-599979: "Sermon CXIII.: The Difference between Walking by Sight, and Walking by Faith" repeats 36.0.11.2-3014197
- WORKSWESLEY07.2-619693: "Sermon CXIV: The Unity of the Divine Being" repeats 36.0.11.2-3033948
- WORKSWESLEY07.2-640214: "Sermon CXV: The Ministerial Office" repeats 36.0.11.2-3054484
- WORKSWESLEY07.2-658183: "Sermon CXVI: Causes of the Inefficacy of Christianity" repeats 36.0.11.2-3073847
- WORKSWESLEY07.2-679523: "Sermon CXVII: On Knowing Christ after the Flesh" repeats 36.0.11.2-3095170
- WORKSWESLEY07.2-693351: "Sermon CXVIII: On a Single Eye" repeats 36.0.11.2-3108952
- WORKSWESLEY07.2-712603: "Sermon CXIX: On Worldly Folly" repeats 36.0.11.2-3128191
- WORKSWESLEY07.2-725250: "Sermon CXX: On the Wedding Garment" repeats 36.0.11.2-3140843
- WORKSWESLEY07.2-740908: "Sermon CXXI: Human Life a Dream" repeats 36.0.11.2-3156519
- WORKSWESLEY07.2-758502: "Sermon CXXII: On Faith" repeats 36.0.11.2-2879032
- WORKSWESLEY07.2-799059: "Sermon CXXIV: The Heavenly Treasure in Earthen Vessels" repeats 36.0.11.2-3214842
- WORKSWESLEY07.2-810468: "Sermon CXXV: On Living without God" repeats 36.0.11.2-3226236
- WORKSWESLEY07.2-823984: "Sermon CXXVI: On the Danger of Increasing Riches" repeats 36.0.11.2-3239701
- WORKSWESLEY07.2-843073: "Sermon CXXVII: The Trouble and Rest of Good Men" repeats 36.0.11.2-3258578
- WORKSWESLEY07.2-859803: "Sermon CXXVIII: Free Grace" repeats 36.0.11.2-3276325
- WORKSWESLEY07.2-891946: "Sermon CXXIX: The Cause and Cure of Earthquakes" repeats 36.0.11.2-3308225
- WORKSWESLEY07.2-923375: "Sermon CXXX: National Sins and Miseries" repeats 36.0.11.2-3339533
- WORKSWESLEY07.2-968482: "Sermon CXXXII: On Laying the Foundation of the New Chapel, near the City-Road, London" repeats 36.0.11.2-3384541
- WORKSWESLEY07.2-1039023: "Sermon CXXXIV: True Christianity Defended" repeats 36.0.11.2-3453757
- WORKSWESLEY07.2-1064237: "Sermon CXXXV: On Mourning for the Dead" repeats 36.0.11.2-3478445
- WORKSWESLEY07.2-1077195: "Sermon CXXXVI: On Corrupting the Word of God" repeats 36.0.11.2-3491308
- WORKSWESLEY07.2-1089681: "Sermon CXXXVII: On the Resurrection of the Dead" repeats 36.0.11.2-3504045
- WORKSWESLEY07.2-1117125: "Sermon CXXXVIII: On Grieving the Holy Spirit" repeats 36.0.11.2-3531407
- WORKSWESLEY07.2-1134333: "Sermon CXXXIX: On Love" repeats 36.0.11.2-3548580
- WORKSWESLEY07.2-1151186: "Sermon CXL: On Public Diversions" repeats 36.0.11.2-3565545
- WORKSWESLEY07.2-1170925: "Sermon CXLI: On the Holy Spirit" repeats 36.0.11.2-3585234

## References the book table refused (2)

A Logos reference to a book outside the sixty-six, or a chapter or verse past the end; it is left out of the row, and the row stays.

- 36.0.11.2-113755: "bible.43.2.13-43.2.16": Logos book 43 is not one of the 66
- 36.0.11.2-1210937: "bible.43.1.12": Logos book 43 is not one of the 66

## Series rows (0)

The ref spans more than three chapters (a whole book or a long text): the row is kept, found by its title and listed by its source, and matched by no passage.

None.

## Logos and the label disagree (3)

Both were read and differ; the row carries Logos's reference.

- 36.0.11.2-1149015: Logos says Mark.9.38; Mark.9.39, the label "Mark 9:38" says Mark.9.38
- 36.0.11.2-1721890: Logos says Rom.8.29; Rom.8.30, the label "Romans 8:29" says Rom.8.29
- 36.0.11.2-2813847: Logos says Ps.8.3; Ps.8.4, the label "Psalm 8:3" says Ps.8.3

## Rows with no ref (27)

Neither a Logos reference nor a label that could be read: the row is kept, found by its title and its source, and by no passage. Ids, in the file's order:

36.0.11.2-678281, 36.0.11.2-717210, 36.0.11.2-912230, 36.0.11.2-947882, 36.0.11.2-3276325, WORKSWESLEY05.3-382380, WORKSWESLEY05.3-410324, WORKSWESLEY05.3-698340, WORKSWESLEY05.3-733498, WORKSWESLEY05.3-771222, WORKSWESLEY05.3-810193, WORKSWESLEY05.3-847316, WORKSWESLEY05.3-888016, WORKSWESLEY05.3-925422, WORKSWESLEY05.3-965487, WORKSWESLEY05.3-1005286, WORKSWESLEY05.3-1040941, WORKSWESLEY05.3-1068276, WORKSWESLEY05.3-1087984, WORKSWESLEY05.3-1110262, WORKSWESLEY05.3-1166864, WORKSWESLEY05.3-1192378, WORKSWESLEY06.2-357057, WORKSWESLEY07.2-309563, WORKSWESLEY07.2-780028, WORKSWESLEY07.2-943737, WORKSWESLEY07.2-995269
