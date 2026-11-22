status: todo
kind: refactor
commit: refactor(ui): the body pane is its own component
scope: internal/**, cmd/**, e2e/**, Makefile, .harness/**
not_before: 2026-01-01
depends_on: 033-urlbar-component

# ui/editor/body

Goal: `request_bodypane`, `request_bodystyle`, `request_filebody` and `request_schemaui` move to
`ui/editor/body` (body text field and its focus claim in `Keys`, JSON schema form, file body). Follow the
component recipe of `.harness/reorg.md`.

## Model
Model: unchanged. The body field and schema-form state move from `reqTab` into the component's per-tab map.

## Acceptance
- Dump tests against a fake `Deps`; body and schema tests move with the component.
- Common acceptance of `.harness/reorg.md`.
