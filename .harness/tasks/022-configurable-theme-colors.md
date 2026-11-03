status: todo

# Configurable colors and selection contrast

## Goal
The palettes are fixed. Selection and find highlights in particular are hard to see for some users, and what looks right on one monitor does not on another.

## Scope
- Settings gain a Colors section: the selection color, the find-match colors, the focus outline and the accent can be changed per theme, with a color picker (hex field plus swatches) and a live preview.
- A contrast setting (Normal, High) strengthens the selection, find and focus colors without touching the rest; High meets at least 7:1 for text on a selected background.
- Every override is checked against the theme's text contrast ratio (the same math as the existing theme test); a color that would make text unreadable is refused with the ratio shown.
- Overrides are stored in the settings file as token to hex, apply to a theme by id, ignore unknown tokens, and can be reset one by one or all at once.
- Themes stay tables: user colors are layered over `widget.Dark()` / `widget.Light()` with `Theme.With`, not a third theme.

## Acceptance
- Tests: the contrast helper accepts and refuses colors at the boundary, overrides layer correctly and survive a theme switch, a corrupt or partial settings file falls back per token.
- `make check` passes; `make shot` for the default and the High contrast variants of both themes.
