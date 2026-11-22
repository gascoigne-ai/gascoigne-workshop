status: todo
kind: refactor
commit: refactor(ui): the MCP pane is its own component
scope: internal/**, cmd/**, e2e/**, Makefile, .harness/**
not_before: 2026-01-01
depends_on: 040-graphql-pane

# ui/editor/mcp

Goal: `protocol_mcpexplorer` and `protocol_mcpform` move to `ui/editor/mcp` (explorer and host of
`mcp/mcpform`). Afterwards `ui/editor` composes the panes by one table row each (the protocol table of
`window_panes.go` moves there; no switch on protocol kind, per `rules.md`). Follow the component recipe of
`.harness/reorg.md`.

## Model
Model: unchanged.

## Acceptance
- Dump tests against a fake `Deps`; explorer, form and sampling tests move with it.
- The `ui/editor` composite has a dump test with all panes registered.
- Common acceptance of `.harness/reorg.md`.
