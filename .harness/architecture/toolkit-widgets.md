## Toolkit: widgets and drawing

-  **Sideways scrolling** (`widget.HScroll`, `widget.ScrollTabs`): a row at natural width inside a
  clipped box with arrow buttons at overflowing ends (they take the click before the tabs below
  them), wheel scrolling, and `KeepInView()` to keep one child in view when it changes. The pane tab
  strips use it, so tabs past the pane width (Vars, Settings) stay reachable; their `ScrollState`
  lives in the request tab.
-  **Scrolling tab strips**: `HScroll` keeps an arrow's width of room at each end when its content
  overflows (so no tab is hidden under an arrow), draws an arrow only at an end with somewhere to
  scroll to (the arrow takes the click before the tab under it), and scrolls with the wheel and a
  touchpad's two-finger sideways swipe (`Input.WheelX`). The request tab strip uses it too
  (`App.tabsScroll`), keeping the active tab in view. That strip is markup (`requesttabs.go`):
  `requesttabs.vml` walks one `requestTabItem` view model per open request (badge and its color,
  name, dirty mark, active, handlers, and the separator's inset, 0 on the first tab so it has none;
  decision 4) inside `<hscroll>`, followed by the "+". 
-  **Panning a strip**: pressing on an overflowing `HScroll` and dragging more than 4 px pans it,
  following the pointer even outside the strip (`Ctx.PointerX`); the tab under the press is still
  selected. So a left drag does not reorder tabs while the request strip overflows
  (`ScrollState.Overflowing`); the middle button always reorders, and with room to spare a left drag
  still does.
-  **Separators and padding**: `view.Separator(inset)` draws a soft hairline (`BorderL`,
  `BorderInset`) on an element's left edge, used between tabs (request and pane strips) and between
  table columns (header and rows) so neighbours read as distinct. Holding an `HScroll` arrow scrolls
  once at the press, then repeats after 20 frames every 3 (a press on one arrow dragged onto the
  other does not). The table cell being edited is inset 3 px in its cell so its rounded outline and
  focus ring have room, and keeps its text where the shown cell had it (`FieldPad`: 3 px inset + 5
  px field padding = 8).
-  **Resizable** (`widget.Resizable`, `widget.ResizeState`): wraps a floating, centered element like
  a dialog. A press within 6 px inside its border (never outside it, where a modal dismisses) grabs
  an edge or corner (resize cursors, diagonal ones included); dragging sets `ResizeState.W` and `H`
  within the given minimum and maximum, and the caller applies them to the element's own size, so
  the state can be saved. It lays its children out top to bottom like `Col`.
-  **Focus scrolling and drag-expand**: after a Tab moves focus, `view.revealFocused` scrolls every
  `Scroll` holding the focused field so the field is inside its box; `App.dragHover` opens a closed
  folder a tree drag has rested over for `dragExpandTicks` (30) frames.
-  **Tooltips and icons**: `view.Tooltip(text)` on any element; `view.Draw` records the innermost
  hovered tip in `Ctx.TipText`, and `widget.TooltipLayer` (mounted last in the window's Stack, with
  the app's `TooltipState`) counts 30 ticks of rest, then draws a small box near the pointer
  (flipped above or inward at the screen edge). A click or moving to another element restarts the
  wait, elements under a modal overlay have none, and while a tip waits the app keeps drawing
  (`TooltipState.Waiting`). Line icons (`Trash`, `CopyIcon`, `SearchIcon`, `Plus`, `PromptIcon`,
  `ChevronUp/Down`, `Cross`) come from one `glyphs` table with a hover color table (`glyphHover`:
  the trash turns red); `IconButton(glyph, ...)` is an outlined chip around one. The response
  toolbar (find, copy, cURL), the find bar, the tab strip's new-tab button and the table delete
  cells use them; buttons and tabs show their shortcut in the tip (`App.hint`).
-  **Button width is a minimum**: `widget.Button` and `PrimaryButton` apply `view.FitW` after their
  options, so a `W` becomes `MinW` plus `Shrink`: the button is as wide as its label when the row
  has room and returns to `W` (label cut with an ellipsis) when the row is tight.
-  **Icons are SVG files** (`uitoolkit/widget/icons/*.svg`, `svg.go`, `icon.go`, task 023):
  `widget.Icon(name, opts...)` draws a parsed icon (the subset is described in `svg.go`: path with
  all commands including arcs, line, polyline, polygon, rect, circle, ellipse, `fill` and `stroke`
  as `none`, `currentColor` or `token(name)`, stroke width, cap and join, `data-hover="token"` for
  the hover color, `none` for no change) as vector paths scaled to the device, in the element's
  `Color` (muted by default). Natural size is the viewBox; `W` and `H` scale it. `RegisterIcons(fs)`
  adds or replaces icons from a folder and reports broken files with file and line; `IconNames`,
  `HasIcon` list them. `CrossIcon`, `TrashIcon` and the other named functions are one-line wrappers,
  the checkbox check (`check`) and the code view fold arrows (`triangle-down`, `triangle-right`) are
  icons too, drawn inside those widgets by `drawIconAt` (a `widget.Span` with an `Icon` is painted
  as that icon in a code view, its Text kept for copy: the WebSocket log marks use `arrow-up`,
  `arrow-down`, `dot`; the changed-setting mark and the release-note bullets use `dot`, the
  environment menu marks the active one with `check`), and templates have `<icon name="trash"/>` and
  `<icon-button icon="search"/>`. An unknown name panics in Go and is a build error in a template,
  both listing the names.
-  **Icon masks** (`widget/icon.go`): an icon drawn in one color (the usual case) is rendered once
  in white per icon, scale and stroke weight (`iconMask`, up to 512 entries, dropped when full),
  then drawn as one image tinted with the element's color; icons with `token(...)` paints are drawn
  from their paths each frame. Adding icons from an open set (Lucide is ISC, Tabler is MIT) is a
  matter of dropping SVG files into `uitoolkit/widget/icons` with the license file; the icons there
  are the toolkit's own.
-  **Icon stroke weight is a theme setting**: a theme file may set `"iconStroke": 1.4` (a
  multiplier, `core.Theme.IconStroke`; absent keeps the base theme's, 0 means 1); `widget.Icon`
  multiplies every stroke width by it, so thin and bold icon sets are a theme choice.
-  **Pixel snapping**: `Ctx.Fill`, `Outline`, `HLine` and `VLine` convert logical edges to device
  pixels with `Ctx.snap` (rounded per edge, so neighbours share an edge with no gap) and draw lines
  `Ctx.LineWidth()` thick, the scale rounded to whole pixels and at least one, so borders stay sharp
  at fractional scales.
-  **Rounded fills** (`Ctx.FillStyled`, `FillStyledCorners`) draw from a cached corner mask in one
  `DrawTriangles`, with no anti-aliased offscreen. The rect's edges snap to device pixels with
  `Ctx.snap` (as `Fill` does, so rounded and square elements share edges at any scale) and the
  radius is rounded to whole pixels and clamped to half the shorter side. `sliceQuads` (pure) cuts
  the rect into nine pieces: four corner squares sampling a quarter of the mask for a rounded
  corner (a solid texel for the others), four edge strips and the centre; the vertical gradient
  rides on the vertex colors (`gradientAt`). The mask (`cornerMasks`, owned by the `Ctx`, keyed by
  radius, built by filling a `2r` square with rounded corners, so a corner is the same pixels the
  anti-aliased fill gave) is dropped by `SetScale` and `SetTheme`, and when `maskLimit` (64) masks
  are held. `Ctx.OutlineStyled` draws a rounded border the same way: `ringQuads` (pure) cuts it into
  four corner squares sampling a quarter of the ring mask and four edge strips `LineWidth` thick
  (the solid texel), in one `DrawTriangles` with the color as vertex color; the ring mask is in the
  same cache, keyed by radius and line width, built by stroking the rounded path in a `2r` square
  (so a corner is the pixels the stroked outline gave); the outer edge snaps like `Fill`'s, and a
  radius below one device pixel is still the square `Outline`. `Ctx.Stats` (a `*core.DrawStats`, nil
  in the app) counts mask fills, ring fills and
  `Width` calls per frame; `BenchmarkDrawStats` in `internal/workspace/ui/app` reports them.
-  **Caret blink** (`core.Ctx.CaretVisible`, `RestartCaret`): the block caret is shown for 500 ms
  and hidden for 500 ms by the clock (`Ctx.Now`, the app's `now`), not by ticks, so the rate does
  not change when the update rate drops from 60 to 30 when idle. Typing, editing keys, clicks and
  clipboard keys restart the phase so the caret is solid while editing; a moving pointer does not.
  The 10 FPS drawing floor (`engine.MinFPS`) guarantees a frame at least every 100 ms, so a phase
  change is drawn within that.
-  **Monospace columns** (`core.Ctx.Advance`, `monoGrid.colX/colAt`, shared by `codeGeometry` and `areaGeometry`):
  the code view and the text area place column `n` at `round(n * advance)` with the exact advance of
  one character, and draw each span at the column it starts in. `Width` is rounded to whole pixels,
  so `n * Width("0")` would let the caret, selection, matches and clicks drift a fraction of a pixel
  per character away from the text drawn by whole-string width (several characters at column 100).
