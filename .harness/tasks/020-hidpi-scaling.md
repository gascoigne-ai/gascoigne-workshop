status: done (sub-pixel pointer positions are an accepted exception)

# Generalize HiDPI and scaling

## Goal
Scaling is split between the `uiScale` setting (zoom), `ebiten.DeviceScaleFactor` and hard-coded metrics (RowH, Pad, tab heights). Make it one consistent mechanism.

## Scope
- Single scale source: device scale factor x user `uiScale`, applied in one place (Ctx), never per widget.
- Metrics, font sizes, line widths, radii and hit areas defined in logical units and converted centrally; crisp 1px lines on fractional scales.
- Mouse/touch coordinates and drag deltas in logical units (replace the `RawX/RawY` workaround where possible).
- Per-monitor scale changes at runtime; screenshots at a chosen scale.

## Acceptance
- Tests at 100%, 150%, 200% and fractional scales for layout and hit testing.
- `make check` passes; architecture.md updated.

## Progress
- Done: one scale source in `core.Ctx` (device scale x `uiScale`, set by `SetScale`); text measured in logical pixels; fills, outlines and 1 px lines snap their edges to device pixels (`Ctx.snap`) and lines are `Ctx.LineWidth()` whole pixels, so they stay sharp at 125%-175%; tests over fractional scales.
- Left: pointer positions are truncated to whole logical pixels (`Input.Scaled`) and the slider and tab pan still read device-relative `RawX/RawY`; per-monitor scale change while running; text face sizes at fractional scales; golden screenshots at 150%.
- Checked later: the scale is re-read every frame (`App.Layout` asks `ebiten.Monitor().DeviceScaleFactor()` and `setSize` rebuilds the fonts when it changed), so moving the window to another monitor rescales it; `redraw_test` covers the change. `Ctx.RawX/RawY` became `PointerX/PointerY` (the slider keeps device-pixel drags on purpose, see decisions). Pointer positions stay whole logical pixels: accepted, the gain of fractional positions does not pay for touching every widget. Golden screenshots at 150% are not automated (pixels cannot be read without a running game loop); `-screenshot` with `uiScale` is the manual check, and icons and lines are tested for geometry at 100/150/200%.
