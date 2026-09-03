# Day Board (zaval)

Personal task board: Go + htmx 4 + SQLite. Single user, single binary.

## Conventions
- Commit messages: short, lowercase first letter, no trailers (no Co-Authored-By).
- Code comments: English, only where the code is not self-explanatory.
- No dates, estimates, timers or WIP limits on tasks — deliberate product decision.
- Templates: `html/template`, one page = layout + page file. htmx responses render partials.
- htmx is pinned to 4.0.0 and vendored in `internal/web/static/`.

## Run
    go run .            # http://localhost:8080, db at ./dayboard.db
    go test ./...
