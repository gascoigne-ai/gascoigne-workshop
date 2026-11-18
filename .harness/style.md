# Visual style

How the app looks, where the toolkit leaves a choice. A change to the look changes this file in the same commit; the numbers below are the tokens and constants in `uitoolkit/core/theme.go`, never literals in a view. Neighbours are `TightGap` (6) apart when they are controls side by side and `LooseGap` (8) apart when they are the blocks of a pane.

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
- The welcome dialog is a message dialog 560 wide: the lead text, then the release notes scrolling in a column (a bold heading and bulleted muted points per section, 12 between sections, 4 inside, points wrapped by hand at 64 characters), and one primary button, Get started.

### File browser

The in-app file chooser (decision 18): width 640, height 520 at most (40 from the window's edges). Title bar: the caller's title and the count of items shown as subtitle. Toolbar: the parent-folder icon button (`chevron-up`, `IconBtnW` x `CompactH`), the path field (`RowH`, `Sunken`, grows), the Hidden files check row. The list fills the rest on `Sunken`: `ListRow`s `RowH` high, padded 12, gap `LooseGap` (8), a muted folder icon 16 wide before a folder's name and the same room empty before a file's, the selected row on `Selected`. Footer: `RowH` buttons plus 12 above and below, padded 16, a `BorderSoft` line over it; the problem in small `Error` text takes the room on the left (while there is none, the same place holds a small `Muted` hint naming the program that gives native dialogs on this desktop: "For native dialogs install zenity"), then Cancel and the primary Open or Save (lit), at least 80 wide, as in message dialogs.

### Sizes and position

A resizable dialog is grabbed within 6 px inside or 4 px outside its border (`edgeSlop`, `edgeOut`), grows about its center and stays inside the window with `ScreenMargin` (decision 15). Size and position are app state that outlives the dialog's rebuild.

### Destination picker

The dialog of Move to... and Copy to... (decision 25): width `dialogW` (440), height 480 at most (40 from the window's edges), a message-dialog footer. Title bar: "Move to" or "Copy to", the node's name as the muted `DialogSubtitle`. The list fills the rest on `Sunken`: `ListRow`s `RowH` high, padded `Gutter` times one plus the folder's depth on the left and `Gutter` on the right, gap `TightGap`; a collection's root row is bold with its `collection.N` stripe (3 x 14) as in the tree, the other rows regular. A row that cannot be chosen is `Muted` with a small muted reason at its end ("itself", "inside itself", "current folder"). Footer: `RowH` buttons, padding 12 above and below and 16 at the sides, a `BorderSoft` line over it; a small muted hint "Variables are not changed" takes the room on the left, then Cancel and the primary "Move here" or "Copy here", at least 80 wide, lit only once a row is picked (before that it is unlit and muted and does nothing). Keys: Up and Down skip refused rows, Enter and a double click pick, Escape or a click outside closes. The dialog is movable by its bar (`MoveState`).

## Request panes

### Proto (gRPC)

Padded `Gutter`, gap `LooseGap` (8), like Auth: a `RowH` row of two choice chips, *Reflection* and *Proto files* (the chosen one lit). Reflection shows one muted line, "Methods come from the server's reflection". Proto files shows two lists, each a caption (FILES, IMPORT PATHS), its rows (a `Sunken` path field `RowH` high that grows, then a Remove icon button `IconBtnW` x `CompactH` with the trash icon, gap `TightGap` (6)) and its add button (*Add file...*, *Add path*). Problems found in the files follow the lists, one small `Error` line each, `file:line:col: message`.

## Sidebar

- With no collection open the tree's place holds, on `Sunken` with padding `Gutter` and gap `LooseGap` (8): a muted note "No collection is open", then the buttons *Open collection folder...* and *New collection...*, `RowH`-high (28) and as wide as the sidebar. Nothing is made on disk until one is pressed.
- A collection's root row looks like a folder row (same height, indent `Gutter`, disclosure triangle, no icon of its own and no extra spacing) except that its name is bold (`bold` on the name's texts, also on the pieces the filter splits it into), so where one collection ends and the next begins reads at a glance; several roots stand one after another at depth 0. Folder and request names stay regular.
- Dragging a node onto a row outlines the row in `AccentLine`; while Alt is held the outline is `OK` (green), the cue that releasing copies instead of moves (decision 25).
