# Day Board (zaval)

Personal task board: Go + htmx 4 + SQLite. Single user, single binary.

Read [PLAN.md](PLAN.md) first: the interface is frozen, and it says why.

## Conventions
- Commit messages: short, lowercase first letter, no trailers (no Co-Authored-By).
- Code comments: English, only where the code is not self-explanatory.
- Templates: `html/template`, one page = layout + page file. Every mutation
  re-renders the page's `app` block and htmx morphs it (`hx-target:inherited="#app"`).
- Navigation is a row of words on top (`topbar`), not a rail. The board is one
  centred column until something is open beside it; then it splits into list +
  sheet (`.split.detail-open`). Journal, releases and projects have no side pane
  at all — what you open expands in place (`.expand`).
- Mutations inherit `#page-ctx` (page name + its parameters, `shell.Ctx`) so an
  edit lands back where it was made; the detail pane adds `#detail-ctx` with the
  selected task. Navigation links carry a full URL and cancel the include with
  `hx-include="unset"` — a selector that matches nothing — to avoid duplicate params.
- htmx is pinned to 4.0.0 and vendored in `internal/web/static/`; static URLs
  carry a content hash, the service worker is network-first.
- Dialogs: `[data-confirm]` opens the in-app modal, never `confirm()`.
- The detail pane is a sheet: white card on the paper ground, with the open
  project's colour on its top edge (`--project`). The same colour tints the
  selected row. Actions — «Готово», the state segment — stay green everywhere,
  so muscle memory survives.
- Lists breathe: no hairlines between rows, sections separated by space.
- Per-viewer chrome (pane width, wide rail) lives in `localStorage` and is applied
  to `:root`, so an htmx morph cannot lose it. The nav item for the current page
  is a `<span>`, not a link — clicking where you already are must do nothing.

## Product decisions (deliberate, do not reintroduce)
- No dates, estimates, timers, WIP limits or task steps.
- Task states: Зараз / Чекаю (with a note) / Беклог / Готово. Adding only via ⌘K.
- A task in a list is one line, not a card: tick, project colour, title, waiting
  note, link count, tag. Link chips live in the detail pane — the list is for
  scanning, and cards fit five tasks where twenty fit.
- The board groups by state or by project (`?g=`, remembered in a cookie). That
  toggle replaces project filtering — there is no separate filter control.
- The daily standup is not a default view: it takes the sheet only when asked
  for («Дейлі» in the header, or `d`). With nothing open the board is just the board.
- Releases: one checklist per work project, lines optionally point at a task;
  «Зарелізено» archives the lines into history and empties the list.
- Journal is a feed: days are headings inside one scroll, no day list to click
  through, one project dropdown as the only filter. Opening a task keeps you there.
- Projects: name, kind, colour, ⌘K tag. No hidden projects, no integrations yet.

## Run
    go run .            # http://localhost:8080, db at ./dayboard.db
    go test ./...
    docker compose up --build   # distroless image, /data volume, see README
