status: todo

# Selectable list row in view

## Goal
Hover versus `Bg(Selected)` with `Control("row")` is repeated at 5 sites.

## Scope
- [open] `view.ListRow(selected bool, opts ...Option)` as a composition; test selected and hover looks.
- [open] Replace: `app/overlay.go:124`, `app/view.go:95`, `app/mcpexplorer.go:139`, `settingsui/fontlist.go:175`, `app/sentlog.go:111` (hover only).

## Acceptance
- Test written first and seen failing for the right reason, then passing; `make check` green; `make shot` PNG checked for UI changes; `architecture.md` and the `view` package doc index updated in the same commit; this task file deleted in the last item's commit.

## Notes
- Follow toolkit API rules 1-11 in `rules.md`; options named `OnX`, flags without arguments, doc comment starting with the name.
