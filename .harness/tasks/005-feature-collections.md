status: todo
kind: refactor
commit: refactor(collections): collections and environments become features
scope: internal/collection/**, internal/feature/collections/**, internal/feature/environments/**
not_before: 2026-01-01
depends_on: 003-feature-history

# Features: collections and environments

Goal: `collection/*` becomes `feature/collections` (tree, transfer, `destpicker`, `store`,
`clientfile`, `clientconfig`) with `importers/` (`postman`, `openapi`, `miniyaml`), and
`collection/envs` becomes `feature/environments`. The `collection` root imports its sibling
`clientconfig` today; in the feature that is an ordinary package of one feature, so the old
"root imports no sibling" gap disappears. `collection/collections` imports `settings` only for
`OpenCollection`, which task 001 moved to kernel. Importers keep their fuzz targets
(`FUZZ_TARGETS` in the Makefile) and limits in `architecture/limits.md`.

## Model
Model: unchanged (collection files keep format and version).

## Acceptance
- A refactor: tests stay green and none changes; fuzz targets still run.
- `FEATURE.md` for each feature; `architecture/collection.md` folded into them.
- The repo check command passes.
