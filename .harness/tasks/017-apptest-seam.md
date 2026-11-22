status: todo
kind: refactor
commit: refactor(app): apptest drives the whole app from outside
scope: internal/app/**, e2e/**, .harness/architecture/app.md
not_before: 2026-01-01
depends_on: 016-app-size-ratchet

# An exported test seam so whole-app tests can leave package app

Goal: about 90% of the app tests start a whole `App` through `harness_test.go` (package `app`, unexported). To let
them live in `e2e/`, add `internal/app/apptest`: `Start(t, ...Option)`, `Step`, `Click`, `Type`, `Until`,
`Dump`, queries, and the fakes (`withAdapters`, sessions, JSON servers). It sits on a minimal exported surface of
`app` (`Step`, `SetSize`, `WaitWrites`). The dump format stays byte-identical.

## Model
Model: unchanged. A test-support package; no production type besides the three exported methods.

## Acceptance
- Test first: `apptest.Start` runs a frame and returns a dump equal to the golden of an existing scene.
- The helpers of `harness_test.go` that need no private field are served by `apptest`; the rest stay and shrink
  in later tasks. `app` does not import `apptest`.
- Common acceptance of `.harness/reorg.md`.
