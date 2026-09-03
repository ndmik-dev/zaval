# Day Board (zaval)

Personal task board: Go + htmx 4 + SQLite. Single user, single binary.

## Conventions
- Commit messages: short, lowercase first letter, no trailers (no Co-Authored-By).
- Code comments: English, only where the code is not self-explanatory.
- Templates: `html/template`, one page = layout + page file. Every mutation
  re-renders the page's `app` block and htmx morphs it (`hx-target:inherited="#app"`).
- The drawer carries `page` + filters in hidden fields (`shell.Ctx`), so an edit
  lands back on the page it was made from (board, journal, releases).
- htmx is pinned to 4.0.0 and vendored in `internal/web/static/`; static URLs
  carry a content hash, the service worker is network-first.
- Dialogs: `[data-confirm]` opens the in-app modal, never `confirm()`.

## Product decisions (deliberate, do not reintroduce)
- No dates, estimates, timers, WIP limits or task steps.
- Task states: Зараз / Чекаю (with a note) / Беклог / Готово. Adding only via ⌘K.
- Releases: one checklist per work project, lines optionally point at a task;
  «Зарелізено» archives the lines into history and empties the list.
- Journal shows closed tasks by day; the daily-standup text lives on the board (button/`d`).
- Projects: name, kind, colour, ⌘K tag. No hidden projects, no integrations yet.

## Run
    go run .            # http://localhost:8080, db at ./dayboard.db
    go test ./...
    docker compose up --build   # distroless image, /data volume, see README
