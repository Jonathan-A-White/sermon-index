# CCEL Spurgeon: what the ThML does not settle

Written by `go run ./tools/ccel-spurgeon --cache <dir> --out sources/spurgeon-ccel --report tools/ccel-spurgeon/REPORT.md`; do not edit by hand. Only numbers, titles and references are listed.

## Counts

```
sermon entries read:                   3539
distinct sermon numbers:               3531
rows:                                  3543
rows with a preaching text that resolves: 3467
  of them in Romans 5:                 22
rows of kind exposition:               5
rows with a date:                      2734
numbers with two preaching texts:      53
entries with none (not an exposition): 71
```

## Two or more preaching texts (53)

The ThML tags more than one `scripCom type="Sermon"`. The row's `ref` is the first; the others are `cited`.

- 17
- 234
- 235
- 249
- 431
- 434
- 441
- 633
- 634
- 664
- 681
- 702
- 792
- 796
- 809
- 973
- 975
- 977
- 991
- 1011
- 1027
- 1127
- 1162
- 1216
- 1336
- 1358
- 1485
- 1569
- 1606
- 1645
- 1698
- 1814
- 1947
- 1959
- 2115
- 2229
- 2310
- 2376
- 2378
- 2412
- 2490
- 2527
- 2548
- 2631
- 2726
- 2739
- 2793
- 2820
- 2831
- 2928
- 2931
- 3075
- 3303

## No preaching text that resolves (71)

Not an exposition, and the ThML has no `scripCom type="Sermon"` with a reading Lampas's book table accepts (a meeting, an address, a dedication, an empty or misspelt field). The row has no `ref`.

- 12 The Peculiar Sleep of the Beloved
- 82 The New Park Street Tracts
- 113 Confession of Sin—A Sermon With Seven Texts
- 154 Fast-Day Service
- 251 The Necessity of the Spirit’s Work
- 268 The Ceremony of Laying the First Stone of the New Tabernacle
- 371 Opening Services
- 372 Meeting Of The Neighboring Churches
- 376 Public Meeting Of Our London Baptist Brethren
- 377 Public Meeting Of The Various Denominations
- 380 Meeting Of Our Own Church
- 382 The Last Census
- 416 The Shield Of Faith
- 417 Scourge For Slumbering Souls
- 460 Faith and Repentance Inseparable
- 478 Christ—Perfect Through Sufferings
- 483 Life And Walk Of Faith
- 490 Gracious Renewal
- 505 The Root Of The Matter
- 524 The Saint'S Horror At The Sinner'S Hell
- 537 Encourage Your Minister!
- 703 Salvation Altogether By Grace
- 793 Nearer and Dearer
- 986 Victor Emmanuel, Emancipator
- 1054 "Waters To Swim In"
- 1328 Christ the Maker of All Things New
- 1448 Forgiveness Made Easy
- 1467 The Oil and the Vessels
- 1472 The Glory, Unity and Triumph of the Church
- 1638 Men Without Heart, Sight, or Hearing
- 1681 Shutting, Sealing and Covering—or, Messiah's Glorious Work
- 1687 The Law Written in the Heart
- 1738 "Glory Be Unto the Father"
- 1816 Sermon for New Year's-Day
- 1817 The Ascension and the Second Advent Practically Considered
- 1819 A Sweet Silver Bell Ringing in Each Believer's Heart
- 1820 Fallen Angels a Lesson To Fallen Men
- 1821 Cords and Cart-Ropes
- 1824 The History of Sundry Fools
- 1837 A Great Gospel for Great Sinners
- 1861 The Lowly King
- 1864 First Things First
- 2050 A Paradox
- 2067 The Spirit and the Wind
- 2107 The Withered Fig Tree
- 2125 Hope for Your Future
- 2143 The Shining of the Face of Moses
- 2198 The Census of Israel
- 2443 The Determination of Christ to Suffer for His People
- 2655 Decided Ungodliness
- 2683 The Bitterness of the Cross
- 2774 Love Pledging Fidelity
- 2794 A Refreshing Canticle
- 2822 Christ in Bonds
- 2970 God's Jewels
- 3006 'The Lord Is My Shepherd'
- 3007 Gadding About
- 3010 Jacob's Model Prayer
- 3044 Spiritual Transformations
- 3119 Mistakes Concerning God
- 3217 An Earnest Warning Against Unbelief
- 3230 The Last Message for the Year
- 3249 Under the Apple Tree
- 3307 Over the Mountains
- 3323 The Believer's Glad Prospects
- 3347 Things to Be Remembered
- 3494 Divine Destruction and Protection
- 3536 A Threefold Slogan
- 3552 The Soul's Desertion
- 3554 Our Magnificent Savior
- 3556 Absconding and Apostasy

## A preaching text given but not resolved (28)

The `scripCom` is there and its `passage` is shown (empty, or misspelt); these are among the list above.

- 416 ""
- 417 ""
- 483 ""
- 490 ""
- 505 ""
- 524 "Psalm 269"
- 537 ""
- 703 "2 Timothy 19"
- 793 "Solomon 5:2-8"
- 1054 "EzekieI47:5."
- 1328 "2 Corinthians, 5:17."
- 1448 "Ephesians 4:82."
- 1472 "John 17:22-3."
- 2143 "Exodus 34:29-36."
- 2198 "Numbers 16:66."
- 2683 "Zechaariah 12:10."
- 2794 "Solomon's Song 1:4."
- 2970 "Maalachi 3:17."
- 3044 "Isaiah 4:13."
- 3119 "Psalm 1:21."
- 3217 "Hebrews 8:18,19."
- 3249 "Solomon's Song 2:3."
- 3307 "Solomon's Song 2:16,17."
- 3323 "Solomon's Song 2:17."
- 3347 "Psalm 38:(Title)."
- 3494 "Ezekiel 17:29."
- 3536 "Luke 10:44."
- 3552 "Solomon's Song 5:6."

## Tagged references the checker refuses (26)

The ThML tags a reference that does not exist in Lampas's numbering (a typing slip in the source: a verse past the chapter, a range that ends before it starts). It is left out of the row.

- 59: ref 'Ps.90.1-Ps.90.22': verse past the chapter
- 385-388: ref '1John.10.21': chapter past the book
- 524: ref 'Ps.269': chapter past the book
- 703: ref '2Tim.19': chapter past the book
- 1169: ref 'John.116': chapter past the book
- 1448: ref 'Eph.4.82': verse past the chapter
- 1472: ref 'John.17.22-John.17.3': the range ends before it starts
- 1545: ref '1John.8.44': chapter past the book
- 1621: ref 'Rev.1119': chapter past the book
- 1938: ref 'Ps.1.15': verse past the chapter
- 2143: ref 'Exod.34.29-Exod.34.36': verse past the chapter
- 2198: ref 'Num.16.66': verse past the chapter
- 2376: ref 'Matt.16.39': verse past the chapter
- 2377: ref 'John.131': chapter past the book
- 2583: ref 'Isa.4.8-Isa.4.20': verse past the chapter
- 2774: ref '1Sam.11.17': verse past the chapter
- 2793: ref 'Isa.45.26': verse past the chapter
- 3011: ref 'Num.519': chapter past the book
- 3044: ref 'Isa.4.13': verse past the chapter
- 3045: ref 'Rom.11.1136': verse past the chapter
- 3119: ref 'Ps.1.21': verse past the chapter
- 3217: ref 'Heb.8.18-Heb.8.19': verse past the chapter
- 3217: ref 'Heb.8.18': verse past the chapter
- 3217: ref 'Heb.8.19': verse past the chapter
- 3494: ref 'Ezek.17.29': verse past the chapter
- 3536: ref 'Luke.10.44': verse past the chapter

## Combined entries (2)

One CCEL page for several numbers: one row each, the same `url`.

- 297-298
- 385-388

## A number with more than one entry (12)

CCEL has two entries with the number; each has its own row, the id told here.

- 1331: ids 1331, 1331-2
- 1451: ids 1451A, 1451B
- 1452: ids 1452A, 1452B
- 1454: ids 1454A, 1454B
- 1457: ids 1457A, 1457B
- 1459: ids 1459A, 1459B
- 1461: ids 1461A, 1461B
- 1463: ids 1463A, 1463B
- 1465: ids 1465A, 1465B
- 1467: ids 1467A, 1467B
- 1634: ids 1634A, 1634B
- 2237: ids 2237, 2237-2

