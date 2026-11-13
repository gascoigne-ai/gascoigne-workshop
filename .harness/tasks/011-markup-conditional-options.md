# 011: conditional options in markup

## Objective
A template can switch a style option off per item, so views like the request tab strip (separator on every tab but the first) and the sidebar drop outline need no Go-side patching and no duplicated templates.

## Why
Row 8 applied the drop outline in Go on the built node; row 10 is blocked: `Separator(12)` only on tabs after the first. Two needs, so the loop rule makes it a toolkit task.

## Scope
1. A color-token attribute (`tokenAttr`: `bg`, `border*`, `edge-*`, `hover-*`, ...) given an empty token value applies no option (today: build error). A nonexistent non-empty token stays an error.
2. `separator` attribute: value 0 applies no option (markup only; Go `Separator(0)` is unchanged).
3. Each item with its test written first (fails, then passes), a row/note in `markup.Reference()` and the package doc, and a decision in `decisions.md` (53 addendum or new number).
4. Switch row 8's drop outline (`treeRow`) to the template: `border="{{dropBorder}}"` with an empty token when not the target. Existing pins and goldens stay unchanged.

## Acceptance
- `make check` green; tests for 1 and 2 fail before, pass after.
- Row 8 pins/testdata/PNGs unchanged by item 4; the Go-side `Border(w.AccentLine)(built)` is gone.
- `architecture.md` and `loop.md` conventions updated (conditional option = empty token / zero separator).

## Validation
`git diff` of test files and testdata in item 4's commit is empty.
