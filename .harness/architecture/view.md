## view: the view DSL

The layers are `core` (primitives), `view` (this section: the element tree, layout, runtime and the
extension point) and `widget` (every component, built on `view.Custom` or on other elements);
decision 3 records how they were drawn. The text inputs, scrolling containers, `Table`, icons and
tooltips named below live in `widget` and are used through the same dot-import.

A view is a Go function from state to a tree of nodes, rebuilt every frame, so the tree is always a
pure function of the state: no diffing and no reactive system.

**The frame builds the view once.** `endFrame` builds the tree, lays it out and runs `view.Update` on
it, and keeps it as `App.drawTree`; `Draw` paints that tree (running `Layout` again, which moves
dragged and scrolled parts) and builds a fresh one only when `App.treeStale` says that something it
shows changed after it was built: an idle frame and a drag frame build once, a frame in which a
handler ran build twice, as before. The flag is set by explicit signals, not by a revision counter
(one missed bump would show old content):

- `Ctx.Fired` counts the handlers that ran: `view` adds one at each `OnClick`, `OnDoubleClick`,
  `OnMiddleClick`, `OnDragOver`, `OnRightClick` and `OnDismiss` call, and a widget adds one where it
  calls a handler itself (`Field` change, `OnEnter`, `OnTab`, `Area`, `Checkbox`, `Slider`) or changes
  state that the app reads when it builds (the width of a table column, the size of a `Resizable`).
  `endFrame` compares the count before and after `view.Update`.
- The steps after `Update` that change what the view shows set the flag themselves: `commitEdit`
  (writes the cell), `closeBulk` (and so `settleBulk`), `dragHover` (opens a folder),
  `saveCollection` (marks the tabs saved, so `workspaceTick`), `App.Send` (a settings message),
  `App.toast`, and the settings `Poll` (an outside change of `settings.json`; the frame that polls is
  always stale). The others do not change the tree: `showGrab` sets the cursor, `stateTick` snapshots
  the state and queues a write whose result comes back through the bus before the next build, and
  `shotUpdate` runs only in a screenshot run, where `Draw` builds fresh every time.
- `setSize` (a new window size or scale) drops `drawTree`; the next `Draw` builds for the new size.

Anything that changes state before the build of the next frame (the bus drain, input keys, settings
loaded) is covered by that build. `frametree_test.go` pins it: a script of frames (tab switch, filter
typing, menus, cell edit, folder drag, dividers, column drag, toast expiry, a response on the bus, the
settings dialog dragged, resized and clicked, window resize) asserts after every step that the tree
`Draw` would paint dumps the same as a fresh build. A path that fails it must set the flag; the test
is never relaxed.

Every constructor takes **attributes first, children after**, so what an element is reads before
what it contains. App view files dot-import `view` (and alias `core` as `w`) so views read like
markup:

```go
Row(H(urlH), PadX(w.Pad), PadY(8), Gap(8), Bg(w.Bg),
	Row(Grow(1), Bg(w.Sunken), Border(w.Border),
		Row(W(64), OnClick(cycleProtocol), Text(protocol, TextCenter(), Grow(1), Color(w.Muted))),
		VRuleInset(8),
		Field(&t.url, Grow(1), Placeholder("Enter URL"), OnEnter(send)),
	),
	PrimaryButton("Send", W(80), OnClick(send)),
)
```

-  **Items**: a constructor's arguments are `Item`s: a child `Node`, or an `Option` that sets an
  attribute. A nil node (a false `If`) adds nothing; options given later override earlier ones,
  which is how components such as `Button` let callers override their defaults.
-  **Elements** (`*El`, aliased `Node`): `Row`, `Col`, `Stack`, `Spacer`, `HRule`, `VRule` and
  `VRuleInset` (one-pixel lines in `Border`, the last shortened at both ends, each taking options
  like the rest), `Text`, `Caption`, `Lines`, `Field`, `TextArea` (`Area`), `Code`, icons
  (`Checkbox`, `Cross`, `Triangle`, `Dots`) and the widgets of `uitoolkit/widget` (`Button`,
  `PrimaryButton`, `SegmentChip` (a primary button when selected, else a plain one), `ListRow` (selection
  background or hover tint), `FormRow` and `CheckRow` (label column and checkbox rows), `ActiveTab`
  (the open tab's look, shared by `Tabs` and the request strip), `Menu`, `Dialog`, `ResizableDialog`
  and `DialogTitle` (dimmed scrim, centered modal panel, title bar with a close button that runs
  `OnDismiss`), `Toolbar` and `TabBar` (bars over a pane, sized by `core.ToolbarH`, `core.TabBarH`,
  `core.TabGap`), `Note` (one line of small colored text under a control), `Tabs`) and `Table`,
  which are ordinary compositions of the primitives. Behaviors are values of one small interface
  (measure, layout, draw, update), not switch cases.
-  **Options**: size (`W`, `H`, `Grow`, `Shrink`, `Optional`), spacing (`Pad`, `PadX`, `PadY`,
  `PadL`, `PadT`, `PadR`, `PadB`, `Gap`; one rule for names: a side is a letter suffix, an axis is X
  or Y; templates spell the same options in kebab-case, `pad-x`), look (`Bg`, `Border`, `BorderT`,
  `BorderB`, `HoverBg`, `HoverColor` (an icon under the pointer), `Lighten`, `Shadow`, `Clip`), text
  (`Color`, `Muted`, `Small`, `Mono`, `Ellipsis`, `TextCenter`), events (`OnClick`, `OnChange`,
  `OnEnter`), overlays (`FloatAt`, `FloatCenter`, `FloatTR`, `Modal`).
-  **Vue-like control flow**: `If` (v-if), `For` (v-for), `Frag` (fragment). `Field(&textField)`
  edits state in place like v-model.
-  **`Scroll(&state, ...)`**: a column taller than its box, moved by the wheel and a draggable
  scrollbar thumb (a click in the track pages). The app owns the `ScrollState` so the position
  survives the per-frame rebuild. A clipping element (`Clip`, `Scroll`) masks the pointer position
  for its children while the pointer is outside it, so rows scrolled out of view cannot be hit; the
  click flag still passes so fields can blur. A scroller that uses the wheel zeroes it, so the
  innermost scroller wins. Drawing skips a child whose device rect, grown by the shadow offset, a
  `drawOverflow` of 4 px and one pixel, does not meet the bounds of the image it draws into (the
  active clip, or the screen), so a long scrolled list costs the rows in view (`reaches` in
  `view/reach.go`; 2000 rows draw 65 elements, `BenchmarkLongList`). A skipped child runs neither
  its `Draw` nor its `DrawOver` and sets no hover flag or tooltip text; a `Behavior` must not keep
  state that only its `Draw` writes and its next `Update` reads (the code view and text area clamp
  their scroll in `Update` too). The tree and the table and note panes use it; the code view scrolls
  itself.
-  **`Split(&state, default, first, second)`**: two children around a one-pixel divider that is
  dragged with the pointer (a few pixels either side grab it, and the drag keeps the offset where it
  was grabbed). The size is pixels, or a fraction of the container with `AsRatio`; `MinFirst` /
  `MinSecond` clamp it, the first child winning when the window is too small for both; double click
  resets to the default; `Vertical` stacks instead of side by side. While the pointer is over the
  divider or dragging it the element asks for the resize cursor (`Ctx.Cursor`, reset every frame and
  applied by the app). The app owns the `SplitState`, so it survives the per-frame rebuild and can
  be saved later. While a table row, a tab or a tree row (over another row) is held and moving, the
  app asks for the move cursor (`core.CursorMove`, Ebitengine has no closed-hand shape) unless a
  resize shape was already asked for (`App.showGrab`).
-  **Modal overlays own the pointer**: while any visible element is `Modal` (the settings dialog, a
  context menu, a dialog), nothing under it gets clicks, wheel or hover, a click outside dismisses
  it, and hover feedback (`HoverBg`, `Lighten`, scrollbar and icon highlights) is off for everything
  drawn outside it (`Ctx.HoverOff`, read through `Ctx.Hovered`); the overlay's own items still light
  up. Toasts are not modal, so they block nothing.
-  **Controls look**: a `Checkbox` is a 14 px square, an accent fill with a check mark when on and a
  plain outline (darker under the pointer) when off; the delete cross turns red under the pointer;
  table rows light up under the pointer; `LinkButton` is a quiet text button (accent text, no
  background) used for secondary actions such as Bulk edit, while `Button` / `PrimaryButton` are the
  filled chips. The filled buttons inset their label by `buttonPadX` (10 px) and shorten it with an
  ellipsis when a fixed width is too small; callers that only need a floor use `MinW`, so a longer
  label grows the button instead of spilling out.
-  **Background writes** (`internal/workspace/ui/app/asyncwrite.go`,
  `internal/workspace/controller/serial`, task 040): the autosave of the collection and of the UI
  state clones what it writes on the UI goroutine and `App.write` runs the write as a Cmd.
  `serial.Queue` allows one write per file at a time and keeps only the newest waiting one, so an
  older write never lands after a newer one; the outcome returns as a bus message. `FlushWorkspace`,
  `SaveState` (quitting) and `openWorkspaceAt` call `waitWrites` and then write synchronously. Tests
  that read the file after a save call `waitWrites`. The files the user names go the same way
  (`App.writeFile`, queued by path, through `port.Folders.Write`): a saved response body, a Postman
  export and the screenshot; the toast or the screenshot's end comes with the outcome. The App does
  no file I/O of its own: no non-test file of `workspace/ui/app` imports `os`
  (`TestTheAppDoesNoFileAccessOfItsOwn`, decision 24).
-  **Socket sends** (`wsui.go`, task 040): sending a composed message and pinging go through
  `App.sendInOrder`, a per-tab outbox run by a Cmd, so a slow network never stops a frame and
  messages keep their order; the composer clears when the message is queued and a failure is logged
  in the message log.
-  **Change keys** (`internal/workspace/ui/app/changekeys.go`, task 040): the autosave of the
  collection and of the state, and the window title, compare JSON forms. `changeKeys` caches them
  and the frame step marks them stale only when input, a bus message, a toast change or the window
  geometry changed; idle frames reuse them. Code that saves, loads or flushes uses
  `collectionKeyNow`. A test that edits the model directly calls `markStale`, since in the app an
  edit comes with input.
-  **Layout does not write state**: `view.Layout` places children for the `ScrollState` it finds
  (clamped locally) and leaves it alone; `view.Update` settles it first (`scrollB.update` clamps
  `Y`; `hscrollB.settle` sets `Overflowing`, clamps `X` and brings the `KeepInView` child into view
  when it changed). Test: `layoutpure_test.go`.
-  **Event rule** (tested in `view_test.go`): a left click reaches the `OnClick` of every element
  under the pointer, innermost first, up through its ancestors; a right click only the innermost
  `OnRightClick`; the wheel the innermost scroller that uses it. `OnClickAt(func(x, y int))` is
  `OnClick` with the pointer position, for a menu that opens at the click; templates (`onclick`)
  cannot take it, so `wsbar.go` and `grpcui.go` still read `a.ctx.In` for theirs.
-  **Overlay primitives**: `FloatAt(x, y)` pins a Stack child's top-left at a point, shifted left
  when it would stick out on the right and opened above the point when it would stick out at the
  bottom; `FloatCenter()` centers it; `FocusScope()` makes Tab cycle only inside that element (the
  topmost scope wins); `OnRightClick(f)` gives the pointer position; a `Modal` element swallows and
  dismisses on a right click outside it as well.
-  **Alignment, justification, bounds and margins** (`uitoolkit/view/flex.go`): `AlignItems` on a
  Row or Col (and `AlignSelf` on a child) places children across the axis: `AlignStart`,
  `AlignCenter`, `AlignEnd`, `AlignStretch`, `AlignBaseline` (text baselines line up, the line is
  centered in the row), or the default (stretch; a fixed cross size is centered in a Row, left in a
  Col). `JustifyContent` (`JustifyStart/Center/End/Between/Around`) spreads free main-axis space and
  does nothing when a child grows. `MinW/MaxW/MinH/MaxH` bound any element's size, including a
  growing child's share. `Margin`/`MarginX`/`MarginY` add space around a child that counts in its
  parent's size. Text hugs its line height when not stretched, which is what makes `AlignStart/End`
  and baselines work. Tabs, tree rows and menu rows use baseline alignment so a small badge sits on
  the same baseline as the name.
-  **Layout** is a flexbox subset. Along its axis a child has a fixed size, a `Grow` share of the
  free space, or its content size; across it, it stretches unless it has a fixed size. `Optional`
  children are dropped when the row is too narrow. When it still overflows, `Shrink` children give
  up the overflow, first to last, each down to its `MinW` (`MinH` in a column); a row that fits is
  laid out as if none had it (decision 14). `Stack` layers children; a `FloatTR(top, right)` child
  is pinned top-right and `Modal` makes it swallow clicks and report an outside click (the toolkit
  offers it; the app's overlays are centered dialogs and menus at a point).
-  **Measure memo**: within one `view.Layout` pass an element's behavior is measured once per
  `maxW` (a text once, whatever its room): `El.size` consults a three-entry memo valid for the pass
  (`Ctx.LayoutGen` advances at every `Layout`, so a reused tree is measured again each frame, and
  a size asked for outside a pass, `Ctx.Laying` false, is never remembered). Fixed-size and clamp
  logic run after the memo. `Ctx.Ellipsize` finds the longest fitting prefix by binary search on
  the rune count.
-  **Frame cycle**: `view.Layout`, then `view.Update` (input goes to the top-most element first and
  runs handlers) or `view.Draw`. `App.Update` and `App.Draw` each build the view, so what is drawn
  already reflects what the handlers just changed.
