status: todo
kind: chore
commit: refactor(toolkit): API follows rules 5, 6, 7, 9 and 10
scope: uitoolkit/*,internal/*,.harness/*
depends_on: 130-settings-getset

# Bring the toolkit API in line with its own rules

Origin: the repo audit of 2026-11-19 (three read-only reviews against the harness rules). The user approved all phases and, with that, the deletions and the gate change named here. Behavior stays the same unless the task says otherwise; tests stay green and none is loosened. Update `architecture.md` in the same commit where it describes what changes.

- Rule 5: `Choice` means three things (`core.Ctx` font pick, `theming`, `widget/button.go`): `core.FontChoice`, `theming.ThemeChoice`, and a segmented-chip name; `Segments`, `TextArea.Styler`, `TableSpec.CellStyler` (same `func(text) []Span`) get one name.
- Rule 6: `ColumnWidths` holds state without the `State` suffix: rename.
- Rule 7: `NewTextField` and `NewTextArea` return values, `NewMenu` a pointer: pick one return kind per family or record the exception in `decisions.md`; ask the user if no answer is obvious.
- Rule 9: `Toolbar` and `DialogSubtitle` have no template element: add `<toolbar>` and `<dialog-subtitle>` with tests and rows in the template tables; half families `EdgeT/L/R` without B and `RoundT/B/BL` (rule 1): complete them or record why in the doc comments.
- Options that panic on the wrong element by bare type assertion (`field.go`, `split.go`, `code.go`) or ignore it silently (`slider.go`): one `behaviorOf[T](element, "Placeholder")` naming option and element; the eight nil-check sites for `SelectHandler`, `DismissHandler`, `ChangeHandler` become `El.Select(i)`, `El.Dismiss()`, `El.Changed()`.
- Exported mutable globals: `engine.KeyNames`, `theming.SizeChoices`, `CornerChoices`, `ContrastChoices`, `CustomColors`: unexport or return copies; `core.Ctx.Menu any` and `FocusOwner any` (widget state type-erased in `core`, rule 11): replace if it is possible without a layer break, else record why in `decisions.md`.
- Slider and `SelectHandler`-style options documented; `Glob` error in `theming.LoadUser` logged at debug.
- Each new option or element has its doc comment, test, doc index line and template row (rule 10).
Acceptance: `make check` green.
