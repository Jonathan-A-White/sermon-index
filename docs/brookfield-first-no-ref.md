# Brookfield First rows with no passage (mw-o9rgo8m)

> **Superseded 2026-11-19 (mw-qtu3zvz.59.38):** the church's Subsplash account was then read through the Governor's signed-in browser and the missing titles, preachers and passages were filled in from it (`docs/brookfield-first-subsplash-inventory.md`). The list and counts below are the snapshot before that; recount with the command below.

A snapshot of `sources/brookfield-first/rows.jsonl` after the Dec 2023 Ruth series was filled in (mw-o9rgo8m): the rows that carry no `ref`, so no place in the Bible's reader lists them (rows match a place only by ref). Listed from the file by a one-off script; a later `go run ./tools/brookfield-youtube` run or `--merge-only` can fill more, so recount with `grep -vc '"ref"' sources/brookfield-first/rows.jsonl` (rows without a ref).

**132 of 355 rows have no ref.** By year (the row's date):

| Year | Rows with no ref |
|---|---|
| 2020 | 12 |
| 2021 | 35 |
| 2022 | 45 |
| 2023 | 31 |
| 2024 | 7 |
| 2025 | 1 |
| 2026 | 1 |
| Total | 132 |

Of them 123 are bare "Online Service | <date>" titles, 4 have a Subsplash page (`alt_url`) and 3 have a preacher.

## Where their titles and passages could come from

- **The church's Subsplash series pages** (subsplash.com/u/brookfieldfirst/media): each series lists its sermons with title, date, speaker and chapter, as the Ruth page did. The Governor reads them in the app and types a line into `passages.tsv` (or sends a Postern message that `subsplash-links --read-messages` turns into one). No tool reads Subsplash: its terms forbid automated download (docs/adding-a-source.md).
- **A YouTube description** names a preacher or a passage for a few videos; `tools/brookfield-youtube` already reads those, so a row still bare after a refresh has none.
- **The sermon notes mails and PDFs** (`tools/subsplash-notes`, `subsplash-links --read-notes`) give a title and passage for a row only if the church mailed notes for it.

Nothing here is guessed from who usually preaches or what the series was.

## The rows

| Date | Id | Title |
|---|---|---|
| 2020-03-22 | `Zh-3oOUjXXw` | Brookfield First - Online Service \| March 22, 2020 |
| 2020-09-27 | `VvUNg8E1aPo` | Brookfield First - Online Service \| September 27, 2020 |
| 2020-10-04 | `wl9DPCXJRQU` | Brookfield First - Online Service \| October 4, 2020 |
| 2020-10-11 | `xL6Do-BIVIY` | Brookfield First - Online Service \| October 11, 2020 |
| 2020-10-25 | `u9WhwosmWs4` | Missions Sunday 2020 |
| 2020-11-08 | `GqC4hIsD1OI` | Brookfield First - Online Service \| November 8, 2020 |
| 2020-11-15 | `NQXObToWy_c` | Brookfield First - Online Service \| November 15, 2020 |
| 2020-11-22 | `MInGxyAEtXE` | Brookfield First - Online Service \| November 22, 2020 |
| 2020-12-06 | `Am6yNJt4TzY` | Brookfield First - Online Service \| December 6, 2020 |
| 2020-12-13 | `7spPk-u9usU` | Brookfield First - Online Service \| December 13, 2020 |
| 2020-12-20 | `DyPvottPLtg` | Brookfield First - Online Service \| December 20, 2020 |
| 2020-12-25 | `gcvih2LjlnM` | Brookfield First - Christmas Eve Online 2020 |
| 2021-01-17 | `6YNZhKe912Y` | Brookfield First - Online Service \| January 17, 2021 |
| 2021-01-24 | `_2_snTNb1jI` | Brookfield First - Online Service \| January 24, 2021 |
| 2021-02-07 | `j4MLeoZigo4` | Brookfield First - Online Service \| February 7, 2021 |
| 2021-02-14 | `kys6RctQS2g` | Brookfield First - Online Service \| February 14, 2021 |
| 2021-02-21 | `rukWhl2zJAg` | Brookfield First - Online Service \| February 21, 2021 |
| 2021-02-28 | `qOwLPwQqnVY` | Brookfield First - Online Service \| February 28, 2021 |
| 2021-03-07 | `cu0ev15P4Kc` | Brookfield First - Online Service \| March 7, 2021 |
| 2021-03-14 | `k-3R7XKyUl4` | Brookfield First - Online Service \| March 14, 2021 |
| 2021-03-21 | `Hj24rdeqeXw` | Brookfield First - Online Service \| March 21, 2021 |
| 2021-04-18 | `I4WqgsscZLs` | Brookfield First - Online Service \| April 18, 2021 |
| 2021-04-25 | `DIFiHC_yrUU` | Brookfield First - Online Service \| April 25, 2021 |
| 2021-05-16 | `DKsL9Nn9S1k` | Brookfield First - Online Service \| May 16, 2021 |
| 2021-05-30 | `-kSMhX2hpzU` | Brookfield First - Online Service \| May 30, 2021 |
| 2021-06-06 | `ptenFGPsx7c` | Brookfield First - Online Service \| June 6, 2021 |
| 2021-06-27 | `eC5s58LUFeo` | Brookfield First - Online Service \| June 27, 2021 |
| 2021-07-11 | `UooB7u-_Ay4` | Brookfield First - Online Service \| July 11, 2021 |
| 2021-07-18 | `2EAtmiCguL0` | Brookfield First - Online Service \| July 18, 2021 |
| 2021-08-01 | `dhWyA9jwP9I` | Brookfield First - Online Service \| August 1, 2021 |
| 2021-08-15 | `bCIP1NENvnM` | Brookfield First - Online Service \| August 15, 2021 |
| 2021-08-29 | `bufto7HOmKE` | Brookfield First - Online Service \| August 29, 2021 |
| 2021-09-05 | `L4F3FmnflSw` | Brookfield First - Online Service \| September 5, 2021 |
| 2021-09-12 | `QOyOiwxyEDM` | Brookfield First - Online Service \| September 12, 2021 |
| 2021-09-19 | `Q2Db15DwM-o` | Brookfield First - Online Service \| September 19, 2021 |
| 2021-09-26 | `E1_XNsrVrKE` | Brookfield First - Online Service \| September 26, 2021 |
| 2021-10-03 | `hWP4sJL46do` | Brookfield First - Online Service \| October 3, 2021 |
| 2021-10-10 | `ID4QiNBIMnA` | Brookfield First - Online Service \| October 10, 2021 |
| 2021-10-17 | `Pm5CfbOK_Ek` | Brookfield First - Online Service \| October 17, 2021 |
| 2021-10-24 | `bJ6_nxi2Xyg` | Brookfield First - Online Service \| October 24, 2021 |
| 2021-10-31 | `1QLIxaqs2rI` | Brookfield First - Online Service \| October 31, 2021 |
| 2021-11-14 | `Jrs4NoOkPSk` | Brookfield First - Online Service \| November 14, 2021 |
| 2021-11-21 | `3457BKVXJng` | Brookfield First - Online Service \| November 21, 2021 |
| 2021-11-28 | `gE3O4Blctow` | Brookfield First - Online Service \| November 28, 2021 |
| 2021-12-05 | `sptg5Ct1h2M` | Brookfield First - Online Service \| December 5, 2021 |
| 2021-12-12 | `dAxIGuk0VPU` | Brookfield First - Online Service \| December 12, 2021 |
| 2021-12-19 | `jCjZvRMiNis` | Brookfield First - Online Service \| December 19, 2021 |
| 2022-01-02 | `2Jxsi-ls4SE` | Brookfield First - Online Service \| January 2, 2022 |
| 2022-01-16 | `N1XxDzOqD54` | Brookfield First - Online Service \| January 16, 2022 |
| 2022-01-30 | `Ulmz_nL9Vps` | Brookfield First - Online Service \| January 30, 2022 |
| 2022-02-06 | `xKcuslZXMXo` | Brookfield First - Online Service \| February 6, 2022 |
| 2022-02-13 | `kxxEHBlyMfg` | Brookfield First - Online Service \| February 13, 2022 |
| 2022-02-20 | `t4gwDvadnoM` | Brookfield First - Online Service \| February 20, 2022 |
| 2022-02-27 | `ctn9rhjBsUI` | Brookfield First - Online Service \| February 27, 2022 |
| 2022-03-06 | `yox3MYM1GVg` | Brookfield First - Online Service \| March 6, 2022 |
| 2022-03-13 | `S7soKliFRvk` | Brookfield First - Online Service \| March 13, 2022 |
| 2022-03-20 | `_ZtVxuuj8as` | Brookfield First - Online Service \| March 20, 2022 |
| 2022-03-27 | `zoH56IaujdI` | Brookfield First - Online Service \| March 27, 2022 |
| 2022-04-03 | `wwAZ_xZlQaU` | Brookfield First - Online Service \| April 3, 2022 |
| 2022-04-10 | `1elchTj3bbU` | Brookfield First - Online Service \| April 10, 2022 |
| 2022-04-16 | `tzrSm1iRQHY` | Brookfield First - Online Service \| Good Friday 2022 |
| 2022-04-18 | `vj7heoop0-s` | Brookfield First - Online Service \| Easter 2022 |
| 2022-04-24 | `-ob6U_Y1Dz4` | Brookfield First - Online Service \| April 24, 2022 |
| 2022-05-01 | `3VHs4I1IMdg` | Brookfield First - Online Service \| May 1, 2022 |
| 2022-05-08 | `tM_k0dikL94` | Brookfield First - Online Service \| May 8, 2022 |
| 2022-05-15 | `kZWiYoqUDeM` | Brookfield First - Online Service \| May 15, 2022 |
| 2022-05-29 | `l8Vsq0D3NAs` | Brookfield First - Online Service \| May 29, 2022 |
| 2022-06-05 | `Hn5xEcXyaz8` | Brookfield First - Online Service \| June 5, 2022 |
| 2022-06-12 | `HRiWdVMO4YA` | Brookfield First - Online Service \| June 12, 2022 |
| 2022-06-19 | `q1THgoxZusQ` | Brookfield First - Online Service \| June 19, 2022 |
| 2022-06-26 | `sZx_7jGg2CY` | Brookfield First - Online Service \| June 26, 2022 |
| 2022-07-03 | `mZsXpKanw68` | Brookfield First - Online Service \| July 3, 2022 |
| 2022-07-10 | `2j9mSFQOjvI` | Brookfield First - Online Service \| July 10, 2022 |
| 2022-07-17 | `NTVKcba8sqo` | Brookfield First - Online Service \| July 17, 2022 |
| 2022-07-24 | `d2kDY4KT2Ng` | Brookfield First - Online Service \| July 24, 2022 |
| 2022-07-31 | `LNdML8hIAbY` | Brookfield First - Online Service \| July 31, 2022 |
| 2022-09-11 | `lMqGdkYfXTg` | Brookfield First - Online Service \| September 11, 2022 |
| 2022-09-18 | `6PAZ_hJ1VSQ` | Brookfield First - Online Service \| September 18, 2022 |
| 2022-09-25 | `wlb4NVvoxgs` | Brookfield First - Online Service \| September 25, 2022 |
| 2022-10-02 | `I8FXyGrPB48` | Brookfield First - Online Service \| October 2, 2022 |
| 2022-10-16 | `E5LpOXe-kSQ` | Brookfield First - Online Service \| October 16, 2022 |
| 2022-10-30 | `rQOrSKJ9KU8` | Brookfield First - Online Service \| October 30, 2022 |
| 2022-11-06 | `bAZSYaiidJQ` | Brookfield First - Online Service \| November 6, 2022 |
| 2022-11-13 | `rP04FESm4WE` | Brookfield First - Online Service \| November 13, 2022 |
| 2022-11-20 | `l1X4T9XHmlE` | Brookfield First - Online Service \| November 20, 2022 |
| 2022-11-27 | `bxJ9CitE6ks` | Brookfield First - Online Service \| November 27, 2022 |
| 2022-12-04 | `lXvpHVGZORE` | Brookfield First - Online Service \| December 4, 2022 |
| 2022-12-11 | `V1GooOhh4wI` | Brookfield First - Online Service \| December 11, 2022 |
| 2022-12-18 | `bEIBELe001I` | Brookfield First - Online Service \| December 18, 2022 |
| 2022-12-26 | `hYwIGU0aeSk` | Brookfield First - Online Service \| Christmas Day 2022 |
| 2022-12-30 | `9rSvG4EWHUk` | Brookfield First - Online Service \| Christmas Eve 2022 |
| 2022-12-30 | `aIVm0KqLf-E` | Brookfield First Sunday Service (Live) |
| 2023-01-22 | `X6iod4NjgwI` | Brookfield First - Online Service \| January 22, 2023 |
| 2023-01-29 | `FUpOG7Ty0aE` | Brookfield First - Online Service \| January 29, 2023 |
| 2023-02-05 | `2Hna4vqryPI` | Brookfield First - Online Service \| February 5, 2023 |
| 2023-02-12 | `qdB3qZ9Dj9o` | Brookfield First - Online Service \| February 12, 2023 |
| 2023-02-19 | `aGTNTlP7I8g` | Brookfield First - Online Service \| February 19, 2023 |
| 2023-02-26 | `ILy5nQsLowU` | Brookfield First - Online Service \| February 26, 2023 |
| 2023-03-05 | `RRiy0gX0MRU` | Brookfield First - Online Service \| March 5, 2023 |
| 2023-03-12 | `PZl1flyzBtA` | Brookfield First - Online Service \| March 12, 2023 |
| 2023-03-19 | `IxKFrCjoIc0` | Brookfield First - Online Service \| March 19, 2023 |
| 2023-03-26 | `m3nT1b7wTd0` | Brookfield First - Online Service \| March 26, 2023 |
| 2023-04-02 | `zJP1d68QiW4` | Brookfield First - Online Service \| April 2, 2023 |
| 2023-04-16 | `ONZDCCeQcGU` | Brookfield First - Online Service \| April 16, 2023 |
| 2023-04-23 | `EWi-cCxHflo` | Brookfield First - Online Service \| April 23, 2023 |
| 2023-04-30 | `NZXw1QTX_ac` | Brookfield First - Online Service \| April 30, 2023 |
| 2023-05-07 | `UhUv6PKaj_M` | Brookfield First - Online Service \| May 7, 2023 |
| 2023-05-14 | `Dz4l728uYKg` | Brookfield First - Online Service \| May 14, 2023 |
| 2023-05-21 | `JPwUyCFnjas` | Brookfield First - Online Service \| May 21, 2023 |
| 2023-06-04 | `phit6QCVVJY` | Brookfield First - Online Service \| June 4, 2023 |
| 2023-06-11 | `AHDbjhrsXn0` | Brookfield First - Online Service \| June 11, 2023 |
| 2023-06-18 | `ESNNqA16jGw` | Brookfield First - Online Service \| June 18, 2023 |
| 2023-06-25 | `vsfXHowcunk` | Brookfield First - Online Service \| June 25, 2023 |
| 2023-07-02 | `Ds8o43AmF_U` | Brookfield First - Online Service \| July 2, 2023 |
| 2023-07-09 | `wt_QDWvLVwY` | Brookfield First - Online Service \| July 9, 2023 |
| 2023-07-14 | `82C-aS8JOKw` | Welcome to the Brookfield First Youtube Channel! |
| 2023-07-16 | `VlIzfw7WpqE` | Brookfield First - Online Service \| July 16, 2023 |
| 2023-07-23 | `Iv6CaELwntk` | Brookfield First - Online Service \| July 23, 2023 |
| 2023-07-30 | `PMIblOZLiuM` | Brookfield First - Online Service \| July 30, 2023 |
| 2023-08-06 | `oCZQHM9-aS8` | Brookfield First - Online Service \| August 6, 2023 |
| 2023-08-13 | `91Dt2J3Utvs` | Brookfield First - Online Service \| August 13, 2023 |
| 2023-08-20 | `xYKgRtjajFU` | Brookfield First - Online Service \| August 20, 2023 |
| 2023-08-27 | `2PVNDh_SxI8` | Brookfield First - Online Service \| August 27, 2023 |
| 2024-01-14 | `WLpfcR2_L-E` | Brookfield First - Online Service \| January 14, 2024 |
| 2024-01-21 | `3nj4hwUAKwU` | Brookfield First - Online Service \| January 21, 2024 |
| 2024-01-28 | `tMMxFNvp4Jc` | Brookfield First - Online Service \| January 28, 2024 |
| 2024-08-18 | `GJI_X_fDUXM` | Brookfield First - Online Service \| August 18, 2024 |
| 2024-11-03 | `76pjZAtjjbg` | Four Things The Bible Says About Persecution and One Thing It Does Not Say |
| 2024-11-10 | `MscTVOvH8AI` | Run the Race |
| 2024-12-26 | `kPTHDXCeoBM` | Brookfield First - Home for Christmas \| Christmas Day 2024 |
| 2025-12-25 | `NW_AI2Bt5l8` | Brookfield First \| Home for Christmas \| Christmas Day 2025 |
| 2026-09-15 | `11kBw-ru00s` | Peter's Priceless Legacy to the Church |
