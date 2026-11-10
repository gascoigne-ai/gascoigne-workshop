status: todo

# Clipboard reporting decorator in core

## Goal
`statusClipboard` (`app/clipboard.go:12-32`) is generic: it wraps a clipboard and reports what was copied.

## Scope
- [open] `core.ReportClipboard(inner Clipboard, report func(string)) Clipboard` with its tests (copy reports `Copied N characters`, paste does not).
- [open] App uses it; delete `statusClipboard`.

## Acceptance
- Test written first and seen failing for the right reason, then passing; `make check` green; `make shot` PNG checked for UI changes; `architecture.md` and the `view` package doc index updated in the same commit; this task file deleted in the last item's commit.

## Notes
- Follow toolkit API rules 1-11 in `rules.md`; options named `OnX`, flags without arguments, doc comment starting with the name.
