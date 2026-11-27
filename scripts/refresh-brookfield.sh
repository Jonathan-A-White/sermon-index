#!/bin/sh
# Regenerate Brookfield First's YouTube list and commit it when it changed.
#
#   scripts/refresh-brookfield.sh
#
# YouTube's Developer Policies (III.E.4) let this rig keep public API data for
# 30 calendar days, then it must be deleted or refreshed. The Laptop's timer
# sermon-index-brookfield-refresh.timer runs this every day at 18:00 UTC, so a
# Morning RePhil episode (Mondays and Fridays) is listed the same day; a run
# spends about 12 units of the API's 10,000 a day. By hand it is the same command.
#
# What it does, in order:
#   1. checks the checkout is on the branch (REFRESH_BRANCH, default main), that
#      sources/brookfield-first, data/morning-rephil.jsonl and sources/allennolan
#      have no edit of anyone's, and brings the branch up to date (fast-forward
#      only);
#   2. reads the key from the vault into one variable, and runs
#        YOUTUBE_API_KEY=<key> go run ./tools/brookfield-youtube
#        YOUTUBE_API_KEY=<key> go run ./tools/allennolan-youtube
#      (allennolan: Pastor Allen Nolan's book playlists, a second source whose
#      YouTube data has the same 30 days) so the key is in the environment of
#      each command alone and nowhere else: it is never written to a file,
#      printed, logged or put on a command line;
#   3. if a tool failed, puts its paths back as they were (the other tool's
#      output is still kept: one source's failure never holds the other's
#      refresh back) and, once the steps below are done, exits non-zero (the
#      timer's OnFailure= then tells the Mayor); the stamp is not touched;
#   4. if the output changed, commits the changed paths; pushes any commit the
#      remote does not have yet (so a push that was refused is retried by the
#      next run);
#   5. loads the sources into the live database, the step that makes the 30 days
#      true: Postgres holds the YouTube data too, so a refreshed file alone
#      leaves the old rows there. In a subshell that reads the service's env
#      file ($SERMON_INDEX_HOME/.config/sermon-index/env, as the install does)
#        sermons load --all <checkout>/sources
#      it runs after an unchanged run as well (the database may lag a landing),
#      and a failed load, or a missing env file, exits non-zero (the timer's
#      OnFailure= then tells the Mayor) with the stamp not touched;
#   6. only then, and only when both tools succeeded, writes the stamp,
#      $SERMON_INDEX_HOME/.local/state/sermon-index/brookfield-refreshed (the
#      epoch second), which scripts/check-brookfield-stale.sh reads: one stamp
#      for both sources, so a stamp over 27 days old is mailed whichever failed.
#      A run that found the lists unchanged is a good refresh: the 30 days start
#      again.
#
# THE LOCK. scripts/subsplash-hourly.sh (every hour) rewrites alt_url in the same
# rows.jsonl this regenerates, commits it and loads it, so the two take one lock,
# $SERMON_INDEX_HOME/.local/state/sermon-index/brookfield.lock (flock), before
# they touch the checkout. The hourly run that finds it held skips the hour; this
# run waits for it (REFRESH_LOCK_WAIT seconds, default 1800) and fails if it
# cannot get it. The regeneration keeps every row's alt_url by row id
# (tools/brookfield-youtube, KeepAltURLs), so a link the hourly run wrote
# survives; a row it linked since the refresh began cannot be lost, for the
# hourly run cannot write while this one holds the lock.
#
# REFRESH_ROOT stands in for the checkout (the tests use it); SERMON_INDEX_HOME
# for $HOME; MW, GO and SERMONS for the commands (the unit sets SERMONS to
# ~/.local/bin/sermons, which is not on its PATH).
set -eu

ROOT=${REFRESH_ROOT:-$(cd "$(dirname "$0")/.." && pwd)}
H=${SERMON_INDEX_HOME:-$HOME}
STAMP=$H/.local/state/sermon-index/brookfield-refreshed
BRANCH=${REFRESH_BRANCH:-main}
MW=${MW:-mw}
GO=${GO:-go}
SERMONS=${SERMONS:-sermons}
ENVFILE=$H/.config/sermon-index/env
BF_DATA="sources/brookfield-first data/morning-rephil.jsonl"
AN_DATA="sources/allennolan"
DATA="$BF_DATA $AN_DATA"
STATE=$H/.local/state/sermon-index
LOCK=$STATE/brookfield.lock
LOCK_WAIT=${REFRESH_LOCK_WAIT:-1800}

die() { echo "refresh-brookfield: $*" >&2; exit 1; }

mkdir -p "$STATE"
exec 9>"$LOCK"
flock -w "$LOCK_WAIT" 9 || die "$LOCK is held (scripts/subsplash-hourly.sh is running?) and was not free in ${LOCK_WAIT}s; nothing was done, the stamp is as it was"

cd "$ROOT"
[ "$(git rev-parse --abbrev-ref HEAD)" = "$BRANCH" ] || die "$ROOT is not on $BRANCH; it will not commit there"
# shellcheck disable=SC2086 # DATA is two paths with no space in either
[ -z "$(git status --porcelain -- $DATA)" ] || die "$DATA has uncommitted changes; leave them to whoever made them"
git fetch -q origin "$BRANCH" || die "cannot fetch origin $BRANCH"
git merge -q --ff-only "origin/$BRANCH" || die "$BRANCH has diverged from origin/$BRANCH; resolve it by hand"

key=$("$MW" secrets get youtube_api_key) || die "mw secrets get youtube_api_key failed; nothing was fetched"
[ -n "$key" ] || die "the vault holds an empty youtube_api_key; nothing was fetched"

# A prefix assignment puts the key in the environment of one command each. A
# tool that fails has its paths put back, and the run goes on with the other's
# output, failing at the end (FAILED) with the stamp untouched.
FAILED=
if ! YOUTUBE_API_KEY=$key "$GO" run ./tools/brookfield-youtube; then
	# shellcheck disable=SC2086
	git checkout -q -- $BF_DATA
	echo "refresh-brookfield: brookfield-youtube failed (see above); its list is as it was" >&2
	FAILED="$FAILED brookfield-youtube"
fi
if ! YOUTUBE_API_KEY=$key "$GO" run ./tools/allennolan-youtube; then
	# shellcheck disable=SC2086
	git checkout -q -- $AN_DATA
	echo "refresh-brookfield: allennolan-youtube failed (see above); its list is as it was" >&2
	FAILED="$FAILED allennolan-youtube"
fi
key=

# shellcheck disable=SC2086
if [ -n "$(git status --porcelain -- $DATA)" ]; then
	# shellcheck disable=SC2086
	git add -- $DATA
	# shellcheck disable=SC2086
	git commit -q -m "Sources: refresh brookfield-first, morning-rephil and allennolan from YouTube's Data API ($(date -u +%F))" -- $DATA
	echo "refresh-brookfield: the list changed; committed $(git rev-parse --short HEAD)"
else
	echo "refresh-brookfield: the list is unchanged"
fi

if [ -n "$(git rev-list "origin/$BRANCH..HEAD")" ]; then
	git push -q origin "$BRANCH" || die "git push origin $BRANCH failed; the commit is kept here and the next run pushes it"
	echo "refresh-brookfield: pushed to origin/$BRANCH"
fi

# The service's own settings (DATABASE_URL) in a subshell, so they reach the load
# alone; the key was cleared above and was never exported.
[ -r "$ENVFILE" ] || die "cannot read $ENVFILE; the sources were not loaded and the stamp is as it was"
if ! (
	unset YOUTUBE_API_KEY
	set -a
	# shellcheck disable=SC1090
	. "$ENVFILE"
	set +a
	exec "$SERMONS" load --all "$ROOT/sources"
); then
	die "sermons load --all $ROOT/sources failed (see above); the commit is kept, the stamp is as it was and the next run loads again"
fi
echo "refresh-brookfield: loaded $ROOT/sources into the database"
# What the hourly run keeps to know the database holds rows.jsonl as it is now.
git hash-object -- sources/brookfield-first/rows.jsonl > "$STATE/subsplash-loaded.new" && mv "$STATE/subsplash-loaded.new" "$STATE/subsplash-loaded"

[ -z "$FAILED" ] || die "${FAILED# } failed (see above); what the other tool fetched is committed and loaded, and the stamp is as it was"

mkdir -p "$(dirname "$STAMP")"
date +%s > "$STAMP.new"
mv "$STAMP.new" "$STAMP"
echo "refresh-brookfield: done; stamp $STAMP"
