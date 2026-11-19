status: todo
kind: refactor
commit: refactor(app): rules that are not UI move to controllers and models
scope: internal/*,.harness/*
depends_on: 122-split-flow-tests

# Move non-UI rules out of the app package

Origin: the repo audit of 2026-11-19 (three read-only reviews against the harness rules). The user approved all phases and, with that, the deletions and the gate change named here. Behavior stays the same unless the task says otherwise; tests stay green and none is loosened. Update `architecture.md` in the same commit where it describes what changes.

Each moves with its own test, test first (pin current behavior, then move):
`pane.badge` assertion count by parsing text (to `assert`); `jsonStatus` and `formatBody` (use `textfmt.CompactJSON`); `splitVars` (beside `model.Expand`); `curlName`; `missingVarsText` (to `transfer`); `composed` hex parsing; `timeoutFor`; `moveCollection` before-index maths and `rememberCollections` / `rememberEnvironments` (`openset.go`); `treeKeys` navigation; `rememberedPane`. A controller never imports the toolkit.
Acceptance: `make check` green; `internal/layers_test.go` green.
