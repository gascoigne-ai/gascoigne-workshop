status: todo
kind: refactor
commit: refactor(ui): the tab strip is its own component
scope: internal/**, cmd/**, e2e/**, Makefile, .harness/**
not_before: 2026-01-01
depends_on: 031-kvtable-component

# ui/tabstrip

Goal: the view half of `requesttabs.go` moves to `ui/tabstrip` (scroll state, close and drag of tabs); the
model half already sits in `tabs.Set` (030). Follow the component recipe of `.harness/reorg.md`.

## Model
Model: unchanged. The strip's scroll offset moves from `frameState` into the component.

## Acceptance
- Dump test against a fake `Deps`; `requesttabs_view_test.go` and the tab flow tests move with it.
- Common acceptance of `.harness/reorg.md`.
