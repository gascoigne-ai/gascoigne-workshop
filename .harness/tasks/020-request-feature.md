status: todo
kind: refactor
commit: refactor(request): the request model becomes a base feature
scope: internal/**, cmd/**, e2e/**, Makefile, .harness/**
not_before: 2026-01-01
depends_on:

# kernel/request and kernel/openset become feature/request

Goal: `kernel/request` (19 files, 2367 lines, 24 importer packages) is the one substantial kernel package. It
becomes `internal/feature/request`, the single base feature every other may import; the 14-line
`kernel/openset` (`OpenCollection`) joins it. Package names stay `request`, so the rewrite is path-only. Follow
the move recipe in `.harness/reorg.md`; 11 fuzz target lines change.

## Model
Model: unchanged. `OpenCollection` keeps its JSON keys (they are part of the settings file, `persistence.md`).

## Acceptance
- The layer table's `base` zone covers `feature/request`; the old kernel rows for `request` and `openset` go.
- Common acceptance of `.harness/reorg.md`.
