# Day Board (zaval)

Personal task board: Go + htmx 4 + SQLite. Single user, single binary.

Read [PLAN.md](PLAN.md) first. The `notebook` branch replaces the board with
the notebook described there; `main` still carries the frozen board.

## Conventions
- Commit messages: short, lowercase first letter, no trailers (no Co-Authored-By).
- Code comments: English, only where the code is not self-explanatory.
- Templates: `html/template`, one page = layout + page file. Every mutation
  re-renders the page's `app` block and htmx morphs it (`hx-target:inherited="#app"`).
- Navigation is a row of words on top (`topbar`): Сьогодні, Беклог, then
  Релізи and Проєкти on the right. The nav item for the current page is a
  `<span>`, not a link — clicking where you already are must do nothing.
- Mutations inherit `#page-ctx` (page name, `shell.Ctx`) so an edit lands back
  where it was made; `respondNotebook` dispatches on it. Navigation links carry
  a full URL and cancel the include with `hx-include="unset"` — a selector that
  matches nothing.
- htmx is pinned to 4.0.0 and vendored in `internal/web/static/`; static URLs
  carry a content hash, the service worker is network-first.
- Dialogs: `[data-confirm]` opens the in-app modal, never `confirm()`.
- Lists breathe: no hairlines between rows, sections separated by space.

## The notebook (branch `notebook`)
- A page per day, a line per task. The stored model did not change: an open
  line is a task in `now`, a struck line is `done` on the day of `done_at`,
  the backlog page is `backlog`. Waiting is the `waiting` note.
- A line is edited as one string and parsed on save (`parseLine`):
  `#tag` picks the project, a URL anywhere becomes a link chip, a standalone
  `?` starts the waiting note, `→ беклог` / `→ сьогодні` at the end moves the
  line, `/release #tag` makes the line a release checklist. `rawText` is the
  inverse; the two must round-trip (tested).
- Without a tag a new line takes the project of the line above it, then the
  `lastp` cookie, then the first project.
- The editor is a textarea that grows with the line, in the same face as the
  rendered text so nothing jumps. Enter saves and opens the next line (or the
  empty one), ↑/↓ walk lines and save on the way, Esc cancels, an unchanged
  blur closes, an emptied line is deleted. ⌘⏎ strikes, ⌥↑/⌥↓ move
  (`/lines/{id}/move`); the project square is the drag handle (`/lines/order`,
  positions only — `SetPositions` never touches waiting or timestamps).
- Keyboard focus is separate from editing: ↑/↓ (j/k) walk lines from the
  moment the page opens (`.ln.focused`, an ink bar on the left), Enter opens
  the focused line (edits it, or follows a project row), Esc steps out of
  editing and then clears the focus; x / ⌘⏎ strike, b sends to the other page.
  Focus survives a morph by key, like the editing state.
- Opening a line puts the caret after the title, before `? waiting`, `#tag`
  and links (`titleEnd`). Behind the transparent textarea a backdrop (`.hl`)
  paints the syntax: links underlined, `#tag` in the project's colour
  (`data-tags` on `#app`), waiting red, `→ command` muted — `paint()`.
- The empty line's placeholder names the tag a new line will take
  (`NewTag`: the line above, then the `lastp` cookie, then the first project).
- No tick buttons: actions are words that appear on hover at the line's end
  («закреслити», «→ беклог», «повернути»); on a phone they are always visible.
- The page is morphed after every save. `app.js` remembers which line was
  being edited (and the caret) at swap time and reopens it, so a blur-save
  never swallows the click that started editing another line. Sortable
  instances are tracked in a WeakSet, not an attribute — a morph erases
  attributes and would otherwise stack handlers.
- `/release #tag`: the task's title is literally `/release`; the checklist
  under it is the project's `release_templates`; «Зарелізено» archives them
  and strikes the line as «Реліз X · n / m». Checklist actions taken from a
  notebook line re-render the notebook (`page != releases`).
- Releases and projects speak the same language: a heading with a mono count,
  lines, words on hover. A checklist item is a notebook line (`check_item`:
  the box toggles, the text edits in place, «→ задача» unfolds the picker);
  release history is dated sections like the feed, its task chips link to
  `/#task-ID`. A project is a line whose editor unfolds under it (`.pedit`).
  There is no task side-editor any more — a task is edited only as its line.

## Product decisions (deliberate, do not reintroduce)
- No dates, estimates, timers, WIP limits or task steps.
- No board, no state picker, no palette: the line's position is its priority,
  the page is its state. Yesterday's struck lines under today are the standup.
- Projects: name, kind, colour, tag. No hidden projects.
- Live chip statuses (Jira, GitHub) are the next step and the reason this is
  not a text file; nothing polls yet.

## Run
    go run .            # http://localhost:8080, db at ./dayboard.db
    go test ./...
    docker compose up --build   # distroless image, /data volume, see README
