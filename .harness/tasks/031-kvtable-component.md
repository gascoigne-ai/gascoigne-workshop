status: todo
kind: refactor
commit: refactor(ui): the key-value table is its own component
scope: internal/**, cmd/**, e2e/**, Makefile, .harness/**
not_before: 2026-01-01
depends_on: 030c-openset-store

# ui/editor/kvtable with logic

Goal: `request_tablepane`, `request_tablerows`, `request_kvrows`, `request_celledit` and `window_colwidths` move
to `ui/editor/kvtable` and `kvtable/logic` (rows, undo, bulk edit, cell edit, column widths). It claims
Undo/Redo in `Keys` when visible and not editing (this replaces its part of `textFocused`). Follow the component
recipe of `.harness/reorg.md`.

## Model
Model: unchanged in meaning: `tableState` (cell edit, row drag, bulk edit, row selection, column widths) moves
from `App` into the component, per tab in a pruned map; undo history stays a bounded stack per table. Report
any field two panes share.

## Acceptance
- Test first for every new `logic` function that replaces an inline decision; table-driven.
- Dump tests against a fake `Deps`; the table flow tests move with the component.
- Common acceptance of `.harness/reorg.md`.
