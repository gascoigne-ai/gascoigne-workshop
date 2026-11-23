status: todo
kind: refactor
commit: refactor(collections): open-set operations leave the app
scope: internal/**, cmd/**, e2e/**, Makefile, .harness/**
not_before: 2026-01-01
depends_on: 030b-tabs-set

# Open-set operations and import entry leave the app

Goal: third part of task 030 (split by the orchestrator). `tree_collections.go`, `tree_openset.go` and
`tree_import.go` stay in `internal/app` after 030a (they mix store I/O, settings, prompts, toasts and the tab list).
Move the open-set operations into `feature/collections/collections` behind a store type that takes the `Store`
port (an Opus design decides the API and which prompts/toasts stay behind `Deps`); the import table: parsing goes
to the importers, attaching goes to `ui/sidebar/logic`. `tree_ctxmenus.go` (tab, table-row and response menus)
is NOT part of this task: it goes with the tabstrip, kvtable and response tasks. Follow `.harness/reorg.md`.

## Model
The open set keeps its single owner (`collections.Set`); the new store type holds the port and the operations,
no second copy of the set. The designer writes access pattern and structure for any new collection.

## Acceptance
- Test first: store operation tests against a fake `Store`.
- Access-only rewrites of the tree/openset app tests.
- Common acceptance of `.harness/reorg.md`; cap lowered.
