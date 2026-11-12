# Operations

Running the sermon index on the Laptop: where to look, what to do after a change, what each failure looks like. The units and scripts are described in [README.md, "On the Laptop"](../README.md#on-the-laptop) and [deploy/laptop/README.md](../deploy/laptop/README.md).

<!-- toc -->

- [Looking](#looking)
- [After a change](#after-a-change)
- [What a wrong setting looks like](#what-a-wrong-setting-looks-like)
- [The Brookfield First notes](#the-brookfield-first-notes)

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

The order is kept so nothing is lost: **the PDF on disk, then the records file in place, and only then the alert claimed** (a claimed alert is gone from Postern, notes and all). A failed download, a size that is not `notes_size` or a sha256 that is not `notes_sha256` is reported on stderr, writes no record for that alert and leaves it unclaimed; the other alerts of the run go on. A second run writes the same bytes and does not download a PDF it already holds with the same sha256; a file there with another sha256 is, when the row's record is the one this alert gives (a second mail for the same sermon, the church's PDF made again), replaced by the newer mail's notes with the older kept beside as `<row id>.prev.pdf`; with no such record it is reported and not replaced (look at it, delete it by hand if the new one is right), and so is a mail for another sermon that falls on a row with a record (reported with both titles). Each refusal is one line saying what to do, and the summary line counts them (`n not replaced`). An alert that carries any other field than the ones the reader knows is dropped, reported and left unclaimed. The no-subcommand reader (a saved reply of `GET /api/alerts`) accepts the two fields, downloads nothing and claims nothing.
