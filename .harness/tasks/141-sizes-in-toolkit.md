status: todo
kind: refactor
commit: refactor(widget): toolkit widgets read sizes from Ctx.Sizes
scope: uitoolkit/**, .harness/style.md
not_before: 2026-01-01
depends_on: 140-sizes-on-ctx

# Toolkit widgets use Ctx.Sizes

Goal: every widget, view primitive and package-level derivation under `uitoolkit/` reads heights and
widths from `Ctx.Sizes` instead of the height constants (`RowH`, `CompactH`, `SlimH`, `LabelH`, `NoteH`,
`IconBtnW`, `CheckboxSize`, `ToolbarH`, `TabBarH`, dialog heights). Package-level `const` derivations (for
example `MenuRowH`) become functions of `Ctx`. Remove the remaining raw literals in `widget/dialog.go` and
`widget/table.go` by giving them roles in the size rules.

## Model
Model: unchanged.

## Acceptance
- Behavior is identical at default settings: golden scenes and `view.Dump` tests stay unchanged and
  unedited.
- A guard test fails when a widget uses a height constant instead of `Ctx.Sizes`.
- `make check` passes.
