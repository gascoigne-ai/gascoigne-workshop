status: todo
kind: feature
commit: refactor(sidebar): the tree builds only the rows in view
scope: internal/ui/sidebar/**, internal/ui/window/**, internal/feature/collections/FEATURE.md, .harness/architecture/toolkit.md
not_before: 2026-01-01
depends_on:

# Sidebar builds only the rows in view

Goal: `sidebar/view.go` builds a template node for every visible tree row. Correction found by the Opus
design: the tree is built only on stale frames (`treeStale`, see "The frame builds once" in
`architecture/toolkit.md`), but `Draw` lays out again on every drawn frame, so layout is O(rows) per drawn
frame and build is O(rows) per stale frame. A windowed build must also rebuild when the scroll position
moves, because a wheel or scrollbar drag changes `ScrollState.Y` in `scrollB.Update` without a rebuild.
The designer could not read `endFrame`: confirm there when a `Tick`-set `treeStale` is rebuilt.

## Step 0: gate
Add `BenchmarkSidebarRows` in `internal/ui/sidebar/bench_test.go` with the package's fake `Deps` and a
headless `core.NewCtx`: one flat expanded collection, `rows=200` and `rows=5000`, reporting `build`
(`c.View(ctx)` including `VisibleRows()`) and `layout` (`view.Layout(ctx, root, 280, 900)`) separately,
with `b.ReportAllocs`. Gate: at 200 rows, build plus layout under about 0.83 ms (5% of a 16.6 ms frame)
means stop: change nothing, write the numbers here, and leave the decision to the user. Commit the
benchmark alone: `test(sidebar): benchmark tree build and layout by row count`.

## Model
No persistence; all on the UI goroutine.
- `logic.Window{First, Last int}` in `internal/ui/sidebar/logic`: half-open range of rows to build;
  `0 <= First <= Last <= n`. Derived, never stored as truth, from
  `logic.VisibleRange(scrollY, viewH, rowH, n)`: `First = clamp(scrollY/rowH - overscan, 0, max(0,
  n - ceil(viewH/rowH) - overscan))`, `Last = min(n, First + ceil(viewH/rowH) + 2*overscan)`. `overscan` is
  a named constant of 3 rows (a wheel step is 48 px plus one row). `rowH <= 0` or `viewH == 0` returns all
  rows (the old behaviour; tests and the first frame).
- `Component.built logic.Window`: the range the current tree was built with; the staleness key.
  `Tick()` returns `c.tree.Tick() || VisibleRange(c.scroll.Y, ...) != c.built`, ORed into `treeStale`.
- `Component.viewH`: window height as an upper bound of the scroll box, set by `SetViewH` from `setSize`.
  Rejected: an exported `ScrollState.ViewH` (toolkit change, wrong on the first frame and after a resize).
- Rows: keep `[]collection.Row` from `VisibleRows`, build `rows[First:Last]`. Access: count plus a
  contiguous range; hundreds of rows now, 5k-50k at 100x. Rejected: a map from node to built `Node`
  reused across builds (invalidation follows select, drag-over, filter, Alt). Revisit: `VisibleRows`/
  `FilteredRows` (a full walk, also called by `Keys` every focused frame) shows in a profile at 100x; then
  cache the row list on the expanded set, the filter and the collection revision.
- Spacers: two `<row h="{{padTop}}"/>` and `<row h="{{padBottom}}"/>` in `sidebarSource`, each with
  `if=`, so a tree that fits emits neither and the existing goldens stay byte-identical. Content height
  stays `n*rowH`, so the thumb and `clampScroll` are unchanged. Rejected: one tall spacer with an offset.
  Revisit: rows stop having one height; then a prefix sum of heights and a binary search.
- App-local, not a toolkit widget: one consumer. Promote to `widget` (rule 10) when a second long list
  (sent log, history) needs it. A toolkit `LazyList` would need a rebuild signal; bumping `Ctx.Fired` on
  every Y change would rebuild the whole window each wheel frame for every `Scroll`.

## What breaks with absent rows (read from code by the designer; recheck)
Only the scroll rebuild (handled by `Tick` plus overscan) and whole-window or e2e tests that find a row by
text beyond the first screen. Keyboard (`Keys` uses `VisibleRows()`), drag and drop (state in `logic.Tree`
by node), hover (off-clip rows are unhittable), filter highlight, tab focus and the context menu do not
need absent rows.

## Tests (test first)
1. `TestTreeRowsHaveOneHeightAndStack`: 50 rows, every row `H == rowH`, `Y == top + i*rowH` (pins the
   fixed-height assumption; passes now).
2. `TestVisibleRange`, table-driven, stub returning `Window{}` first: empty, fits, top, middle with
   overscan, bottom, `scrollY` past the content, `scrollY` not a multiple of `rowH`, `viewH=0`, `rowH<=0`.
3. `TestSidebarBuildsOnlyRowsInView`: 5000 rows, Y in the middle; the dump holds about `viewH/rowH + 6`
   rows plus two spacers; laid-out content height `5000*rowH`.
4. `TestSidebarSmallTreeHasNoSpacers`: existing goldens unchanged, never regenerated.
5. `TestSidebarTickAsksRebuildWhenRangeMoves`: same rows false; past a row boundary true; a collapse that
   shrinks n true.
6. `window/frametree_test.go`: add frames that wheel the long tree down and drag the scrollbar; the painted
   tree dumps the same as a fresh build. Fails before the `Tick` check exists.
7. Rerun `BenchmarkSidebarRows`; 5000 rows should be near the 200-row numbers. Numbers in the commit.

## Docs (same commit)
`architecture/toolkit.md` "The frame builds once": a view that windows a long list reports a scroll-range
change from its `Tick`, because `Draw` re-lays out without a rebuild. `internal/feature/collections/
FEATURE.md`: the tree builds only the rows in view plus overscan. Why, rejected alternative and revisit
signal in the doc comments of `VisibleRange`, `built` and `viewH`.

## Skip
Scrolling the selection into view on keyboard or `Reveal` (an existing gap, separate task), caching
`VisibleRows`, a generic `widget` virtual list, windowing the sent log, drag auto-scroll.

## Acceptance
- Test first, failing for the stated reason; say so in the report.
- `make check` passes; run the whole-window and e2e suites and report any test that addressed a row beyond
  the first screen.
