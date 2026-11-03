status: todo

# Table row operations and change handling

## Goal
Tables only support editing a cell, toggling and deleting a row. Everything around a row change is missing: adding in the middle, ordering, undo, and keeping the rest of the app in step.

## Scope
- Insert above/below, duplicate and delete a row from the keyboard (Ctrl+Enter inserts below while editing a cell, Ctrl+D duplicates, Delete removes the row when no cell is being edited, Alt+Up/Down moves it) and by dragging a row handle.
- A header checkbox enables or disables all rows; a bulk-edit mode turns the table into a text area of `name: value` lines and back.
- Undo and redo of table changes (Ctrl+Z / Ctrl+Y when no text is being edited), grouped per edit, shared by Params, Headers, Vars and form fields.
- One place applies a table change (`changed` hook): it rewrites the URL for Params, marks the tab dirty, drops a cell edit that points at a row that no longer exists, and keeps the selection sensible.
- Duplicate names are allowed (many APIs repeat query keys) but a duplicate header is flagged; empty names are kept in the table but never sent.

## Acceptance
- Table-driven tests for each operation, including the edit state after a delete or move, undo across a URL rewrite, and bulk edit round trips (values with `:` and leading spaces).
- `make check` passes.
