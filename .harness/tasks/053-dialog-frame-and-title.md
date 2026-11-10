status: todo

# Dialog frame and title bar in view

## Goal
Modal frame (dim scrim + centered `Control("menu")` with `FocusScope`, `Modal`, `OnDismiss`) is built three times; the title bar twice.

## Scope
- [open] `view.Dialog(opts ...Item)` with `OnDismiss`, optional `Resizable`; replace `app/overlay.go:236-253`, `app/calltrace.go:165-171`, `settingsui/dialog.go:146-160`.
- [open] `view.DialogTitle(title string, opts...)` with a slot for filter/search and `OnClose`; replace `calltrace.go:176-191`, `settingsui/dialog.go:183-194`.

## Acceptance
- Test written first and seen failing for the right reason, then passing; `make check` green; `make shot` PNG checked for UI changes; `architecture.md` and the `view` package doc index updated in the same commit; this task file deleted in the last item's commit.

## Notes
- Follow toolkit API rules 1-11 in `rules.md`; options named `OnX`, flags without arguments, doc comment starting with the name.
