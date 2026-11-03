status: todo

# Multi-line text editor widget

## Goal
The request body has no editor. Build `widget.TextArea` as the reusable multi-line editor.

## Scope
- Lines, cursor, selection (shift+arrows, drag, double and triple click, Ctrl+A), Up/Down with a remembered column, Home/End, PageUp/PageDown.
- Enter inserts a newline with the current indentation; Tab inserts spaces (Shift+Tab dedents the selection).
- Copy, cut, paste (multi-line), undo/redo shared with `TextField`'s history type.
- Line numbers, scrolling (vertical and horizontal) via task 004, current-line highlight, optional JSON syntax colors by a `Style` callback.
- VML node `TextArea(*widget.TextArea, ...)` bound like `Field`.

## Acceptance
- Table-driven `widget` tests for cursor movement across lines, column memory, selection ranges, indent/dedent, paste with newlines, undo.
- `make check` passes.
