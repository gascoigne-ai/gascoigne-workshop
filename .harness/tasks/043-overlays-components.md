status: todo
kind: refactor
commit: refactor(ui): overlays are components under ui/overlays
scope: internal/**, cmd/**, e2e/**, Makefile, .harness/**
not_before: 2026-01-01
depends_on: 042-keymap-settings-host

# filebrowser, welcome, menus, toasts and the overlays composite

Goal: `window_filebrowser`, `window_welcome`, `window_menu`, `window_menuicons`, `window_overlay`,
`window_hosts` and the toast drawing move to `ui/overlays/{filebrowser,welcome,menus,toasts}` and the
`overlays` composite (open and close state, key order for Escape, the calltrace host). Follow the component
recipe of `.harness/reorg.md`.

## Model
Model: unchanged. `feature/workspace/overlay` keeps the stack rules; widget state moves into the components.

## Acceptance
- Dump tests against fake `Deps`; filebrowser, dialog drag, focus order and cursor tests move with them.
- Common acceptance of `.harness/reorg.md`.
