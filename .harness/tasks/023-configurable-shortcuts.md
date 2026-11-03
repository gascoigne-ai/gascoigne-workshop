status: todo

# Configurable shortcuts

## Goal
The shortcut table in `internal/ui/shortcuts.go` is fixed. Users need to change bindings that clash with their window manager or habits.

## Scope
- Settings gain a Shortcuts section listing every action with its binding; clicking a binding records the next key combination (Escape cancels, Backspace clears), and a Reset button restores the default.
- Bindings are stored in the settings file as action id to a key string such as `Ctrl+Shift+Tab`; unknown actions are ignored and missing ones use the default.
- Conflicts are caught while recording: a combination already used by another action is refused with that action named; combinations the text widgets own (Ctrl+A/C/V/X/Z/Y, plain letters) are refused.
- Every action has a stable id and a handler in the table; the settings menu, the help list and the Ctrl+key behavior all read the same table, so there is one source of truth.
- The widget-level text shortcuts (undo, clipboard, select all) stay fixed in this task, but are listed read-only.

## Acceptance
- Tests: parse and print key strings round-trip, conflicts and reserved combinations are refused, a saved binding replaces the default and fires, a removed binding fires nothing, and defaults have no duplicates.
- `make check` passes.
