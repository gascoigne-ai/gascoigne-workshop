status: done for what does not need window state (the rest stays in internal/ui)

# Organize packages by functionality

## Goal
Packages are grouped by layer more than by feature, so related code is spread out (`internal/ui` holds panes, menus, settings dialog, themes, state sync, filters together).

## Scope
- Group code by functionality (request editor, response viewer, settings, themes, collections, protocols) with one package per feature and a narrow API between them.
- Keep dependency direction: `uitoolkit/core` -> `uitoolkit/vml` -> `uitoolkit/engine`; features depend on them, never the reverse.
- Move tests with the code; keep hard-coded screen coordinates helpers in one place.

## Acceptance
- No behavior change; `make check` passes after every move.
- `.harness/architecture.md` describes the new layout in the same commit.

## Progress
- Done: feature packages `internal/respview` (response bodies to lines, compact filter), `internal/textfmt`, `internal/appearance` (themes, colors, user themes, choices), `internal/collection` (tree rows, filter, sample), next to the existing `model`, `proto`, `store`, `settings`, `state`, `jsonview`. Toolkit moved to `uitoolkit/` (task 019).
- Left in `internal/ui` on purpose: panes, environments, WebSocket/MCP tab, workspace saving, overlays and shortcuts all read and write the `App` state (tabs, settings, collection). Splitting them further needs an explicit app-state interface; do that when one of them gets a second user.
