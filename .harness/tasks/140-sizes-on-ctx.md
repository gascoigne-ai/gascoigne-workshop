status: todo
kind: feature
commit: feat(core): sizes derive from the UI font's line height
scope: uitoolkit/core/**, uitoolkit/theming/**, .harness/style.md
not_before: 2026-01-01

# Sizes struct on Ctx

Goal: control heights follow the UI font, so a user-chosen text size never clips. The rules are in
`style.md` (Sizes): heights are line height plus twice a padding, gaps are steps of a 2 px unit.
Today they are constants for the default faces (`BaseLine` 16, `BaseSmallLine` 14). Make them a value
computed from the faces' real line heights.

## Model
`core.Sizes{Line, SmallLine, Row, Compact, Slim, Label, Note, Toolbar, TabBar, DialogTitle,
DialogToolbar, IconBtnW, WordBtnW, Checkbox int}`, built by a pure `SizesFor(line, small int) Sizes`.
Invariants: every height >= its line height, `height - line` even, Row > Compact > Slim. Line is clamped
to 13..22 so an installed font with a huge ascent cannot blow up the rows. Spacing steps stay constants
(they do not follow the text). Stored on `Ctx` as `Ctx.Sizes`, recomputed in `SetTheme` and `SetFonts`,
never in `SetScale` (scale is applied at draw time; recomputing on it would jitter while the slider
moves). One source of truth: the old height constants are deleted once `Sizes` exists. Access pattern:
read every frame by layout, written on theme or font change; a value struct with fields. Rejected: a map
keyed by scope name (untyped). Revisit when a component needs a size per face (mono rows): add a field.

## Scope
1. Test first: table test of `SizesFor` over lines (13,11), (15,13), (16,14), (19,15), (30,20) incl. the
   clamp and the invariants; `TestCtxSetThemeUpdatesSizes`; default faces still give the current values.
2. `Sizes`, `SizesFor`, `Ctx.Sizes`, and `Sizes.Scope()` returning the template scope names (`rowH`,
   `compactH`, `gutter`, ...) in one place.
3. Keep the old constant names as thin wrappers only if a task after this one still needs them; list
   which. Update `style.md` Sizes.

## Acceptance
- Default settings: no golden scene or `view.Dump` changes.
- `make check` passes.
