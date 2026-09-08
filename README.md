# Day Board

Personal task board for two work projects and a few pet projects. What is in
progress, what waits on someone, a backlog, a journal for the daily standup and
a release checklist per project.

Self-hosted, single user, server-rendered. Go + htmx 4 + SQLite, one binary,
one dependency (`modernc.org/sqlite`, pure Go), no frontend build. The
interface is Ukrainian.

Status and the decision not to redesign again: [PLAN.md](PLAN.md).

## Run

    go run .            # http://localhost:8080, dayboard.db in the working directory
    go test ./...

No password is needed on a loopback address. Anywhere else the server refuses
to start without `PASSWORD` (or `INSECURE=1`).

## Configuration

Read from the environment, falling back to `.env` in the working directory.
A real environment variable always wins over the file.

| Variable     | Default        | Purpose                                              |
|--------------|----------------|------------------------------------------------------|
| `ADDR`       | `:8080`        | listen address                                       |
| `DB_PATH`    | `dayboard.db`  | SQLite file                                          |
| `PASSWORD`   | —              | the one password; required off loopback              |
| `BACKUP_DIR` | unset          | nightly `VACUUM INTO` copies here, last 30 kept      |
| `TZ`         | system         | timezone for day boundaries and the journal          |
| `ENV_FILE`   | `.env`         | path to the env file                                 |
| `INSECURE`   | unset          | `1` allows running without a password off loopback   |

## Deploy

    docker compose up --build

Multi-stage into distroless, `CGO_ENABLED=0`, runs as nonroot, database and
backups on a `/data` volume. `/data` is created in the image owned by uid
65532: a fresh volume inherits that, so the nonroot process can create the
database on first start.

The image has no shell, so `-healthcheck` makes the binary call its own
`/healthz` — that is what the compose healthcheck runs.

On Dokploy the service is a **Compose** application: `dokploy-network` is
external and joined by the service, the domain is added in the Domains tab
(service `zaval`, port `8080`), and `PASSWORD` goes in the Environment tab.

Backups: with `BACKUP_DIR` set (the image sets `/data/backups`) the server
writes `dayboard-YYYY-MM-DD.db` every night at 03:00 and keeps 30. To restore,
stop the container and copy a file over `/data/dayboard.db`.
