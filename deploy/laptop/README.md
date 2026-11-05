# deploy/laptop: the sermon index as two systemd user units

Two user units, installed by [`scripts/install-laptop.sh`](../../scripts/install-laptop.sh) (no sudo; the way back is [`scripts/uninstall-laptop.sh`](../../scripts/uninstall-laptop.sh)). In both, `%h` is the user's home directory, which systemd fills in. The pattern is the Laptop's `postern-backend.service`: an `EnvironmentFile`, `%h` paths, `Restart=on-failure`.

| What | Where |
| --- | --- |
| Postgres 17 | Docker container `sermon-index-pg`, published on `127.0.0.1:5433` only (never the network) |
| The service | `~/.local/bin/sermons serve` on `10.88.0.2:8790` (the WireGuard address; the VPS's nginx reaches it there) |
| The data | `~/.local/state/sermon-index/pg` (the Postgres data directory) |
| The secrets | `~/.config/sermon-index/pg.env` and `env`, mode 0600 |

## sermon-index-pg.service

```ini
[Unit]
Description=Sermon index: Postgres 17 in Docker (the Laptop; 127.0.0.1:5433)
After=network-online.target docker.service

[Service]
ExecStartPre=-/usr/bin/docker rm -f sermon-index-pg
ExecStart=/usr/bin/docker run --rm --name sermon-index-pg -v %h/.local/state/sermon-index/pg:/var/lib/postgresql/data -p 127.0.0.1:5433:5432 --env-file %h/.config/sermon-index/pg.env postgres:17
ExecStop=/usr/bin/docker stop sermon-index-pg
Restart=on-failure
RestartSec=5

[Install]
WantedBy=default.target
```

- `Description`: what `systemctl --user status` shows.
- `After`: start after the network and the system's Docker daemon are up. (A user unit cannot `Requires` a system unit, so this only orders; if Docker is not up yet the start fails and `Restart` tries again.)
- `ExecStartPre`: the leading `-` lets it fail. It removes a container of this name that a hard kill left behind (a container run with `--rm` is removed on a clean stop only), which would otherwise make `docker run` fail with "name already in use".
- `ExecStart`: `docker run` in the foreground, so systemd owns the process. `--rm` removes the container when it stops; the data survives in the mounted directory. `--name sermon-index-pg` is the name `docker exec` and `docker stop` use. `-v ...:/var/lib/postgresql/data` keeps the database on the host. `-p 127.0.0.1:5433:5432` publishes Postgres on the loopback only, port 5433 (5432 is left for any Postgres of the user's own). `--env-file` reads `POSTGRES_USER`, `POSTGRES_PASSWORD` and `POSTGRES_DB`; the image applies them **only when it first fills an empty data directory**, so changing the password in the file later does not change the database's. `postgres:17` is the major version, pinned: a major upgrade is a dump and a restore, never a changed tag (the image's data layout changes at 18).
- `ExecStop`: `docker stop`, which sends Postgres SIGTERM (a clean shutdown) and waits up to ten seconds.
- `Restart`: restart when the process exits non-zero or is killed; a `systemctl --user stop` is not restarted.
- `RestartSec`: wait five seconds between restarts, so a fault does not spin.
- `WantedBy`: `default.target`, started when the user's manager starts (at boot with lingering on, at login otherwise: `loginctl show-user $USER -p Linger`).

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
