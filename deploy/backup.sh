#!/bin/sh
# Consistent SQLite backup (safe while the app runs); keeps the last 30.
# Cron on the host: 0 3 * * * docker compose -f /opt/zaval/deploy/compose.yml exec -T dayboard backup
set -eu
db="${DB_PATH:-/data/dayboard.db}"
dir="$(dirname "$db")/backups"
mkdir -p "$dir"
sqlite3 "$db" ".backup '$dir/dayboard-$(date +%F).db'"
ls -1t "$dir"/dayboard-*.db | tail -n +31 | xargs -r rm --
