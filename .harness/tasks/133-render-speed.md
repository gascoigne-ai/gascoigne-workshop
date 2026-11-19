status: todo
kind: feature
commit: perf(render): rounded fills and outlines batch, the frame builds once
scope: uitoolkit/*,internal/workspace/ui/app/*,internal/settings/*,Makefile,.harness/architecture.md,.harness/style.md

# Speed up rendering

Origin: a profiling review of 2026-11-19 (temporary benchmarks, nothing committed) of a drag frame with the settings dialog open at 1280x800: about 2.2 ms CPU per frame, `a.view()` 290 µs, `view.Layout` 121 µs, `view.Draw` about 1.4 ms, and 42 anti-aliased gradient fills, 37 rounded outlines and 716 `Width` calls. The GPU cost is not in these numbers, so the first item measures it. The look must not change: golden files stay as they are, and a `make shot` PNG is looked at after every item that touches drawing.

Measure before and after each item and write the numbers into the commit body: draw commands per frame (`go run -tags ebitenginedebug ./cmd/workshop`), frame time during a dialog drag, and for the CPU items a benchmark checked in next to the code. A frame that gets slower does not ship.

## Scope
Work in this order, one commit per item.

1. **Baseline.** Add the benchmarks (`view()` build, `Layout`, `Draw` with a dialog open) and write the baseline numbers into this file. Nothing else changes.
2. **Rounded fills stop using the unbatched anti-aliased path.** `uitoolkit/core/ctx.go:401` (`DrawTriangles` with `AntiAlias`), reached from `FillStyled` and `FillStyledCorners`, makes Ebitengine clear and blend a full-window offscreen per fill. Step a: draw each fill into a sub-image of its rect plus one pixel, so the offscreen shrinks. Step b: a cached corner mask per (radius, scale) drawn as a 9-slice with vertex colors for the gradient, as `widget/icon.go` does with `iconMask`. The mask geometry is pure and unit-tested; the cache has a bound.
3. **Rounded outlines use the same mask cache** (a ring), instead of `vector.StrokePath` per element (`ctx.go:480`, called at `view/runtime.go:347`), which forces a vector flush each.
4. **`settings.Changed` computes the defaults map once** and the current map once per `View`, not two JSON round trips per row (`settingsui/form.go:266`, `settings/schema.go:277`).
5. **`Draw` reuses the tree `Update` built** (`app.go:650` and `:673`) and only runs `Layout` again, unless a handler changed state this tick (a flag set where state changes; a test that a changed state still shows).
6. **Layout measures a text once per pass**: memoize `(maxW) -> (w, h)` on `El` for one `Layout` call (`view/layout.go:109,138`). `Ellipsize` (`ctx.go:310`) searches by halves, as `CursorAt` does.
7. **Templates do less per build** (`markup/component.go`): attributes without `{{` become options at `Parse`, paths are split once, the element spec is looked up once per node.
8. **Children outside the active clip are skipped** (`view/runtime.go:339`); a shadow still draws when its rect reaches the clip.

Not in this task: caching the panes in an offscreen image during a drag. After items 2 to 5 the frame is cheap, and a cache can show stale content (a response arriving, the caret, a toast). Reopen it only if a measurement still shows a slow drag.

## Model
No persisted data and no new state besides two caches: the corner mask cache, keyed by (radius, scale) and bounded, cleared when the scale changes; the per-pass text measure on `El`, dropped at the start of each `Layout`. Alternative rejected: one offscreen image per pane, because it has to know every cause of a change. Revisit if a drag frame is still over budget after item 5.

## Acceptance
- `make check` green; the benchmarks from item 1 show the before and after of every item in its commit body.
- Draw commands per drag frame go down after items 2 and 3 (numbers in the commits).
- `make shot` PNGs of the main window and the settings dialog look the same as before.
- `architecture.md` describes the mask cache and the one-build frame; `style.md` is updated only if a corner or outline pixel changes.
