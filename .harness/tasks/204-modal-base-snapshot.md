status: todo
kind: feature
commit: refactor(window): draw the window under a dialog once, from a snapshot
scope: uitoolkit/view/**, uitoolkit/core/**, internal/ui/window/**, .harness/architecture/app.md, .harness/architecture/toolkit.md
not_before: 2026-01-01
depends_on:

# Snapshot of the window under a modal dialog

Goal: while a modal dialog is open (the settings dialog first), every dirty frame repaints the whole
window: moving the dialog, a caret blink or typing in it all pay for the base tree again (one whole-window
dirty flag, no layers; `BenchmarkDragFrame` is about 1.4 ms). Render the base (the base column plus the dim
scrim, never the dialog panel or its shadow) once into an offscreen image, and on later frames blit it and
draw only the layers above it. The user asked for this approach. The snapshot skips only the drawing: each
frame still builds the tree, lays it out twice and runs `Update`.

## Step 0: gain ceiling (stop rule)
Before any code, measure with a throwaway benchmark: `view.Draw` of each `root.Children()[L:]` (the part
that stays per frame) against `BenchmarkDrawTree`. Gain = (DrawTree - DrawTop) / `BenchmarkDragFrame`,
counting the `over.Tick` poll. Under 20%: stop, write the numbers and "no-go" here, change nothing.
Also list, as an implementer item, every `now()`, `Ctx.Now`, `time.Now`, `Tick` and `CaretVisible` read on
base build or draw paths and write the list here (clock-derived content such as elapsed "Sending..." text,
relative times, spinners): each either arrives on a detected signal, gets a new signal (for example
in-flight sends count as `stale` every frame), or stop and ask.

## Model
All state in memory, owned by `window.Component`, never persisted.

What is split. `Frag` flattens into its parent's children, so the window `<stack>` root has these direct
children: the base col, the settings scrim Row, the settings panel (`Modal()`), the font list, then picker,
browser, popup menu, dialog scrim and panel, call log, toasts and tooltips, each only while open.
- Snapshot: the root's own pre-children drawing plus `kids[0:L)`, where `L` is the index of the FIRST
  visible direct child that is `modal`. Drawn every frame: `kids[L:]` (the settings panel, font list, menus,
  the "Unsaved settings" confirm, call log, toasts, tooltips).
- Why the first modal: a confirm stacked over the settings panel would otherwise freeze the panel's hover
  and caret inside the snapshot. With the split at the first modal the snapshot holds no modal element, so
  nothing in it reacts to the pointer (`HoverOff` is on for all of it).

Toolkit (`uitoolkit/view`, no app knowledge):
- `view.ModalLayer(root Node) int`, pure: returns `L`, or -1 when no direct child is a visible modal (a
  modal nested deeper gives a full draw), or when the root draws after its children (`border`, the style
  border of its control, `edgeTok`, an `OverDrawer` behavior), because a split there changes paint order.
- `view.DrawBelow(ctx, root, layer, dst)`: resets `TipText`, sets `HoverOff = true`, draws the root's
  pre-children part and `kids[:layer]` with the same clip and cull as `El.draw`.
- `view.DrawFrom(ctx, root, layer, dst)`: resets `TipText`, sets `HoverOff = true` (the modal turns it off
  inside itself, as today), draws `kids[layer:]` into the root's clip.
- Invariant: `DrawBelow` then `DrawFrom` on one image equals `Draw`, element for element. `El.draw` splits
  into `drawSelf` (bg, hover bg, borders T/L/R/B, `b.Draw`), the children loop and `drawAfter`; `Draw` calls
  all three.
- `core.Ctx.CaretReads int` counts `CaretVisible` calls, so a drawing kept across frames knows whether it
  shows the blink phase. Rejected: deciding by focus owner (misses a Behavior that blinks without focus).

Window (`internal/ui/window/snapshot.go`):
- `snapshotPhase` enum `snapNone | snapFresh | snapStale`. Invariant: `image != nil` iff phase is not
  `snapNone` (instead of a valid bool beside a nil-able image).
- `snapshotGeometry` comparable struct: `size image.Point` from `screen.Bounds()` (not `w*scale`, so a
  fractional scale cannot mismatch), `scale float64`, `layer int`, `caret caretUse` (enum `caretUnused |
  caretShown | caretHidden`; `caretUnused` when `CaretReads` did not move while the snapshot was drawn).
- `frameSignals{drained, pruned, keys, fired, stale, resized, everyFrame}` and pure `changesBase(frameSignals)
  bool` (their OR). `keys` = input beyond pointer move or held: any click, wheel, rune, key or Tab; compute
  by clearing X, Y and Held and comparing with the zero `core.Input` if comparable, else list the fields plus
  a `reflect` field-count test that fails when one is added.
- `baseSnapshot{phase, image, geometry}` and pure `snapshotStep(phase, kept, current snapshotGeometry,
  hasModal bool) action` with `action` one of `reuse`, `redraw`, `reallocate`, `release`.

| from | event | to / action |
|---|---|---|
| none | draw, L >= 0 | allocate, draw below, fresh |
| fresh | `changesBase` true in any Step since the last Draw | stale (sticky: Ebitengine can run several Updates per Draw) |
| fresh | draw, geometry equal | reuse |
| fresh or stale | draw, size differs | deallocate, allocate, draw below, fresh |
| fresh | draw, scale, layer or caret differs | clear, draw below, fresh |
| stale | draw | clear, draw below, fresh |
| any | draw, L = -1 | `Deallocate`, none |
| any | `drawEveryFrame` (capture) | stale at every Step |

Collections: none added. The tree's children stay a slice read by position; `ModalLayer` is one linear
scan of about 10 root children per Draw. Rejected: a layer marker stored on elements (new `El` state where
position already says it). Revisit: a modal below a non-modal overlay that must stay live in the snapshot,
or a deeper modal that needs the split.

Every way the base can change while a modal is open, and what detects it:
1. Bus messages (send results, ws/SSE/gRPC/MCP events, `wsRetry` timers, file dialog answers, `prefs.Later`,
   write outcomes, the font scan): `drained`.
2. Handlers in `view.Update`, including live settings edits (theme, fonts, scale, layout, response mode):
   `Ctx.Fired` delta. `Send` also calls `Stale()`.
3. Key handlers in `a.Keys`, which run before the build (Enter on a dialog's primary button, `askSave` then
   revert or persist, shortcut recording): `keys`.
4. `Tick` steps after `Update` (`editor.Tick`, `resp.Tick`, `over.Tick`, which polls `settings.json` and
   always reports stale every `PollTicks` frames; `saveCollection`, toasts shown, explicit `Stale()`): the
   final `treeStale` of `endFrame` is `stale`.
5. Toast expiry: `pruned` (conservative; toasts are drawn every frame anyway).
6. Size, device scale, moving monitors (`setSize`): `resized` plus the geometry key.
7. Caret blink in a base field that kept focus: `caret` in the geometry, so the base is redrawn twice a
   second only in that case and the pixels match a full draw. Decided, not asked: if the user prefers to
   freeze it, that saves two base redraws a second and the oracle test must then skip the caret.
8. Clock-derived content built into the tree with no message: see the audit in step 0.
9. Hover and tooltips in the base: none; `HoverOff` freezes them under a modal. toolkit.md's rule that "a
   Behavior keeps no state that only its Draw writes" is what makes skipping the base draw safe.

Image lifecycle: `ebiten.NewImage(size)` (`Unmanaged: true` only if measured to help); on a size change
`Deallocate` then allocate (`Dispose` is deprecated in the pinned Ebitengine); when `L = -1` `Deallocate`
(closing fires a handler, so a dirty Draw follows). Memory is 4 bytes per device pixel only while a modal is
open: about 4 MB at 1280x800@1x, 16 MB at 2560x1600, 33 MB at 4K, 59 MB at 5K. Blit with identity GeoM,
nearest filter, default source-over: the base col (`bg="frame"`) is an opaque non-antialiased fill, so where
it covers, source-over equals a copy, and a device edge column the snap leaves uncovered keeps its old pixels
exactly as a full draw does today. Rejected: `BlendCopy` (it would make those uncovered pixels transparent);
the `make shot` check at fractional scale settles this.

Zones: `view` (`ModalLayer`, `DrawBelow`, `DrawFrom`, the `El.draw` split), `core` (`CaretReads`),
`internal/ui/window` (`snapshot.go`, calls in `step.go`, `redraw.go`, `endFrame`). No port is touched.
Toolkit rule 10: the new exported `view` functions get doc comments, tests and a line in the `view` package
doc index; no markup row (runtime calls, not options).

## Tests (test first; stubs so they fail on assertions)
1. `TestModalLayer` (view, table): no modal gives -1; a modal at index 2 gives 2; two modals give the first;
   a hidden modal is skipped; a modal nested in a child gives -1; a root with a border, `OverDrawer` or edge
   gives -1.
2. `TestDrawBelowAndDrawFromDrawWhatDrawDraws` (view): the `ctx.Stats.Elements` sum equals `Draw`;
   `DrawFrom` alone counts only `kids[L:]`; `TipText` is set only from the top part.
3. `TestCaretVisibleCountsReads` (core).
4. `TestChangesBase` (window, table over each signal; pointer move alone or held alone gives false) and
   `TestPointerOnlyInputCoversEveryField`.
5. `TestSnapshotStep` (window, table over the transition table), including `release` on `L = -1` and
   `reallocate` on a size change.
6. `TestTheWindowTreeSplitsAtTheSettingsPanel`: settings open, `ModalLayer` points at the panel, the scrim is
   `L-1` (pins how the markup slots flatten).
7. `TestADragUnderTheSettingsDialogDrawsOnlyTheDialog`: open settings, Draw once, Step a pointer move, Draw:
   `Elements` equals the per-frame part's count and is less than the full count.
8. `TestABaseChangeUnderTheDialogRedrawsTheBase` (table, settings open): drained send result, ws message, key
   frame (Enter on the confirm), live theme edit, resize, toast, caret phase flip with base focus. Each gives
   a full `Elements` count on the next Draw; also several Steps per Draw with the event in the first Step.
9. Oracle `TestTheSnapshotMatchesTheBaseAcrossASession` (mirrors
   `TestTheDrawnTreeMatchesAFreshBuildAcrossASession`): a script with the dialog open (drag, type, clicks in
   the dialog, a send result, a ws message, a toast, a live scale or theme edit, a resize, a fake-clock
   advance with a send in flight, a confirm over settings, Escape); after each frame, while the phase is
   fresh, the `view.Dump` of the base children equals the dump taken when the snapshot was drawn. Blind
   spot: Behavior internals the dump does not print; the `make shot` check covers those.
10. Benchmarks: `BenchmarkDragFrame` before and after, and `elements/frame` in `BenchmarkDrawStats`. Stop
    rule: under 20% gain on `BenchmarkDragFrame`: revert and record numbers here.
11. Visual: `make shot` of the settings scene at UI scale 1.0, 1.25 and 1.5, `cmp` against shots from the
    base commit: pixel-identical (capture sets `drawEveryFrame`, so the shots run the draw-below, blit and
    draw-from path; the reuse path's pixels follow from the oracle test). Existing golden scenes with a
    dialog open pass unchanged. Say whether the display was available; if not, say the visual check was not
    run. Smoke: drag the dialog in the running app at a fractional scale.

## Docs (same commit)
- `architecture/app.md` "Drawing only when something changed": the snapshot, what is in it, what is drawn
  every frame, the signals that make it stale, release when the modal closes.
- `architecture/toolkit.md` Runtime rules: `ModalLayer`, `DrawBelow`, `DrawFrom` beside "Modal owns the
  pointer", and the `CaretReads` note.
- Doc comment of `baseSnapshot` with the why and rejected alternatives: a damage rect on a persistent frame
  (`SetScreenClearedEveryFrame(false)` already keeps it) still walks the base, needs damage tracking per
  element, and the scrim's alpha needs the base under the old and new dialog rects; a cheaper full redraw
  keeps the cost growing with the size of the base; a fingerprint of the base tree cannot see a custom
  Behavior's state. Revisit when build and layout dominate the drag frame, when memory at 5K matters, or when
  a non-modal overlay wants the same treatment.

## Risks
Stale snapshot means wrong pixels, most likely from clock-derived build content (item 8); covered by the
oracle test, the per-signal tests, `everyFrame` for captures and the clock audit. Typing in a dialog is a key
frame, so the base is redrawn on each keystroke (no worse than today); only drag, resize and caret blink
reuse the snapshot. `SetScreenClearedEveryFrame(false)` keeps old pixels; the blit rewrites everything the
base covers.

## Acceptance
- Test first, failing for the stated reason; say so in the report.
- `make check` passes; step 0 and the benchmark gate decide whether the task lands.
