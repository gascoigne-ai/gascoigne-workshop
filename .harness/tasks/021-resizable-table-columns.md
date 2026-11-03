status: todo

# Resizable table columns

## Goal
Table columns have fixed proportions (28/40/32 for name, value and description), so long values are cut off and short ones waste space.

## Scope
- Drag the divider between two header cells to resize the columns on either side; the cursor shows a resize shape over the handle, double click resets that table to the default widths.
- Widths are fractions of the table width, so they survive window resizing; a column never gets narrower than a minimum that still shows its header caption.
- Each table keeps its own widths (Params, Headers, Vars, form fields, response headers); `vml.Table` takes a `Widths *[]float64` owned by the app, like `ScrollState`.
- Widths are part of the remembered UI state (task 020) and can be reset from the table's right-click menu (task 018).
- Keyboard: with a header cell focused, Alt+Left/Right adjusts the column by a step.

## Acceptance
- Pure tests for the width math: drag clamps at the minimum on both sides, the total stays 100%, a window resize keeps the proportions.
- vml test: pressing on a divider and dragging changes the widths the rows use; a click on a cell next to the divider still edits it.
- `make check` passes; `make shot` with a narrowed column.
