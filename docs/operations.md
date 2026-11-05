# Operations

Running the sermon index on the Laptop: where to look, what to do after a change, what each failure looks like. The units and scripts are described in [README.md, "On the Laptop"](../README.md#on-the-laptop) and [deploy/laptop/README.md](../deploy/laptop/README.md).

<!-- toc -->

- [Looking](#looking)
- [After a change](#after-a-change)
- [What a wrong setting looks like](#what-a-wrong-setting-looks-like)

<!-- /toc -->

## Looking

```sh
systemctl --user status sermon-index sermon-index-pg
journalctl --user -u sermon-index -f          # the service: one line a request on stderr, and its start and stop
journalctl --user -u sermon-index-pg -f       # Postgres, as the container prints it
docker logs sermon-index-pg                   # the same, from Docker
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

  The service reads the database on every request, so there is nothing to restart; its answers carry an ETag and may be cached for five minutes (docs/api.md).
- **New code**: `git pull`, `scripts/install-laptop.sh`: it rebuilds, restarts the service if the binary changed, applies migrations and loads every source. A migration that adds a column the loader fills needs the load (the install runs it).
- **A unit file changed**: the install copies it and runs `daemon-reload`; a changed Postgres unit is left running (the install says so): apply it with `systemctl --user restart sermon-index-pg`.
- **Backup and move**: README, "On the Laptop".

## What a wrong setting looks like

| Assumed | A wrong one shows as |
| --- | --- |
| `10.88.0.2` (the WireGuard address, `SERMONS_ADDR`) | `sermon-index` fails to start with `listen tcp 10.88.0.2:8790: bind: cannot assign requested address` when the interface is down or the address differs; the install's smoke test says "service down" |
| The user is in the `docker` group | the install stops at once with "cannot talk to Docker"; the Postgres unit fails with `permission denied while trying to connect to the Docker daemon socket` |
| `/usr/bin/docker` (the unit's absolute path) | `status` shows `Failed to locate executable /usr/bin/docker` (status 203/EXEC); `which docker` says where it is, edit the unit |
| `~/.local/bin` on the user manager's `PATH` | not needed by the units (absolute `%h` paths); the install's `go` must be on the shell's `PATH` or it stops with "go is not on PATH" |
| `~/.config/sermon-index/env` | a missing file: the unit fails with `Failed to load environment files: No such file or directory`; a file without `DATABASE_URL`: `no database URL: set DATABASE_URL` in the journal |
| The password in `pg.env` matches `env` | `sermon-index` starts, `/v1/health` answers 503, the journal says `password authentication failed`; the password in `pg.env` only counts when the data directory is first filled |
| Port 5433 free on the loopback | `sermon-index-pg` fails with `port is already allocated`; the install's wait for Postgres times out |
| User lingering (`loginctl enable-linger`) | the units run only while the user is logged in; after a reboot nothing answers until login |
