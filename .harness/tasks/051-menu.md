status: todo

# Popup menu in view

## Goal
The context menu (`app/overlay.go:34-189,303-346`, `menuicons.go`) is generic and used by 15 `openMenu` calls, the variable-completion menu and the settings page menu.

## Scope
- [open] `view.MenuItem{Label, Shortcut, Hint, Icon, Disabled, Separator, Run}` and `view.MenuState{X, Y, Items, Hot}` with `Menu(state, opts...)`; key navigation (Up/Down skip separators and disabled, Enter, Esc) with `nextSelectable`; width measured at layout through `core.Ctx`.
- [open] Layout is pure and unit-tested without a GPU: placement keeps the menu inside the window.
- [open] The label-to-icon map (`menuicons.go`) stays in the app and fills `MenuItem.Icon`; `App.openMenu` becomes a thin wrapper.
- [open] Replace the three app menus; delete `menuRow`, `menuWidth`, `menuHasIcons`, `nextSelectable` from the app.

## Acceptance
- Test written first and seen failing for the right reason, then passing; `make check` green; `make shot` PNG checked for UI changes; `architecture.md` and the `view` package doc index updated in the same commit; this task file deleted in the last item's commit.

## Notes
- Follow toolkit API rules 1-11 in `rules.md`; options named `OnX`, flags without arguments, doc comment starting with the name.
- Needs task 043 (`Input.Consume`) and 046 (`ListRow`).
