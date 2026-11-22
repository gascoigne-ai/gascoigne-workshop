status: todo
kind: refactor
commit: refactor(ui): the GraphQL pane is its own component
scope: internal/**, cmd/**, e2e/**, Makefile, .harness/**
not_before: 2026-01-01
depends_on: 039-grpc-pane

# ui/editor/graphql

Goal: `protocol_graphql` moves to `ui/editor/graphql` (query, variables, schema fetch). Follow the component
recipe of `.harness/reorg.md`.

## Model
Model: unchanged. The variables field state moves from `reqTab` into the component's per-tab map.

## Acceptance
- Dump tests against a fake `Deps`; GraphQL schema and view tests move with it.
- Common acceptance of `.harness/reorg.md`.
