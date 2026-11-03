status: todo

# Right-click menus

## Goal
Common actions are only reachable through buttons and shortcuts. Add right-click context menus where users expect them, on top of the menu primitive from task 012.

## Scope
- Request tab: Close, Close others, Close tabs to the right, Duplicate, Copy URL.
- Tree row: New request, New folder, Rename, Duplicate, Delete, Copy URL (the CRUD actions come from task 013; until then only the actions that exist).
- Table row: Insert row above, Insert row below, Duplicate, Delete, Enable all, Disable all (task 019 provides the operations).
- Text fields and the text area: Undo, Redo, Cut, Copy, Paste, Select all, with the enabled state and the shortcut shown.
- Response: Copy body, Copy as cURL, Save body (task 010).
- Menus open at the pointer, flip to stay on screen, close on Escape or an outside click, and are keyboard navigable (Up/Down/Enter).

## Acceptance
- Tests for each menu's items and enabled states, placement near the window edges, and that choosing an item runs the same code as its button or shortcut.
- A right click on a text field must not clear its selection.
- `make check` passes; `make shot` of an open menu.
