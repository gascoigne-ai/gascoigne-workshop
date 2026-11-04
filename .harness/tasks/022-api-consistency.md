status: todo

# Toolkit API consistency debt

## Goal
Bring the remaining exported names of `uitoolkit` in line with the rules in `rules.md` ("Toolkit API rules"). Found by reading the whole exported API once; fixed already: sides and axes (border, padding, margin), icons end in `Icon`, `Ctx.Color` (was `Col`), `core.Gutter` (was `Pad`), `Seg` merged into `Span`, `Modal()` + `OnDismiss`, positional callbacks of `Slider` and `Tabs` became `OnSlide` and `OnSelect`.

## Scope (rule in brackets)
- `NewTextField` returns a value, `NewTextArea` a pointer, `NewMultiline` a value [7]: pick one (pointer) and change the owners (`reqTab.url`, `find`, `timeout`, ...).
- State type naming: `TextField`, `TextArea`, `CodeView` vs `ScrollState`, `SplitState`, `SliderState` [6]: keep, or give all a common shape.
- `Separator(show bool, inset int)` takes a bool [4]: use `If`-style at the call site, keep `Separator(inset)`.
- `OnHeldOver` should read `OnHover`-like [3]; `OnTab(func(back bool))` is fine.
- `Center()` (text) next to `AlignItems(AlignCenter)` [5]: rename the text one `TextCenter()` or `Centered()`.
- `Reveal()` (option, keep a tab in view) and `TextField.Reveal(x, w)`, `CodeView.RevealLine` [5].
- `Lighten()` option in `vml` and `core.Lighten` function [5].
- `Float(top, right)`, `FloatAt(x, y)`, `FloatCenter()`: one family, check the argument order and names [1].
- `core.ParseHex` and `core.ParseColor` both parse colors [8].
- `CodeView.Scroll(dy, contentH, viewH)` method and `vml.Scroll` element share a name [5].
- Template attributes missing for options that could have one [9]: `h`/`w` have them; `Edges`, `Float*`, `HoverBg`, `Tooltip` variants, `Separator`, `BorderInset`.

## Acceptance
- Each item fixed or recorded as an accepted exception in `decisions.md`; `make check` passes after every step.
