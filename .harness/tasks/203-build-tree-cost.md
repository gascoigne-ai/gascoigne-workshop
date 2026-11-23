status: todo
kind: feature
commit: refactor(markup): building a template allocates less
scope: uitoolkit/markup/**, uitoolkit/view/**, uitoolkit/core/**, uitoolkit/widget/paragraph.go, internal/ui/sizes/**, internal/ui/window/render_bench_test.go, .harness/architecture/toolkit.md
not_before: 2026-01-01
depends_on:

# Cheaper window tree build

Goal: `BenchmarkBuildTree` is about 338 us, 894 KB and 5267 allocs per build of the settings-dialog scene,
and `BenchmarkIdleStep` (about 170 us) is mostly that build. The Opus design could not run grep or the
profiler, so the step order is a guess: step 0 measures and reorders steps 1-5 by measured bytes and
objects. Static options are already built once at Parse time (`node.prepare`/`attr.prepareOption`), so
"precompile attributes" is mostly done; what is left is below.

## Findings (from code; verify with the profile)
- `params.go`: `scoped[T].read` calls `lookup` -> `splitPath` -> `strings.Split` on every build; `interpolated`,
  `tokenParam`, `textParam` call `parseInterp` on every build. `architecture/toolkit.md` (Templates) says
  Parse does this; the code does not.
- `build.go`: each `for` item copies the whole scope (`child := Scope{}; maps.Copy(child, sc)`), usually the
  merged size scope of about 70 entries; `asList` adds a `[]any` per loop.
- `internal/ui/sizes/sizes.go` `Scope` builds a fresh 26-entry map (`core/sizes.go`), merges 38 `fixedSizes`
  and `own` into a new map on every component build.
- `parseInterp` keeps empty literals around each `{{path}}`, missing the single-literal fast path in
  `interp.eval`; non-strings go through `fmt.Fprint`.
- `view.flatten` allocates a slice per element with children even with no fragment; `newEl` grows `kids`
  one child at a time; `Custom` copies its items; markup `buildOne` grows `kids` and `opts` from nil and
  `elementParts.items()` copies them into a third slice.
- `El` is about 50 fields (700-900 B), probably most of the bytes; shrinking it is out of scope.

## Model
Nothing persisted; every structure is owned by the UI goroutine during a build, template fields are fixed
after Parse.
- Loop binding chain (unexported `binding{name; item any; index int; outer *binding}`): read by the first
  path step, innermost first, then the caller's scopes; written once per item; lives for one `build`. One
  binding per `for`, overwritten per item. Rejected: a scope copy per item (today); a slice of bindings.
  Revisit: a build output starts keeping a reference to the scope; then one binding per item.
- Fallback scopes: `Template.Build(sc Scope, fallbacks ...Scope)` (and `MustBuild`): read by key, own first,
  then each fallback in order, never written by markup. Rejected: caching the merged base map and cloning it
  per build; a magic `$base` key. Revisit: more than about 3 layers or a fallback that changes per item.
- Sizes scope memo on `core.Ctx` (`sizeScope struct{ of Sizes; names map[string]any }`, read via
  `Ctx.SizesScope()`): a one-entry memo checked against the `Sizes` value on each read, so a direct
  assignment to `Ctx.Sizes` cannot leave it stale. Rejected: rebuilding only in SetTheme/SetFonts.
  Revisit: sizes become per face or per pane.
- Pre-parsed params: `paramBinding` gets `parts interp` and `path []string` at Parse. Immutable after.
- Presized slices in `buildOne` and `newEl`: exact capacity. Rejected: `sync.Pool` (a built tree keeps its
  slices, so reuse would alias between frames).

Invariants: a build never writes into a scope it is given; shadowing is unchanged (a loop variable or
`$index` shadows its own scope, its own scope shadows the fallbacks, an inner `$index` shadows an outer
one); error texts stay byte-identical, an unknown name still reports its full path.

## Steps (test first)
0. Measure: `go test ./internal/ui/window -run '^$' -bench BuildTree -benchmem -memprofile mem.out`, then
   `go tool pprof -sample_index=alloc_space` and `alloc_objects`. Write the baseline here. Reorder steps
   1-5. Check a `view.Dump` golden of the settings-dialog scene exists; if not add one and see it pass.
1. `for` without a scope copy (`build.go`, `scope.go` `lookupPath`, `asList`). Pin first (pass today):
   `TestForHandlersKeepTheirOwnItem`, `TestNestedForIndexShadowsOuterIndex`,
   `TestLoopVariableShadowsScopeEntry`, error-text cases. Failing (`!race` file):
   `TestForCostDoesNotGrowWithScopeSize` (AllocsPerRun on a 50-item `for` with a 5-entry and a 200-entry
   scope must be equal). Thread an unexported build environment through `build`, `buildOne`, `options`,
   `args`, `bindHandler`, `interp.eval`, `param.read`; walk non-`[]any` slices with `reflect.Value.Index`.
2. Pre-parsed params (`params.go`, `parse.go` `bindParam`). Failing (`!race`): `TestParamReadAllocatesNoSplit`
   (a template whose only dynamic part is `<scroll state="s">` against the same with a literal param).
3. Fallback scopes plus the Ctx sizes memo. Land the signature as a stub that ignores fallbacks so the tests
   fail on assertions. Tests: `TestBuildReadsFallbackScopesInOrder` (table: own wins, first fallback beats a
   later one, a loop variable beats all, an unknown name gives the same error),
   `TestBuildLeavesItsScopesUnchanged`, `TestSizesScopeFollowsAnAssignedSizes` (core),
   `TestScopeOrderOwnOverFixedOverCtx` (sizes; pin before the change). Replace
   `tmpl.MustBuild(sizes.Scope(ctx, own))` and `a.scope(own)` with `tmpl.MustBuild(own, sizes.Fixed,
   ctx.SizesScope())` or a `sizes.Build` helper. Check every call site for writes into the returned scope.
4. El slices (`view/el.go` `flatten`, `newEl`; `custom.go`; markup `buildOne`, `elementParts`). Failing
   (`!race`): `TestBuildingAContainerAllocatesNoSpliceCopy`, `TestCustomAllocatesNoItemCopy`,
   `TestBuildOneDoesNotGrowSlices`. `flatten` returns its input when there is no fragment; `newEl` presizes;
   `Custom` sets `custom` before applying items; markup presizes `opts` and builds one `items` slice.
5. Interpolation fast path (`scope.go`). Pin `TestInterpFormatsLikeFprint` (int, bool, float, Token,
   Stringer against `fmt.Sprint`). Failing (`!race`): `TestSinglePathTextAllocatesNothingForAString`. Drop
   empty literals at Parse; a lone path returns a string as is, an `int` via `strconv.Itoa`, the rest fmt.
6. Only if a fresh profile of `BenchmarkDragFrame` still shows paragraph wrapping (not a build cost):
   `Paragraph` keeps `flow` (text with collapsed whitespace, once per build) and lines are substrings of it;
   `wrapParagraph` no longer allocates `line + " " + word`. Pin `TestParagraphWrapsLikeBefore` first
   (multiple spaces, newline, a word wider than the line, empty text); failing
   `TestWrapParagraphAllocatesOnlyItsLines`. Rejected: a wrap cache keyed by face, text and width (a second
   bounded cache). Revisit: wrapping still at 5% or more of DragFrame.

AllocsPerRun tests go in `//go:build !race` files; `make check` runs `-race`.

## Stop rule
After each step run `-count=10` and compare with benchstat. A step that improves `BenchmarkBuildTree`
ns/op by less than 5% (step 6: `BenchmarkDragFrame`) is not landed; stop the sequence and record the numbers
here. Every step: all `view.Dump` goldens and `TestTheDrawnTreeMatchesAFreshBuildAcrossASession` stay
byte-identical, no golden regenerated.

## Skip
Sharing a parse-time `[]Option` for all-static nodes (a race if `view.With` or a constructor appends into
it; revisit only with a `s[:n:n]` slice and a test that appends), a memo of dynamic closures on the template
(shared mutable state across parallel builds), shrinking `El` (its own task and design), caching reflect
`FieldByName`, removing the second build in Draw when `treeStale` is set (changes the contract), tree reuse
with `sync.Pool` (aliasing with `drawTree`).

## Docs (same commit as each step)
`architecture/toolkit.md`: Templates bullet (`for` binds without copying, params pre-parsed, fallback
scopes), a Model line for the Ctx memo beside the width cache. Package doc of `markup/parse.go` and the
`Build`/`MustBuild` comments. `core.Sizes`/`Ctx.SizesScope` doc with why, rejected alternative, revisit
signal. `internal/ui/sizes` `Scope` doc, or remove `Scope` when no caller is left. Check that
`uitoolkit/examples` still compiles and passes.

## Acceptance
- Test first, failing for the stated reason; say so in the report.
- `make check` passes.
