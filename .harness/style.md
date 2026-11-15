# Visual style

How the app looks, where the toolkit leaves a choice. A change to the look changes this file in the same commit; the numbers below are the tokens and constants in `uitoolkit/core/theme.go`, never literals in a view.

## Dialogs

Every dialog is a modal floating panel over a dimmed scrim (`widget.Dialog`, `ResizableDialog`, `MessageDialog`), centered until the user drags it, built in this order from the top:

1. **Title bar** (`widget.DialogTitle`), then optionally 2. **toolbar** (`widget.DialogToolbar`), then 3. the **body**.

### Title bar

- Height `DialogTitleH` (44), side padding 16, gap 8, items centered vertically, flush with the panel's edges (the panel itself has no padding).
- The bar is drawn like the window's header, which is the app's identity: `Hero` with the `hero` gradient control (`Control("hero")`), so it differs from the body in every theme, its two top corners rounded to the panel's radius (`RoundT("menu")`, anti-aliased with alpha, so the bar follows the panel's outline instead of poking out of the corners), with a `Border` line under it (the toolbar's line is the soft `BorderSoft`). Text on it is `HeroText`, icons and the subtitle `HeroMuted`; never `Text`/`Muted`, which are for the body.
- Content from the left: the **title** (bold, `HeroText`; a short noun: "Settings", "Log", "Rename"), an optional **subtitle** (`widget.DialogSubtitle`: `HeroMuted`, small, ellipsized, gives way first; the page shown, a count, the file; never a sentence or an action), free room, then the **close button** (32 x `CompactH`, no chip, `CrossIcon` in `HeroMuted`, darkened under the pointer with `Shadow` (alpha) in the `row` control's corners, so it follows the Corners setting like the bar, tooltip "Close (Esc)").
- No controls in the bar: no fields, no buttons except close. Those go in the toolbar.
- The close button, Escape and a click outside all run the dialog's `OnDismiss`, so they cancel alike. A dialog that must not be cancelled by an outside click still has the button.
- The bar is the drag handle (`DragHandle`): a press on the free part (not the border, not the close button) drags the dialog, a double click centers it. The grab cursor (`CursorMove`) shows only from the press until the release, never while hovering, so an idle bar looks like a title and not like a control. The dialog stays inside the window (`Moved`, `MoveState`; decision 15).
- Not done: an icon before the title. The bar lays out title, then items, then room, then close, so an icon would have to be an item placed after the title; add it only with a place for it in `DialogTitle`.

### Toolbar

- Height `DialogToolbarH` (40), side padding 16, gap 8, `BorderSoft` line under it; controls are `CompactH` high (fields `RowH`), a field that takes the room has `Grow(1)`.
- Holds what acts on the whole dialog: a search or filter field, Save, Clear, a mode switch. Settings: search, Save, JSON/Form. Log: filter, Clear.
- A dialog with no such control has no toolbar (message dialogs).

### Corners

Everything that fills a corner of the dialog panel (the title bar, the settings page list, the Log text) rounds that corner to the panel's radius with `RoundT`, `RoundB` or `RoundBL` (`round-t`, `round-b`, `round-bl` in templates; the argument is the panel's control, `menu`), drawn with alpha at the edge. The radius is the control's after the user's Corners setting (`Ctx.Radius`), so square, soft and round all fit; a square fill in a rounded panel is a bug.

### Message dialogs

Title bar (title only), then a body padded 16 with gap 12: a muted `Lines` text, any form, the input line, then the buttons on the right (`RowH` high, gap 8, the primary one lit). Width `MessageDialogW` (380).

### Sizes and position

A resizable dialog is grabbed within 6 px inside or 4 px outside its border (`edgeSlop`, `edgeOut`), grows about its center and stays inside the window with `ScreenMargin` (decision 15). Size and position are app state that outlives the dialog's rebuild.
