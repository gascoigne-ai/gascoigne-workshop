status: todo
kind: refactor
commit: refactor(kernel): shared leaf packages live in internal/kernel
scope: internal/**, cmd/**, e2e/**, .harness/architecture.md, .harness/architecture/**, .harness/overview.md, uitoolkit/CLAUDE.md, Makefile
not_before: 2026-01-01
depends_on:

# Kernel: the shared leaves

Goal: layout C (feature slices) needs a place for what several features share. Measured with
`go list`: every cross-domain edge except the `request` model is a small shared piece:
`workspace.Level`/`Error`/`Info` (severity, from `workspace/severity.go`; used by `calltrace`,
`sentlog`, `settingsui`, `prefs`), `settings/fileversion` and `settings/userdir` (used by
`store`, `clientfile`, `history`, `statefile`, `settingsfile`), `response/textfmt` and
`response/jsonview` (used by `calltrace`, `respview`, `settingsui`), and `settings.OpenCollection`
(used by `collection/collections`). Move them to `internal/kernel/<name>`; kernel imports no other
`internal` package. Rewrite the layout section of `internal/CLAUDE.md` for kernel, feature, platform
and app, and the module table and diagram in `.harness/architecture.md`.

## Model
Model: unchanged. One owner per type; `OpenCollection` moves with its schema row, so persistence and
the version table are unchanged (fixtures stay).

## Acceptance
- A refactor: tests stay green and none changes (test files move with their code).
- `go list` shows no `internal/{request,collection,response,settings,workspace}` package importing
  another of these domains except through `kernel` and the `request` model.
- The repo check command passes.
