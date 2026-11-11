# deploy/laptop: the sermon index as systemd user units

Two user units for the service, four more for the Brookfield First refresh and two for the hourly Subsplash linking (both below), installed by [`scripts/install-laptop.sh`](../../scripts/install-laptop.sh) (no sudo; the way back is [`scripts/uninstall-laptop.sh`](../../scripts/uninstall-laptop.sh)). In all of them, `%h` is the user's home directory, which systemd fills in. The pattern is the Laptop's `postern-backend.service`: an `EnvironmentFile`, `%h` paths, `Restart=on-failure`.

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
- [The hourly Subsplash linking](#the-hourly-subsplash-linking)

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
| `sermon-index-brookfield-refresh.service` | `scripts/refresh-brookfield.sh`: key from `mw secrets get youtube_api_key` into the environment of the one `go run ./tools/brookfield-youtube`; if the data changed, a commit and a push; then `sermons load --all ~/sermon-index/sources` into the live database (also when the list is unchanged, since the database may lag a landing); only then the stamp `~/.local/state/sermon-index/brookfield-refreshed`. `OnFailure=mw-unit-failed@%n.service` mails the Mayor when it fails; `TimeoutStartSec=15min`; `EnvironmentFile=-%h/.config/mw/dispatch.env` supplies only `PATH` (for `mw`, `go`, `git`); `Environment=SERMONS=%h/.local/bin/sermons` names the load command, and the script reads `~/.config/sermon-index/env` (`DATABASE_URL`) itself, in a subshell the YouTube key never enters. |
| `sermon-index-brookfield-stale.timer` | Daily at 13:00 UTC, `Persistent=true`. |
| `sermon-index-brookfield-stale.service` | `scripts/check-brookfield-stale.sh`: exits non-zero when the stamp is missing, not a number, or older than 27 days (`STALE_DAYS`), and the same `OnFailure=` mails the Mayor. 27 days is two refreshes missed and still three inside YouTube's 30. |

The key is never in a unit, a file, the journal or a command line: the script holds it in one shell variable and hands it to the one command through a prefix assignment (`YOUTUBE_API_KEY=$key go run ...`). The script refuses to run if the checkout is not on `main`, if the two data paths have uncommitted changes, or if `main` has diverged from `origin/main`; when the tool fails it puts the two paths back as they were and leaves the stamp alone; a push that is refused leaves the commit, and the next run pushes it. A load that fails (or a missing `~/.config/sermon-index/env`) fails the run and leaves the stamp alone, so the stale check still counts the days; the commit is kept and the next run loads again. A run that finds the list unchanged commits nothing, loads, and still writes the stamp (the 30 days start again). The refresh takes the lock `~/.local/state/sermon-index/brookfield.lock` first and waits for it (see the hourly Subsplash linking, below), and after its `load --all` notes what it loaded for the hourly run. To run it by hand: `sh ~/sermon-index/scripts/refresh-brookfield.sh`; to see the next runs: `systemctl --user list-timers 'sermon-index-*'`.

## The hourly Subsplash linking

A sermon's Subsplash page is the row's `alt_url`, and the Governor sends the evidence as it comes: the notes mail (Postern keeps it as an alert) and the short links he shares (they land in Postern's inbox). Two more units, `sermon-index-subsplash-hourly.service` and `.timer` (`OnCalendar=hourly`, `Persistent=true`), run [`scripts/subsplash-hourly.sh`](../../scripts/subsplash-hourly.sh) of the checkout at `~/sermon-index` so that a link he sends shows in Lampas within the hour. The install enables the timer like the others; the unit has `ConditionPathExists=` on the script, so before the checkout has it a start is skipped, not failed.

| Step | What |
| --- | --- |
| lock | `~/.local/state/sermon-index/brookfield.lock`, the one lock the refresh also takes. An hourly run that finds it held says so and exits 0 (the next hour links); the refresh waits for it (`REFRESH_LOCK_WAIT`, 30 minutes) and fails if it never comes free. So the two never rewrite `rows.jsonl` at once, and the refresh keeps every row's `alt_url` by row id (`tools/brookfield-youtube`, `KeepAltURLs`). |
| checkout | On `main`, tree clean, not behind `origin/main`: otherwise the run says why, exits 1 and has changed nothing (it has run nothing yet). |
| notes | `go run ./tools/subsplash-notes fetch` (the key `~/.config/sermon-index/postern-alerts.key`, `POSTERN_URL` from the unit): `data/subsplash-notes.jsonl`, git-ignored. |
| link list | Every `https://brookfieldfirst.subspla.sh/<code>` in `~/.local/state/mw/postern/inbox/*.message.json`, in a temporary file outside the checkout, and each `*.pdf` in the inbox that `pdftohtml` reads. |
| link | `go run ./tools/subsplash-links` over those and the records. It sends one HEAD for each short link it has never seen (the cache, `data/subsplash-links-cache.jsonl`, keeps the rest) and appends one line to `~/.local/state/sermon-index/subsplash-progress.jsonl`: `time`, `records`, `links`, `rows_linked`, `rows_total`, `new`, `oldest`, `newest` (the dates of the linked rows). |
| commit | If `rows.jsonl` changed: one commit, `Subsplash links: N new (hourly)`, then a push; a commit the remote lacks (a refused push) is pushed by the next run. The tree is never left dirty. |
| load | `sermons load ~/sermon-index/sources/brookfield-first` (`DATABASE_URL` from `~/.config/sermon-index/env`), only when `rows.jsonl` is not what the last load put there: the git hash of the loaded file is kept in `~/.local/state/sermon-index/subsplash-loaded`, written after a good load (and by the refresh after its `load --all`). A failed load is tried again next hour. |

A step that fails does not hide the later ones that can still run (Postern down does not stop the inbox's links); the run exits 1 at the end and `OnFailure=mw-unit-failed@%n.service` tells the Mayor. To read a run: `journalctl --user -u sermon-index-subsplash-hourly`; the next runs: `systemctl --user list-timers | grep subsplash-hourly`; by hand: `sh ~/sermon-index/scripts/subsplash-hourly.sh`. The progress file is private to the Laptop and is not in the repository.
