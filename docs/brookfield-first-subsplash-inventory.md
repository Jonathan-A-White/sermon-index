# Brookfield First: the church's Subsplash account against the index (mw-qtu3zvz.59.38)

<!-- toc: none -->

Written by `go run ./tools/subsplash-notes inventory-fill` from the inventory `go run ./tools/subsplash-notes inventory` read through the Governor's signed-in browser (his allowed use of his own account, polite pace; docs/operations.md, "Reading Subsplash through the signed-in browser"). The inventory itself (`data/subsplash-inventory.jsonl`: link, date, title, preacher, passage, series, themes, notes link) is git-ignored like the links file; this report holds counts and ids only. A re-run rewrites it.

**384 sermons** inventoried, 2026-11-15 back to 2020-01-05 (the Morning RePhil episodes are not sermons and are left out; `data/morning-rephil.jsonl` tracks them).

| | Sermons |
|---|---|
| Matched to one index row (by date, `MatchNote`) | 352 |
| On Subsplash, no row in the index | 32 |
| Rows of the index with no sermon on Subsplash | 13 |
| Left alone: two or more sermons claim one row, so none is chosen | 18 |
| Matched, and the row's title, preacher or passage still is not Subsplash's (the row's own is kept, not changed) | 23 |

## What went into passages.tsv

137 lines have `from` = `browse`: each names its row by id and carries a title (the row's own unless it was a generic "Online Service" one), 118 a preacher and 119 a passage, set where the row lacked them. A row that already had a line in the list, and a row with nothing missing that Subsplash has, got none; a second run adds nothing.

## Verses and themes

| | Sermons |
|---|---|
| With verses (a passage, from Subsplash or the row) | 370 |
| With themes (a theme linked in `sermon-topics.jsonl`, or the row's own topics; Subsplash shows none) | 14 |
| With neither | 14 |

Subsplash's sermon page and its notes name no theme or tag, and the sermon-notes mails (Postern kind `sermon-notes`) carry a title, a passage, a date and sometimes a speaker, no theme; so every theme counted here is one the index already holds.

## On Subsplash, not in the index

| Date | Code | Title | Passage | Why no row |
|---|---|---|---|---|
| 2026-11-15 | `wvnvwrc` | Why the Mission Cannot Wait | Revelation 5:9-10 | no row within one day of Sunday 2026-11-15 |
| 2026-05-24 | `pzxx57g` | Brookfield First Sunday Service (Live) |  | no row within one day of Sunday 2026-05-24 |
| 2025-12-25 | `7zkjt9v` | Home for Christmas \| Christmas Day 2025 |  | Thursday 2025-12-25 is a Thursday: only a Sunday, or a Saturday or Monday beside one, is taken |
| 2025-09-12 | `qnkjtp6` | Celebration of Life Service \| William R. Mazzulla Jr. \| September 12, 2025 (12pm ET) |  | Friday 2025-09-12 is a Friday: only a Sunday, or a Saturday or Monday beside one, is taken |
| 2025-03-05 | `y8xmkp3` | Sculpted for Service | 1 Samuel 16:1-13 | Wednesday 2025-03-05 is a Wednesday: only a Sunday, or a Saturday or Monday beside one, is taken |
| 2025-02-05 | `cgfh37f` | Metamorphosis | John 17:11-19 | Wednesday 2025-02-05 is a Wednesday: only a Sunday, or a Saturday or Monday beside one, is taken |
| 2024-12-04 | `2pbdc26` | The Other King | Matthew 2:1-20 | Wednesday 2024-12-04 is a Wednesday: only a Sunday, or a Saturday or Monday beside one, is taken |
| 2024-11-06 | `yzfpvrc` | Holy Dissatisfaction | Judges 6:1-13 | Wednesday 2024-11-06 is a Wednesday: only a Sunday, or a Saturday or Monday beside one, is taken |
| 2024-10-02 | `8gx55bw` | Five Characteristics of Faith-Sharers | Luke 5:27-32 | Wednesday 2024-10-02 is a Wednesday: only a Sunday, or a Saturday or Monday beside one, is taken |
| 2024-09-04 | `8f4yrww` | God's Hammer | Jeremiah 23:21-29 | Wednesday 2024-09-04 is a Wednesday: only a Sunday, or a Saturday or Monday beside one, is taken |
| 2024-05-01 | `jwhq9r2` | GPS |  | Wednesday 2024-05-01 is a Wednesday: only a Sunday, or a Saturday or Monday beside one, is taken |
| 2024-04-03 | `r2sjx6b` | What the Harvest Says to Us |  | Wednesday 2024-04-03 is a Wednesday: only a Sunday, or a Saturday or Monday beside one, is taken |
| 2024-03-06 | `dsk3m35` | The Power of Partnership | John 14-16 | Wednesday 2024-03-06 is a Wednesday: only a Sunday, or a Saturday or Monday beside one, is taken |
| 2024-02-07 | `x2kvcrc` | Living Your Highest Calling |  | Wednesday 2024-02-07 is a Wednesday: only a Sunday, or a Saturday or Monday beside one, is taken |
| 2024-01-05 | `tmcj6wp` | Devotional: Day 5 |  | Friday 2024-01-05 is a Friday: only a Sunday, or a Saturday or Monday beside one, is taken |
| 2024-01-04 | `44nrdhv` | Devotional: Day 4 |  | Thursday 2024-01-04 is a Thursday: only a Sunday, or a Saturday or Monday beside one, is taken |
| 2024-01-03 | `gbd65xq` | Devotional: Day 3 |  | Wednesday 2024-01-03 is a Wednesday: only a Sunday, or a Saturday or Monday beside one, is taken |
| 2024-01-02 | `sxgcywq` | Devotional: Day 2 |  | Tuesday 2024-01-02 is a Tuesday: only a Sunday, or a Saturday or Monday beside one, is taken |
| 2023-04-07 | `mb2vqmj` | Good Friday 2023 | Matthew 27:11-50 | Friday 2023-04-07 is a Friday: only a Sunday, or a Saturday or Monday beside one, is taken |
| 2022-04-17 | `sjkbm3w` | Easter 2022 | Mark 16 | more than one row within one day of Sunday 2022-04-17 (tzrSm1iRQHY, vj7heoop0-s): not guessed |
| 2022-04-15 | `8pn2z88` | Good Friday 2022 | Mark 15:33-47 | Friday 2022-04-15 is a Friday: only a Sunday, or a Saturday or Monday beside one, is taken |
| 2021-04-04 | `qcf94bt` | Easter 2021 | John 20:1-29 | more than one row within one day of Sunday 2021-04-04 (I3XhhpYd8xk, dacrnNNpyAU): not guessed |
| 2021-04-02 | `y8s2nr3` | Good Friday 2021 | John 19:1-5 | Friday 2021-04-02 is a Friday: only a Sunday, or a Saturday or Monday beside one, is taken |
| 2020-12-24 | `m3y65nq` | Christmas Eve 2020 |  | Thursday 2020-12-24 is a Thursday: only a Sunday, or a Saturday or Monday beside one, is taken |
| 2020-03-01 | `6n3jcqc` | The Untouchables | Mark 1:40-45 | no row within one day of Sunday 2020-03-01 |
| 2020-02-23 | `ms4xbmw` | Master Priorities | Mark 1:35-39 | no row within one day of Sunday 2020-02-23 |
| 2020-02-16 | `q8bh3vv` | God Unlimited | Mark 1:29-34 | no row within one day of Sunday 2020-02-16 |
| 2020-02-09 | `fsc4y4m` | As One Having Authority | Mark 1:21-28 | no row within one day of Sunday 2020-02-09 |
| 2020-02-02 | `k8wvccm` | Fishers of Men | Mark 1:16-20 | no row within one day of Sunday 2020-02-02 |
| 2020-01-26 | `6f5q9kc` | War of the Worlds | Mark 1:11-15 | no row within one day of Sunday 2020-01-26 |
| 2020-01-12 | `9njf5vd` | John the Baptizer | Mark 1:2-8 | no row within one day of Sunday 2020-01-12 |
| 2020-01-05 | `nxmfjgq` | A King for Fire and Failure | Mark 1:1 | no row within one day of Sunday 2020-01-05 |

## In the index, not on Subsplash

| Date | Row | Title |
|---|---|---|
| 2026-11-17 | `QYbJqGWgXnc` | Why the Mission Cannot Wait \| Revelation 5:9-10 \| Pastor Phil Morgan |
| 2026-09-15 | `11kBw-ru00s` | Peter's Priceless Legacy to the Church |
| 2025-12-25 | `NW_AI2Bt5l8` | Brookfield First \| Home for Christmas \| Christmas Day 2025 |
| 2024-12-26 | `kPTHDXCeoBM` | Brookfield First - Home for Christmas \| Christmas Day 2024 |
| 2023-07-14 | `82C-aS8JOKw` | Welcome to the Brookfield First Youtube Channel! |
| 2023-04-07 | `DZnys21Mi0U` | The Cross of Christ |
| 2022-12-30 | `9rSvG4EWHUk` | Brookfield First - Online Service \| Christmas Eve 2022 |
| 2022-12-30 | `aIVm0KqLf-E` | Brookfield First Sunday Service (Live) |
| 2022-04-18 | `vj7heoop0-s` | Brookfield First - Online Service \| Easter 2022 |
| 2022-04-16 | `tzrSm1iRQHY` | Brookfield First - Online Service \| Good Friday 2022 |
| 2021-04-04 | `I3XhhpYd8xk` | Surprised by Joy |
| 2021-04-03 | `dacrnNNpyAU` | Behold the Man |
| 2020-12-25 | `gcvih2LjlnM` | Brookfield First - Christmas Eve Online 2020 |
