status: todo
kind: refactor
commit: refactor(ui): the assertions pane is its own component
scope: internal/**, cmd/**, e2e/**, Makefile, .harness/**
not_before: 2026-01-01
depends_on: 035-auth-component

# ui/editor/assertions

Goal: `request_assertpane` (and `requestsettings.go` if only the editor uses it) move to `ui/editor/assertions`.
Follow the component recipe of `.harness/reorg.md`.

## Model
Model: unchanged. The tests field state moves from `reqTab` into the component's per-tab map.

## Acceptance
- Dump tests against a fake `Deps`; assertion pane tests move with the component.
- Common acceptance of `.harness/reorg.md`.
