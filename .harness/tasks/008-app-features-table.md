status: todo
kind: refactor
commit: refactor(app): features register through a table
scope: internal/**, cmd/**, e2e/**, .harness/architecture.md, .harness/architecture/**, .harness/overview.md, uitoolkit/CLAUDE.md, Makefile
not_before: 2026-01-01
depends_on: 012-websocket-view-host

# App: a Features table

Goal: `internal/app` is about 9.4k source lines and 406 methods on one `App` (Go cannot declare a
method in another package, so one `App` stays). Add a `Features` table (name, constructor, the
`Host` it needs) so a new feature is a table row, not a branch in `App`, and register the six
features through it (history, settings, collections, environments, send, workspace).

Findings and the user's answers: the implementer found the six features share no constructor or
`Host` (they are `App` fields, and each `Host` is the `App` itself), so a table built now would only
forward calls. The user chose to run this task after 011 (connection state model) and 012
(websocket view), when each feature has its own state value and a `Load`/`Restore`/`Save` method
set the table can loop over. Re-measure then; if the features still share no lifecycle, report
blocked again. No logic moves out of `App` here.

## Model
Model: unchanged. The table is static data: name, constructor, the `Host` it needs.

## Acceptance
- Test first: a test of the table (every feature registered once, constructors build with a fake host).
- Existing `app` tests stay green and none changes.
- The repo check command passes; `architecture/app.md` lists the table.
