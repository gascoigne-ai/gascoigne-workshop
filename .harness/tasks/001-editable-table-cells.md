status: todo

# Editable table cells

## Goal
Params, Headers and Vars tables are display-only except checkbox and delete. Make every cell editable.

## Scope
- Click a cell to edit it in place with a `widget.TextField` (selection, copy/paste already exist); Enter or click elsewhere commits, Escape cancels.
- Tab / Shift+Tab move to the next / previous cell; Tab past the last cell of the placeholder row adds a row.
- Typing into the placeholder row creates a real row (enabled by default).
- Params edits rewrite the URL (`model.WithParams`); URL edits keep updating Params (`MergeParams`). Header and Var edits update the model.
- VML: a `Table` option for editable cells; cells stay plain text when the table is read-only (response headers).

## Acceptance
- vml test: click a cell, type, Enter fires `OnEdit(row, col, text)`; Tab order; placeholder row adds a row.
- ui test: editing `q` in Params changes the URL; editing the URL changes the Params row.
- `make check` passes; `make shot` shows an open cell editor.
