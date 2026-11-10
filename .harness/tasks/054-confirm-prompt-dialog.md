status: todo

# Confirm and prompt dialogs in view

## Goal
`dialog`, `dialogButton`, `confirm`, `prompt`, `dialogDefault` (`app/overlay.go:54-66,194-255`) are generic except for `form *mcpform.Form`.

## Scope
- [open] `view.DialogState{Title, Body, Input, Buttons}` and rendering through `view.Dialog` (task 053); the MCP form becomes a body slot supplied by the app.
- [open] `App.confirm` and `App.prompt` stay as thin wrappers over it (8 uses plus `askSaveSettings` in `settingshost.go`).

## Acceptance
- Test written first and seen failing for the right reason, then passing; `make check` green; `make shot` PNG checked for UI changes; `architecture.md` and the `view` package doc index updated in the same commit; this task file deleted in the last item's commit.

## Notes
- Follow toolkit API rules 1-11 in `rules.md`; options named `OnX`, flags without arguments, doc comment starting with the name.
- Needs task 053 and 052.
