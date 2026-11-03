status: todo

# Focus model and global shortcuts

## Goal
Focus is per widget (`Focused` flags) and there are no keyboard shortcuts.

## Scope
- One focus owner in `widget.Ctx`: clicking outside any focusable clears it; Tab / Shift+Tab cycle focusables in tree order; the focused field shows a focus outline.
- Shortcuts table in `internal/ui`: Ctrl+Enter send, Ctrl+L focus URL, Ctrl+T new request, Ctrl+W close tab, Ctrl+Tab / Ctrl+Shift+Tab next / previous tab, Ctrl+, open settings, Escape closes the menu or cancels an edit.
- Shortcuts are a table (key, modifiers, action name, handler), not a switch, and are listed in the settings menu.

## Acceptance
- Tests: focus cycles in order and wraps; a shortcut fires once per press; shortcuts are ignored while a text field consumes the key (Ctrl+L inside a field still works, plain letters do not trigger anything).
- `make check` passes.
