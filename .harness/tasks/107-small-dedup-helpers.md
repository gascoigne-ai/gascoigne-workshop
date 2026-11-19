status: todo
kind: refactor
commit: refactor(core): one hex parser, one color mix, stdlib map helpers
scope: uitoolkit/*,internal/*
depends_on: 106-spacing-tokens

# Share the small duplicated helpers

Origin: the repo audit of 2026-11-19 (three read-only reviews against the harness rules). The user approved all phases and, with that, the deletions and the gate change named here. Behavior stays the same unless the task says otherwise; tests stay green and none is loosened. Update `architecture.md` in the same commit where it describes what changes.

- One hex-color parser in `core` (alpha form and a strict flag) replacing `core/color.go`, `theming/themefile.go`, `widget/svgraster.go`; one `core.Mix(a, b, t)` replacing the four lerp closures; delete `roundedPath` in favor of `cornersPath`; one device-rect helper in `core/ctx.go`.
- `slices.Sorted(maps.Keys(m))` for the nine collect-and-sort copies; `maps.Clone` / `maps.Copy` for the hand-written map copies.
- One exported `with` helper replacing the three copies (`view/layout.go`, `widget/button.go`, `markup/component.go`).
- `wordRange` uses `wordAt`; one `folded(i)` and one `spanText(spans)` in `codeview.go`.
- `HScroll`: one `clamp` helper for the six clamps; merge the two identical arms of `Update`; split `PreUpdate` into arrows, pan and hold.
- `ScrollState`/`scrollB` reuse `layoutBars` and `barDrag` from `bars.go` instead of a second press implementation.
- Section-comment dividers removed by splitting (`textfield.go` into state, history, edit, lines).
Acceptance: behavior and goldens unchanged; `make check` green.
