## Toolkit: widgets and drawing

- **Sideways strips.** `widget.HScroll` shows a row at natural width in a clipped box, with an
  arrow only at an end that can still scroll, the wheel, a touchpad swipe (`Input.WheelX`) and
  `KeepInView()` for one child. A press dragged a few pixels pans it, following `Ctx.PointerX`
  outside the strip. So the request strip (`requesttabs.go`, `App.tabsScroll`) reorders by left
  drag only while it fits (`ScrollState.Overflowing`); the middle button always reorders. The pane
  tab strips (`ScrollTabs`) use it too.
- **Tooltips.** `view.Tooltip(text)` on any element; `view.Draw` records the innermost hovered tip
  in `Ctx.TipText`, and `widget.TooltipLayer`, mounted last in the window's Stack, shows it after a
  rest. While a tip waits the app keeps drawing (`TooltipState.Waiting`). Buttons and tabs show
  their shortcut in the tip (`App.hint`).
- **Icons are SVG files** (`widget/icons/*.svg`; the subset is in `svg.go`). `widget.Icon(name)`
  draws one as vector paths in the element's `Color`; `RegisterIcons(fs)` adds or replaces icons
  from a folder. Glyphs inside widgets are icons too: the checkbox check, the fold arrows, a
  `Span.Icon` in a code view. The icons are the toolkit's own; icons from an open set (Lucide is
  ISC, Tabler is MIT) go in with their license file. One-color icons are cached as white masks and
  tinted (`iconMask`). A theme's `iconStroke` scales every stroke, so icon weight is a theme choice.
- **Sharp edges.** `Ctx.Fill`, `Outline`, `HLine` and `VLine` snap each logical edge to device
  pixels (`Ctx.snap`), so neighbours share an edge, and draw lines `Ctx.LineWidth()` thick. Rounded
  fills and outlines (`FillStyled`, `OutlineStyled`) snap the same way and draw from cached corner
  masks in one `DrawTriangles` (`core/cornermask.go`). `Ctx.Stats` counts mask builds and `Width`
  calls; `BenchmarkDrawStats` in `internal/workspace/ui/app` reports them.
- **Caret blink** follows the clock (`Ctx.CaretVisible`), not ticks, so its rate holds when the idle
  app drops from 60 to 30 updates. `engine.MinFPS` (10) draws at least every 100 ms, so a phase
  change shows.
- **Monospace columns.** `CodeView` and `TextArea` share `monoGrid`, which places columns by the
  exact advance so the caret, selection, matches and clicks stay on the text.
