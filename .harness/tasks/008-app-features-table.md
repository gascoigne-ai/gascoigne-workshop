status: todo
kind: refactor
commit: refactor(app): features register through a table
scope: internal/app/**, internal/feature/**
not_before: 2026-01-01
depends_on: 003-feature-history, 004-feature-settings, 005-feature-collections, 006-feature-send, 007-feature-workspace-and-platform

# App: a Features table

Goal: `internal/app` is 9.4k source lines and about 406 methods on one `App` (Go cannot declare a
method in another package, so one `App` stays). Add a `Features` table (name, constructor, the
`Host` it needs) so a new feature is a table row, not a branch in `App`; move the feature logic the
`App` still holds (`protocol_wsui.go` 476, `statesync.go` 391, `window_filebrowser.go` 362) into its
feature behind a `Host` of at most about 5 methods, tested with a fake host.

OPEN QUESTION (ask the user): one commit per feature moved? Recommended yes, websocket view first.

## Model
Per feature: the `Host` interface, who owns the state, what persists.

## Acceptance
- Test first for each `Host` seam: a fake host test states the behavior.
- Existing `app` tests for a moved area move with it and stay green.
- The repo check command passes; `architecture/app.md` lists the table.
