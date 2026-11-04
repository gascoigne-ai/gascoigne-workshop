status: todo

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
