status: todo

# Shortcut bindings live in the toolkit, not the app

## Goal
`app/shortcuts.go`, `shortcutpage` hold binding logic that duplicates or belongs next to `engine.Binding`.

## Scope
- [done] `shortcut` holds an `engine.Binding` default instead of `key, ctrl, shift, alt`; delete `shortcut.label()` (`app/shortcuts.go:44`) and use `Binding.String()`; `action()` in `keybindings.go:10` loses its rebuild. Tests that call `label()` (`app_flow_test.go:758,3070`) switch to `Binding.String()` with the same expectations.
- [done] `Binding.Matches(in core.Input, key ebiten.Key) bool` from `runShortcuts` (`shortcuts.go:70`).
- [open] `engine.BindingFrom(in core.Input, key)` from `shortcutpage.RecordKey` (`shortcutpage.go:200-215`).
- [open] `Binding.TypesText()` and `Binding.IsFunctionKey()` from `shortcutpage.TypesText/isFunctionKey` (`:82-101`); `Refuse` keeps its app-flavoured messages in `shortcutpage`.

## Acceptance
- Test written first and seen failing for the right reason, then passing; `make check` green; `make shot` PNG checked for UI changes; `architecture.md` and the `view` package doc index updated in the same commit; this task file deleted in the last item's commit.

## Notes
- Follow toolkit API rules 1-11 in `rules.md`; options named `OnX`, flags without arguments, doc comment starting with the name.
- Check first whether `engine` may import `core.Input` per the layer rules (`engine` depends downward only).
