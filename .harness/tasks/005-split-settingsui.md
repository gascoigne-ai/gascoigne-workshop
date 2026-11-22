status: todo
kind: refactor
commit: refactor(settings): split settingsui into form, dialog and pages
scope: internal/settings/ui/**
not_before: 2026-01-01
depends_on: 002-settings-workspace-cycle

# Split settingsui

Goal: `settings/ui/settingsui` is 1626 lines (`form.go` 537) next to the page packages
`shortcutpage` (520), `colorpage` and `systempage`. The form engine and the dialog shell sit in one
package, so a page change touches both. Separate the generic form (rows, fields, validation display)
from the dialog shell, behind the `Host` interface the pages already use. The task designer
chooses the exact cut; each result stays under about 800 lines and imports no other `ui` package.

## Model
Model: unchanged (no new state; the form keeps its `Model` and typed messages).

## Acceptance
- A refactor: existing tests stay green and none changes.
- No `ui` package imports another `ui` package.
- The repo check command passes.
