# Day Board

Personal task board for a couple of work projects and pet projects.
Go + htmx 4 + SQLite, one binary, one user.

## Run locally

    go run .                       # http://localhost:8080, db at ./dayboard.db

No password locally unless `PASSWORD` is set.

## Deploy

    cp deploy/compose.yml /opt/zaval/deploy/   # or clone the repo there
    echo PASSWORD=... > /opt/zaval/deploy/.env
    docker compose -f /opt/zaval/deploy/compose.yml up -d --build

Port 8080 is bound to localhost; put Caddy or nginx with TLS in front.
The cookie is marked Secure when the proxy sends `X-Forwarded-Proto: https`.

Backups: `docker compose exec -T dayboard backup` writes `/data/backups/dayboard-YYYY-MM-DD.db` (keeps 30).

## Env

| var        | default          |
|------------|------------------|
| `ADDR`     | `:8080`          |
| `DB_PATH`  | `dayboard.db`    |
| `PASSWORD` | empty = no login |
| `TZ`       | system           |
