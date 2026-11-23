# Life Bible Ministry, Through the Bible: what the pages do not settle

<!-- toc: none -->

Written by `go run ./tools/lifebible-ttb --cache .lifebible-cache --out sources/lifebible-ttb`; do not edit by hand. Only addresses, titles, passages and YouTube ids are listed or kept: never audio, video, a transcript or a page's description.

## Counts

```
through-the-bible addresses in the sitemap:                        1074
rows written:                                                      995
  of them series (a whole book over 3 chapters):                   2
  of them over 3 chapters, not series:                             52
addresses skipped:                                                 79
  the Through the Bible list page:                                 1
  a book page (the list of a book's teachings):                    66
  no passage in the page title:                                    10
  no YouTube video on the page:                                    1
  the same address (or row id) as an earlier one in the sitemap:   1
```

Every `through-the-bible` address of `https://www.lifebibleministry.com/sitemap.xml` is a row of `rows.jsonl` or is named below: rows + skipped = addresses.

A row's `id` is the address's path below `/through-the-bible/` with `/` written `.` (`isaiah/1` is `isaiah.1`). Its `url` is the page, `alt_url` the YouTube watch address of the video the page embeds, so Lampas's `videoOf()` plays it. The row has **no date**: the page's `uploadDate` is the day the site imported its pages (2026-04-18 and 2026-04-19 for almost all of them), not the day the teaching was given, so it is not written as one.

**Wide rows.** Through the Bible teaches up to seventeen chapters at a time ("Isaiah 1-4:1", "Job 15–31"), so a row whose ref spans more than 3 chapters is the teaching's real text, not an overview, and is kept as a plain row with its full ref: it shows at each chapter it teaches (the Governor's "Isaiah 1" sheet needs this). `sermons check` warns of each (one warning a wide row); that is answered here, not by `series`. Only a title that names a whole book of more than 3 chapters ("Lamentations: ...") is a `series` row, with its first chapter as ref.

## Terms and robots.txt (read 2026-11-23)

The site has **no terms of use**: `/terms`, `/privacy`, `/terms-of-use`, `/legal` and `/copyright` all answer 404 and the page footer has no legal link. Its footer reads "© 2026 Life Bible Ministry · Pastor Paul LeBoutillier" and its `/about` page says Life Bible Ministry is "a global teaching resource — where every message, transcript, and study is available completely free of charge to anyone in the world" and, under Our Mission, "To make thorough, verse-by-verse Bible teaching available to every person in the world — completely free of charge" (statements of purpose, not a clause on automated access or on linking). No terms say yes and none say no; the Governor read this on 2026-11-23 (his tap, txid direct:cbd35263b1ee4f0d463e475a928c110f09e551f0493cbc44081935fd559c5dcf) and said to go ahead because `robots.txt` allows it and we keep only links, titles, passages and YouTube ids.

`https://www.lifebibleministry.com/robots.txt`, the block for every robot, word for word:

```
User-Agent: *
Allow: /
Disallow: /api/
Disallow: /auth/
Disallow: /console/
Disallow: /locked/
Disallow: /background
```

Its blocks for Googlebot, Bingbot, Google-Extended, GPTBot, ChatGPT-User, OAI-SearchBot, ClaudeBot, anthropic-ai, PerplexityBot, cohere-ai, Amazonbot, YouBot, Applebot, Applebot-Extended, FacebookBot, facebookexternalhit and Twitterbot all say `Allow: /` and disallow `/api/`, `/auth/` and `/console/`; the file ends `Sitemap: https://www.lifebibleministry.com/sitemap.xml`. The tool asks only for the sitemap and for pages under `/through-the-bible/`, never a disallowed path, one request at a time, a second or more apart, with the User-Agent `sermon-index-lifebible-ttb (+https://github.com/Jonathan-A-White/sermon-index)`.

## Skipped: the Through the Bible list page (1)

- https://www.lifebibleministry.com/through-the-bible

## Skipped: a book page (the list of a book's teachings) (66)

- https://www.lifebibleministry.com/through-the-bible/genesis
- https://www.lifebibleministry.com/through-the-bible/exodus
- https://www.lifebibleministry.com/through-the-bible/leviticus
- https://www.lifebibleministry.com/through-the-bible/numbers
- https://www.lifebibleministry.com/through-the-bible/deuteronomy
- https://www.lifebibleministry.com/through-the-bible/joshua
- https://www.lifebibleministry.com/through-the-bible/judges
- https://www.lifebibleministry.com/through-the-bible/ruth
- https://www.lifebibleministry.com/through-the-bible/1-samuel
- https://www.lifebibleministry.com/through-the-bible/2-samuel
- https://www.lifebibleministry.com/through-the-bible/1-kings
- https://www.lifebibleministry.com/through-the-bible/2-kings
- https://www.lifebibleministry.com/through-the-bible/1-chronicles
- https://www.lifebibleministry.com/through-the-bible/2-chronicles
- https://www.lifebibleministry.com/through-the-bible/ezra
- https://www.lifebibleministry.com/through-the-bible/nehemiah
- https://www.lifebibleministry.com/through-the-bible/esther
- https://www.lifebibleministry.com/through-the-bible/job
- https://www.lifebibleministry.com/through-the-bible/psalms
- https://www.lifebibleministry.com/through-the-bible/proverbs
- https://www.lifebibleministry.com/through-the-bible/ecclesiastes
- https://www.lifebibleministry.com/through-the-bible/song-of-solomon
- https://www.lifebibleministry.com/through-the-bible/isaiah
- https://www.lifebibleministry.com/through-the-bible/jeremiah
- https://www.lifebibleministry.com/through-the-bible/lamentations
- https://www.lifebibleministry.com/through-the-bible/ezekiel
- https://www.lifebibleministry.com/through-the-bible/daniel
- https://www.lifebibleministry.com/through-the-bible/hosea
- https://www.lifebibleministry.com/through-the-bible/joel
- https://www.lifebibleministry.com/through-the-bible/amos
- https://www.lifebibleministry.com/through-the-bible/obadiah
- https://www.lifebibleministry.com/through-the-bible/jonah
- https://www.lifebibleministry.com/through-the-bible/micah
- https://www.lifebibleministry.com/through-the-bible/nahum
- https://www.lifebibleministry.com/through-the-bible/habakkuk
- https://www.lifebibleministry.com/through-the-bible/zephaniah
- https://www.lifebibleministry.com/through-the-bible/haggai
- https://www.lifebibleministry.com/through-the-bible/zechariah
- https://www.lifebibleministry.com/through-the-bible/malachi
- https://www.lifebibleministry.com/through-the-bible/matthew
- https://www.lifebibleministry.com/through-the-bible/mark
- https://www.lifebibleministry.com/through-the-bible/luke
- https://www.lifebibleministry.com/through-the-bible/john
- https://www.lifebibleministry.com/through-the-bible/acts
- https://www.lifebibleministry.com/through-the-bible/romans
- https://www.lifebibleministry.com/through-the-bible/1-corinthians
- https://www.lifebibleministry.com/through-the-bible/2-corinthians
- https://www.lifebibleministry.com/through-the-bible/galatians
- https://www.lifebibleministry.com/through-the-bible/ephesians
- https://www.lifebibleministry.com/through-the-bible/philippians
- https://www.lifebibleministry.com/through-the-bible/colossians
- https://www.lifebibleministry.com/through-the-bible/1-thessalonians
- https://www.lifebibleministry.com/through-the-bible/2-thessalonians
- https://www.lifebibleministry.com/through-the-bible/1-timothy
- https://www.lifebibleministry.com/through-the-bible/2-timothy
- https://www.lifebibleministry.com/through-the-bible/titus
- https://www.lifebibleministry.com/through-the-bible/philemon
- https://www.lifebibleministry.com/through-the-bible/hebrews
- https://www.lifebibleministry.com/through-the-bible/james
- https://www.lifebibleministry.com/through-the-bible/1-peter
- https://www.lifebibleministry.com/through-the-bible/2-peter
- https://www.lifebibleministry.com/through-the-bible/1-john
- https://www.lifebibleministry.com/through-the-bible/2-john
- https://www.lifebibleministry.com/through-the-bible/3-john
- https://www.lifebibleministry.com/through-the-bible/jude
- https://www.lifebibleministry.com/through-the-bible/revelation

## Skipped: the page could not be read (0)

None.

## Skipped: no passage in the page title (10)

- https://www.lifebibleministry.com/through-the-bible/romans/1-1 (title: Romans (Part 1) :1-17: Paul, a servant of Christ Jesus)
- https://www.lifebibleministry.com/through-the-bible/2-corinthians/2026/1-2 (title: 2 Corinthians 1 (Part 2) :12 24: You stand firm in your faith)
- https://www.lifebibleministry.com/through-the-bible/1-corinthians/small-group/1cor-2025-intro (title: How To Use This Small Group Study: HOW TO USE THIS STUDY)
- https://www.lifebibleministry.com/through-the-bible/matthew/in-depth (title: Life Bible Ministry: Study Through All 66 Books of the Bible)
- https://www.lifebibleministry.com/through-the-bible/1-corinthians/small-group (title: Life Bible Ministry: Study Through All 66 Books of the Bible)
- https://www.lifebibleministry.com/through-the-bible/matthew/by-chapter (title: Life Bible Ministry: Study Through All 66 Books of the Bible)
- https://www.lifebibleministry.com/through-the-bible/romans/romans-2016 (title: Life Bible Ministry: Study Through All 66 Books of the Bible)
- https://www.lifebibleministry.com/through-the-bible/1-corinthians/in-depth (title: Life Bible Ministry: Study Through All 66 Books of the Bible)
- https://www.lifebibleministry.com/through-the-bible/2-corinthians/2017 (title: Life Bible Ministry: Study Through All 66 Books of the Bible)
- https://www.lifebibleministry.com/through-the-bible/2-corinthians/2026 (title: 2 Corinthians (2026): Through the Bible)

## Skipped: no YouTube video on the page (1)

- https://www.lifebibleministry.com/through-the-bible/2-timothy/2-2 (title: 2 Timothy 2 (Part 2) :14-26: Rightly Dividing the Word of Truth)

## Skipped: the same address (or row id) as an earlier one in the sitemap (1)

- https://www.lifebibleministry.com/through-the-bible/nehemiah/9-10 (title: Nehemiah 9–10)

