status: todo
kind: refactor
commit: refactor(app): move self-contained areas out of internal/app
scope: internal/app/**, internal/*/ui/**
not_before: 2026-01-01
depends_on:

# Move self-contained areas out of internal/app

Goal: `internal/app` is 9.4k source lines and about 406 methods on one `App`; tests are 17k.
One `App` stays (Go cannot declare a method in another package). Areas that need little of the
`App` become their own `ui` or controller package behind a `Host` interface, tested with a fake
host, as `calltrace`, `sentlog` and `destpicker` already are. Candidates from the file prefixes
(`protocol_wsui.go` 476, `statesync.go` 391, `window_filebrowser.go` 362): the websocket view, the
file browser window, state sync. The task designer picks by how few `App` fields each needs.

OPEN QUESTION (ask the user): how many areas in this task? Recommended: one area per commit, the
websocket view first, and re-measure before the next.

## Model
Per area: the `Host` interface (at most about 5 methods), who owns the state, what persists.

## Acceptance
- Test first for each `Host` seam: a fake host test states the behavior the area needs.
- The existing `app` tests for the area move with it and stay green.
- The repo check command passes; `.harness/architecture/app.md` lists the new package.
