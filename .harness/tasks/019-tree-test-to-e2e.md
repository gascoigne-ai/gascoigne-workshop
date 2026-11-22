status: todo
kind: refactor
commit: refactor(e2e): whole-app guard tests move to e2e
scope: internal/app/**, e2e/**, .harness/architecture/app.md
not_before: 2026-01-01
depends_on: 045-window-composite

# First whole-app tests move to e2e

Goal: move `TestTheDrawnTreeMatchesAFreshBuildAcrossASession` (it guards `treeStale`) and the `*_flow_test.go`
files that use only exported helpers into `e2e/<area>/` (areas: sidebar, request, response, protocols, settings,
persistence) through `apptest`. Access-only rewrites are allowed (user decision). The report counts the test
files that still read private `App` fields and names them; later extraction tasks move those with their
component.

## Model
Model: unchanged.

## Acceptance
- Every moved test keeps its name and assertions; goldens move with `git mv`.
- Common acceptance of `.harness/reorg.md`.

## Decision
Deferred to run after 045 (orchestrator, same reasoning as 018; the user chose "defer" for 018). The Opus
design found: `TestTheDrawnTreeMatchesAFreshBuildAcrossASession` compares `frameTree` (what `Tree()` returns)
with a fresh private build, so it cannot move through `apptest` without two new exports (`FreshTree`,
`Resize(w, h, scale)`); and the 7 flow files read (123 tests: tree_flow, window_layout_flow, request_auth_flow,
request_send_flow, request_tabs_flow, request_table_flow, frametree) write private state (`pointTabAt`, `paneTab`,
`expanded`, `settings.*`, `ctx.Clipboard`) with no exported path. They move with the component tasks that own
that state. After 045 the guard moves with `ui/window`; if it still needs `FreshTree` and `Resize`, add them
then, test first, with the area name `window` in `e2e/window` (area not in the six listed above; ask).
Notes: moved e2e tests carry `//go:build !release`, no `t.Parallel` (`apptest.Start` uses `t.Setenv`), goldens
referenced as `../testdata/<name>.dump`; the golden `internal/app/testdata/window-empty-wide.dump` is read by
`apptest_test.go`, so it must not be `git mv`ed away from it.
