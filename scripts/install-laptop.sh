#!/bin/sh
# Install the sermon index on the Laptop as two systemd user units: Postgres
# 17 in Docker and the service. No sudo; safe to run again (a second run
# changes nothing and says so).
#
#   scripts/install-laptop.sh
#
# What it does, in order:
#   1. builds `sermons` (go build, the git describe stamped in as the version)
#      to ~/.local/bin/sermons;
#   2. writes ~/.config/sermon-index/pg.env and env (mode 0600) if absent,
#      with a password made once by `openssl rand -hex 24`, never printed;
#   3. copies deploy/laptop/*.service to ~/.config/systemd/user and runs
#      `systemctl --user daemon-reload` when one changed;
#   4. enables and starts the Postgres unit, waits for it (PG_WAIT seconds,
#      default 60), runs `sermons migrate` and `sermons load --all sources/`;
#   5. enables and starts the service (the service refuses a database with a
#      migration not applied, so it starts after migrate), then scripts/smoke.sh.
#
# SERMON_INDEX_HOME stands in for $HOME (the tests set it to a temporary
# directory); the units themselves say %h, the user's real home.
set -eu

ROOT=$(cd "$(dirname "$0")/.." && pwd)
H=${SERMON_INDEX_HOME:-$HOME}
BIN=$H/.local/bin/sermons
CONF=$H/.config/sermon-index
UNITS=$H/.config/systemd/user
DATA=$H/.local/state/sermon-index/pg
PG_WAIT=${PG_WAIT:-60}
PG_UNIT=sermon-index-pg.service
SVC_UNIT=sermon-index.service

CHANGED=0
changed() { CHANGED=1; echo "$*"; }
die() { echo "install-laptop: $*" >&2; exit 1; }
need() { command -v "$1" >/dev/null 2>&1 || die "$1 is not on PATH: $2"; }

need go "the factory's Go is pinned under ~/.local; put it on PATH for this run"
need docker "Docker is not installed"
need openssl "openssl makes the database password"
need systemctl "this installs systemd user units"

# Docker may be refused to this session (and to the user manager that runs the
# Postgres unit) because it started before the user joined the docker group;
# `sg docker -c` takes the group from /etc/group, so then every docker call of
# the install goes through it. The unit does the same.
DOCKER_SG=0
if ! docker info >/dev/null 2>&1; then
	if command -v sg >/dev/null 2>&1 && sg docker -c 'docker info' >/dev/null 2>&1; then
		DOCKER_SG=1
		echo "note: docker is refused to this session (it started before the docker group was added); using sg docker -c for the install's docker calls"
	else
		die "cannot talk to Docker, with or without sg docker -c: is this user in the docker group (getent group docker), and is the Docker daemon running?"
	fi
fi
# docker_run ARGS...: docker with the fixed arguments of this script (none has
# a space or a quote), through sg when the probe above needed it.
docker_run() {
	if [ "$DOCKER_SG" = 1 ]; then sg docker -c "docker $*"; else docker "$@"; fi
}

# 1. The binary.
mkdir -p "$H/.local/bin"
VERSION=$(git -C "$ROOT" describe --tags --always --dirty 2>/dev/null || echo dev)
(cd "$ROOT" && go build -ldflags "-X main.version=$VERSION" -o "$BIN.new" ./cmd/sermons) || { rm -f "$BIN.new"; die "go build failed"; }
BIN_CHANGED=0
if [ -f "$BIN" ] && cmp -s "$BIN" "$BIN.new"; then
	rm -f "$BIN.new"
else
	mv "$BIN.new" "$BIN"
	BIN_CHANGED=1
	changed "built $BIN ($VERSION)"
fi

# 2. The env files: written only when absent, so the password is made once.
# pg.env (what the Postgres container reads) and env (what the service reads)
# share it; if one file survives, its password is the one kept.
mkdir -p "$CONF"
chmod 700 "$CONF"
PW=
if [ -f "$CONF/pg.env" ]; then
	PW=$(sed -n 's/^POSTGRES_PASSWORD=//p' "$CONF/pg.env" | head -n 1)
elif [ -f "$CONF/env" ]; then
	PW=$(sed -n 's#^DATABASE_URL=postgres://sermons:\([^@]*\)@.*#\1#p' "$CONF/env" | head -n 1)
fi
if [ ! -f "$CONF/pg.env" ] || [ ! -f "$CONF/env" ]; then
	[ -n "$PW" ] || PW=$(openssl rand -hex 24)
fi
if [ ! -f "$CONF/pg.env" ]; then
	( umask 077; printf 'POSTGRES_USER=sermons\nPOSTGRES_PASSWORD=%s\nPOSTGRES_DB=sermons\n' "$PW" > "$CONF/pg.env" )
	chmod 600 "$CONF/pg.env"
	changed "wrote $CONF/pg.env (0600)"
fi
if [ ! -f "$CONF/env" ]; then
	( umask 077; printf 'DATABASE_URL=postgres://sermons:%s@127.0.0.1:5433/sermons\nSERMONS_ADDR=10.88.0.2:8790\nSERMONS_ORIGINS=https://lampas.allmymind.org,http://localhost:5173\n' "$PW" > "$CONF/env" )
	chmod 600 "$CONF/env"
	changed "wrote $CONF/env (0600)"
fi
PW=
ADDR=$(sed -n 's/^SERMONS_ADDR=//p' "$CONF/env" | head -n 1)
[ -n "$ADDR" ] || die "$CONF/env has no SERMONS_ADDR"

# 3. The units.
mkdir -p "$UNITS" "$DATA"
UNITS_CHANGED=0
PG_UNIT_CHANGED=0
SVC_UNIT_CHANGED=0
for u in "$PG_UNIT" "$SVC_UNIT"; do
	if [ -f "$UNITS/$u" ] && cmp -s "$ROOT/deploy/laptop/$u" "$UNITS/$u"; then
		continue
	fi
	cp "$ROOT/deploy/laptop/$u" "$UNITS/$u"
	UNITS_CHANGED=1
	changed "installed $UNITS/$u"
	[ "$u" = "$PG_UNIT" ] && PG_UNIT_CHANGED=1
	[ "$u" = "$SVC_UNIT" ] && SVC_UNIT_CHANGED=1
done
[ "$UNITS_CHANGED" = 0 ] || systemctl --user daemon-reload

# 4. Postgres.
if systemctl --user is-enabled "$PG_UNIT" >/dev/null 2>&1 && systemctl --user is-active "$PG_UNIT" >/dev/null 2>&1; then
	if [ "$PG_UNIT_CHANGED" = 1 ]; then
		echo "note: $PG_UNIT changed and is left running; apply it with: systemctl --user restart $PG_UNIT"
	fi
else
	systemctl --user enable --now "$PG_UNIT"
	changed "enabled and started $PG_UNIT"
fi

echo "waiting for Postgres (up to ${PG_WAIT}s)..."
waited=0
# -h 127.0.0.1: while the image initialises a new data directory it listens
# on its socket only, and would answer ready before it is.
until docker_run exec sermon-index-pg pg_isready -h 127.0.0.1 -U sermons -d sermons >/dev/null 2>&1; do
	if [ "$waited" -ge "$PG_WAIT" ]; then
		die "Postgres did not answer in ${PG_WAIT}s: sg docker -c 'docker logs sermon-index-pg' (or docker logs sermon-index-pg); journalctl --user -u sermon-index-pg"
	fi
	sleep 1
	waited=$((waited + 1))
done

# The sermons command reads DATABASE_URL from the same file the service does.
set -a
. "$CONF/env"
set +a
out=$("$BIN" migrate) || { echo "$out"; die "sermons migrate failed"; }
echo "$out"
case "$out" in *applied*) CHANGED=1 ;; esac
"$BIN" load --all "$ROOT/sources" || die "sermons load failed"

# 5. The service.
if systemctl --user is-enabled "$SVC_UNIT" >/dev/null 2>&1 && systemctl --user is-active "$SVC_UNIT" >/dev/null 2>&1; then
	if [ "$BIN_CHANGED" = 1 ] || [ "$SVC_UNIT_CHANGED" = 1 ]; then
		systemctl --user restart "$SVC_UNIT"
		changed "restarted $SVC_UNIT"
	fi
else
	systemctl --user enable --now "$SVC_UNIT"
	changed "enabled and started $SVC_UNIT"
fi

SERMONS_URL="http://$ADDR" sh "$ROOT/scripts/smoke.sh" || die "the smoke test failed: journalctl --user -u sermon-index"

if [ "$CHANGED" = 0 ]; then
	echo "nothing changed: the binary, env files and units were already as they should be, and migrate and load found nothing to do."
else
	echo "installed: the sermon index answers at http://$ADDR"
fi
cat <<WAYBACK

The way back:
  sh $ROOT/scripts/uninstall-laptop.sh
stops and disables both units and removes them and $BIN. It leaves, and
names again when it runs:
  $DATA   (the database)
  $CONF    (the password: pg.env and env)
WAYBACK
