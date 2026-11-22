status: todo
kind: refactor
commit: refactor(app): features register through a table
scope: internal/**, cmd/**, e2e/**, .harness/architecture.md, .harness/architecture/**, .harness/overview.md, uitoolkit/CLAUDE.md, Makefile
not_before: 2026-01-01
depends_on: 003-feature-history, 004-feature-settings, 005-feature-collections, 006-feature-send, 007-feature-workspace-and-platform

# App: a Features table

Goal: `internal/app` is about 9.4k source lines and 406 methods on one `App` (Go cannot declare a
method in another package, so one `App` stays). Add a `Features` table (name, constructor, the
`Host` it needs) so a new feature is a table row, not a branch in `App`, and register the six
features through it (history, settings, collections, environments, send, workspace).

User's answer (one area per task): this task moves one area only, the websocket view
(`protocol_wsui.go`, 476 lines), into `internal/feature/send` behind a `Host` of at most about 5
methods, tested with a fake host. Further areas (`statesync.go` 391, `window_filebrowser.go` 362)
are separate follow-up tasks, queued after this one, each re-measured first.

## Model
The `Host` interface of the websocket view, who owns its state, what persists (unchanged).

## Acceptance
- Test first for each `Host` seam: a fake host test states the behavior.
- Existing `app` tests for a moved area move with it and stay green.
- The repo check command passes; `architecture/app.md` lists the table.
