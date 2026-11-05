#!/bin/sh
# The sermon index's smoke test: is the service up, loaded and right?
#
#   scripts/smoke.sh                       # http://10.88.0.2:8790 (the Laptop)
#   SERMONS_URL=http://127.0.0.1:8790 scripts/smoke.sh
#
# Checks that /v1/health answers 200 with at least two sources, and that
# /v1/sermons?ref=rom.5.8 answers 200 with "Love's Commendation" (Spurgeon
# 104, whose preaching text is Romans 5:8) first. Exits 0 when both hold and
# 1 otherwise, saying which failed. SMOKE_WAIT (default 30) is how many
# seconds to wait for /v1/health to answer 200 (the service has just been
# started). Needs curl, nothing else.
set -u

URL=${SERMONS_URL:-http://10.88.0.2:8790}
WAIT=${SMOKE_WAIT:-30}

fail() { echo "smoke: FAIL: $*" >&2; exit 1; }

# get <url> sets STATUS and BODY. curl prints the body, then the status on a
# line of its own; a connection that fails prints 000.
get() {
	resp=$(curl -sS --max-time 10 -w '\n%{http_code}' "$1" 2>/dev/null) || true
	STATUS=$(printf '%s\n' "$resp" | tail -n 1)
	BODY=$(printf '%s\n' "$resp" | sed '$d')
}

waited=0
while :; do
	get "$URL/v1/health"
	[ "$STATUS" = 200 ] && break
	if [ "$waited" -ge "$WAIT" ]; then
		fail "service down: GET $URL/v1/health answered $STATUS (is sermon-index.service running? journalctl --user -u sermon-index)"
	fi
	sleep 1
	waited=$((waited + 1))
done

sources=$(printf '%s\n' "$BODY" | tr -d '\n' | sed -n 's/.*"sources": *\([0-9][0-9]*\).*/\1/p')
[ -n "$sources" ] || fail "/v1/health has no sources count: $BODY"
[ "$sources" -ge 2 ] || fail "fewer than two sources loaded ($sources): run 'sermons load --all sources/'"

get "$URL/v1/sermons?ref=rom.5.8"
[ "$STATUS" = 200 ] || fail "GET /v1/sermons?ref=rom.5.8 answered $STATUS"
first=$(printf '%s\n' "$BODY" | grep -o '"title": *"[^"]*"' | head -n 1 | sed 's/^"title": *"\(.*\)"$/\1/')
[ "$first" = "Love's Commendation" ] || fail "wrong first sermon for rom.5.8: '$first', want 'Love's Commendation'"

echo "smoke: ok: $sources sources; Romans 5:8 starts with $first"
