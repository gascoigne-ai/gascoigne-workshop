status: todo
kind: refactor
commit: refactor(ui): the auth pane is its own component
scope: internal/**, cmd/**, e2e/**, Makefile, .harness/**
not_before: 2026-01-01
depends_on: 034-body-component

# ui/editor/auth

Goal: `request_auth`, `request_cookies` and `request_trust` move to `ui/editor/auth` (auth fields, cookie jar
view, trust prompt). Follow the component recipe of `.harness/reorg.md`.

## Model
Model: unchanged. Auth field state moves from `reqTab` into the component's per-tab map.

## Acceptance
- Dump tests against a fake `Deps`; auth flow and view tests move with the component.
- Common acceptance of `.harness/reorg.md`.
