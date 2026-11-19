status: todo
kind: refactor
commit: refactor(widget): sizes come from theme tokens, not literals
scope: uitoolkit/*,internal/*,.harness/style.md
depends_on: 105-rename-shadowing-params

# Replace spacing literals with tokens

Origin: the repo audit of 2026-11-19 (three read-only reviews against the harness rules). The user approved all phases and, with that, the deletions and the gate change named here. Behavior stays the same unless the task says otherwise; tests stay green and none is loosened. Update `architecture.md` in the same commit where it describes what changes.

`.harness/rules.md`: UI sizes come from `uitoolkit/core/theme.go`.
- Use existing tokens: `dialog.go` (`W(32)` is `IconBtnW`, `Gap(8)` `LooseGap`), `rows.go` (`formLabelGap` is `Gutter`, `Gap(6)` `TightGap`), `menu.go` (`MenuRowH` is `CompactH`), and the literals in `button.go`, `tabs.go`, `table.go`, `checkbox.go` (the `14`), `field.go`, `slider.go`, `codeview.go`, `textarea.go`, `view/runtime.go` (shadow offset 4, edge width 2).
- Add tokens where none fits: `DialogPad`, `CheckboxSize`, `CellPad`; dialog margin 40, max height 560 and title height 44 duplicated in `calltrace.go`, `settingsui/dialog.go`, `destpicker.go` become one set in `core`.
- One constant for the 6 px edge shared by `view/overlay.go` `dragEdge` and `widget/resizable.go` `edgeSlop`.
Acceptance: `make check` green and golden files unchanged; `make shot` PNG looked at and identical in layout.
