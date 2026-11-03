status: todo

# Draggable splitters

## Goal
Sidebar width and the request/response split are fixed.

## Scope
- `vml.Split` container with a draggable divider (cursor change, min sizes, double-click resets).
- Request/response can stack vertically; the choice is a setting.
- Every layout element with a size can be changed, not only the two main splits: sidebar width, the request/response ratio, the height of the find bar and toolbar rows only where it makes sense (fixed rows stay fixed), the width of the settings menu; each has a sensible minimum and maximum and a reset.
- A single `vml.Split` (and an equivalent for a fixed-size side) is the mechanism, so a new resizable area is one element, not new code.
- Sizes persist through the remembered UI state (task 020).

## Acceptance
- Tests for clamping, drag math and persistence of the ratios.
- `make check` passes.
