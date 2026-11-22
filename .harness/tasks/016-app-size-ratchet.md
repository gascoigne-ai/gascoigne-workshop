status: todo
kind: ci
commit: ci(e2e): the app package may not grow
scope: e2e/**, .harness/architecture/app.md
not_before: 2026-01-01
depends_on: 015-layers-new-zones

# Ratchet on the size of internal/app

Goal: `internal/app` has 63 non-test files. A test caps that number so the old package cannot grow while it is
dissolved; each extraction task lowers the cap (see `.harness/reorg.md`).

## Model
Model: unchanged.

## Acceptance
- `e2e/appsize_test.go` counts the non-test `.go` files of `internal/app` and fails above the cap (63). Test
  first: it fails with the cap at 62.
- The failure message says how to lower the cap and that it only goes down.
- The repo check command passes.
