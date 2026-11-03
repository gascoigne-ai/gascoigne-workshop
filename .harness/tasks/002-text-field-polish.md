status: todo

# Text field polish

## Goal
The URL field edits text but lacks the movement and scrolling users expect.

## Scope
- Ctrl+Left/Right word movement, Ctrl+Backspace/Delete word deletion, with shift to select.
- Double-click selects a word, triple-click selects all.
- Horizontal scroll keeps the cursor visible in a field narrower than its text; selection auto-scrolls while dragging past the edge.
- Undo/redo (Ctrl+Z, Ctrl+Shift+Z / Ctrl+Y) with coalesced typing.
- Context for later tasks: keep all of this inside `widget.TextField` so the table cells and the text area reuse it.

## Acceptance
- Table-driven `widget` tests for word boundaries (punctuation, unicode), click counting, scroll offset, undo coalescing.
- `make check` passes.
