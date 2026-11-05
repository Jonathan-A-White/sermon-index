#!/bin/sh
# The way back from scripts/install-laptop.sh: stop and disable both units,
# remove them and the binary. The database and the password files are left,
# so installing again finds the same data; this script names them and says
# how to remove them. Safe to run again.
#
# SERMON_INDEX_HOME stands in for $HOME (the tests set it).
set -eu

H=${SERMON_INDEX_HOME:-$HOME}
UNITS=$H/.config/systemd/user
BIN=$H/.local/bin/sermons
CONF=$H/.config/sermon-index
DATA=$H/.local/state/sermon-index/pg

# The service first: it Requires Postgres, and stopping Postgres first would
# stop it anyway.
for u in sermon-index.service sermon-index-pg.service; do
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

cat <<LEFT

Left in place (the install finds them again):
  $DATA   the database (owned by the container's postgres user)
  $CONF    pg.env and env: the database password

To remove them too, for good, back up first (README, "On the Laptop"), then:
  docker run --rm -v $DATA:/d alpine find /d -mindepth 1 -delete
  rmdir $DATA
  rm -r $CONF
LEFT
