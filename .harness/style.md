# Visual style

How the app looks, where the toolkit leaves a choice. A change to the look changes this file in the same commit; the numbers below are the tokens and constants in `uitoolkit/core/theme.go`, never literals in a view.

## Dialogs

Every dialog is a modal floating panel over a dimmed scrim (`widget.Dialog`, `ResizableDialog`, `MessageDialog`), centered until the user drags it, built in this order from the top:

1. **Title bar** (`widget.DialogTitle`), then optionally 2. **toolbar** (`widget.DialogToolbar`), then 3. the **body**.

### Title bar

- Height `DialogTitleH` (44), side padding 16, gap 8, items centered vertically, flush with the panel's edges (the panel itself has no padding).
- Color `Frame` (the window chrome, so the bar reads as a window title and not as a row of the body), `HoverBg(Hover)` so it answers the pointer, a `Border` line under it (the toolbar's line is the soft `BorderSoft`).
- Content from the left: the **title** (bold, `Text`; a short noun: "Settings", "Log", "Rename"), an optional **subtitle** (`widget.DialogSubtitle`: muted, small, ellipsized, gives way first; the page shown, a count, the file; never a sentence or an action), free room, then the **close button** (32 x `CompactH`, `CrossIcon`, tooltip "Close (Esc)").
- No controls in the bar: no fields, no buttons except close. Those go in the toolbar.
- The close button, Escape and a click outside all run the dialog's `OnDismiss`, so they cancel alike. A dialog that must not be cancelled by an outside click still has the button.
- The bar is the drag handle (`DragHandle`): the free part (not the border, not the close button) shows the move cursor, a press drags the dialog, a double click centers it. The dialog stays inside the window (`Moved`, `MoveState`; decision 56).
- Not done: an icon before the title. The bar lays out title, then items, then room, then close, so an icon would have to be an item placed after the title; add it only with a place for it in `DialogTitle`.

### Toolbar

- Height `DialogToolbarH` (40), side padding 16, gap 8, `BorderSoft` line under it; controls are `CompactH` high (fields `RowH`), a field that takes the room has `Grow(1)`.
- Holds what acts on the whole dialog: a search or filter field, Save, Clear, a mode switch. Settings: search, Save, JSON/Form. Log: filter, Clear.
- A dialog with no such control has no toolbar (message dialogs).

### Message dialogs

Title bar (title only), then a body padded 16 with gap 12: a muted `Lines` text, any form, the input line, then the buttons on the right (`RowH` high, gap 8, the primary one lit). Width `MessageDialogW` (380).

### Sizes and position

A resizable dialog is grabbed within 6 px inside or 4 px outside its border (`edgeSlop`, `edgeOut`), grows about its center and stays inside the window with `ScreenMargin` (decisions 56, 57). Size and position are app state that outlives the dialog's rebuild.
