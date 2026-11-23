status: todo
kind: feature
commit: refactor(widget): CodeView draws a large body with fewer allocations
scope: uitoolkit/widget/**, uitoolkit/core/**, .harness/architecture/toolkit.md
not_before: 2026-01-01
depends_on:

# CodeView draw cost on a large body

Goal: `BenchmarkCodeViewDraw` (40k lines) still costs about 1.3 ms and 11.7k allocs (2.2 MB) per call
after the layout cache. The design is by an Opus task-designer that could not open the Ebitengine source,
so every claim about its internals is an inference: step 0 measures first and decides steps 4 and 5.

## Model
All new state is derived, owned by the UI goroutine and never persisted.
- `digits int` in `textCache` (`codeview_cache.go`): the digit count of the widest line number. Access:
  read 4-8 times a frame, written when the lines change. Same `linesKey` invalidation as `runes`.
  Rejected: recomputing per call (a loop over all lines, 4 times per Draw), a separate cache with its own
  key. Revisit: a caller edits `Line.Num` in place without `SetLines`.
- `advances map[faceKey]float64` on `core.Ctx`, keyed like `widthKey`, cleared where `widths` is
  (`SetScale`, `SetTheme`, `SetFonts`). Access: read 4-8 times a frame by every mono widget. Rejected:
  one `monoAdvance` field (breaks `Advance(face)` for other faces); measuring per call. Revisit: more than
  about 16 faces live at once.
- `rowBuf []Span` scratch on `CodeView`: overwritten per row, read once by `drawSpans`. Rejected:
  `sync.Pool` (one goroutine), allocating per row. Nothing keeps it after `Draw` returns.
- `textOps text.DrawOptions` scratch (and `glyphs []text.Glyph` if step 5 lands) on `core.Ctx`, fully
  reset per pass. Rejected: a pointer per call.

## Steps (test first)
0. Gate, no code: run `go test ./uitoolkit/widget -run '^$' -bench CodeViewDraw -benchmem -memprofile
   mem.out -cpuprofile cpu.out`, then `go tool pprof -sample_index=alloc_objects -top` and `alloc_space`.
   Write the top 10 into this file. If over 80% sits under `text.Draw`/`text.AppendGlyphs`, steps 4 and 5
   go ahead; else re-rank from the profile.
1. One geometry per call, digits cached. `contentWidth`, `bars`, `clampScroll`, `updateBars` take the
   `codeGeometry` already built (today a Draw builds it 4 times, each build scans all lines in
   `numberDigits` and measures "0"). First pin `TestCodeViewNumberDigitsFollowsWidestNumber` (by position,
   by `Num`, `Lines` reassigned without `SetLines`; passes now). Failing:
   `TestCodeViewCountsLineNumberDigitsOncePerContent` (counter like `layoutBuilds`: 1 after three
   Update+Draw, 2 after `SetLines`). Grep the callers of `CodeView` in `internal/` for in-place `Line.Num`
   edits first.
2. `Ctx.Advance` cached. Failing: `TestAdvanceIsMeasuredOncePerFont` (a `Stats.Advances` counter or
   AllocsPerRun 0 after the first call), `TestAdvanceFollowsScale`.
3. Row pieces without rune slices. Replace `cutSpans` with `appendRowSpans(dst, spans, from, to)` that
   slices `Span.Text` by byte offset; `monoGrid.drawSpans` walks the string, not `[]rune(sp.Text)`. Pin
   `TestRowSpansCutsByRune` first (ASCII, `"żółw"`, a cut inside the 2nd span, an icon span, empty and
   whole ranges). Failing: `TestRowSpansAllocateNothingIntoABuffer`, `TestSpanStringCountsLikeRunes`
   (tabs, multibyte, against `columns.span`). Keep `TextArea` compiling; it shares `monoGrid`/`columns`.
4. `Ctx.Text` without per-pass allocations: `textPasses` returns `[2]float32`; a pure `textOptions(...)`
   resets `GeoM` and `ColorScale` before scaling (`ScaleWithColor` multiplies). Failing:
   `TestTextOptionsDoNotCarryOverBetweenCalls`, `TestTextOptionsAllocateNothing`. No alloc test on `Text`
   itself (it counts Ebitengine internals).
5. Only if step 0 gates it: shape once, draw twice. `text.AppendGlyphs` once into `c.glyphs[:0]`, draw the
   same glyphs in both stem-boost passes. The two passes stay (a documented look decision). Read the
   `AppendGlyphs` doc for glyph image lifetime and placement first. Failing: a pure
   `glyphOps(glyph, origin, scale)` table test against the translation `text.Draw` uses. Look check:
   `make shot` of a response body before and after, pixel-identical; say that it was run.
6. Rerun `BenchmarkCodeViewDraw`; before and after in the commit body. `make check`.

AllocsPerRun tests go in a `//go:build !race` file (`make check` runs `-race`).

## Skip
Hoisting `selRange` (no allocs), `shownSpans` for folded rows, `strconv.Itoa` per row, `drawMatches`
over all matches (needs a `matches` sub-bench first), a single-pass or shader stem boost (look decision).
A raster cache of drawn rows is the biggest possible win but a real design fork: the user decides after
step 5's numbers.

## Acceptance
- Test first for every step, failing for the stated reason, then the smallest code; say so in the report.
- Docs in the same commits: `.harness/architecture/toolkit.md` (CodeView layout cache, Width cache and a
  Text drawing line), doc comments on `codeview_cache.go`, `Ctx.Text`, `textPasses`.
- `make check` passes.
