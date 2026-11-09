#!/bin/sh
# Exit non-zero when Brookfield First's YouTube list was last refreshed more
# than 27 days ago, or never (no stamp, or one that is not a number).
#
#   scripts/check-brookfield-stale.sh
#
# The stamp is $SERMON_INDEX_HOME/.local/state/sermon-index/brookfield-refreshed,
# written by scripts/refresh-brookfield.sh only after a good run. YouTube lets
# the data be kept 30 days; the refresh runs at least every 25, so 27 means two
# runs in a row failed. The Laptop's sermon-index-brookfield-stale.timer runs
# this daily, and its OnFailure= tells the Mayor. STALE_DAYS changes the 27.
set -eu

H=${SERMON_INDEX_HOME:-$HOME}
STAMP=$H/.local/state/sermon-index/brookfield-refreshed
STALE_DAYS=${STALE_DAYS:-27}

stale() { echo "check-brookfield-stale: $*" >&2; exit 1; }

[ -f "$STAMP" ] || stale "no refresh stamp at $STAMP: the list has never been refreshed here (run scripts/refresh-brookfield.sh)"
at=$(head -n 1 "$STAMP")
case $at in
'' | *[!0-9]*) stale "$STAMP does not hold an epoch second: $at" ;;
esac
now=$(date +%s)
age=$((now - at))
if [ "$age" -gt $((STALE_DAYS * 86400)) ]; then
	stale "the last good refresh of the Brookfield First list was $((age / 86400)) days ago ($(date -u -d "@$at" +%F)); YouTube allows 30. Run scripts/refresh-brookfield.sh and read the journal: journalctl --user -u sermon-index-brookfield-refresh"
fi
echo "check-brookfield-stale: the last good refresh was $((age / 86400)) days ago; fine"
