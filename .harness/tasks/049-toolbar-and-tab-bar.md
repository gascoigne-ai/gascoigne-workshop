status: todo

# Toolbar and tab bar in view

## Goal
`Row(H(RowH+8), PadX(Gutter), Gap(6), AlignCenter, BorderB)` and the tab-strip wrapper repeat.

## Scope
- [open] `view.Toolbar(opts ...Item)`; replace `app/panes.go:484,494,519` and `app/wsbar.go:18` (36 px variant, keep as an option).
- [open] `view.TabBar(...)` for `Row(H(paneTabsH), PadT(tabGap), BorderB, Tabs)`; replace `app/view.go:358`, `app/mcpexplorer.go:133`.

## Acceptance
- Test written first and seen failing for the right reason, then passing; `make check` green; `make shot` PNG checked for UI changes; `architecture.md` and the `view` package doc index updated in the same commit; this task file deleted in the last item's commit.

## Notes
- Follow toolkit API rules 1-11 in `rules.md`; options named `OnX`, flags without arguments, doc comment starting with the name.
