status: todo
kind: feature
commit: feat(settings): the text size is a number the user sets
scope: internal/settings/**, uitoolkit/theming/**, uitoolkit/core/theme.go, cmd/workshop/**, .harness/style.md, .harness/architecture/settings.md
not_before: 2026-01-01
depends_on: 142-sizes-in-app-views, 143-sizes-in-feature-views

# Numeric text size

Goal: the user sets the UI font size as a number (default 14) instead of small, medium or large. The
small and mono faces derive from it, and controls follow through `Ctx.Sizes`.

## Model
`Settings.TextSize string` becomes `FontSize int` (px, 10..24, default 14), clamped by `settings.Clamp`.
Small = FontSize - 2 (min 9), mono = FontSize - 1; one function in `theming` owns the derivation, theme
files no longer carry font sizes. Old files: version bump through the `fileversion` table, a reader maps
`small`, `medium`, `large` to 13, 14, 16 and unknown values to 14. The settings form shows a number
field (step buttons plus typed input) in place of the chips; the schema row carries the range. Alternatives
rejected: keeping presets (no free value), a float (sub-pixel sizes blur text; revisit if hi-dpi needs it).

## Scope
1. Tests first: the version reader for old values, `Clamp`, the derivation table, the form row.
2. Settings, schema row, migration, fixtures of real old output.
3. `theming`: `SizeChoices` removed, `Prefs.FontSize`, derivation; `appearance.ThemeFor` callers.
4. Golden scenes `large-text` and `small-text` use 16 and 13; regenerate the goldens the way `release.md`
   says; check `TestEveryFuzzTargetIsInTheMakefile` and add a fuzz case for the new reader.
5. `style.md` Type and Sizes, `architecture/settings.md`, CHANGELOG line.

## Acceptance
- At 14 nothing changes on screen; at 10 and 24 no control clips or overlaps (a `view.Dump` or golden
  scene at each end of the range).
- `make check` passes.
