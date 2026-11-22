status: todo
kind: refactor
commit: refactor(ui): the response pane is its own composite
scope: internal/**, cmd/**, e2e/**, Makefile, .harness/**
not_before: 2026-01-01
depends_on: 036-assertions-component

# ui/response composite

Goal: `responsepane` and `response_respfile` move to `ui/response`, which composes the `respview` and `sentlog`
hosts (moved in 028); `responseStatus` moves into it from the statusbar's neighbour code. The find field claims
its keys in `Keys`. Follow the component recipe of `.harness/reorg.md`.

## Model
Model: unchanged. Find and view-mode state stay in `send/viewer`; widget state moves into the component.

## Acceptance
- Dump tests against a fake `Deps`; `response_*_test.go` move with the component.
- Common acceptance of `.harness/reorg.md`.
