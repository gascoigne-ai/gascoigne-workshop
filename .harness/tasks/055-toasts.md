status: todo

# Toast stack in view

## Goal
`controller/toasts.Stack` and `toastsView` (`app/overlay.go:257-299`) are generic: info fades, errors stay, dedupe, max 5.

## Scope
- [open] Move the stack to `view.ToastStack` with its tests; `view.Toasts(stack, now, window, opts...)` takes the anchor (bottom right above a bottom inset) instead of `statusH`, `a.w`, `a.h`.
- [open] Re-point the 38 `a.toast` and 9 `prefs.Toast` callers; delete `workspace/controller/toasts`.

## Acceptance
- Test written first and seen failing for the right reason, then passing; `make check` green; `make shot` PNG checked for UI changes; `architecture.md` and the `view` package doc index updated in the same commit; this task file deleted in the last item's commit.

## Notes
- Follow toolkit API rules 1-11 in `rules.md`; options named `OnX`, flags without arguments, doc comment starting with the name.
- The layer test forbids controllers importing the toolkit, so the stack cannot stay a controller that the view reads; check `internal/layers_test.go` and `decisions.md` 31 before moving.
