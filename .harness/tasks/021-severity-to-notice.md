status: todo
kind: refactor
commit: refactor(request): message severity joins the base feature
scope: internal/**, cmd/**, e2e/**, Makefile, .harness/**
not_before: 2026-01-01
depends_on: 020-request-feature

# kernel/severity moves to workspace/notice

Goal: `kernel/severity` (27 lines; importers: app, sentlog, calltrace, prefs, formpage, settingsui) becomes
`internal/feature/workspace/notice`, a new package and the future owner of the status line and the toasts.

OPEN QUESTION (ask the user before moving): `settings/prefs` would then import `workspace/notice`, which breaks
"features never import each other". Recommend the fallback: put `severity` in `feature/request` (no new
cross-feature edge). Record the answer under `## Decision`.

## Model
Model: unchanged. `Severity` and its values are unchanged.

## Acceptance
- Common acceptance of `.harness/reorg.md`.

## Decision
`severity` goes into `feature/request` (user answer, the recommended option), not `workspace/notice`: no new
cross-feature import edge. Package `severity` stays its own small package? No: it becomes a file of package
`request` (`severity.go`), so importers write `request.Severity` (check for a name clash with existing
identifiers first; if one exists, stop and ask). `workspace/notice` (task 029) imports it from `request`.
Title above is the original; the move target is `internal/feature/request`.
