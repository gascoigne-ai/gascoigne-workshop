status: mostly done (remaining items are small)

# Toolkit API consistency debt

## Goal
Bring the remaining exported names of `uitoolkit` in line with the rules in `rules.md` ("Toolkit API rules"). Found by reading the whole exported API once; fixed already: sides and axes (border, padding, margin), icons end in `Icon`, `Ctx.Color` (was `Col`), `core.Gutter` (was `Pad`), `Seg` merged into `Span`, `Modal()` + `OnDismiss`, positional callbacks of `Slider` and `Tabs` became `OnSlide` and `OnSelect`.

## Done
- `NewTextArea` returns a value like `NewTextField` and `NewMultiline` [7] (all editors are values the owner stores).
- `Separator(inset)` plus `When(cond, opt)` instead of a bool parameter [4].
- `OnHeldOver` is `OnDragOver`; text `Center()` is `TextCenter()` (template `text-center`); the option `Reveal()` is `KeepInView()`; `Float(top, right)` is `FloatTopRight` next to `FloatAt` and `FloatCenter`; `CodeView.Scroll` is `ScrollBy` [3, 5].

## Left
- State type naming: `TextField`, `TextArea`, `CodeView` vs `ScrollState`, `SplitState`, `SliderState` [6]: accepted as it is (rule 6 describes it); revisit only if a widget gets both.
- `core.ParseHex` and `core.ParseColor` both parse colors, in `theming` now [8]: make `ParseColor` build on `ParseHex`.
- `core.Lighten` (a function) and `vml.Lighten()` (an option) share a name; they are the same idea at two levels [5]: accepted, or rename the function `Brighten`.
- Template attributes missing for options that could have one [9]: `Edges`, `FloatAt`/`FloatCenter`/`FloatTopRight`, `HoverBg`, `Separator`, `BorderInset`, `Tooltip` is there as `tip`.

## Acceptance
- Each item fixed or recorded as an accepted exception in `decisions.md`; `make check` passes after every step.
