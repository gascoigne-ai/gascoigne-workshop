status: todo
kind: refactor
commit: refactor(workspace): remembered state and writes leave the app
scope: internal/**, cmd/**, e2e/**, Makefile, .harness/**
not_before: 2026-01-01
depends_on: 043-overlays-components

# statesync, changekeys, workspace.go and asyncwrite leave app

Goal: `statesync` (391 lines), `changekeys`, `workspace.go` (282) and `asyncwrite` move to
`feature/workspace/state`, `autosave` and `serial`, and to `feature/collections/collections` for the open-set
persistence. `app` holds no persistence code; the stores are built from `Options` in `app.go`.

## Model
Model: unchanged in meaning. `persistState` (state path, debounces, write queue, change keys, welcomed,
trusted, unreadable) leaves `App` and is owned by the workspace controllers; file formats and versions in
`persistence.md` are untouched.

## Acceptance
- Test first for every new unit that replaces inline logic (change detection, debounce); statesync and
  autosave tests move with it.
- Common acceptance of `.harness/reorg.md`.
