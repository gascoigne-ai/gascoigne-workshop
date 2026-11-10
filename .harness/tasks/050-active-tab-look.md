status: todo

# Active tab look is one implementation

## Goal
`app/view.go:224` copies the active-tab styling of `view/components.go:83`; decision 14 said to reopen when a second consumer appears.

## Scope
- [open] Export the look (`ActiveTab()` option) or extend `TabItem` with slots (`Prefix`, `Suffix`, `OnClose`, `OnMiddle`, `OnRight`, `OnDragOver`); ask the user which if both stay open after reading the code.
- [open] `App.requestTab` is built from the toolkit tab element; delete the copy.
- [open] Record the outcome in `decisions.md` (decision 14 reopened).

## Acceptance
- Test written first and seen failing for the right reason, then passing; `make check` green; `make shot` PNG checked for UI changes; `architecture.md` and the `view` package doc index updated in the same commit; this task file deleted in the last item's commit.

## Notes
- Follow toolkit API rules 1-11 in `rules.md`; options named `OnX`, flags without arguments, doc comment starting with the name.
- Do after task 049.
