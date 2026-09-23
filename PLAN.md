# Day Board — where it stands

A decision log, not a roadmap. Read before touching the interface again.

## Verdict, September 2026

Seven rounds of design in one week produced a competent board with no opinion:
correct everywhere, delightful nowhere. The cause is not the last mockup — it is
that every round adjusted the previous one and nothing was ever thrown away.

As a task manager this loses to a text file or Things. The only parts with
value a file does not have are:

1. links that classify themselves into chips (Jira, GitHub, GitLab, Slack, Figma);
2. a standup that assembles itself from what was closed and what is in «Зараз»;
3. a release checklist whose lines point at tasks.

The board, the states, the journal are scaffolding for those three.

## Rule until further notice

**No more redesign.** The current interface (`03e8eab` onward — row of words on
top, one column, sheet only when something is open) is frozen. Use it for
thirty days or do not use it; either answer is worth more than an eighth round.

## The direction, if it survives

«Блокнот»: a page per day instead of a board, a line instead of a task.
`#atl` picks the project, a pasted link becomes a chip, `? відповіді` marks
waiting, `/release atl` unfolds the checklist in place. Yesterday's page sits
under today's already struck through — that *is* the standup. Backlog is the
second and last page. Chips carry live status from Jira and GitHub, which is
the one thing a text file cannot do.

Mockups: https://claude.ai/code/artifact/c4d792fb-1dce-44c2-9881-b19e87674bb5

**How to test it without building it:** keep a plain text file in Obsidian for
one week using exactly that syntax. If the file survives the week, the notebook
is worth a week of building on this backend (Go, SQLite, deploy, PWA all stay;
the model changes to lines and days). If it does not, neither is the app.

**Decision, 9 September 2026:** built anyway, on the `notebook` branch, without
the text-file week. Stage 1 is there: today + the struck-lines feed, the
backlog page, the line editor with the syntax above, `/release` as a line,
⌘⏎ / ⌥↑↓ / drag. The stored model did not change, so `main` and `notebook`
read the same database. What is not there yet, in order:

1. **Live chips** — Jira and GitHub status on the chip, and the «змерджено —
   закреслити?» nudge. Needs two tokens and one poller; this is the item that
   makes it not a text file.
2. Phone pass: the line editor on iOS keyboard (Enter, no ⌥ — a move handle).
3. Delete the board's leftovers once the notebook has lived a month: the
   `/releases` history page becomes struck `/release` lines in the feed.

## What would make the current app indispensable instead

Only one thing: live statuses on the chips — `ATL-412 · In review`,
`PR #128 · merged` — and a standup that reads merged PRs. Two days of work on
the existing code. Do this *before* any notebook, because a redesign without
new capability is round eight.

## Small UX debts, noted so they are not forgotten

Not to be done during the freeze. If the app is still in use after thirty days,
these are the first hour of work:

- Rows look different between the two groupings: a colour square in «за
  станом», none in «за проєктом». One row shape in both.
- A day with nothing closed drops the «Готово сьогодні» section without a
  trace. One quiet line — «сьогодні ще нічого» — keeps the page's shape.
- Empty sections say what to do (⌘K) — keep that; the empty *board* on a fresh
  database does not yet.

## Deliberately absent

Dates, estimates, timers, WIP limits, task steps, hidden projects, a second way
to create a task besides ⌘K. The list in CLAUDE.md is the contract.
