#!/bin/sh
# The way back from scripts/install-laptop.sh: stop and disable both units and
# the three Brookfield timers, remove them and the binary. The database, the
# password files and the refresh stamp are left, so installing again finds the
# same data; this script names them and says how to remove them. Safe to run
# again.
#
# SERMON_INDEX_HOME stands in for $HOME (the tests set it).
set -eu

H=${SERMON_INDEX_HOME:-$HOME}
UNITS=$H/.config/systemd/user
BIN=$H/.local/bin/sermons
CONF=$H/.config/sermon-index
DATA=$H/.local/state/sermon-index/pg

# The Brookfield First refresh, its daily check and the hourly Subsplash linking first: the timers, then the
# services they run.
# The sermon index service next: it Requires Postgres, and stopping Postgres
# first would stop it anyway.
for u in sermon-index-brookfield-refresh.timer sermon-index-brookfield-stale.timer sermon-index-subsplash-hourly.timer \
	sermon-index-brookfield-refresh.service sermon-index-brookfield-stale.service sermon-index-subsplash-hourly.service \
	sermon-index.service sermon-index-pg.service; do
	systemctl --user disable --now "$u" >/dev/null 2>&1 || true
	if [ -f "$UNITS/$u" ]; then
		rm -f "$UNITS/$u"
		echo "removed $UNITS/$u"
	fi
done
systemctl --user daemon-reload
if [ -f "$BIN" ]; then
	rm -f "$BIN"
	echo "removed $BIN"
fi

# The docker line printed below goes through sg when this session is refused
# Docker (it started before the user joined the docker group) and sg answers.
DOCKER_RUN="docker run --rm -v $DATA:/d alpine find /d -mindepth 1 -delete"
if ! docker info >/dev/null 2>&1 && command -v sg >/dev/null 2>&1 && sg docker -c 'docker info' >/dev/null 2>&1; then
	DOCKER_RUN="sg docker -c '$DOCKER_RUN'"
fi

cat <<LEFT

Left in place (the install finds them again):
  $DATA   the database (owned by the container's postgres user)
  $CONF    pg.env and env: the database password
  $H/.local/state/sermon-index/brookfield-refreshed   the refresh stamp

To remove them too, for good, back up first (README, "On the Laptop"), then:
  $DOCKER_RUN
  rmdir $DATA
  rm -r $CONF
LEFT
