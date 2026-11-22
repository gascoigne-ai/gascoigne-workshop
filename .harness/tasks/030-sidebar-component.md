status: todo
kind: refactor
commit: refactor(ui): the sidebar owns its tree, menus and drag
scope: internal/**, cmd/**, e2e/**, Makefile, .harness/**
not_before: 2026-01-01
depends_on: 029-statusbar-component

# ui/sidebar with logic, and tabs.Set

Goal: `tree_sidebar`, `tree_ctxmenus`, `treemenu`, `treedrag`, `treeops`, `tree_collections`, `tree_import`,
`tree_openset` move to `ui/sidebar` and `ui/sidebar/logic` (selection, expansion, drag and drop decisions,
reveal). Open-set changes go to `feature/collections/collections`. New `feature/workspace/tabs.Set` (tab list,
current tab, `Open`, close) wraps today's `tabs.Update`. `sidebar.Keys` is the first key claimer carved out of
`routeKeys`. Follow the component recipe of `.harness/reorg.md`; the designer reports fields shared with other
areas before the implementer starts.

## Model
New `tabs.Set`: owner of the open tabs and the current one; invariant: current is nil or a member; a slice (order
is the strip order; sizes are tens), no map. `sidebar/logic.Tree`: selected node, expanded set (map keyed by
node id, looked up per row per frame), drag state as one value with a transition table (idle, pressed,
dragging, dropping), not a flag pair. Rejected: `treeState` fields on `App`.

## Acceptance
- Test first: `logic` unit tests (selection, expand, drag transitions, drop allowed); a dump test of the
  sidebar against a fake `Deps`; `tabs.Set` tests.
- `routeKeys` offers the key to `sidebar.Keys` before its old tree code; the old code is deleted.
- Common acceptance of `.harness/reorg.md`.
