# deploy/laptop: the sermon index as systemd user units

Two user units for the service, and four more for the Brookfield First refresh (below), installed by [`scripts/install-laptop.sh`](../../scripts/install-laptop.sh) (no sudo; the way back is [`scripts/uninstall-laptop.sh`](../../scripts/uninstall-laptop.sh)). In both, `%h` is the user's home directory, which systemd fills in. The pattern is the Laptop's `postern-backend.service`: an `EnvironmentFile`, `%h` paths, `Restart=on-failure`.

| What | Where |
| --- | --- |
| Postgres 17 | Docker container `sermon-index-pg`, published on `127.0.0.1:5433` only (never the network) |
| The service | `~/.local/bin/sermons serve` on `10.88.0.2:8790` (the WireGuard address; the VPS's nginx reaches it there) |
| The data | `~/.local/state/sermon-index/pg` (the Postgres data directory) |
| The secrets | `~/.config/sermon-index/pg.env` and `env`, mode 0600 |

<!-- toc -->

- [sermon-index-pg.service](#sermon-index-pgservice)
  - [Why `sg docker -c`](#why-sg-docker--c)
- [sermon-index.service](#sermon-indexservice)
- [The env files](#the-env-files)
- [The Brookfield First refresh](#the-brookfield-first-refresh)

<!-- /toc -->

## sermon-index-pg.service

```ini
[Unit]
Description=Sermon index: Postgres 17 in Docker (the Laptop; 127.0.0.1:5433)
After=network-online.target docker.service

[Service]
ExecStartPre=-/usr/bin/sg docker -c 'docker rm -f sermon-index-pg'
ExecStart=/usr/bin/sg docker -c 'exec docker run --rm --name sermon-index-pg -v %h/.local/state/sermon-index/pg:/var/lib/postgresql/data -p 127.0.0.1:5433:5432 --env-file %h/.config/sermon-index/pg.env postgres:17'
ExecStop=/usr/bin/sg docker -c 'docker stop sermon-index-pg'
Restart=on-failure
RestartSec=5

[Install]
WantedBy=default.target
```

- `Description`: what `systemctl --user status` shows.
- `After`: start after the network and the system's Docker daemon are up. (A user unit cannot `Requires` a system unit, so this only orders; if Docker is not up yet the start fails and `Restart` tries again.)
- `ExecStartPre`: `/usr/bin/sg docker -c '...'` runs the docker command with the `docker` group taken from `/etc/group` (see "Why `sg docker -c`" below). The leading `-` lets it fail. It removes a container of this name that a hard kill left behind (a container run with `--rm` is removed on a clean stop only), which would otherwise make `docker run` fail with "name already in use".
- `ExecStart`: `docker run` (under `sg docker -c`; the `exec` makes it replace sg's shell) in the foreground, so systemd owns the process. `--rm` removes the container when it stops; the data survives in the mounted directory. `--name sermon-index-pg` is the name `docker exec` and `docker stop` use. `-v ...:/var/lib/postgresql/data` keeps the database on the host. `-p 127.0.0.1:5433:5432` publishes Postgres on the loopback only, port 5433 (5432 is left for any Postgres of the user's own). `--env-file` reads `POSTGRES_USER`, `POSTGRES_PASSWORD` and `POSTGRES_DB`; the image applies them **only when it first fills an empty data directory**, so changing the password in the file later does not change the database's. `postgres:17` is the major version, pinned: a major upgrade is a dump and a restore, never a changed tag (the image's data layout changes at 18).
- `ExecStop`: `docker stop` (under `sg docker -c`), which sends Postgres SIGTERM (a clean shutdown) and waits up to ten seconds.
- `Restart`: restart when the process exits non-zero or is killed; a `systemctl --user stop` is not restarted.
- `RestartSec`: wait five seconds between restarts, so a fault does not spin.
- `WantedBy`: `default.target`, started when the user's manager starts (at boot with lingering on, at login otherwise: `loginctl show-user $USER -p Linger`).

### Why `sg docker -c`

A systemd user manager gets its groups when it starts, at the user's first login. The Laptop's started (2026-10-14) before `jwhite` was added to the `docker` group, so the manager, every unit under it and every session shell started from it lack the group even though `getent group docker` lists the user: `docker info` answers `permission denied` at `/var/run/docker.sock`, and a plain `ExecStart=/usr/bin/docker run ...` fails the same way. Restarting the manager (or rebooting) would cure it but ends every session, so the unit runs docker through `/usr/bin/sg docker -c '...'`, which starts the command with the `docker` group read from `/etc/group` now. For a manager that has the group, `sg` does the same thing and changes nothing. `%h` is still expanded by systemd inside the single quotes. `scripts/install-laptop.sh` does likewise for its own docker calls when `docker info` is refused and `sg docker -c 'docker info'` answers.

## sermon-index.service

```ini
[Unit]
Description=Sermon index: the read-only HTTP API (the Laptop; 10.88.0.2:8790, behind the VPS)
After=sermon-index-pg.service
Requires=sermon-index-pg.service

[Service]
ExecStart=%h/.local/bin/sermons serve
EnvironmentFile=%h/.config/sermon-index/env
Restart=on-failure
RestartSec=5

[Install]
WantedBy=default.target
```

- `Description`: what `systemctl --user status` shows.
- `After`: start after the Postgres unit has started (the unit is "started" when its `docker run` begins, not when Postgres answers: the service starts without a database and answers 503 until it does, and `Restart` covers the one case it does not, a database with a migration unapplied).
- `Requires`: if the Postgres unit is stopped, this one stops too; starting this one starts Postgres.
- `ExecStart`: the binary the install built; an absolute `%h` path, because the user manager's `PATH` does not include `~/.local/bin`.
- `EnvironmentFile`: `DATABASE_URL` (`postgres://sermons:<password>@127.0.0.1:5433/sermons`), `SERMONS_ADDR` (`10.88.0.2:8790`) and `SERMONS_ORIGINS` (the origins CORS allows). A missing file fails the start.
- `Restart`: restart on a non-zero exit; `sermons serve` exits non-zero when the database has a migration not applied, so a deploy that forgot `sermons migrate` shows as a restart loop, not a wrong answer.
- `RestartSec`: five seconds between restarts.
- `WantedBy`: `default.target`, as above.

## The env files

`pg.env` (read by the container):

```
POSTGRES_USER=sermons
POSTGRES_PASSWORD=<48 hex characters, openssl rand -hex 24>
POSTGRES_DB=sermons
```

`env` (read by the service, and sourced by the install for `sermons migrate` and `sermons load`):

```
DATABASE_URL=postgres://sermons:<the same password>@127.0.0.1:5433/sermons
SERMONS_ADDR=10.88.0.2:8790
SERMONS_ORIGINS=https://lampas.allmymind.org,http://localhost:5173
```

The install writes each file only when it is absent, so the password is made once and never printed. To move the service to another address, edit `SERMONS_ADDR` and `systemctl --user restart sermon-index`.

## The Brookfield First refresh

YouTube's Developer Policies (III.E.4) let the rig keep public API data for 30 calendar days, then delete or refresh it, so `sources/brookfield-first` and `data/morning-rephil.jsonl` are regenerated by a timer. Four more units, installed and enabled by the same `scripts/install-laptop.sh` (the two services are run by their timers and never enabled themselves). All four run the scripts of the checkout at `~/sermon-index`.

| Unit | What it does |
| --- | --- |
| `sermon-index-brookfield-refresh.timer` | `OnCalendar=*-*-01,26 04:00:00 UTC`, `Persistent=true`: the 1st and the 26th of each month, so never more than 25 days between runs (the 1st to the 26th is the longest gap) and a Laptop that was off runs it on waking. |
| `sermon-index-brookfield-refresh.service` | `scripts/refresh-brookfield.sh`: key from `mw secrets get youtube_api_key` into the environment of the one `go run ./tools/brookfield-youtube`; if the data changed, a commit and a push; then the stamp `~/.local/state/sermon-index/brookfield-refreshed`. `OnFailure=mw-unit-failed@%n.service` mails the Mayor when it fails; `TimeoutStartSec=15min`; `EnvironmentFile=-%h/.config/mw/dispatch.env` supplies only `PATH` (for `mw`, `go`, `git`). |
| `sermon-index-brookfield-stale.timer` | Daily at 13:00 UTC, `Persistent=true`. |
| `sermon-index-brookfield-stale.service` | `scripts/check-brookfield-stale.sh`: exits non-zero when the stamp is missing, not a number, or older than 27 days (`STALE_DAYS`), and the same `OnFailure=` mails the Mayor. 27 days is two refreshes missed and still three inside YouTube's 30. |

The key is never in a unit, a file, the journal or a command line: the script holds it in one shell variable and hands it to the one command through a prefix assignment (`YOUTUBE_API_KEY=$key go run ...`). The script refuses to run if the checkout is not on `main`, if the two data paths have uncommitted changes, or if `main` has diverged from `origin/main`; when the tool fails it puts the two paths back as they were and leaves the stamp alone; a push that is refused leaves the commit, and the next run pushes it. A run that finds the list unchanged commits nothing and still writes the stamp (the 30 days start again). To run it by hand: `sh ~/sermon-index/scripts/refresh-brookfield.sh`; to see the next runs: `systemctl --user list-timers 'sermon-index-*'`.
