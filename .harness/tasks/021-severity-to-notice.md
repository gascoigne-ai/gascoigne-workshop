status: todo
kind: refactor
commit: refactor(notice): message severity lives with the notices
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
