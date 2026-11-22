status: todo
kind: refactor
commit: refactor(ui): window composes the screen, app only wires
scope: internal/**, cmd/**, e2e/**, Makefile, .harness/**
not_before: 2026-01-01
depends_on: 044-workspace-state

# ui/window, ui/editor and ui/header take view, step and keys

Goal: `window.go`, `window_panes`, `window_title`, `window_grab` and `redraw` move to `ui/window`; `ui/editor`
and `ui/header` (with `envchip` from `env.go`) compose their children. `app` ends with at most 6 files
(`options`, `ports`, `app`, `game`, `frame`, plus `drive/` and `apptest/`), builds the stores and
`window.New(Stores{...})`, and holds no handler. Delete `appstate.go`, `routeKeys` and `textFocused`. Tighten
the `app` row: no `app` to feature adapters.

OPEN QUESTION (ask the user before deleting): confirm the deletion of `harness_test.go` and `appstate.go`
(rule: ask before deleting). Record the answer under `## Decision`.

## Model
Model: unchanged. `frameState` fields go to the window composite or the components that own them; `treeStale`
becomes the OR of the children's `Tick` results.

## Acceptance
- `TestTheDrawnTreeMatchesAFreshBuildAcrossASession` (now in e2e) stays green.
- `internal/app` non-test file count is at most 6 and `e2e/appsize_test.go` says so.
- Common acceptance of `.harness/reorg.md`.
