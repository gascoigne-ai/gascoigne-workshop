status: todo

# Scrolling and scrollbars

## Goal
Only the code view scrolls. The tree, tables and panes silently clip.

## Scope
- `vml.Scroll(...)` container: vertical (and optional horizontal) scrolling with the wheel, a draggable scrollbar thumb that appears on hover or scroll, and scroll-into-view for the focused child.
- Use it for the collections tree, request and response tables, note panes and the pane bodies.
- Code view: horizontal scrolling for long lines (Shift+wheel and a bottom scrollbar), scrollbar thumb for vertical.
- Keyboard: PageUp/PageDown/Home/End in the code view.

## Acceptance
- vml test: content taller than the box scrolls and clamps; thumb size and position math; wheel over a nested scroller scrolls the innermost.
- `make check` passes; `make shot` with a long tree.
