# Operations

Running the sermon index on the Laptop: where to look, what to do after a change, what each failure looks like. The units and scripts are described in [README.md, "On the Laptop"](../README.md#on-the-laptop) and [deploy/laptop/README.md](../deploy/laptop/README.md).

<!-- toc -->

- [Looking](#looking)
- [After a change](#after-a-change)
- [What a wrong setting looks like](#what-a-wrong-setting-looks-like)
- [The Brookfield First notes](#the-brookfield-first-notes)
- [Reading Subsplash through the signed-in browser](#reading-subsplash-through-the-signed-in-browser)
  - [The inventory of every sermon](#the-inventory-of-every-sermon)

<!-- /toc -->

## Looking

```sh
systemctl --user status sermon-index sermon-index-pg
journalctl --user -u sermon-index -f          # the service: one line a request on stderr, and its start and stop
journalctl --user -u sermon-index-pg -f       # Postgres, as the container prints it
docker logs sermon-index-pg                   # the same, from Docker (if refused: sg docker -c 'docker logs sermon-index-pg')
scripts/smoke.sh                              # is it up, loaded and right: exit 0 or 1 with the reason
curl -s http://10.88.0.2:8790/v1/health       # {status, version, sources, sermons}; 503 when the database does not answer
```

`/v1/health`'s `version` is the git describe the binary was built from.

## After a change

- **A new source directory, or a changed row, landed in `sources/`**: `sermons check --all sources`, then load it into the live database with the service's own settings:

  ```sh
  set -a; . ~/.config/sermon-index/env; set +a
  ~/.local/bin/sermons load --all <the clone>/sources     # idempotent: an unchanged row is left alone; prints the counts
  ```

  The Brookfield refresh (`scripts/refresh-brookfield.sh`, README "The Brookfield First refresh") does this load itself: it ends with `sermons load --all <the clone>/sources`, after the commit and push and before the stamp, so Postgres never keeps YouTube API data past the 30 days the Developer Policies allow. A failed load fails the run (the Mayor is mailed) and leaves the stamp as it was.

  The service reads the database on every request, so there is nothing to restart; its answers carry an ETag and may be cached for five minutes (docs/api.md).
- **New code**: `git pull`, `scripts/install-laptop.sh`: it rebuilds, restarts the service if the binary changed, applies migrations and loads every source. A migration that adds a column the loader fills needs the load (the install runs it).
- **A unit file changed**: the install copies it and runs `daemon-reload`; a changed Postgres unit is left running (the install says so): apply it with `systemctl --user restart sermon-index-pg`.
- **Backup and move**: README, "On the Laptop".

## What a wrong setting looks like

| Assumed | A wrong one shows as |
| --- | --- |
| `10.88.0.2` (the WireGuard address, `SERMONS_ADDR`) | `sermon-index` fails to start with `listen tcp 10.88.0.2:8790: bind: cannot assign requested address` when the interface is down or the address differs; the install's smoke test says "service down" |
| The user is in the `docker` group, in `/etc/group` | the install stops at once with "cannot talk to Docker, with or without sg docker -c" (when only the session lacks the group, the install and the unit use `sg docker -c` and work); the Postgres unit fails with `permission denied while trying to connect to the Docker daemon socket` |
| `/usr/bin/docker` (the unit's absolute path) | `status` shows `Failed to locate executable /usr/bin/docker` (status 203/EXEC); `which docker` says where it is, edit the unit |
| `~/.local/bin` on the user manager's `PATH` | not needed by the units (absolute `%h` paths); the install's `go` must be on the shell's `PATH` or it stops with "go is not on PATH" |
| `~/.config/sermon-index/env` | a missing file: the unit fails with `Failed to load environment files: No such file or directory`; a file without `DATABASE_URL`: `no database URL: set DATABASE_URL` in the journal |
| The password in `pg.env` matches `env` | `sermon-index` starts, `/v1/health` answers 503, the journal says `password authentication failed`; the password in `pg.env` only counts when the data directory is first filled |
| Port 5433 free on the loopback | `sermon-index-pg` fails with `port is already allocated`; the install's wait for Postgres times out |
| User lingering (`loginctl enable-linger`) | the units run only while the user is logged in; after a reboot nothing answers until login |

## The Brookfield First notes

`go run ./tools/subsplash-notes fetch` (the package comment of `tools/subsplash-notes/main.go` is its contract) lists the family's sermon-notes alerts at Postern and, for each it can match to one Brookfield First row, writes a record (`data/subsplash-notes.jsonl`, gitignored) and claims the alert.

An alert whose mail had the church's notes PDF carries `notes_sha256` and `notes_size`; its bytes are at the signed `GET /api/alerts/{id}/notes`. `fetch` downloads them for each alert it will record, checks the size and the sha256, and keeps the PDF at **`~/.local/state/sermon-index/notes/<row id>.pdf`** (`--notes <dir>` to change; the directory is mode 0700, the files 0600). That is the Laptop only, outside the checkout: the PDF holds the church's notes and the Governor's own fill-ins, so it is **never committed**, and a `--notes` directory inside a git work tree is refused. Only a record (Sunday, title, passage) is ever written under the checkout.

The order is kept so nothing is lost: **the PDF on disk, then the records file in place, and only then the alert claimed** (a claimed alert is gone from Postern, notes and all). A failed download, a size that is not `notes_size` or a sha256 that is not `notes_sha256` is reported on stderr, writes no record for that alert and leaves it unclaimed; the other alerts of the run go on. A second run writes the same bytes and does not download a PDF it already holds with the same sha256; a file there with another sha256 is, when the row's record is the one this alert gives (a second mail for the same sermon, the church's PDF made again), replaced by the newer mail's notes with the older kept beside as `<row id>.prev.pdf`; with no such record it is reported and not replaced (look at it, delete it by hand if the new one is right), and so is a mail for another sermon that falls on a row with a record (reported with both titles). Each refusal is one line saying what to do, and the summary line counts them (`n not replaced`). An alert may also carry a `speaker` (the preacher the mail names): the record keeps it, and `subsplash-links --read-notes` writes it as the line's speaker only for a row that has no preacher. An alert that carries any other field than the ones the reader knows is dropped, reported and left unclaimed. The no-subcommand reader (a saved reply of `GET /api/alerts`) accepts the two fields, downloads nothing and claims nothing.

## Reading Subsplash through the signed-in browser

`go run ./tools/subsplash-notes browse` is the second way in beside the mail: it reads, through the Laptop's `mw browser` on the Governor's own signed-in Subsplash session, what he would read by hand, and writes what the by-hand job writes. It is the Governor's allowed use of his own account at a polite pace, with the account's risk his (his B on card `direct:09581219`, 2026-11-16); Subsplash's terms (A.5 and A.6) bar automated means, so the rules below are the whole of what it does.

**Run it.**

1. `mw browser open https://subsplash.com/brookfieldfirst/media` and sign in himself over VNC (the tool types no password and keeps no session; `mw browser close` deletes the session). `mw browser status` prints the DevTools address (`http://127.0.0.1:9222`, which is `--cdp`'s default; only an address on this machine is taken).
2. From `~/sermon-index`: `go run ./tools/subsplash-notes browse`. It needs `pdftotext` only for `--verify`.
3. Then `go run ./tools/subsplash-links --links data/subsplash-browser-links.txt`, which files the addresses on their rows with no request at all (the address is in the file already, so there is nothing to look up). The hourly script reads that file too (SUBSPLASH_BROWSER_LINKS names another), so this step is only needed between hourly runs.

**What it reads.** One tab, one page load at a time, at least 10 s between two loads (`--pace` cannot go lower): the sermon list (`https://subsplash.com/brookfieldfirst/media`, then its pages 2 to the last, `?page=n`, the address its pager gives them), each sermon's page (`/u/brookfieldfirst/media/d/<code>-<slug>`), its notes (the Notes button's page) and each Morning RePhil episode's page. That list holds only the newest 200 items, so after it browse reads the church site's series index (`https://brookfieldfirst.org/media/`, then `/media/page/n/`; `--series-index ''` leaves it out) and each series' page with View More pressed until it shows every sermon, and reads each sermon of them not already seen, as it does the list's (the Morning RePhil series has its own pass). An address that is none of those is refused before it is loaded. The share link is the short link its code makes (`https://brookfieldfirst.subspla.sh/<code>`): the code is in the sermon's address, so the Share button is never pressed.

**What it writes.** Newest to oldest. A sermon with a record of its title and its address already on a row (or in the links file) is not read again. For any other: the record the mail would have made (`data/subsplash-notes.jsonl`: id, date, title, ref, and the speaker the page names), the notes' text as `<row id>.txt` in `~/.local/state/sermon-index/notes` beside the PDFs (0600, never committed, refused inside a git work tree) and the sermon's full address in `data/subsplash-browser-links.txt` (git-ignored). A sermon with no Notes button is not recorded, only its address is kept. A sermon whose notes name a passage or a date the matcher cannot place is reported, and not marked read, so the next run reads it again. A record already on file is never replaced (a different one is reported). The matching is `MatchNote`, the mail's own; there is no second matcher.

**The Morning RePhil series.** After the sermon list it reads the series page on the church's site (`https://brookfieldfirst.org/media/series/4fmk6cd/morning-rephil`, 12 episodes a press of View More, 223 in all): each press is one paced load, and every episode's number, title, date and share code is on that page, so no episode page is loaded (`--episode-pages` also loads each new one, to keep any text it carries; the pages read so far carry none; `--series-only` reads only the series). Episodes are written to `data/morning-rephil.jsonl` (tracked), oldest first: `number`, `title`, `date`, the YouTube `url` where `tools/brookfield-youtube` has the video, `share` (`https://brookfieldfirst.subspla.sh/<code>`) and, with `--episode-pages`, `notes_file` (that text, kept in the notes directory, not in the repo, which is public). The number is the one the title prints; an episode whose title prints none is numbered by its place among all the episodes, the oldest being 1 (it is said when it happens; the 2026-11-16 run had none). An episode already on file from YouTube is given its share link only, matched by number, or by day when its YouTube title prints no number. A re-run adds only episodes with no share link on file, and presses nothing when every episode has one. `tools/brookfield-youtube`'s refresh keeps the `share` and `notes_file` fields and the episodes only Subsplash has. The sermon list of the Subsplash site holds only the newest 200 items (10 pages of 20, Morning RePhil episodes among them), so older sermons are not reached this way.

**When it stops.** At the first login prompt, captcha, HTTP 429 or 403, or page that is not the one expected (a "Page not found", the wrong address, no date on a sermon page, notes that do not open with a quoted title), it prints why on stderr and exits 1 with nothing more written. The next run resumes where it stopped and changes nothing already written: sign in again first if it was asked to. `--limit n` reads at most n sermon and episode pages, then ends normally.

**Check it against the mail.** `go run ./tools/subsplash-notes browse --verify 3` reads the newest three sermons that already have a mail-made record, says for each whether its title, date, row id, passage, link code and notes text (compared with the PDF's words, spaces left out) are the same, and writes nothing. A date line in the notes can differ by one day: the PDF a Saturday mail made prints that Saturday, the page the Sunday.

**The Jobs page.** `--feed` (default `$POSTERN_DATA/jobs/subsplash-browse.json`, `~/.local/state/postern/jobs/` otherwise) is written after each sermon or episode with one tick ("read") for each item the list has shown, newest first, and the oldest read one as the place. The records it writes also tick "mail" and, once `subsplash-links` has run, "link" on the hourly feed `subsplash-links`.

### The inventory of every sermon

`go run ./tools/subsplash-notes inventory` reads every sermon on the Governor's signed-in account, newest to oldest, by the rules above (one tab, one load at a time, at least 10 s between two loads, the same stops at a login prompt, captcha, HTTP 429 or 403 or unexpected page; his B of 2026-11-16 covers it, the account's risk his). It walks the sermon list (newest 200), then the church site's series pages for the older ones, and loads **every** sermon's page, even one browse has read before, because the inventory needs what only that page shows: the preacher, the passage, the series (the line "More from <series>") and, from the Notes button, the address of the notes page. The Morning RePhil episodes are not sermons and are left to `browse`.

It writes `data/subsplash-inventory.jsonl` (git-ignored, like the links file; `--inventory` names another), one line a sermon, newest first: `code`, `address`, `share`, `date`, `title`, `preacher`, `passage` (as printed), `ref` (OSIS), `series`, `themes`, `has_notes`, `notes_link`, and `row` (the id `MatchNote` gave it, the mail's matcher; there is no second one) or `no_row` with the reason. Each sermon's address is also added to the links file, as `browse` does. A sermon already in the inventory is not loaded again, so a second run resumes where the first stopped and changes no line already written; `--limit n` reads at most n sermon pages. The Jobs page has its own feed, `subsplash-inventory.json`.

`themes` stays empty: the app's sermon page and its notes name no theme or tag, and the sermon-notes mails (Postern kind `sermon-notes`) carry a title, passage, date and sometimes a speaker. The "Related by Topic" the page prints lists other sermons, not tags.

`go run ./tools/subsplash-notes inventory-fill` needs no browser. It matches each entry to its row with `MatchNote`, and for each row that lacks something Subsplash has (a generic title such as "Online Service | <date>", no preacher, no passage) appends one line to `sources/brookfield-first/passages.tsv` (`from` = `browse`, the row's id in `row`; a row the list already has a line for, and a row two sermons claim, are left alone; what the row already has is kept), then writes `docs/brookfield-first-subsplash-inventory.md`: what Subsplash has that the index lacks and the reverse (counts and ids), and the counts of sermons with verses, with themes and with neither. Run `go run ./tools/brookfield-youtube --merge-only` after it to put the lines on the rows and `go run ./tools/subsplash-links --links data/subsplash-browser-links.txt` to file the addresses. A second run adds nothing.
