# Allen Nolan on YouTube: report

<!-- toc: none -->

Written by `go run ./tools/allennolan-youtube` from the YouTube Data API v3 (channel Pastor Allen Nolan, `UCSpQNTtLDeQ5Z571g9j0LRg`, @allennolan); a re-run rewrites it. Only titles, dates, links and the passage a title, a playlist's title or a description names are kept: no video, audio, caption or description text.

## Counts

- Playlists of the channel: 163; **book playlists** (named by a Bible book, or a chapter of one): 33; left out although their title names a book: 7; the rest (topical, one-off, year, podcast, shorts and clip lists): 123
- Playlist entries read: 303 (a video in two book playlists is read twice and counted once below)
- Videos found: 241
- Rows written: 116, in `rows.jsonl`
  - passage in the video's title (verses): 32
  - chapter in the video's title, in a playlist of chapters: 18
  - the chapter(s) the playlist's own title names: 49
  - the one passage of the book the description names: 17
- Skipped: 125 (every video of a book playlist is a row or is counted here)
  - no passage: 125
  - private or deleted: 0
- Share of the listed videos with a passage: 48% (116 of 241)
- Date range: 2018-11-13 to 2025-10-22
- Quota used: 40 units (1 channels.list, 39 playlists.list and playlistItems.list, 1 unit each; the daily allowance is 10,000)

## Stop rule

If fewer than 30% of the videos of the book playlists carry a passage the tool can read, it writes this report and no rows. It did not apply: 48% of 241 videos carry one.

## Book playlists

A playlist is a book series when its title names a Bible book and says it is a study of it ("Book of", "Gospel of", "Bible Study", "Verse by Verse", "Chapter") or names a chapter of it ("Isaiah 53"). Topical playlists, one-offs, years, podcasts and clip lists are out. The videos are the API's count for the playlist; a video in two of them is one row.

| Playlist id | Title | Videos | Book | Text of every video with none of its own |
| --- | --- | ---: | --- | --- |
| `PLaPig5gEoOs3xPx70PVslkkPbooxp3Tya` | The Meaning of Isaiah 53 Verse by Verse | 5 | Isa | Isa.53 |
| `PLaPig5gEoOs10eIWJ4LPfadnEGzEA5FeM` | 1 Corinthians 13 \| Pastor Allen Nolan | 3 | 1Cor | 1Cor.13 |
| `PLaPig5gEoOs3anPohVf8FuFpA8kdP5TC6` | Revelation Bible Study by Chapters | 18 | Rev | none |
| `PLaPig5gEoOs0VWOUwW8Tmj2yDpcaSkfEi` | Abraham and Issac Book of Genesis Sermons \| Pastor Allen Nolan | 1 | Gen | none |
| `PLaPig5gEoOs2zWtpz7oXVRheijkGnz4at` | Abraham: The Book of Genesis \| Pastor Allen Nolan Sermons | 3 | Gen | none |
| `PLaPig5gEoOs3UUNDXiY-ntfeqlGKNvMCR` | The Book of Ezekiel \| Pastor Allen Nolan | 1 | Ezek | none |
| `PLaPig5gEoOs1GSPd4j9orD7p52oFRT-4a` | The Book of Daniel \| Pastor Allen Nolan | 1 | Dan | none |
| `PLaPig5gEoOs358pwqn0z-IpCQVC84YIM8` | The Book of Leviticus \| Pastor Allen Nolan | 1 | Lev | none |
| `PLaPig5gEoOs0HynfQTgaGCZYcekqQ2u4a` | The Book of Exodus \| Pastor Allen Nolan | 1 | Exod | none |
| `PLaPig5gEoOs3ZGGbnVHkhHRwVVT_KtFhG` | The Word: Gospel of John Series with Pastor Allen Nolan | 2 | John | none |
| `PLaPig5gEoOs3_YvLa9aQcko0TWe75dU9n` | Book of Proverbs \| Pastor Allen Nolan Sermons | 1 | Prov | none |
| `PLaPig5gEoOs0z13rlyL5CHds_M-XY_ksA` | Isaiah 53 Verse by Verse Bible Study \| Allen Nolan | 14 | Isa | Isa.53 |
| `PLaPig5gEoOs33Jy9Qp0gKqJntdBlXD9hI` | Revelation Chapters 21 and 22 Explained \| Allen Nolan | 5 | Rev | Rev.21-Rev.22 |
| `PLaPig5gEoOs0ltcscTzFIFupnqtkITTjZ` | Revelation Chapters 19 and 20 Explained \| Allen Nolan | 8 | Rev | Rev.19-Rev.20 |
| `PLaPig5gEoOs3kCvRY1-Ey1EHZbtdyPIGS` | Revelation Chapters 17 and 18 Explained \| Allen Nolan | 3 | Rev | Rev.17-Rev.18 |
| `PLaPig5gEoOs19c5AcIG15OauB0Lb3Gh6S` | Revelation Chapters 15 and 16 Explained \| Allen Nolan | 3 | Rev | Rev.15-Rev.16 |
| `PLaPig5gEoOs1PwT1PqKGmHRMe_aE2fcY9` | Revelation Chapter 14 \| Allen Nolan | 2 | Rev | Rev.14 |
| `PLaPig5gEoOs02YfbRAHV6ZsjrqUbVDSsQ` | Revelation Chapter 13 \| Allen Nolan | 3 | Rev | Rev.13 |
| `PLaPig5gEoOs3eQlM_4iF9S6mmcnq-_Eum` | Revelation Chapter 12 \| Allen Nolan | 3 | Rev | Rev.12 |
| `PLaPig5gEoOs1tQwjPgvSDRSmpct-H_QZa` | Revelation Chapter 11 \| Allen Nolan | 2 | Rev | Rev.11 |
| `PLaPig5gEoOs2Vb0NKqdjzkQ9p0ihzPhvI` | Revelation Chapter 10 \| Allen Nolan | 2 | Rev | Rev.10 |
| `PLaPig5gEoOs3cK3-FqCDwbFQrG36_8tmN` | Revelation Chapter 9 \| Allen Nolan | 2 | Rev | Rev.9 |
| `PLaPig5gEoOs0n87ej4jSgWLMKePr0Zn79` | Revelation Chapter 8 \| Allen Nolan | 3 | Rev | Rev.8 |
| `PLaPig5gEoOs2ndPLoyQu16mYehHnqrK8a` | Revelation Chapter 7 \| Allen Nolan | 2 | Rev | Rev.7 |
| `PLaPig5gEoOs1EsAlkuQEmeuSIgHyOboB4` | Revelation Chapter 6 \| Allen Nolan | 6 | Rev | Rev.6 |
| `PLaPig5gEoOs10jLHZGgPfjiOwpt0QCtyj` | Revelation Chapter 5 \| Allen Nolan | 3 | Rev | Rev.5 |
| `PLaPig5gEoOs2GvrXy_GM4IHkYYMHrpdZC` | Revelation Chapter 4 \| Allen Nolan | 3 | Rev | Rev.4 |
| `PLaPig5gEoOs01noycyUaNt0GU-LyZQb6M` | Revelation Chapter 3 \| Allen Nolan | 4 | Rev | Rev.3 |
| `PLaPig5gEoOs0fkqT4tRIpgf26wsBuwkcp` | Revelation Chapter 2 \| Allen Nolan | 5 | Rev | Rev.2 |
| `PLaPig5gEoOs1fiitSLd58UaWvdEZYT0XM` | Revelation Chapter 1 \| Allen Nolan | 5 | Rev | Rev.1 |
| `PLaPig5gEoOs01RTVypvwb7hhVOd8e023W` | Book of Genesis Bible Study | 89 | Gen | none |
| `PLaPig5gEoOs2cBHbHE87evL8kSTGZCbe7` | The Book of Revelation Bible Study Verse by Verse | 67 | Rev | none |
| `PLaPig5gEoOs3EHYw0dw6EiMO_5Pw1af1b` | Book of Amos Bible Study Verse by Verse | 32 | Amos | none |

### Playlists that name a book and were left out

Their title names a book but says nothing of a study of it, so they are topical and their videos were not read:

- `PLUudUmRXBi9o` Genesis Mysteries Explained: Nephilim, Enoch, the Firmament & the Flood (videos: 12)
- `PLaPig5gEoOs1pTb4yiC14TSwGkUN0CtpV` Secrets of Amos Revealed with Pastor Allen Nolan (videos: 1)
- `PLaPig5gEoOs1lRLnJp4p8M_cDmDjwnFPm` Genesis: The Story of Joseph \| Pastor Allen Nolan (videos: 2)
- `PLaPig5gEoOs004kn727twWYcu94ZXX4vD` The Miraculous Sign: Isaiah's Amazing Prophecy \| Allen Nolan (videos: 2)
- `PLaPig5gEoOs2d_wWrmoVTgk1z0kH7Ch6t` Ezekiel and Revelation \| Allen Nolan (videos: 3)
- `PLaPig5gEoOs1uWR2ZFMtIaHM9Idvuq0Mw` Daniel and Revelation \| Allen Nolan (videos: 7)
- `PLaPig5gEoOs1B728r4uwVMWvn6L75KhUN` The Spiritual Gifts (Manifestation Gifts of 1 Corinthians) (videos: 15)

## What was skipped

The passage rules are conservative: a verse in the title is read; a bare chapter in a title only in a playlist of chapters (in "Genesis Bible Study 5" or "Amos 6" the number is the part of the study, not a chapter); a playlist named by a chapter ("Isaiah 53") gives it to a video that names none; a description gives the one passage of the book with verses that it names. A whole book, a summary over more than 3 chapters, a part number or a passage the book table refuses is no passage, and nothing is guessed.

- `XEocYnLTpVA` "Leaving Your First Love: The Sin Jesus Won't Tolerate: Revelation Part 2 | Pastor Allen Nolan": no passage: its title and description name none the tool can read
- `-cgr5gHf9Ow` "Why Did It Go This Far? Abraham and Isaac | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `YkfO59EbhBI` "Abraham and God's Spectacular Promise | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `R33KRrSBMM8` "Do You Want Amazing Faith? Step Out Like Abraham | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `9lG6iSOgq2Q` "How One Person's Faith Changed Everything: The Life of Abraham | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `wG0BWKjsiNM` "Ezekiel's Prophecies Decoded: What They Tell Us About What's Coming": no passage: its title and description name none the tool can read
- `d4gaUFD9w0U` "Daniel's Four Kingdoms and What Comes Next": no passage: its title and description name none the tool can read
- `JFGKv71e2so` "Most Underrated Book in the Bible? Leviticus' Powerful Purpose! | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `djweo-IBjhI` "Exodus: Biblical Principles and Amazing Facts | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `RoJVzRclCi8` "3 Must Know Facts About Jesus: The Word 2 | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `8miQag-Sr1Q` "Essential Biblical Wisdom: Book of Proverbs Bible Study | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `AW2dZ6oQ0_c` "5 Things Jesus Predicted For The End Times That Are Happening Now": no passage: its title and description name none the tool can read
- `hXABIzBTXLU` "666 Means This—Not What You've Been Taught": no passage: its title and description name none the tool can read
- `8RqbI8i0W5w` "Why the Reformation Period Matters to You | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `HEnY1RcthnM` "Synagogue of Satan Exposed | End Times Prophecy Decoded": no passage: its title and description name none the tool can read
- `h-Aq0xHQnUI` "In-Depth Genesis Bible Study: Book of Genesis Explained | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `_wOP6fjN7Cs` "Young Earth or Old Earth? Here's What the Bible Says | Genesis Study 2": no passage: its title and description name none the tool can read
- `_7dSKtSX_64` "Does the Bible Contradict Science? | Book of Genesis Bible Study 3 | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `-xxOACvYwlU` "The Genesis Big Bang Theory? | Book of Genesis Bible Study 4 | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `a62keLmgWH8` "Everything You Need to Know About the First Days of Creation: Genesis 5 with Pastor Allen Nolan": no passage: its title and description name none the tool can read
- `_2HH2RJQeK4` "Firmament Around the Earth? Book of Genesis Bible Study 6 | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `z97-OR_jh2U` "What the Bible Really Says About the Fall: Genesis Part 7 | Pastor Allen Nolan": no passage: its title and description name none the tool can read
- `gKcAqKAIh0Y` "How Does Sin Affect the World? Book of Genesis Bible Study 8 | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `FqlVQWsTOUA` "Story of Adam and Eve | Book of Genesis Bible Study 9 | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `b52ZM7vc_3s` "Eve's Changed Perception | Book of Genesis Bible Study 10 | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `0B_SKImrUlE` "What Are The Curses of God? Book of Genesis Bible Study 11 | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `cdoFH6AgWAA` "Gospel of Jesus Christ in Genesis? Genesis Bible Study 12 | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `j1LGq7DiyTo` "Curse on the Woman? Book of Genesis Bible Study 13 | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `gy5SDt8an5U` "What Was The Actual Curse on Adam? Book of Genesis 14 | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `lOlscjW02Os` "Did You Know This About the Sons of God? Book of Genesis Bible Study 15 | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `Ub7KSGIaumI` "Why the Nephilim Are More Than Just Giants | Genesis 6 Deep Dive": no passage: its title and description name none the tool can read
- `ixvMxE2-WK4` "WHY God FLOODED the Earth | Genesis Bible Study 17 | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `8TH6aKqH0hk` "What Are Demons and Fallen Angels? Genesis Explained Bible Study 18 | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `4WqsaN2miaY` "The Real Story of Noah | Book of Genesis Bible Study 19 | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `fB4vD2qXXmM` "When Did the Flood Occur? | Book of Genesis Explained Bible Study 20 | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `UsBYwKvc_Sc` "Who Was Methuselah? Genesis Explained Bible Study 21 | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `c-xOjbi4LE4` "Inside Noah's Ark | Book of Genesis Explained Bible Study 22 | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `2b54N8H1kvE` "God Remembered Noah | Book of Genesis Explained Bible Study 23 | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `Br8fm0FxJPs` "Why Bad Things Happen to Good People | Book of Genesis Episode 24 | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `y5tAizzpacs` "The Noahic Covenant Explained | Book of Genesis Episode 25 | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `2isyGaOR5xU` "Be Fruitful and Multiply | Book of Genesis Explained Bible Study 26 | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `-2ldrDiRxZU` "Capital Punishment and Christians | Book of Genesis Bible Study 27 | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `rzYTxIVNYG8` "The Tower of Babel Explained | Book of Genesis Explained Bible Study 28 | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `zDXLV_fum-A` "Forming of the Nations | Book of Genesis Explained Bible Study 29 | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `NYXFm5TO89w` "Life of the Patriarchs | Book of Genesis Explained Bible Study 30 | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `LGSRmwK0Rrg` "Lineage of Abraham | Book of Genesis Explained Bible Study 31 | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `NwB5DT7x1R0` "Abraham's Divine Calling | Book of Genesis Explained Bible Study Part 32 | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `F9NWVXxLqP0` "Going Out In Faith | Book of Genesis Explained Bible Study 33 | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `CQFMqTjNgcI` "Abraham's Big Mistake | Book of Genesis Explained Bible Study 34 | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `veb20qtqjXs` "Leaving Egypt Disgraced | Book of Genesis Explained Bible Study 35 | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `n4sKe0qcTug` "Lot's Greedy Choice | Book of Genesis Explained Bible Study 36 | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `upVP20__RQc` "Doctrine of Justification by Faith | Book of Genesis Bible Study 37 | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `Q0icdPtSK1M` "Abraham Talking to God | Book of Genesis Explained Bible Study 38 | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `z-a6rlerL_U` "Abraham and Hagar | Book of Genesis Explained Bible Study 39 | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `ZtEhSH6RGdA` "Biblical Purpose of CIRCUMCISION | Book of GENESIS Bible Study 40 | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `1psYrDp3aGw` "Sign of the Abrahamic Covenant | Book of Genesis Bible Study 41 | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `dhM2J74TBUk` "God and Angels Visit Abraham | Book of Genesis Explained Bible Study 42 | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `MZy6n91Ikos` "Judgement On Sodom and Gomorrah | Book of GENESIS Bible Study 43 | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `5bicYHzS8G8` "Sodom and Gomorrah | Book of Genesis Explained Bible Study 44 | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `YpfeQA8WJRA` "Abraham's Camp | Book of Genesis Explained Bible Study 45 | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `3G7n8x8pLDs` "Birth of Isaac | Book of Genesis Explained Bible Study 46 | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `Ekm3LaCSqoE` "Abraham and Isaac | Book of Genesis Explained Bible Study 47 | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `ZAnyGM3oFU4` "Book of Genesis Bible Study Part 48 | The Death of Sarah | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `rnQcI1kvKlw` "Book of Genesis Explained Bible Study 49 | Marriage of Isaac and Rebekah | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `1ksczckbt84` "Rebekah Becomes Isaac's Wife | Book of Genesis Explained Bible Study 50 | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `dq83C4svS2Q` "Book of Genesis Explained Bible Study 51 | End of Abraham's Life | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `N3SmmqJnvYs` "Book of Genesis Explained Bible Study 52 | Generations of Isaac | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `aNJlr-EbupA` "Calvinism Refuted: Unconditional Election | Book of Genesis Episode 53 | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `Z_A2aCdE-jw` "Book of Genesis Bible Study Part 54 | Esau and the Birthright | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `fFmiWivLLh4` "Book of Genesis Bible Study Part 55 | Stolen Blessing | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `yduBTNoFqZ0` "Jacob and Esau | Book of Genesis Episode 56 | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `Dwcm_I4Aui8` "Book of Genesis Explained Bible Study 57 | Esau Wants to Kill Jacob | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `tjomL_uK6tY` "Book of Genesis Bible Study Part 58 | Jacob Becomes A Patriarch | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `dfvDUoB3WLY` "Book of Genesis Bible Study 59 | Jacob's 20 Years Outside of Canaan | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `s9tKezraCo8` "Book of Genesis Bible Study Part 60 | Laban's Hospitality Runs Out | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `xRPaCMZ7a5g` "Book of Genesis Bible Study Part 61 | Jacob Falls In Love | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `xQwchOwYy8I` "Book of Genesis Bible Study Part 62 | God Opens Leah's Womb | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `gF6mZbvjK4E` "Book of Genesis Bible Study Part 63 | Jacob Does the Right Thing | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `xvPtTv38U2I` "Book of Genesis Bible Study Part 64 | Jacob Wishes To Go Home | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `SZE_r5dCofE` "Book of Genesis Bible Study Part 65 | A Change In Laban's Attitude | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `Rb_Z-Skj-2E` "Book of Genesis Bible Study Part 66 | Why Have You Stolen My gods? | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `SEzwqk7Ovpo` "Book of Genesis Bible Study Part 67 | Jacob's Greatest Enemy | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `-cwzaD68To8` "Book of Genesis Bible Study Part 68 | Jacob and Esau's Reunion | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `1-2rK8gXVp0` "GENESIS Bible Study Part 69 | Abduction and Rape of Dinah | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `EsyxGo7M0dA` "Book of Genesis Bible Study Part 70 | Prince of Shechem | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `5FVkijh71T0` "Book of Genesis Bible Study Part 71 | Setting Up Joseph | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `wFqLwH2YZ68` "Book of Genesis Bible Study Part 72 | Why Joseph Was The Favorite | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `58I7-zprC2w` "Book of Genesis Bible Study Part 73 | Joseph and Potiphar | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `dWroV9irVRM` "Joseph's Rise to Power | Book of Genesis Explained Bible Study 74 | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `38HT7sLruLQ` "Book of Genesis Bible Study Part 75 | How Joseph Became A Leader | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `ImC2A4BVQVo` "Book of Genesis Bible Study Part 76 | Jacob's Sons Go To Egypt | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `NSWQmOo59yo` "The Genesis Famine: A Warning for Today? Gensis 77 | Pastor Allen Nolan": no passage: its title and description name none the tool can read
- `7zH9j-8CW0I` "The Real Reason Joseph Kept the Silver Cup: Genesis 78 | Pastor Allen Nolan": no passage: its title and description name none the tool can read
- `KVTRUyUZqGc` "Book of Genesis Bible Study 79 | Joseph Reveals His True Identity | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `n_4DqTXGvKg` "The Secret to Joseph’s Success: Genesis 80 | Pastor Allen Nolan Genesis Study": no passage: its title and description name none the tool can read
- `UHvf52Kf6r8` "Jacob Goes to Egypt | Book of Genesis Explained Bible Study 81 | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `_byb3QnbXFo` "Joseph Saves Egypt | Genesis Explained Bible Study 82 | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `eOWJqakQOGA` "Before Jacob's Death | Book of Genesis Explained Bible Study 83 | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `I8KPONBATBQ` "Key to the Old Testament | Book of Genesis Explained Bible Study 84 | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `s80OQ1gbJLM` "Blessings of Jacob | Book of Genesis Explained Bible Study 85 | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `43Uh4pLFZcM` "Destiny of the Tribes of Israel | Genesis Explained Bible Study 86 | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `A2Zm_zKHI88` "Twelve Tribes of Israel | Book of Genesis Bible Study 87 | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `fKPcyYBcWzY` "Jacob’s Last Blessing: The End of an Era | Genesis 88": no passage: its title and description name none the tool can read
- `psxmOWhSNtA` "God's Justice Will Prevail | Book of Genesis Explained Bible Study 89 | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `0COrgPXrihc` "Daniel's Greatest Revelation: The Dream That Predicted the Future | Revelation Part 19": no passage: its title and description name none the tool can read
- `JSSBKAIgnys` "Why Daniel's Vision Unlocks Revelation's Final Prophecy | Revelation Part 21": no passage: its title and description name none the tool can read
- `2PF1GyOx3ww` "The Ancient Prophecy About the Antichrist That's Coming to Pass: Revelation Part 22": no passage: its title and description name none the tool can read
- `UVBLOzA-aA0` "Who Are the Two Witnesses In Revelation? Revelation Bible Study | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `yqdMfwQUXJw` "Book of Revelation Explained 47: Summary of Revelation 1-14 | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `3qi_4X8Tm-4` "Book of Revelation Explained: The False Prophet (Bonus Episode) Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `lu6Ow0KykIM` "Book of Amos In-Depth Bible Study 1: FOUNDATION of Amos | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `mWppNyMfrSA` "Book of Amos Bible Study 2: INTRODUCTION to Amos | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `Gw5cXRL2tCc` "Book of Amos Bible Study 3: OUTLINE of Amos | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `aPhFB4fPjR8` "Amos Part 4: The Verse That Reframes the Entire Bible": no passage: its title and description name none the tool can read
- `z_h0folQ4Kc` "Book of Amos 5: The Forgotten Prophecy About Jerusalem and the Festival of Shelters": no passage: its title and description name none the tool can read
- `c8YT3zriedE` "Book of Amos Bible Study: Syrians TORTURING Their Prisoners of War: Amos 6": no passage: its title and description name none the tool can read
- `S1KGpqt4agk` "Did God Endorse Slavery? What Deuteronomy and Amos Actually Reveal: Amos 7": no passage: its title and description name none the tool can read
- `oPQHTp8740U` "Book of Amos In-Depth Bible Study 8: The Bible and SLAVERY Part 2 | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `QD6uhXfGcJM` "Book of Amos Bible Study: What Does The Bible And Human Trafficking | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `Wlj-Jo70eac` "Book of Amos In-Depth Bible Study 11: Edom's JUDGEMENT Part 2 | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `j21CaxKhtHk` "What Does the Bible Say About Abortion? Book of Amos Bible Study 12 | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `RDkkJ-CNtMo` "Burial or Cremation? What the Bible and Church History Say": no passage: its title and description name none the tool can read
- `3LKdwL-gieM` "Book of Amos In-Depth Bible Study 18: Does God Cause Natural Disasters? Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `X-7AcpoW2G0` "Book of Amos In-Depth Bible Study 27: The 5 VISIONS of Amos | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read
- `DPrF3BGMC2I` "Book of Amos In-Depth Bible Study 29: Ethical Monotheism | Pastor Allen Nolan Sermon": no passage: its title and description name none the tool can read

## What was read

Only the YouTube Data API v3, with the Governor's API key (read from the environment, never written to a file, a log or this report): `channels.list`, `playlists.list` and `playlistItems.list`, one request at a time and a second apart. Nothing was downloaded: no video, audio, caption or page. No Subsplash or allennolan.com page was read, and no Subsplash address or embed: Subsplash's terms forbid automated reading, and the channel gives all that is needed.

## Licence

YouTube API Services Developer Policies, section III.E ("Handling YouTube Data and Content"), https://developers.google.com/youtube/terms/developer-policies, quoted word for word:

> API Clients may temporarily store limited amounts of Non-Authorized Data for as long as is necessary for the purposes of the API Client but not longer than 30 calendar days. As in section (III.E.4.c) immediately above, this means that after 30 calendar days, the API Client must either delete or refresh the stored data.
>
> Aside from the permissions and rights granted in this section, you and your API Clients have no further permissions or rights to API Data, including to temporarily stored API Data.

WATCH THE 30 DAYS: public API data may be kept for 30 calendar days and then must be deleted or refreshed, so this list has to be regenerated by the tool (`YOUTUBE_API_KEY="$(mw secrets get youtube_api_key)" go run ./tools/allennolan-youtube`) at least every 30 days. A copy that cannot be refreshed (an offline bundle) is the Governor's to decide on. YouTube's terms bind the videos the links open, not these rows.
