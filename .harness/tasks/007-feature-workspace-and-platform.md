status: todo
kind: refactor
commit: refactor(workspace): workspace feature and platform adapters
scope: internal/**, cmd/**, e2e/**, .harness/architecture.md, .harness/architecture/**, .harness/overview.md, uitoolkit/CLAUDE.md, Makefile
not_before: 2026-01-01
depends_on: 004-feature-settings, 005-feature-collections

# Feature: workspace; and platform

Goal: `workspace/{tabs,state,filebrowser,autosave,bus,overlay,serial,ui}` become
`internal/feature/workspace`; the OS adapters (`filedialog`, `dockicon`, `folders`, `systemclock`,
`statefile` stays with the feature) go to `internal/platform`. After task 001 nothing else imports
`workspace` for severity, so the edge into this feature comes only from `app`.

Finding from task 006: `workspace/tabs` imports the `send` root for `MCPServer`. Once workspace is a
feature this is a feature-to-feature import; move `MCPServer` (a type both use) to kernel in this
task, or record why not.

## Model
Model: unchanged (remembered-state file keeps format and version).

## Acceptance
- A refactor: tests stay green and none changes.
- `FEATURE.md`; `architecture/workspace.md` folded into it.
- The repo check command passes.
