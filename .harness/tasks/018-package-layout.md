status: todo

# Organize packages by functionality

## Goal
Packages are grouped by layer more than by feature, so related code is spread out (`internal/ui` holds panes, menus, settings dialog, themes, state sync, filters together).

## Scope
- Group code by functionality (request editor, response viewer, settings, themes, collections, protocols) with one package per feature and a narrow API between them.
- Keep dependency direction: `ui/widget` -> `ui/vml` -> `ui/engine`; features depend on them, never the reverse.
- Move tests with the code; keep hard-coded screen coordinates helpers in one place.

## Acceptance
- No behavior change; `make check` passes after every move.
- `.harness/architecture.md` describes the new layout in the same commit.
