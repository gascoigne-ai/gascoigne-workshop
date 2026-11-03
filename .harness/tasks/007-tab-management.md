status: todo

# Request tab management

## Goal
Tabs can be opened and switched but not closed, reordered or marked as changed.

## Scope
- Close button on each tab, middle-click closes, Ctrl+W closes; closing the last tab opens a blank request.
- Dirty marker when a request differs from its saved state (needs task 014 for saved state; until then, from its state at open).
- Drag to reorder; overflow scrolls horizontally with the active tab kept in view.
- `+` creates a request in the selected folder.

## Acceptance
- Tests for close (active index fix-up), reorder, dirty detection after an edit and after undoing it.
- `make check` passes.
