## Toolkit: widgets and drawing

- **Sideways strips.** `widget.HScroll` shows a row at natural width in a clipped box, with an
  arrow only at an end that can still scroll, the wheel, a touchpad swipe (`Input.WheelX`) and
  `KeepInView()` for one child. A press dragged a few pixels pans it, following `Ctx.PointerX`
  outside the strip. So the request strip (`requesttabs.go`, `App.tabsScroll`) reorders by left
  drag only while it fits (`ScrollState.Overflowing`); the middle button always reorders. The pane
  tab strips (`ScrollTabs`) use it too.
- **Tooltips.** `view.Tooltip(text)` on any element; `view.Draw` records the innermost hovered tip
  in `Ctx.TipText`, and `widget.TooltipLayer`, mounted last in the window's Stack, shows it after a
  rest. While a tip waits the app keeps drawing (`TooltipState.Waiting`). Their look is in `style.md`.
- **Icons** (look and sources: `style.md`, Icons). The SVG subset is in `svg.go`; `RegisterIcons(fs)`
  adds or replaces icons from a folder; one-color icons are cached as white masks and tinted
  (`iconMask`).
- **Sharp edges.** `Ctx.Fill`, `Outline`, `HLine` and `VLine` snap each logical edge to device
  pixels (`Ctx.snap`), so neighbours share an edge, and draw lines `Ctx.LineWidth()` thick. Rounded
  fills and outlines (`FillStyled`, `OutlineStyled`) snap the same way and draw from cached corner
  masks in one `DrawTriangles` (`core/cornermask.go`). `Ctx.Stats` counts mask builds and `Width`
  calls; `BenchmarkDrawStats` in `internal/workspace/ui/app` reports them.
- **Caret blink** follows the clock (`Ctx.CaretVisible`), not ticks, so its rate holds when the idle
  app drops from 60 to 30 updates. `engine.MinFPS` (10) draws at least every 100 ms, so a phase
  change shows.
- **Monospace columns.** `CodeView` and `TextArea` share `monoGrid`, which places columns by the
  exact advance so the caret, selection, matches and clicks stay on the text. A tab advances to
  the next stop of `core.TabStop` columns along its line (`core.ColumnAfter`); the wrap layout keeps
  each row's start column, so drawing, the caret, selection, matches, clicks, wrapping and the
  horizontal extent all count it the same way. The text keeps the real tab (display only).
