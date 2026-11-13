# 011: choice, form-row, check-row, lines and note elements in markup

## Objective
Templates stop spelling out widget compositions that exist as Go constructors: `widget.Choice`, `widget.FormRow`, `widget.CheckRow`, `Lines` and `widget.Note`.

## Why
Written out in several views already (see `grep -rn "written out\|spelled out" --include=*.vml` and loop.md's gap list): choice (urlbar, responsepane, authchip*, explorerlist, settingsui), form-row/check-row (settingspane, authpane; mcpform is next), lines (authpane, others), note. Rows 15-20 would copy them again.

## Scope (one element per commit, in this order)
For each: element and attributes mirror the Go constructor (rule 9: kebab-case names; rule 10: doc comment, test comparing template `Dump`/behavior to the Go call written first and seen failing, `markup.Reference()` row, package doc index line). Reuse the pattern of `icon-button` (task 012, `widgets_test.go`).
1. `choice` (label text, `selected` flag attribute; today's conditional pairs `primary-button if` / `button if="!x"` become one element where the Go constructor does that).
2. `form-row` (label attribute, children as the control; the label gap is the unexported literal in widget, keep it there).
3. `check-row` (state binding as Decision 52, label, `on-change`/`on-click` as the Go option).
4. `lines` and `note` (same).
5. Replace the written-out copies in all `.vml` files with the elements; the existing pin tests, testdata and golden PNGs stay unchanged (zero diff in this commit). Remove the "no X element" comments.
6. `architecture.md` element list and `loop.md` gap list updated; delete this file in this last commit.

## Acceptance
`make check` green; each new-element test fails before and passes after; item 5 commit has zero test/testdata/PNG diff. If an element's Go constructor cannot be mirrored without a new option, stop and report instead of widening scope.
