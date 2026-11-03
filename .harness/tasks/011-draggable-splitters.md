status: todo

# Draggable splitters

## Goal
Sidebar width and the request/response split are fixed.

## Scope
- `vml.Split` container with a draggable divider (cursor change, min sizes, double-click resets).
- Request/response can stack vertically; the choice is a setting.
- Sizes persist in settings.

## Acceptance
- Tests for clamping, drag math and persistence of the ratios.
- `make check` passes.
