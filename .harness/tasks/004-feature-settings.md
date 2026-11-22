status: todo
kind: refactor
commit: refactor(settings): settings is one feature package tree
scope: internal/**, cmd/**, e2e/**, .harness/architecture.md, .harness/architecture/**, .harness/overview.md, uitoolkit/CLAUDE.md
not_before: 2026-01-01
depends_on: 003-feature-history

# Feature: settings

Goal: move `settings` (schema, `prefs`, `theme`, the system adapters, `settingsui` and the three
pages) to `internal/feature/settings`. `settings/ui/settingsui` is 1626 lines: split the generic
form (`form.go` 537) from the dialog shell, behind the `Host` the pages use. OS adapters
(`displayscale`, `fonts`, `keyrepeat`, `systeminfo`) go to `internal/platform`. After task 001
this feature imports no other feature. `FEATURE.md` replaces `architecture/settings.md`.

## Model
Model: unchanged (settings file schema and version stay; fixtures stay).

## Acceptance
- A refactor: tests stay green and none changes.
- No `ui` package imports another `ui` package; each result stays under about 800 lines.
- The repo check command passes.
