# Brookfield First on YouTube: report

<!-- toc: none -->

Written by `go run ./tools/brookfield-youtube` from the YouTube Data API v3 (channel @BrookfieldFirst); a re-run rewrites it. Only titles, dates, links, and a preacher or a passage where a title or description names one are kept: no video, audio, caption or description text.

## Counts

- Videos seen: 527 (the entries of the channel's uploads playlist)
- Skipped: 0
- Sermons: 356 (every video that is not a Morning RePhil), in `rows.jsonl`
  - with a preacher: 238
  - with a passage: 343 (series among them: 2, each a text over more than 3 chapters, kept with its first chapter and matched by no passage)
  - with neither: 11
- Morning RePhil episodes: 225, in `data/morning-rephil.jsonl`
- Date range: 2020-03-15 to 2026-11-23
- Quota used: 12 units (1 channels.list and 11 playlistItems.list, 1 unit each; the daily allowance is 10,000)

## Dates

A row's date is the day its title prints ("August 23, 2020"), which is the Sunday of the service: 334 rows. The others, and every Morning RePhil episode, carry the day YouTube published the video (UTC, so an evening upload can be a day later than the day in New York).

## What was not sorted

- **Videos kept with no passage and no preacher: 11.** Most are whole services ("Online Service | October 25, 2026") whose title and description name neither. They are listed by title and date, found by search, and matched by no passage.
- **Videos with a passage but no preacher: 107; with a preacher but no passage: 2.**
- **Descriptions that print a sermon title: 26.** The title of the sermon is not kept (the story keeps only the video's own title, and the licence note keeps description text out); the preacher and the passage beside it are.
- **Passages the book table refuses: 0**.
- **Entries skipped: 0** (none was private, deleted or listed twice).
- **Morning RePhil episodes whose title carries no number: 1**. Each is numbered by its place in publish order among all the Morning RePhil episodes (1 is the oldest); that number is not the series' own and can repeat one that a title prints:
  - `MznqGO5oI2U`: number 171

## The church's list

`passages.tsv` is the church's own list of its sermons (series, Sunday, title, speaker, passage), typed by hand from screenshots of the church's app that the Governor took, or (the lines with a `row`) copied from a message in which he gave a sermon's title, date, preacher and passage as he read them in the app, for a sermon that has no notes; no tool reads the app. Each entry is matched to the one YouTube row of its date (a line with a `row` to that row, which tools/subsplash-links found within a day of the date), and that row takes the sermon's title, its preacher and its passage and keeps its watch link. An entry that cannot be matched to exactly one row, or whose passage the book table refuses, is named here and changes nothing; nothing is guessed.

- Entries in the list: 344
- Merged into a row: 344
- Of them from the Governor's messages: 65
- Of them from the notes mails: 111
- Of them from the church's app, through the Governor's browser: 137
- Rows with a preacher: 238 of 356 (109 before the speakers of three rows were read: the Governor's word of 2026-11-12 for "When Angels Go to Church", and the header of the notes PDFs of two others). The mails name no speaker and the YouTube titles and descriptions name none beyond those already read, so the other 118 have none; nothing is guessed from who usually preaches:
  - `gcvih2LjlnM` 2020-12-25, Brookfield First - Christmas Eve Online 2020
  - `tzrSm1iRQHY` 2022-04-16, Brookfield First - Online Service | Good Friday 2022
  - `vj7heoop0-s` 2022-04-18, Brookfield First - Online Service | Easter 2022
  - `hYwIGU0aeSk` 2022-12-26, Brookfield First - Online Service | Christmas Day 2022
  - `9rSvG4EWHUk` 2022-12-30, Brookfield First - Online Service | Christmas Eve 2022
  - `aIVm0KqLf-E` 2022-12-30, Brookfield First Sunday Service (Live)
  - `82C-aS8JOKw` 2023-07-14, Welcome to the Brookfield First Youtube Channel!
  - `RosBFCA-dUA` 2023-09-10, Dear 21st Century Church
  - `ZJN1he-Hhbs` 2023-09-17, All the Called
  - `R1m8CtFiHBc` 2023-09-24, Is Christ Divided?
  - `AO7g1n2CCRY` 2023-10-01, The Subversive Power of the Cross
  - `BIUjJJlPbtA` 2023-10-08, God's Strange Plan
  - `xzhbkgC8GMg` 2023-10-15, The Cure for Nerf Gun Ministry
  - `LDtHvuvsH8I` 2023-10-22, Wisdom by the Spirit
  - `DqzKNZy8t7M` 2023-11-05, Our Privilege of Partnership
  - `oSCKhNSiSY8` 2023-11-12, The Carnal Christian
  - `xSLYZDhSLew` 2023-11-19, Built to Last
  - `vjyLGESSrQI` 2023-11-26, Found Faithful
  - `lUM4I5QGV4M` 2023-12-31, Following Jesus into 2024
  - `-6kwEEFZJk4` 2024-01-07, If You Walk With God
  - `52ea_kadhDU` 2024-02-04, The Now and the Not Yet
  - `Kb_hfL752jg` 2024-02-11, Daddy D̶a̶y̶ Discipleship Care
  - `HsqiEtRGkXI` 2024-02-18, The Surprising Grace of Church Discipline
  - `3AOcy3sUbVY` 2024-02-25, The Wrong Way to Be Right
  - `SLUcMxK4hRo` 2024-03-03, Such Were Some of You
  - `Y9LdjpS5kx0` 2024-03-10, Bumper Sticker Christianity
  - `EFmdijdm2q4` 2024-03-17, You Are Not Your Own
  - `4WgJX2PLMjA` 2024-03-24, What To Do When I Don't Want God
  - `DKtzIeedA9A` 2024-04-07, A New Life Without Fear
  - `kQe3k_vW2e4` 2024-04-14, The Gifts of Marriage and Singleness
  - `itjXWulLKsg` 2024-04-21, Marriage Matters
  - `h4h_wJwJLjI` 2024-04-28, Bloom Where You're Planted
  - `qawSL17O0O0` 2024-05-05, Satisfied in Every Season
  - `EjOUUeZV26U` 2024-05-12, She Believed and Received
  - `EcUkCrdmV2o` 2024-05-19, Why Are We Pentecostal?
  - `mJ1uzv32PIQ` 2024-05-26, Love Will Keep Us Together
  - `Z70MxD1eRLc` 2024-06-02, The End of Entitlement
  - `XOS76QklSXw` 2024-06-09, Gospel-formed Freedom
  - `RKO7lrvQlmU` 2024-06-23, Presuming on Privilege
  - `Yt3X0lBhYkk` 2024-06-30, Flee From Idolatry
  - `m7U7AiOm0Zk` 2024-07-07, Love, Liberty and the Pursuit of God's Glory
  - `RHv9aES72Ek` 2024-07-28, What the Lord's Supper Isn't
  - `YlW2X-9UM5Q` 2024-08-04, What the Lord's Supper IS
  - `GggUbYjr-3k` 2024-08-25, Unleashing the Power of Everybody
  - `bHmXpIDSBDg` 2024-09-01, The More Excellent Way
  - `iFn-zYQYjD8` 2024-09-15, The Gospel-Centered Life
  - `DudjtC56Ufc` 2024-09-29, Resurrection Bodies
  - `YDEiWpWanIM` 2024-10-06, Managing the Mission
  - `Jv82OGpyMDs` 2024-10-13, Maturity Matters
  - `blhOjMlCKVg` 2024-10-20, The King's Kindness
  - `HwwLdYKdyVY` 2024-10-27, The Invitation of Jesus
  - `MscTVOvH8AI` 2024-11-10, Run the Race
  - `NaHOmgIE_p0` 2024-12-01, Mary's Song of Praise
  - `hUXd8n69YfA` 2024-12-08, Zechariah's Song of Faith
  - `CSxQVA7fvBk` 2024-12-15, The Angels' Song of Joy
  - `iYpBxede3E8` 2024-12-22, Simeon's Song of Hope
  - `kPTHDXCeoBM` 2024-12-26, Brookfield First - Home for Christmas | Christmas Day 2024
  - `Zz8HNUGBhSg` 2025-01-19, The Adventure of Faith
  - `KaeYFIQx5wI` 2025-02-02, Covenant Faith
  - `adlZ6o6zeKg` 2025-02-09, Stumbling Faith
  - `K1RBmN8HQ5I` 2025-02-16, Praying Faith
  - `HIEXSCD4LfQ` 2025-02-23, Refined Faith
  - `SBwZflGUJKs` 2025-03-02, Finishing Faith
  - `hpn3UHK4PPI` 2025-03-16, Dying to Live
  - `qdKaxB5XMuw` 2025-03-23, Our Plans, God's Promises
  - `tGI8NsyoN1Y` 2025-03-30, A Careful Love
  - `NzvpNtbyXpo` 2025-04-06, Kingdom Contrasts
  - `0693wJtSExg` 2025-04-13, The Long and Winding Road
  - `Tlpzx5DwBn4` 2025-04-27, Living Letters
  - `vntqTUI77KA` 2025-05-04, From Glory to Glory
  - `cqEOs0o_WRg` 2025-05-11, The Multiplied Mom
  - `0_7ersV9awk` 2025-05-18, Reasons to Never Lose Heart
  - `1X68sihElJ8` 2025-05-25, The Clay Pot Paradox
  - `Cc2rZ218lOk` 2025-06-01, Gospel Hope for Weary Pilgrims
  - `zgnC7lvWsjQ` 2025-06-08, The Life-Giving Spirit
  - `3zJIBJvR_IU` 2025-06-15, Homecoming
  - `MT0urZj5qyw` 2025-06-22, We Must All Appear
  - `EG7J3jRygGw` 2025-06-29, Walk the Walk
  - `Q7dc4xZJ6A0` 2025-07-06, Motivated for Life
  - `27M6bVS9VjY` 2025-07-13, What's So New About New Creation?
  - `XSNAvaCKyVw` 2025-07-20, Protocols of Heaven's Embassy
  - `RzUTnvfZ2Y4` 2025-07-27, Heart Surgeries
  - `zlOYAN3rRAc` 2025-08-03, The Grace of Godly Grief
  - `cc8vmwWQUGw` 2025-08-24, The Storehouse
  - `gSTnbQ8Fe14` 2025-08-31, The Cheerful Giver
  - `nJQv2yqXILI` 2025-09-07, The Weapons of Our Warfare
  - `JWR9_aEJub8` 2025-09-14, The Measure That Matters
  - `1Ry2KCLNEOc` 2025-09-21, Gospel Simplicity
  - `oSKPg6wTCAo` 2025-09-28, Boasting in Bruises
  - `fTfNJSaiaN4` 2025-10-05, The House That God Builds
  - `i4gPozBFFGw` 2025-10-12, The Care of Souls
  - `FhOTBXf3S90` 2025-10-19, Lightning in Clay Pots
  - `AEChlXKiLeQ` 2025-10-26, If God Is For Us?
  - `HqSiDkBcnvU` 2025-11-02, Pierce My Ear
  - `ArOiEm8ti5E` 2025-11-09, Putting Hope Within Reach
  - `rWotu8TC7-c` 2025-11-23, Thanks in the Face of Thorns
  - `aGXSMjnqDtI` 2025-11-30, The Promise Foretold
  - `812ksCvm-es` 2025-12-07, The People Who Waited
  - `LoWuVVLCMr8` 2025-12-14, The King Who Came
  - `Nz9wIUBPmFM` 2025-12-21, The Hope Who Returns
  - `NW_AI2Bt5l8` 2025-12-25, Brookfield First | Home for Christmas | Christmas Day 2025
  - `u22VPCdGIe0` 2026-01-04, The Lion Fear
  - `2hnWkvgswf4` 2026-01-11, The God Who is to Be Feared
  - `OVprCuyYDDg` 2026-01-18, The Grace of Fear
  - `ZTuy1G9FiuE` 2026-01-25, Walking in the Fear of the Lord
  - `0_FQ9fss7fw` 2026-02-01, Behind a Frowning Providence
  - `CHDXYiutIMc` 2026-02-15, The Falls of Babylon
  - `xvtgyhu1tFE` 2026-02-22, When Fields Fail
  - `1vY8oe6vNvU` 2026-03-01, How to Hear a Sermon
  - `QcMBjiGzDv0` 2026-05-10, A Faith That Outlasts Pain
  - `omdA4p7TBhA` 2026-06-14, Heart Recalibration
  - `0jAthJ_RBsE` 2026-06-21, Pictures of Our Father
  - `11kBw-ru00s` 2026-09-15, Peter's Priceless Legacy to the Church
  - `wAsyvhEJVbA` 2026-10-04, When the Man Comes Around
  - `bMemJB2uH9I` 2026-10-11, Balaam's Madness
  - `KeNFhuTVUj0` 2026-11-09, Unfinished Business
  - `QYbJqGWgXnc` 2026-11-17, Why the Mission Cannot Wait | Revelation 5:9-10 | Pastor Phil Morgan
  - `HNtqweEIBpI` 2026-11-22, How to Be a Mat Carrier | Mark 2:1-12 | Pastor Sam Morgan
- Not merged: 0.
- Unsure, left out of the list and never merged: 2:
  - 2026-10-04 "When the Man Comes Around" (Sam Morgan, 2 Peter 2:4-10): its range overlaps the other unsure sermon's at 2 Peter 2:10, so neither is certain; not merged.
  - 2026-10-11 "Balaam's Madness" (Phil Morgan, 2 Peter 2:10-17): its range overlaps the other unsure sermon's at 2 Peter 2:10, so neither is certain; not merged.

## Terms

See `source.toml`: YouTube's API Services Developer Policies let a client keep public API data for 30 calendar days and then require it to be deleted or refreshed, so this list has to be regenerated at least every 30 days.
