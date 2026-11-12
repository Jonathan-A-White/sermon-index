# Brookfield First on YouTube: report

<!-- toc: none -->

Written by `go run ./tools/brookfield-youtube` from the YouTube Data API v3 (channel @BrookfieldFirst); a re-run rewrites it. Only titles, dates, links, and a preacher or a passage where a title or description names one are kept: no video, audio, caption or description text.

## Counts

- Videos seen: 522 (the entries of the channel's uploads playlist)
- Skipped: 0
- Sermons: 354 (every video that is not a Morning RePhil), in `rows.jsonl`
  - with a preacher: 109
  - with a passage: 214 (series among them: 2, each a text over more than 3 chapters, kept with its first chapter and matched by no passage)
  - with neither: 137
- Morning RePhil episodes: 168, in `data/morning-rephil.jsonl`
- Date range: 2020-03-15 to 2026-11-09
- Quota used: 12 units (1 channels.list and 11 playlistItems.list, 1 unit each; the daily allowance is 10,000)

## Dates

A row's date is the day its title prints ("August 23, 2020"), which is the Sunday of the service: 334 rows. The others, and every Morning RePhil episode, carry the day YouTube published the video (UTC, so an evening upload can be a day later than the day in New York).

## What was not sorted

- **Videos kept with no passage and no preacher: 137.** Most are whole services ("Online Service | October 25, 2026") whose title and description name neither. They are listed by title and date, found by search, and matched by no passage.
- **Videos with a passage but no preacher: 108; with a preacher but no passage: 3.**
- **Descriptions that print a sermon title: 26.** The title of the sermon is not kept (the story keeps only the video's own title, and the licence note keeps description text out); the preacher and the passage beside it are.
- **Passages the book table refuses: 0**.
- **Entries skipped: 0** (none was private, deleted or listed twice).
- **Morning RePhil episodes whose title carries no number: 1**. Each is numbered by its place in publish order among all the Morning RePhil episodes (1 is the oldest); that number is not the series' own and can repeat one that a title prints:
  - `y-hVqcfV8Lw`: number 168

## The church's list

`passages.tsv` is the church's own list of its sermons (series, Sunday, title, speaker, passage), typed by hand from screenshots of the church's app that the Governor took, or (the lines with a `row`) copied from a message in which he gave a sermon's title, date, preacher and passage as he read them in the app, for a sermon that has no notes; no tool reads the app. Each entry is matched to the one YouTube row of its date (a line with a `row` to that row, which tools/subsplash-links found within a day of the date), and that row takes the sermon's title, its preacher and its passage and keeps its watch link. An entry that cannot be matched to exactly one row, or whose passage the book table refuses, is named here and changes nothing; nothing is guessed.

- Entries in the list: 199
- Merged into a row: 199
- Of them from the Governor's messages: 65
- Of them from the notes mails: 107
- Not merged: 0.
- Unsure, left out of the list and never merged: 2:
  - 2026-10-04 "When the Man Comes Around" (Sam Morgan, 2 Peter 2:4-10): its range overlaps the other unsure sermon's at 2 Peter 2:10, so neither is certain; not merged.
  - 2026-10-11 "Balaam's Madness" (Phil Morgan, 2 Peter 2:10-17): its range overlaps the other unsure sermon's at 2 Peter 2:10, so neither is certain; not merged.

## Terms

See `source.toml`: YouTube's API Services Developer Policies let a client keep public API data for 30 calendar days and then require it to be deleted or refreshed, so this list has to be regenerated at least every 30 days.
