status: done (option A)

# Move the theme files out of core into a dedicated package

## Goal
`uitoolkit/core` should hold the fundamentals only. Today it also holds the whole theme pipeline: the `Theme` struct and tokens (`theme.go`), the JSON theme parser and the `Dark()`/`Light()` palettes (`themefile.go`, `themes/dark.json`, `themes/light.json`) and color math (`color.go`: hex parsing and formatting, contrast ratio, `EnsureContrast`, shading). Preference mechanics already live in `uitoolkit/theming`.

## The constraint (why this is a task and not a move)
`core.Ctx` holds a `Theme` and resolves tokens (`Ctx.Color`), and `theming` imports `core`. A package that `core` depends on cannot import `core`, so the theme data cannot simply move into `theming` while `Ctx` stays in `core`.

## Options
- **A. Split by direction.** Keep the types (`Token`, `Theme`, `ControlStyle`, token constants) in `core` because `Ctx` needs them. Move what only builds themes into `theming`: the JSON parser (`ParseTheme`, `ThemeExtends`), the palettes and their files (`Dark`, `Light`, `themes/*.json`) and the color math. No import cycle: `theming` imports `core`. App code that wrote `core.Dark()` writes `theming.Dark()`. Tokens stay `core.Bg` etc.
- **B. New bottom package `theme`.** `theme` holds tokens, `Theme`, parser, palettes and color math and imports nothing; `core` and `theming` import it. Tokens become `theme.Bg`, so views would qualify them differently (`w.Bg` becomes `theme.Bg`), a large mechanical rename of about 600 call sites, and the short alias convention changes.
- **C. Make `Ctx` independent of `Theme`.** `Ctx` takes a `Colors` interface (`Color(Token)`, `Style(name)`), so `core` no longer needs the `Theme` type and everything theme-related moves to `theming`. Cleanest layering, but touches `Ctx` and the fonts (fonts are built from theme sizes).

Recommendation: **A** now (smallest change, no cycle, tokens stay short); revisit C if a second renderer appears.

## Scope for option A
- Move `themefile.go`, `themes/*.json`, `color.go` and the palettes to `uitoolkit/theming`; keep `theme.go` types in `core`.
- Tests that call `NewCtx(Dark())` in `core`'s own internal tests would need `theming` (cycle): give those tests a small test-only theme builder in `core` (`testTheme()`), and move the theme-file tests to `theming`.
- Update every `core.Dark()`, `core.Light()`, `core.ParseTheme`, `core.ParseHex`, `core.FormatHex`, `core.ContrastRatio`, `core.EnsureContrast` use (app, `appearance`, `vml` and `component` tests, examples).
- Update `uitoolkit/doc.go`, the layers test (`core` still imports nothing; `theming` imports `core` only), architecture.md and rules.md.

## Acceptance
- `core` has no JSON, no palettes and no color math beyond what `Ctx` itself needs; `make check` passes; the layers test is unchanged except for the file moves.
- No behavior change: screenshots in dark and light match before and after.
