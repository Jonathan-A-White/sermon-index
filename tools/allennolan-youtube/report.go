package main

import (
	"fmt"
	"strings"
)

// The sentences of YouTube's API Services Developer Policies, section III.E
// ("Handling YouTube Data and Content"), word for word, as read on 2026-11-27
// at terms_url (the page last updated 2026-11-06 UTC) and as
// sources/brookfield-first/source.toml quotes them.
const (
	policyStore  = "API Clients may temporarily store limited amounts of Non-Authorized Data for as long as is necessary for the purposes of the API Client but not longer than 30 calendar days. As in section (III.E.4.c) immediately above, this means that after 30 calendar days, the API Client must either delete or refresh the stored data."
	policyRights = "Aside from the permissions and rights granted in this section, you and your API Clients have no further permissions or rights to API Data, including to temporarily stored API Data."
	termsURL     = "https://developers.google.com/youtube/terms/developer-policies"
)

// Summary is the few lines the tool prints.
func (r *Result) Summary() string {
	return fmt.Sprintf("playlists %d (book series %d), entries %d, videos %d, rows %d, skipped %d (%d no passage), share with a passage %.0f%%, dates %s to %s, quota used %d units\n",
		r.Playlists, len(r.Series), r.Entries, r.Videos, len(r.Rows), len(r.Skipped), len(r.Skipped)-r.Private(), 100*r.Share(), r.First, r.Last, r.Units)
}

// Markdown is REPORT.md: the numbers, the book playlists, what was skipped and
// why, and the licence. It holds no key and nothing of a video's description.
func (r *Result) Markdown() string {
	var b strings.Builder
	w := func(format string, a ...any) { fmt.Fprintf(&b, format, a...) }
	w("# Allen Nolan on YouTube: report\n\n<!-- toc: none -->\n\n")
	w("Written by `go run ./tools/allennolan-youtube` from the YouTube Data API v3 (channel %s, `%s`, %s); a re-run rewrites it. Only titles, dates, links and the passage a title, a playlist's title or a description names are kept: no video, audio, caption or description text.\n\n", r.Channel.Title, r.Channel.ID, r.Channel.Handle)

	w("## Counts\n\n")
	w("- Playlists of the channel: %d; **book playlists** (named by a Bible book, or a chapter of one): %d; left out although their title names a book: %d; the rest (topical, one-off, year, podcast, shorts and clip lists): %d\n", r.Playlists, len(r.Series), len(r.LeftOut), r.Playlists-len(r.Series)-len(r.LeftOut))
	w("- Playlist entries read: %d (a video in two book playlists is read twice and counted once below)\n", r.Entries)
	w("- Videos found: %d\n", r.Videos)
	w("- Rows written: %d, in `rows.jsonl`", len(r.Rows))
	if r.Stopped {
		w(" (none: the stop rule applied, see below)")
	}
	w("\n")
	if len(r.Rows) > 0 {
		w("  - passage in the video's title (verses): %d\n", r.By[fromTitle])
		w("  - chapter in the video's title, in a playlist of chapters: %d\n", r.By[fromTitleChapter])
		w("  - the chapter(s) the playlist's own title names: %d\n", r.By[fromPlaylist])
		w("  - the one passage of the book the description names: %d\n", r.By[fromDescription])
	}
	noPassage := len(r.Skipped) - r.Private()
	w("- Skipped: %d (every video of a book playlist is a row or is counted here)\n", len(r.Skipped))
	w("  - no passage: %d\n", noPassage)
	w("  - private or deleted: %d\n", r.Private())
	w("- Share of the listed videos with a passage: %.0f%% (%d of %d)\n", 100*r.Share(), len(r.Rows), r.Videos-r.Private())
	w("- Date range: %s to %s\n", r.First, r.Last)
	w("- Quota used: %d units (1 channels.list, %d playlists.list and playlistItems.list, 1 unit each; the daily allowance is 10,000)\n\n", r.Units, max(r.Units-1, 0))

	w("## Stop rule\n\n")
	w("If fewer than %d%% of the videos of the book playlists carry a passage the tool can read, it writes this report and no rows. ", int(minShare*100))
	if r.Stopped {
		w("**It applied:** %.0f%% of %d videos, so `rows.jsonl` and `source.toml` were not written.\n\n", 100*r.Share(), r.Videos-r.Private())
	} else {
		w("It did not apply: %.0f%% of %d videos carry one.\n\n", 100*r.Share(), r.Videos-r.Private())
	}

	w("## Book playlists\n\nA playlist is a book series when its title names a Bible book and says it is a study of it (\"Book of\", \"Gospel of\", \"Bible Study\", \"Verse by Verse\", \"Chapter\") or names a chapter of it (\"Isaiah 53\"). Topical playlists, one-offs, years, podcasts and clip lists are out. The videos are the API's count for the playlist; a video in two of them is one row.\n\n")
	w("| Playlist id | Title | Videos | Book | Text of every video with none of its own |\n| --- | --- | ---: | --- | --- |\n")
	for _, s := range r.Series {
		scope := s.Scope
		if scope == "" {
			scope = "none"
		}
		w("| `%s` | %s | %d | %s | %s |\n", s.ID, escapeCell(s.Title), s.Count, s.Book, scope)
	}
	w("\n")
	if len(r.LeftOut) > 0 {
		w("### Playlists that name a book and were left out\n\nTheir title names a book but says nothing of a study of it, so they are topical and their videos were not read:\n\n")
		for _, p := range r.LeftOut {
			w("- `%s` %s (videos: %d)\n", p.ID, escapeCell(p.Title), p.Count)
		}
		w("\n")
	}

	w("## What was skipped\n\nThe passage rules are conservative: a verse in the title is read; a bare chapter in a title only in a playlist of chapters (in \"Genesis Bible Study 5\" or \"Amos 6\" the number is the part of the study, not a chapter); a playlist named by a chapter (\"Isaiah 53\") gives it to a video that names none; a description gives the one passage of the book with verses that it names. A whole book, a summary over more than %d chapters, a part number or a passage the book table refuses is no passage, and nothing is guessed.\n\n", maxChapters)
	if len(r.Skipped) == 0 {
		w("None.\n\n")
	}
	for _, x := range r.Skipped {
		w("- `%s` %q: %s\n", x.ID, x.Title, x.Why)
	}
	if len(r.Skipped) > 0 {
		w("\n")
	}

	w("## What was read\n\n")
	w("Only the YouTube Data API v3, with the Governor's API key (read from the environment, never written to a file, a log or this report): `channels.list`, `playlists.list` and `playlistItems.list`, one request at a time and a second apart. Nothing was downloaded: no video, audio, caption or page. No Subsplash or allennolan.com page was read, and no Subsplash address or embed: Subsplash's terms forbid automated reading, and the channel gives all that is needed.\n\n")

	w("## Licence\n\nYouTube API Services Developer Policies, section III.E (\"Handling YouTube Data and Content\"), %s, quoted word for word:\n\n", termsURL)
	w("> %s\n>\n> %s\n\n", policyStore, policyRights)
	w("WATCH THE 30 DAYS: public API data may be kept for 30 calendar days and then must be deleted or refreshed, so this list has to be regenerated by the tool (`YOUTUBE_API_KEY=\"$(mw secrets get youtube_api_key)\" go run ./tools/allennolan-youtube`) at least every 30 days. A copy that cannot be refreshed (an offline bundle) is the Governor's to decide on. YouTube's terms bind the videos the links open, not these rows.\n")
	return b.String()
}

func escapeCell(s string) string { return strings.ReplaceAll(s, "|", "\\|") }

// SourceTOML is source.toml.
func (r *Result) SourceTOML() string {
	var b strings.Builder
	w := func(format string, a ...any) { fmt.Fprintf(&b, format, a...) }
	w("# Pastor Allen Nolan's book-by-book preaching: %d videos of the playlists of his YouTube\n", len(r.Rows))
	w("# channel @allennolan that are named by a Bible book, written by\n#\n")
	w("#   YOUTUBE_API_KEY=\"$(mw secrets get youtube_api_key)\" go run ./tools/allennolan-youtube\n#\n")
	w("# from YouTube's official Data API (v3): channels.list, playlists.list and\n# playlistItems.list, public metadata only, with a key. This file is rewritten\n# by every run; edit the tool, not the file. Nothing is downloaded: no video,\n# no audio, no caption. No Subsplash or allennolan.com page was read.\n#\n")
	w("# What the tool counted (REPORT.md lists what it could not sort):\n")
	w("#   %d book playlists, %d videos found, %d rows (each with a passage),\n#   %d skipped, dates %s to %s.\n#\n", len(r.Series), r.Videos, len(r.Rows), len(r.Skipped), r.First, r.Last)
	w("# The licence below is YouTube API Services Developer Policies, section III.E,\n# \"Handling YouTube Data and Content\" (the page last updated 2026-11-06 UTC),\n# word for word.\n\n")
	w("id = %q\n", sourceID)
	w("name = %q\n", "Pastor Allen Nolan's book-by-book preaching (YouTube)")
	w("kind = \"sermons\"\n\n")
	w("licence = \"\"\"\n%s\n%s\"\"\"\n\n", policyStore, policyRights)
	w("licence_note = \"\"\"\nRead by the official YouTube Data API v3 with a Google API key (the Governor's, restricted to that API), which allows reading a channel's public metadata. The index keeps only titles, dates, the preacher's name, the passage a video's title, its playlist's title or its description names, and the YouTube watch link; never video, audio, a caption, or description text beyond a passage. YouTube's terms bind the videos the links open, not these rows.\nWATCH THE 30 DAYS: the Developer Policies quoted above allow public API data to be kept for 30 calendar days and then require it to be deleted or refreshed, so this list is to be regenerated by the tool at least every 30 days. A copy that cannot be refreshed (an offline bundle) is the Governor's to decide on. The pastor's own site (allennolan.com) and the Subsplash pages it embeds were read by no tool: Subsplash's terms forbid automated reading.\"\"\"\n\n")
	w("credit = %q\n", "Pastor Allen Nolan; his YouTube channel @allennolan")
	w("homepage = %q\n", "https://www.youtube.com/@allennolan")
	w("terms_url = %q\n", termsURL)
	w("added = 2026-11-27\n")
	w("on_by_default = false\n")
	return b.String()
}
