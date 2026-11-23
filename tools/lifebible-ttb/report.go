package main

import (
	"fmt"
	"strings"

	"github.com/Jonathan-A-White/sermon-index/internal/sources"
)

// robotsStar is the block of the site's robots.txt that applies to this tool
// (User-Agent: *), word for word as read on 2026-11-23 from
// https://www.lifebibleministry.com/robots.txt.
const robotsStar = `User-Agent: *
Allow: /
Disallow: /api/
Disallow: /auth/
Disallow: /console/
Disallow: /locked/
Disallow: /background`

// Summary is the counts, as printed by the tool.
func (r *Report) Summary() string {
	var b strings.Builder
	line := func(label string, n int) { fmt.Fprintf(&b, "%-66s %d\n", label, n) }
	line("through-the-bible addresses in the sitemap:", r.Pages)
	line("rows written:", r.Rows)
	line("  of them series (a whole book over 3 chapters):", r.Series)
	line(fmt.Sprintf("  of them over %d chapters, not series:", sources.WideChapters), r.Wide)
	line("addresses skipped:", len(r.Skipped))
	for _, reason := range skipReasons {
		if n := r.count(reason); n > 0 {
			line("  "+reason+":", n)
		}
	}
	return b.String()
}

var skipReasons = []string{ReasonList, ReasonBook, ReasonFetch, ReasonNoPassage, ReasonNoVideo, ReasonRepeat}

func (r *Report) count(reason string) int {
	n := 0
	for _, s := range r.Skipped {
		if s.Reason == reason {
			n++
		}
	}
	return n
}

// Markdown is REPORT.md.
func (r *Report) Markdown() string {
	var b strings.Builder
	b.WriteString("# Life Bible Ministry, Through the Bible: what the pages do not settle\n\n<!-- toc: none -->\n\n")
	b.WriteString("Written by `go run ./tools/lifebible-ttb --cache .lifebible-cache --out sources/lifebible-ttb`; do not edit by hand. Only addresses, titles, passages and YouTube ids are listed or kept: never audio, video, a transcript or a page's description.\n\n")
	b.WriteString("## Counts\n\n```\n" + r.Summary() + "```\n\n")
	b.WriteString("Every `through-the-bible` address of `https://www.lifebibleministry.com/sitemap.xml` is a row of `rows.jsonl` or is named below: rows + skipped = addresses.\n\n")
	b.WriteString("A row's `id` is the address's path below `/through-the-bible/` with `/` written `.` (`isaiah/1` is `isaiah.1`). Its `url` is the page, `alt_url` the YouTube watch address of the video the page embeds, so Lampas's `videoOf()` plays it. The row has **no date**: the page's `uploadDate` is the day the site imported its pages (2026-04-18 and 2026-04-19 for almost all of them), not the day the teaching was given, so it is not written as one.\n\n")
	b.WriteString("**Wide rows.** Through the Bible teaches up to seventeen chapters at a time (\"Isaiah 1-4:1\", \"Job 15–31\"), so a row whose ref spans more than 3 chapters is the teaching's real text, not an overview, and is kept as a plain row with its full ref: it shows at each chapter it teaches (the Governor's \"Isaiah 1\" sheet needs this). `sermons check` warns of each (one warning a wide row); that is answered here, not by `series`. Only a title that names a whole book of more than 3 chapters (\"Lamentations: ...\") is a `series` row, with its first chapter as ref.\n\n")
	b.WriteString("## Terms and robots.txt (read 2026-11-23)\n\n")
	b.WriteString("The site has **no terms of use**: `/terms`, `/privacy`, `/terms-of-use`, `/legal` and `/copyright` all answer 404 and the page footer has no legal link. Its footer reads \"© 2026 Life Bible Ministry · Pastor Paul LeBoutillier\" and its `/about` page says Life Bible Ministry is \"a global teaching resource \u2014 where every message, transcript, and study is available completely free of charge to anyone in the world\" and, under Our Mission, \"To make thorough, verse-by-verse Bible teaching available to every person in the world \u2014 completely free of charge\" (statements of purpose, not a clause on automated access or on linking). No terms say yes and none say no; the Governor read this on 2026-11-23 (his tap, txid direct:cbd35263b1ee4f0d463e475a928c110f09e551f0493cbc44081935fd559c5dcf) and said to go ahead because `robots.txt` allows it and we keep only links, titles, passages and YouTube ids.\n\n")
	b.WriteString("`https://www.lifebibleministry.com/robots.txt`, the block for every robot, word for word:\n\n```\n" + robotsStar + "\n```\n\n")
	b.WriteString("Its blocks for Googlebot, Bingbot, Google-Extended, GPTBot, ChatGPT-User, OAI-SearchBot, ClaudeBot, anthropic-ai, PerplexityBot, cohere-ai, Amazonbot, YouBot, Applebot, Applebot-Extended, FacebookBot, facebookexternalhit and Twitterbot all say `Allow: /` and disallow `/api/`, `/auth/` and `/console/`; the file ends `Sitemap: https://www.lifebibleministry.com/sitemap.xml`. The tool asks only for the sitemap and for pages under `/through-the-bible/`, never a disallowed path, one request at a time, a second or more apart, with the User-Agent `" + userAgent + "`.\n\n")
	for _, reason := range skipReasons {
		var list []Skip
		for _, s := range r.Skipped {
			if s.Reason == reason {
				list = append(list, s)
			}
		}
		fmt.Fprintf(&b, "## Skipped: %s (%d)\n\n", reason, len(list))
		if len(list) == 0 {
			b.WriteString("None.\n\n")
			continue
		}
		for _, s := range list {
			line := "- " + s.URL
			if s.Title != "" {
				line += " (title: " + s.Title + ")"
			}
			if s.Err != "" {
				line += " (" + s.Err + ")"
			}
			b.WriteString(line + "\n")
		}
		b.WriteString("\n")
	}
	return b.String()
}
