status: todo
kind: refactor
commit: refactor(e2e): whole-app guard tests move to e2e
scope: internal/app/**, e2e/**, .harness/architecture/app.md
not_before: 2026-01-01
depends_on: 017-apptest-seam

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
