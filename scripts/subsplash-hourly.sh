#!/bin/sh
# Link Brookfield First's sermons to their Subsplash pages as the Governor's
# mails and shared links arrive, and put them in the live database. The
# Laptop's timer sermon-index-subsplash-hourly.timer runs this every hour; by
# hand it is the same command.
#
#   scripts/subsplash-hourly.sh
#
# What it does, in order:
#   0. takes the lock it shares with scripts/refresh-brookfield.sh
#      ($SERMON_INDEX_HOME/.local/state/sermon-index/brookfield.lock): if the
#      refresh (or another hourly run) holds it, this run says so and exits 0,
#      the next hour's run does the work; the refresh WAITS for this one;
#   1. checks the checkout is on the branch (HOURLY_BRANCH, default main), that
#      its tree is clean and that it is not behind origin. If any is not so it
#      says why, exits 1 and changes NOTHING (it has run nothing yet): a dirty
#      tree is somebody's work, and a refresh's fast-forward would stop on it;
#   2. subsplash-notes fetch: the notes mails Postern kept (the records file
#      data/subsplash-notes.jsonl, git-ignored; the notes PDFs stay outside the
#      checkout);
#   2b. the sermons with no notes: subsplash-links --read-messages reads the
#      Governor's messages in the Postern inbox that end 'No notes' (a title, an
#      optional series, the date, the preacher and the passage, as he read them
#      in the church's app), only those signed by his key (the public key in
#      $GOVERNOR_KEY_FILE, default ~/.config/sermon-index/governor.pub), and adds
#      each to sources/brookfield-first/passages.tsv and no-notes.tsv; then
#      brookfield-youtube --merge-only (no API, no key) puts the title, preacher
#      and passage on the sermon's row (2d). A message it cannot read whole is
#      named by txid and changes nothing;
#   2c. the notes mails' own title and passage: subsplash-links --read-notes
#      adds a passages.tsv line (row = the record's row id, last column 'notes')
#      for each record whose row has no passage and only a generic title, since
#      data/subsplash-notes.jsonl is git-ignored and a row keeps what is in the
#      tracked list; a row his message already has a line for is not replaced
#      (a difference is named);
#   2d. brookfield-youtube --merge-only (no API, no key) puts passages.tsv on the
#      rows. Before the link step, so that a link whose address names such a
#      sermon finds its row by the new title;
#   3. the link list: every https://brookfieldfirst.subspla.sh/<code> in the
#      Postern inbox's messages ($POSTERN_INBOX, default ~/.local/state/mw/
#      postern/inbox), in a temporary file outside the checkout, and every
#      search PDF in the inbox that pdftohtml can read;
#   4. subsplash-links over those and the records, which writes each row's
#      alt_url (one HEAD per short link it has never seen, the cache keeps the
#      rest) and appends one line to $SERMON_INDEX_HOME/.local/state/sermon-index/
#      subsplash-progress.jsonl;
#   5. if rows.jsonl, passages.tsv, no-notes.tsv, source.toml or REPORT.md
#      changed: commits them ('Subsplash links: N new (hourly)', with ', M with
#      no notes' when no-notes.tsv gained M lines) and pushes, so the tree is never left dirty; a commit the remote does not
#      have yet (a refused push) is pushed by the next run;
#   6. asks the live service (SERMONS_URL, default https://sermons.allmymind.org)
#      for every Brookfield row it serves and compares each with rows.jsonl
#      (title, ref, date, preacher, url, alt_url: subsplash-links --compare).
#      If any differs, whatever wrote the row, it loads sources/brookfield-first
#      into the live database (sermons load, with DATABASE_URL from
#      ~/.config/sermon-index/env in a subshell) and compares again; if rows
#      still differ it writes one line naming the count and up to five row ids
#      and fails, which is the alarm. With the two equal nothing is loaded. A
#      service that cannot be asked fails the run; the git hash of the last load
#      (.../subsplash-loaded) is then the guide to whether to load.
#   7. writes the 'Subsplash links' job feed for Postern's Jobs page
#      ($POSTERN_DATA/jobs/subsplash-links.json, POSTERN_DATA defaulting to
#      ~/.local/state/postern): every Brookfield row newest first, with mail (a
#      notes record names it), link (it holds a Subsplash address) and live (the
#      service serves that address), and his place (subsplash-links --feed).
#      Titles and dates only. If the service cannot be asked the old file is
#      left and the run fails, so the page's "updated" goes stale.
# A failed step does not hide the later ones that can still run (a Postern that
# is down does not stop the inbox's links); the script exits 1 at the end if
# any step failed, and the timer's OnFailure= tells the Mayor.
#
# HOURLY_ROOT stands in for the checkout (the tests use it); SERMON_INDEX_HOME
# for $HOME; GO and SERMONS for the commands (the unit sets SERMONS to
# ~/.local/bin/sermons, which is not on its PATH).
set -eu

ROOT=${HOURLY_ROOT:-$(cd "$(dirname "$0")/.." && pwd)}
H=${SERMON_INDEX_HOME:-$HOME}
STATE=$H/.local/state/sermon-index
LOCK=$STATE/brookfield.lock
PROGRESS=$STATE/subsplash-progress.jsonl
LOADED=$STATE/subsplash-loaded
INBOX=${POSTERN_INBOX:-$H/.local/state/mw/postern/inbox}
BRANCH=${HOURLY_BRANCH:-main}
GO=${GO:-go}
SERMONS=${SERMONS:-sermons}
SERVICE=${SERMONS_URL:-https://sermons.allmymind.org}
ENVFILE=$H/.config/sermon-index/env
ROWS=sources/brookfield-first/rows.jsonl
KEYFILE=${GOVERNOR_KEY_FILE:-$H/.config/sermon-index/governor.pub}
FEED=${POSTERN_DATA:-$H/.local/state/postern}/jobs/subsplash-links.json
SRC=sources/brookfield-first

say() { echo "subsplash-hourly: $*"; }
die() { echo "subsplash-hourly: $*" >&2; exit 1; }

# 0. The lock.
mkdir -p "$STATE"
exec 9>"$LOCK"
if ! flock -n 9; then
	say "another run (the Brookfield refresh, or the last hourly run) holds $LOCK; nothing done, the next hour's run will link"
	exit 0
fi

# 1. The checkout.
cd "$ROOT"
[ "$(git rev-parse --abbrev-ref HEAD)" = "$BRANCH" ] || die "$ROOT is not on $BRANCH; nothing done"
dirty=$(git status --porcelain)
if [ -n "$dirty" ]; then
	die "the tree of $ROOT is not clean, so nothing was done (somebody's work is in it, and the refresh's fast-forward would stop on it): $(echo "$dirty" | head -n 3 | tr '\n' ';')"
fi
git fetch -q origin "$BRANCH" || die "cannot fetch origin $BRANCH; nothing done"
behind=$(git rev-list --count "HEAD..origin/$BRANCH")
[ "$behind" = 0 ] || die "$BRANCH is $behind commit(s) behind origin/$BRANCH (the refresh or a landing will bring it up to date); nothing done"

TMP=$(mktemp -d "${TMPDIR:-/tmp}/subsplash-hourly.XXXXXX")
trap 'rm -rf "$TMP"' EXIT
status=0

# 2. The notes mails.
if ! "$GO" run ./tools/subsplash-notes fetch; then
	echo "subsplash-hourly: subsplash-notes fetch failed (see above); going on with the records and links there are" >&2
	status=1
fi

# 2b. The sermons with no notes, from the Governor's messages, onto passages.tsv.
if [ -r "$KEYFILE" ] && [ -n "$(tr -d ' \t\r\n' < "$KEYFILE")" ]; then
	GKEY=$(tr -d ' \t\r\n' < "$KEYFILE")
	if "$GO" run ./tools/subsplash-links --read-messages "$INBOX" --key "$GKEY" --rows "$ROWS" > "$TMP/nonotes.out"; then :; else
		echo "subsplash-hourly: subsplash-links --read-messages failed (see above)" >&2
		status=1
	fi
	cat "$TMP/nonotes.out"
else
	echo "subsplash-hourly: cannot read the Governor's public key at $KEYFILE; his 'No notes' messages were not read" >&2
	status=1
fi

# 2c. The title and passage of each notes mail, onto passages.tsv: the records
# file is git-ignored, so only a line of the tracked list keeps them on the row.
if "$GO" run ./tools/subsplash-links --read-notes --rows "$ROWS" --passages "$SRC/passages.tsv" > "$TMP/notes.out"; then :; else
	echo "subsplash-hourly: subsplash-links --read-notes failed (see above)" >&2
	status=1
fi
cat "$TMP/notes.out"

# 2d. passages.tsv onto the rows (no API, no key). Before the link step, so that
# a link whose address names such a sermon finds its row by the new title.
if ! "$GO" run ./tools/brookfield-youtube --merge-only --out "$SRC"; then
	echo "subsplash-hourly: brookfield-youtube --merge-only failed (see above)" >&2
	status=1
fi

# 3. The link list, from the messages in the inbox, and the search PDFs.
: > "$TMP/links.txt"
if [ -d "$INBOX" ]; then
	find "$INBOX" -maxdepth 1 -name '*.message.json' -print0 | xargs -0 -r cat 2>/dev/null |
		grep -o 'brookfieldfirst\.subspla\.sh/[A-Za-z0-9]*' | sort -u | sed 's|^|https://|' > "$TMP/links.txt" || true
fi
say "$(wc -l < "$TMP/links.txt" | tr -d ' ') shared links in $INBOX"
set -- --links "$TMP/links.txt" --progress "$PROGRESS"
if [ -d "$INBOX" ] && command -v pdftohtml >/dev/null 2>&1; then
	for f in "$INBOX"/*.pdf; do
		[ -f "$f" ] || continue
		if pdftohtml -stdout -xml -i "$f" >/dev/null 2>&1; then
			set -- "$@" --pdf "$f"
		else
			echo "subsplash-hourly: $f is not a PDF pdftohtml can read; skipped" >&2
		fi
	done
fi

# 4. The links.
if "$GO" run ./tools/subsplash-links "$@" > "$TMP/links.out"; then :; else
	echo "subsplash-hourly: subsplash-links failed (see above)" >&2
	status=1
fi
cat "$TMP/links.out"

# 5. Commit and push what changed.
PATHS=
for f in rows.jsonl passages.tsv no-notes.tsv source.toml REPORT.md; do
	[ -e "$SRC/$f" ] && PATHS="$PATHS $SRC/$f"
done
# shellcheck disable=SC2086
if [ -n "$(git status --porcelain -- $PATHS)" ]; then
	# shellcheck disable=SC2086
	git add -- $PATHS
	new=$(git diff --cached --numstat -- "$ROWS" | cut -f1)
	[ -n "$new" ] || new=0
	nn=$(git diff --cached --numstat -- "$SRC/no-notes.tsv" | cut -f1)
	[ -n "$nn" ] || nn=0
	# a new no-notes.tsv counts its header line too
	if [ "$nn" -gt 0 ] && ! git cat-file -e "HEAD:$SRC/no-notes.tsv" 2>/dev/null; then nn=$((nn - 1)); fi
	if [ "$nn" -gt 0 ]; then
		msg="Subsplash links: $new new, $nn with no notes (hourly)"
	else
		msg="Subsplash links: $new new (hourly)"
	fi
	# shellcheck disable=SC2086
	if ! git commit -q -m "$msg" -- $PATHS; then
		# shellcheck disable=SC2086
		git reset -q -- $PATHS || true
		for f in $PATHS; do
			if git ls-files --error-unmatch "$f" >/dev/null 2>&1; then git checkout -q -- "$f"; else rm -f -- "$f"; fi
		done
		die "git commit failed; $SRC was put back as it was"
	fi
	say "committed $(git rev-parse --short HEAD): $msg"
fi
if [ -n "$(git rev-list "origin/$BRANCH..HEAD")" ]; then
	if git push -q origin "$BRANCH"; then
		say "pushed to origin/$BRANCH"
	else
		echo "subsplash-hourly: git push origin $BRANCH failed; the commit is kept here and the next run pushes it" >&2
		status=1
	fi
fi

# 6. Load what the live service lacks. THE SERVICE decides, not this run's own
# changes: a row written by a story, a hand edit or the refresh is as unloaded as
# one this run wrote. subsplash-links --compare asks the service
# ($SERVICE/v1/sermons) for every Brookfield row it serves and sets it beside
# rows.jsonl (title, ref, date, preacher, url, alt_url); if any differs it is
# loaded, and the two are compared again. If rows still differ the run says so
# in one line (the count and up to five row ids) and fails, which is the alarm:
# the unit's OnFailure=. The git hash of what was last loaded (.../subsplash-loaded,
# written after a good load and by the refresh) is only the fallback for a
# service that cannot be asked: the load goes on when the hash says it is new.
compare() {
	cmp_n=
	cmp_ids=
	if "$GO" run ./tools/subsplash-links --compare "$SERVICE" --rows "$ROWS" > "$TMP/compare.out" 2> "$TMP/compare.err"; then
		cmp_n=$(sed -n 's/^compare: \([0-9][0-9]*\) of .*/\1/p' "$TMP/compare.out" | head -n 1)
		cmp_ids=$(sed -n 's/^ids: //p' "$TMP/compare.out" | head -n 1)
		[ -n "$cmp_n" ] && return 0
	fi
	echo "subsplash-hourly: cannot compare rows.jsonl with $SERVICE: $(head -n 3 "$TMP/compare.err" | tr '\n' ' ')" >&2
	return 1
}

want=$(git hash-object -- "$ROWS")
have=$(cat "$LOADED" 2>/dev/null || true)
need=
if compare; then
	if [ "$cmp_n" = 0 ]; then
		say "the service holds rows.jsonl as it is; nothing loaded"
		echo "$want" > "$LOADED.new"
		mv "$LOADED.new" "$LOADED"
	else
		say "$cmp_n row(s) differ from the service ($cmp_ids); loading"
		need=1
	fi
else
	status=1
	[ "$want" = "$have" ] || need=1
fi
if [ -n "$need" ]; then
	if [ ! -r "$ENVFILE" ]; then
		echo "subsplash-hourly: cannot read $ENVFILE; $SRC was not loaded" >&2
		status=1
	elif (
		set -a
		# shellcheck disable=SC1090
		. "$ENVFILE"
		set +a
		exec "$SERMONS" load "$ROOT/$SRC"
	); then
		echo "$want" > "$LOADED.new"
		mv "$LOADED.new" "$LOADED"
		say "loaded $ROOT/$SRC into the database"
		if compare; then
			if [ "$cmp_n" = 0 ]; then
				say "the service now holds rows.jsonl as it is"
			else
				echo "subsplash-hourly: after the load $cmp_n row(s) still differ from the service ($cmp_ids)" >&2
				status=1
			fi
		else
			status=1
		fi
	else
		echo "subsplash-hourly: sermons load $ROOT/$SRC failed (see above); the next run loads again" >&2
		status=1
	fi
fi

# 7. The job feed, last, so that "live" is what the service holds now.
if ! "$GO" run ./tools/subsplash-links --feed "$FEED" --service "$SERVICE" --rows "$ROWS" --no-notes-file "$SRC/no-notes.tsv"; then
	echo "subsplash-hourly: the job feed $FEED was not written (see above)" >&2
	status=1
fi

[ "$status" = 0 ] || die "done with errors (see above)"
say "done"
