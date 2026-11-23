package main

import (
	"fmt"
	"io"
	"strings"
)

// termsRead is what was read by hand at biblethinker.org on 2026-11-23
// (mw-qtu3zvz.280.2), before the tool was written, and is printed in every
// report. The robots.txt is not here: the tool fetches and quotes the live one.
const termsRead = `## Terms of use: none

Read on 2026-11-23, one request at a time with the User-Agent "sermon-index-check (+https://github.com/Jonathan-A-White/sermon-index)".

**The site has no terms of use.** These addresses were fetched and each answered 404 (not found): ` + "`/terms`, `/terms-of-use`, `/terms-and-conditions`, `/legal`, `/privacy-policy`, `/copyright`" + `. The footer has no link to any, and prints only "Copyright © 2026 BibleThinker. All Rights Reserved."

**The only legal page is /disclaimer** (https://biblethinker.org/disclaimer/), and it is a privacy policy, not terms of use. It opens: "BibleThinker.org provides powerful tools for the studying of God’s Word through our free online video reference library, with study tools that are grounded in the historical, conservative Christian faith." Its sections are the privacy policy summary, what passive and active data the site collects, California Do Not Track, how and when data is shared with third parties (donation processing, aggregate statistics), safeguards, and the reader's privacy rights (including the GDPR's). It says nothing about copying, scraping, automated access, linking or indexing: **not addressed**. Silence is not a permission (docs/adding-a-source.md, rule 2); the Governor's word is.

**The Governor's word (2026-11-23 17:37:46Z, Postern txid direct:cbd35263b1ee4f0d463e475a928c110f09e551f0493cbc44081935fd559c5dcf):** "A: Go ahead: robots.txt allows it, no terms say no; we keep only links, titles, passages and YouTube ids".

**What is kept:** a post's title, its passage, its address, the id of the YouTube video the post itself embeds, the length and the date the listing prints. Never the media, the audio (Castos), a transcript, the lesson notes or a description.`

// notRead are the site's other categories, none of them a book series, as the
// REST API listed them on 2026-11-23.
const notRead = "Answering Progressive Christians, Catholicism Refuted, Christians React to 20 Atheist Arguments Against God, Contending for the Faith Webinar, Debates, Evidence for the Bible, Exposing False Prophets & Cover-Up Culture, Homosexuality, How to Find Jesus in the Old Testament, Interviews, Learning about Cults, Marriage Questions Conference '22, Marriage, Divorce and Remarriage, Past Livestreams & Misc. Topical Teachings, Past Q&A Sessions, Pop-Atheism and Its Problems, PSA Defended: Scripture, History, Philosophy, & Rhetoric, The Mark Series (an empty category; the Mark series is in Mark), The Passion Project, Thinking Biblically Reviews, Understanding the Old Testament Law, Wisdom in the Word (topical talks), Women In Ministry, World Mission Society Church of God - WMSCOG, Worship Songs"

// SeriesRun is a series with what the site gave and what was made of it.
type SeriesRun struct {
	Data *SeriesData
	Res  SeriesResult
}

// WriteReport writes REPORT.md.
func WriteReport(w io.Writer, runs []SeriesRun, robots string) {
	var seen, rows, skipped int
	for _, r := range runs {
		seen += r.Res.Seen
		rows += len(r.Res.Rows)
		skipped += len(r.Res.Skips)
	}
	p := func(f string, a ...any) { fmt.Fprintf(w, f+"\n", a...) }

	p("# biblethinker: Bible Thinker's book series, Mike Winger")
	p("")
	p("Written by `go run ./tools/biblethinker --cache .biblethinker-cache --out sources/biblethinker --report sources/biblethinker/REPORT.md`; this file is rewritten by every run.")
	p("")
	p("One row an episode of a book-by-book series through the Bible on https://biblethinker.org: the post's address (`url`), the YouTube video the post itself embeds (`alt_url`, a watch address, which Lampas's `videoOf()` plays), the passage, the preacher Mike Winger, the date and the length. The posts of every other category (Q&A sessions, livestreams, topical talks, debates, interviews) are not read.")
	p("")
	p("## Counts")
	p("")
	p("- Book series found: %d.", len(runs))
	p("- Episodes found on the series pages: **%d**.", seen)
	p("- Rows written: **%d**.", rows)
	p("- Episodes skipped: **%d** (each is named below).", skipped)
	p("")
	p("| Series | Page | Archive pages | Category count | Episodes on the pages | Rows | Skipped |")
	p("| --- | --- | ---: | ---: | ---: | ---: | ---: |")
	for _, r := range runs {
		p("| %s | %s | %d | %d | %d | %d | %d |", r.Res.Series.Name, r.Res.Series.URL(), len(r.Data.Pages), r.Data.Count, r.Res.Seen, len(r.Res.Rows), len(r.Res.Skips))
	}
	p("")
	p("Every episode on a series page is a row or is named below. The skipped, by reason:")
	p("")
	counts := map[string]int{}
	for _, r := range runs {
		for _, s := range r.Res.Skips {
			counts[reasonKind(s.Reason)]++
		}
	}
	for _, k := range []string{"no passage", "no YouTube id", "not in the series", "passage not in the book"} {
		p("- %s: %d", k, counts[k])
	}
	p("")
	p("## The skipped episodes")
	for _, r := range runs {
		if len(r.Res.Skips) == 0 {
			continue
		}
		p("")
		p("### %s", r.Res.Series.Name)
		p("")
		for _, s := range r.Res.Skips {
			p("- %s (%s): %s", s.Title, s.URL, s.Reason)
		}
	}
	p("")
	p("## Passages")
	p("")
	p("A passage is read from the title (`1 Peter 1:3-7`, `(1:1-8)`, `(Mk 5:1-20)`, `(15:40-16:8)`, `Romans 6`) and, where the title prints none, from the digits that end the post's address (`...-1-peter-13-7` is 1 Peter 1:3-7; the digits are cut into chapter and verse in the one way the book has, and a post whose digits can be cut two ways is skipped). Where both give a passage and they differ, the title is used and the post is named here.")
	p("")
	any := false
	for _, r := range runs {
		for _, d := range r.Res.Disagree {
			any = true
			p("- %s", d)
		}
	}
	if !any {
		p("- No post's title and address name different passages.")
	}
	p("")
	nodate := 0
	for _, r := range runs {
		nodate += r.Res.NoDate
	}
	p("Rows with no date (the REST API gave none for the post): %d.", nodate)
	p("")
	p("## Not read")
	p("")
	p("The site's other categories, none of them a book series: %s.", notRead)
	p("")
	p(termsRead)
	p("")
	p("**robots.txt** (https://biblethinker.org/robots.txt, fetched by this run, word for word):")
	p("")
	p("```")
	p("%s", strings.TrimRight(robots, "\n"))
	p("```")
	p("")
	p("`User-agent: *` with an empty `Disallow:` forbids nothing, and the tool still obeys any Disallow a later robots.txt prints (it refuses to fetch the path). It fetches one request at a time, %s apart, with the User-Agent `%s`; a page it has fetched is kept in the git-ignored `.biblethinker-cache/` and not fetched again.", "two seconds", userAgent)
}

func reasonKind(reason string) string {
	switch {
	case strings.HasPrefix(reason, "not in the "):
		return "not in the series"
	case strings.Contains(reason, "passage in the title is not in"):
		return "passage not in the book"
	case strings.HasPrefix(reason, "no passage") || strings.Contains(reason, "digits ending the post"):
		return "no passage"
	default:
		return "no YouTube id"
	}
}
