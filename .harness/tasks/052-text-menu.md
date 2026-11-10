status: todo

# Text-field context menu in view

## Goal
`textMenu` (`app/ctxmenus.go:126-160,207`) offers undo, redo, cut, copy, paste, select all; it is used 24 times and `TextMenu` is a method on 3 to 4 `Host` interfaces.

## Scope
- [open] `view.TextMenu(field *TextField, opts...)` option that opens the toolkit menu on right click.
- [open] Remove `Host.TextMenu` from `shortcutpage.go:38`, calltrace, settingsui, colorpage and re-point the 24 uses.
- [open] `statusClipboard` replacement (task 044) stays app-side wiring.

## Acceptance
- Test written first and seen failing for the right reason, then passing; `make check` green; `make shot` PNG checked for UI changes; `architecture.md` and the `view` package doc index updated in the same commit; this task file deleted in the last item's commit.

## Notes
- Follow toolkit API rules 1-11 in `rules.md`; options named `OnX`, flags without arguments, doc comment starting with the name.
- Needs task 051.
