status: todo

# The layer test enforces what the MVU layout promises

## Goal
`internal/layers_test.go` rejects the violations the analysis found, and has no stale cases.

## Scope
- [done] Controllers may not import `ebiten`, `net/http` or anything under `uitoolkit`; a case proves each fails (the scan records the two external packages as `external/...` pseudo paths).
- [done] A `ui` package may not import another `ui` package (the composition root keeps its access to other domains' ui).
- [open] Delete the stale cases that name `request/controller/assert` (now `request/model/assert`, decision 22) and `collection/ui/tree` (does not exist), or point them at real packages.
- [open] `architecture.md` lists the final package map.

## Acceptance
- The test fails for each deliberate violation and passes on the tree; `make check` green.

## Notes
- Do last, after tasks 036 to 038 have settled the package layout.
