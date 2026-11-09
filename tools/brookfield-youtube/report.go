package main

import (
	"fmt"
	"strings"
)

// Summary is the few lines the tool prints.
func (r *Result) Summary() string {
	return fmt.Sprintf("videos seen %d, sermons %d (%d with a preacher, %d with a passage, %d series), Morning RePhil %d, skipped %d, dates %s to %s, quota used %d units\n",
		r.Seen, len(r.Rows), r.WithPreacher, r.WithPassage, r.SeriesRows, len(r.Rephil), len(r.Skipped), r.First, r.Last, r.Units)
}

// Markdown is REPORT.md: the numbers and what could not be sorted. It holds
// no key and nothing of a video's description.
func (r *Result) Markdown() string {
	var b strings.Builder
	w := func(format string, a ...any) { fmt.Fprintf(&b, format, a...) }
	w("# Brookfield First on YouTube: report\n\n<!-- toc: none -->\n\n")
	w("Written by `go run ./tools/brookfield-youtube` from the YouTube Data API v3 (channel %s); a re-run rewrites it. Only titles, dates, links, and a preacher or a passage where a title or description names one are kept: no video, audio, caption or description text.\n\n", r.Handle)
	w("## Counts\n\n")
	w("- Videos seen: %d (the entries of the channel's uploads playlist)\n", r.Seen)
	w("- Skipped: %d\n", len(r.Skipped))
	w("- Sermons: %d (every video that is not a Morning RePhil), in `rows.jsonl`\n", len(r.Rows))
	w("  - with a preacher: %d\n", r.WithPreacher)
	w("  - with a passage: %d (series among them: %d, each a text over more than 3 chapters, kept with its first chapter and matched by no passage)\n", r.WithPassage, r.SeriesRows)
	w("  - with neither: %d\n", r.neither())
	w("- Morning RePhil episodes: %d, in `data/morning-rephil.jsonl`\n", len(r.Rephil))
	w("- Date range: %s to %s\n", r.First, r.Last)
	w("- Quota used: %d units (%d channels.list and %d playlistItems.list, 1 unit each; the daily allowance is 10,000)\n\n", r.Units, min(1, r.Units), max(r.Units-1, 0))

	w("## Dates\n\n")
	w("A row's date is the day its title prints (\"August 23, 2020\"), which is the Sunday of the service: %d rows. The others, and every Morning RePhil episode, carry the day YouTube published the video (UTC, so an evening upload can be a day later than the day in New York).\n\n", r.TitleDates)

	w("## What was not sorted\n\n")
	w("- **Videos kept with no passage and no preacher: %d.** Most are whole services (\"Online Service | October 25, 2026\") whose title and description name neither. They are listed by title and date, found by search, and matched by no passage.\n", r.neither())
	w("- **Videos with a passage but no preacher: %d; with a preacher but no passage: %d.**\n", r.WithPassage-r.both(), r.WithPreacher-r.both())
	w("- **Descriptions that print a sermon title: %d.** The title of the sermon is not kept (the story keeps only the video's own title, and the licence note keeps description text out); the preacher and the passage beside it are.\n", r.DescriptionTitles)
	w("- **Passages the book table refuses: %d**", len(r.BadRefs))
	if len(r.BadRefs) == 0 {
		w(".\n")
	} else {
		w(" (the video is kept without that passage):\n")
		for _, x := range r.BadRefs {
			w("  - `%s`: %q\n", x.ID, x.Text)
		}
	}
	w("- **Entries skipped: %d**", len(r.Skipped))
	if len(r.Skipped) == 0 {
		w(" (none was private, deleted or listed twice).\n")
	} else {
		w(":\n")
		for _, x := range r.Skipped {
			w("  - `%s` %q: %s\n", x.ID, x.Title, x.Why)
		}
	}
	w("- **Morning RePhil episodes whose title carries no number: %d**", len(r.Unnumbered))
	if len(r.Unnumbered) == 0 {
		w(".\n")
	} else {
		w(". Each is numbered by its place in publish order among all the Morning RePhil episodes (1 is the oldest); that number is not the series' own and can repeat one that a title prints:\n")
		for _, id := range r.Unnumbered {
			n := 0
			for _, x := range r.Rephil {
				if strings.HasSuffix(x.URL, "="+id) {
					n = x.Number
				}
			}
			w("  - `%s`: number %d\n", id, n)
		}
	}
	if len(r.RepeatedNumbers) > 0 {
		w("- **Morning RePhil numbers carried by two or more videos:** %v.\n", r.RepeatedNumbers)
	}
	w("\n## Terms\n\nSee `source.toml`: YouTube's API Services Developer Policies let a client keep public API data for 30 calendar days and then require it to be deleted or refreshed, so this list has to be regenerated at least every 30 days.\n")
	return b.String()
}

func (r *Result) both() int {
	n := 0
	for _, x := range r.Rows {
		if x.Ref != nil && x.Preacher != nil {
			n++
		}
	}
	return n
}

func (r *Result) neither() int {
	n := 0
	for _, x := range r.Rows {
		if x.Ref == nil && x.Preacher == nil {
			n++
		}
	}
	return n
}

// SourceTOML is source.toml. It is written by the tool, with the counts of
// this run in its comments, so a re-run rewrites it too.
func (r *Result) SourceTOML() string {
	return fmt.Sprintf(`# Brookfield First (Brookfield Assembly of God, Brookfield CT), %[1]d sermon
# videos of its YouTube channel %[2]s, written by
#
#   YOUTUBE_API_KEY="$(mw secrets get youtube_api_key)" go run ./tools/brookfield-youtube
#
# from YouTube's official Data API (v3): channels.list and playlistItems.list,
# public metadata only, with a key. This file is rewritten by every run; edit
# the tool, not the file. Nothing is downloaded: no video, no audio, no caption.
#
# What the tool counted (REPORT.md lists what it could not sort):
#   %[3]d videos seen, %[1]d sermons, %[4]d with a preacher, %[5]d with a passage
#   (%[6]d of them series, matched by no passage), %[7]d Morning RePhil episodes
#   (in data/morning-rephil.jsonl, not here), dates %[8]s to %[9]s.
#
# The licence below was read on 2026-11-09 at the address in terms_url
# (YouTube API Services Developer Policies, section III.E, "Handling YouTube
# Data and Content", the page last updated 2026-11-06 UTC). The same sentences,
# word for word:

id = "brookfield-first"
name = "Brookfield First's sermons (YouTube)"
kind = "sermons"

licence = """
API Clients may temporarily store limited amounts of Non-Authorized Data for as long as is necessary for the purposes of the API Client but not longer than 30 calendar days. As in section (III.E.4.c) immediately above, this means that after 30 calendar days, the API Client must either delete or refresh the stored data.
Aside from the permissions and rights granted in this section, you and your API Clients have no further permissions or rights to API Data, including to temporarily stored API Data."""

licence_note = """
Read by the official YouTube Data API v3 with a Google API key (the Governor's, restricted to that API), which allows reading a channel's public metadata. The index keeps only titles, dates, the preacher and the passage where a video's title or description names one, and the YouTube watch link; never video, audio, a caption, or description text beyond a preacher's name and a passage. YouTube's terms bind the videos the links open, not these rows.
WATCH THE 30 DAYS: the Developer Policies quoted above allow public API data to be kept for 30 calendar days and then require it to be deleted or refreshed, so this list is to be regenerated by the tool at least every 30 days. A copy that cannot be refreshed (an offline bundle) is the Governor's to decide on. Brookfield First's own site (Subsplash) forbids automated download in its terms and was not read by any tool."""

credit = "Brookfield First (Brookfield Assembly of God), Brookfield CT"
homepage = "https://www.youtube.com/%[2]s"
terms_url = "https://developers.google.com/youtube/terms/developer-policies"
added = 2026-11-09
on_by_default = false
`, len(r.Rows), r.Handle, r.Seen, r.WithPreacher, r.WithPassage, r.SeriesRows, len(r.Rephil), r.First, r.Last)
}
