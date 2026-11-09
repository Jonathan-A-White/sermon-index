#!/bin/sh
# Regenerate Brookfield First's YouTube list and commit it when it changed.
#
#   scripts/refresh-brookfield.sh
#
# YouTube's Developer Policies (III.E.4) let this rig keep public API data for
# 30 calendar days, then it must be deleted or refreshed. The Laptop's timer
# sermon-index-brookfield-refresh.timer runs this on the 1st and the 26th of
# every month (never more than 25 days apart); by hand it is the same command.
#
# What it does, in order:
#   1. checks the checkout is on the branch (REFRESH_BRANCH, default main), that
#      sources/brookfield-first and data/morning-rephil.jsonl have no edit of
#      anyone's, and brings the branch up to date (fast-forward only);
#   2. reads the key from the vault into one variable, and runs
#        YOUTUBE_API_KEY=<key> go run ./tools/brookfield-youtube
#      so the key is in the environment of that one command and nowhere else:
#      it is never written to a file, printed, logged or put on a command line;
#   3. if the tool failed, puts the two paths back as they were and exits
#      non-zero (the timer's OnFailure= then tells the Mayor); the stamp is not
#      touched;
#   4. if the output changed, commits the two paths; pushes any commit the
#      remote does not have yet (so a push that was refused is retried by the
#      next run);
#   5. only then writes the stamp, $SERMON_INDEX_HOME/.local/state/sermon-index/
#      brookfield-refreshed (the epoch second), which scripts/check-brookfield-stale.sh
#      reads. A run that found the list unchanged is a good refresh: the 30
#      days start again.
#
# REFRESH_ROOT stands in for the checkout (the tests use it); SERMON_INDEX_HOME
# for $HOME; MW and GO for the commands.
set -eu

ROOT=${REFRESH_ROOT:-$(cd "$(dirname "$0")/.." && pwd)}
H=${SERMON_INDEX_HOME:-$HOME}
STAMP=$H/.local/state/sermon-index/brookfield-refreshed
BRANCH=${REFRESH_BRANCH:-main}
MW=${MW:-mw}
GO=${GO:-go}
DATA="sources/brookfield-first data/morning-rephil.jsonl"

die() { echo "refresh-brookfield: $*" >&2; exit 1; }

cd "$ROOT"
[ "$(git rev-parse --abbrev-ref HEAD)" = "$BRANCH" ] || die "$ROOT is not on $BRANCH; it will not commit there"
# shellcheck disable=SC2086 # DATA is two paths with no space in either
[ -z "$(git status --porcelain -- $DATA)" ] || die "$DATA has uncommitted changes; leave them to whoever made them"
git fetch -q origin "$BRANCH" || die "cannot fetch origin $BRANCH"
git merge -q --ff-only "origin/$BRANCH" || die "$BRANCH has diverged from origin/$BRANCH; resolve it by hand"

key=$("$MW" secrets get youtube_api_key) || die "mw secrets get youtube_api_key failed; nothing was fetched"
[ -n "$key" ] || die "the vault holds an empty youtube_api_key; nothing was fetched"

# A prefix assignment puts the key in the environment of this one command.
if ! YOUTUBE_API_KEY=$key "$GO" run ./tools/brookfield-youtube; then
	key=
	# shellcheck disable=SC2086
	git checkout -q -- $DATA
	die "brookfield-youtube failed (see above); the list and the stamp are as they were"
fi
key=

# shellcheck disable=SC2086
if [ -n "$(git status --porcelain -- $DATA)" ]; then
	# shellcheck disable=SC2086
	git add -- $DATA
	# shellcheck disable=SC2086
	git commit -q -m "Sources: refresh brookfield-first and morning-rephil from YouTube's Data API ($(date -u +%F))" -- $DATA
	echo "refresh-brookfield: the list changed; committed $(git rev-parse --short HEAD)"
else
	echo "refresh-brookfield: the list is unchanged"
fi

if [ -n "$(git rev-list "origin/$BRANCH..HEAD")" ]; then
	git push -q origin "$BRANCH" || die "git push origin $BRANCH failed; the commit is kept here and the next run pushes it"
	echo "refresh-brookfield: pushed to origin/$BRANCH"
fi

mkdir -p "$(dirname "$STAMP")"
date +%s > "$STAMP.new"
mv "$STAMP.new" "$STAMP"
echo "refresh-brookfield: done; stamp $STAMP"
