status: todo
kind: refactor
commit: refactor(history): history is one feature package tree
scope: internal/**, cmd/**, e2e/**, .harness/architecture.md, .harness/architecture/**, .harness/overview.md, uitoolkit/CLAUDE.md
not_before: 2026-01-01
depends_on: 001-kernel-shared-leaves, 002-kernel-model

# Feature: history

Goal: the smallest feature goes first to prove the layout. `response/adapter/history` (231 lines)
and `response/ui/sentlog` (179) become `internal/feature/history/{adapter,ui}` with a `model.go`
and a `FEATURE.md` (behavior, limits, known gaps, moved out of `architecture/response.md`).
It imports kernel and platform only, never another feature; `sentlog` keeps its `Host` seam.

## Model
Model: unchanged (history file format and version stay; its fixture stays).

## Acceptance
- A refactor: tests stay green and none changes.
- `FEATURE.md` exists and `architecture/response.md` no longer holds the history text.
- The repo check command passes.
