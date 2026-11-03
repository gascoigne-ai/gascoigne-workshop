status: todo

# Dialogs, context menus and toasts

## Goal
The only overlay is the settings menu. CRUD and confirmation need more.

## Scope
- `vml` overlays: context menu (right-click, keyboard navigable), modal dialog (title, body, buttons, text input), toast (auto-dismiss, errors persist until clicked).
- Text widgets get a context menu: Cut, Copy, Paste, Select all.
- Replace the status-bar-only reporting of clipboard errors with toasts.

## Acceptance
- Tests for menu placement near window edges, keyboard navigation, dialog focus trap, toast timing with an injected clock.
- `make check` passes.
