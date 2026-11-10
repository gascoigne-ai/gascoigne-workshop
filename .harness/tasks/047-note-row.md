status: todo

# Note row (inline error or muted text) in view

## Goal
`Row(H(NoteH), Text(note, Small, Color(Error), Ellipsis, Grow(1)))` is copied 6 times.

## Scope
- [open] `view.Note(text string, opts ...Option)` with flags `Muted()` and `Error()` (check no name clash, rule 5).
- [open] Replace: `settingsui/form.go:182,312`, `colorpage.go:139`, `app/raw.go:198`, `shortcutpage.go:321,323`.

## Acceptance
- Test written first and seen failing for the right reason, then passing; `make check` green; `make shot` PNG checked for UI changes; `architecture.md` and the `view` package doc index updated in the same commit; this task file deleted in the last item's commit.

## Notes
- Follow toolkit API rules 1-11 in `rules.md`; options named `OnX`, flags without arguments, doc comment starting with the name.
