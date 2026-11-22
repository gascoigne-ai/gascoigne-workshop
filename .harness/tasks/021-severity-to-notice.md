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
cross-feature edge). Record the answer under `## Decision
`severity` goes under the base feature (user answer, the recommended option), not `workspace/notice`: no new
cross-feature import edge. Target: `internal/feature/request/severity` (own small package, so importers keep
writing `severity.Level`, `severity.Info`; merging into package `request` would add the generic names `Level`,
`Info`, `Error`, `All` to it). The `base` zone already covers sub-packages of `feature/request`; if the layer
table does not, fix it test first. Update the doc comment ("lives in the kernel package") and docs. Task 029's
`workspace/notice` imports it from there.
