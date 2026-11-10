status: todo

# Input helpers move to core

## Goal
`swallowMenuKeys`, `hasKey` and `restartsCaret` are pure functions of `core.Input`.

## Scope
- [done] `core.Input.Consume(keys ...ebiten.Key)` replaces `swallowMenuKeys`/`hasKey` (`app/overlay.go:329,348`).
- [open] `core.Input.RestartsCaret()` replaces `restartsCaret(in)` (`app/redraw.go:36`).

## Acceptance
- Test written first and seen failing for the right reason, then passing; `make check` green; `make shot` PNG checked for UI changes; `architecture.md` and the `view` package doc index updated in the same commit; this task file deleted in the last item's commit.

## Notes
- Follow toolkit API rules 1-11 in `rules.md`; options named `OnX`, flags without arguments, doc comment starting with the name.
- Do before task 051 (menu), which uses `Consume`.
