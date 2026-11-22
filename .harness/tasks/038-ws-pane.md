status: todo
kind: refactor
commit: refactor(ui): the websocket pane is its own component
scope: internal/**, cmd/**, e2e/**, Makefile, .harness/**
not_before: 2026-01-01
depends_on: 037-response-component

# ui/editor/ws

Goal: `protocol_wsui` (467 lines) moves to `ui/editor/ws`; connection state and events the pane reads come from
`sends`. Follow the component recipe of `.harness/reorg.md`. If the file is still over about 300 lines, split
it by sub-component (message list, composer), not by file prefix.

## Model
Model: unchanged. Message-composer field state moves into the component's per-tab map.

## Acceptance
- Dump tests against a fake `Deps`; `protocol_wsui_test.go` and messages view tests move with it.
- Common acceptance of `.harness/reorg.md`.
