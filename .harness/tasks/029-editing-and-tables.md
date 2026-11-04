status: todo

# Editing, tables and tree polish

## Goal
Small gaps found while finishing earlier tasks.

## Scope
- Table columns: resize from the keyboard (Alt+Left/Right on a header cell needs a header focus concept).
- Table rows: a drag handle to reorder with the mouse (keyboard and menu moves exist); bulk edit for form fields.
- Tree: match highlight in the filter; expand a folder while a drag hovers over it.
- Scroll-into-view of a focused child inside a `vml.Scroll`.
- Shortcuts: list the text widgets' own keys (Ctrl+A/C/V/X/Z/Y) in the settings, read-only; plain-key bindings are refused today, allow them with a warning.
- Drag and drop of text and files into fields and the tree.
- Native file dialogs: shell out to zenity or kdialog, or build a file picker widget. Decide first and record in decisions.md; tasks 025 and 026 need it.

## Acceptance
- A test per item where behavior is testable headless; `make check` passes; architecture.md updated.

## Progress
- Done: the file dialog decision (decisions 12): zenity or kdialog when installed, a text prompt otherwise.
- Left: the rest of the scope.
