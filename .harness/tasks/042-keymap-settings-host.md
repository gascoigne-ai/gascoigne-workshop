status: todo
kind: refactor
commit: refactor(settings): key bindings leave the app and run from a keymap
scope: internal/**, cmd/**, e2e/**, Makefile, .harness/**
not_before: 2026-01-01
depends_on: 041-mcp-pane

# feature/settings/keymap and the settings host leave app

Goal: `settingshost.go` (272 lines), `settings_keybindings` and `settings_shortcuts` move to
`feature/settings/keymap` (binding resolution; runs the `Commands()` that components offer) and
`ui/overlays/settingsdialog` (the host of its pages). `routeKeys` shrinks to the `window.Keys` chain: overlays,
sidebar, editor, response, then `keymap.Run`.

OPEN QUESTION (ask the user before building): where action ids are owned and where bindings are persisted.
Recommend: ids declared by each component's `commands.go`, bindings persisted by `settings` as today. Record the
answer under `## Decision`.

## Model
New `keymap.Keymap`: owner of the resolved binding per action id; invariant: one binding resolves to one action
(conflicts reported, as the shortcut page does now); a map from key chord to action id (looked up per key event),
built when settings change. Rejected: a switch on key in `app`.

## Acceptance
- Test first: resolution and conflict tests, `Run` calls the offered command only for unclaimed keys.
- The settings dialog tests keep their assertions.
- Common acceptance of `.harness/reorg.md`.
