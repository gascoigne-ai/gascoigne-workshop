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

User's answers: first, one area per task; then, after the implementer measured the websocket view
(about 15 `App` methods, state in `reqTab` shared with `driver.go`, `game.go`, `request_send.go`,
`statesync.go`), this task builds the table only. No logic moves out of `App` here. The state type
is task 011 and the view move is task 012.

## Model
Model: unchanged. The table is static data: name, constructor, the `Host` it needs.

## Acceptance
- Test first: a test of the table (every feature registered once, constructors build with a fake host).
- Existing `app` tests stay green and none changes.
- The repo check command passes; `architecture/app.md` lists the table.
