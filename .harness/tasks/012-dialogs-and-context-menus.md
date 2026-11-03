status: todo

# Dialogs, context menus and toasts

## Goal
The only overlay is the settings menu. CRUD and confirmation need more.

## Scope
- `vml` overlays: context menu (right-click, keyboard navigable), modal dialog (title, body, buttons, text input), toast (auto-dismiss, errors persist until clicked).
- The context menu primitive only (items, shortcuts, disabled state, submenu-free, keyboard navigation); the menus for tabs, tree, tables, text and the response are task 018.
- Replace the status-bar-only reporting of clipboard errors with toasts.

## Acceptance
- Tests for menu placement near window edges, keyboard navigation, dialog focus trap, toast timing with an injected clock.
- `make check` passes.
