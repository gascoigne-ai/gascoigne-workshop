## view: the view DSL

A view is a Go function from state to a tree of elements, so the tree is a pure function of the
state: no diffing and no reactive system. Elements and options are indexed in the `view` and
`widget` package docs; naming is `rules.md` (Toolkit API rules). State a widget edits
(`ScrollState`, `SplitState`, `TextField`...) is owned by the app, so it survives a rebuild.

### The frame builds once

- `App.endFrame` builds the tree, lays it out, runs `view.Update` and keeps it (`drawTree`).
  `Draw` lays it out again (drags and scrolls move) and paints it. It builds again only when
  `treeStale` is set; `setSize` drops `drawTree`.
- `treeStale` comes from explicit signals, not a revision counter, because one missed bump would
  show old content. `Ctx.Fired` counts the handlers that ran: `view` for its `On*` options, a
  widget where it calls a handler or changes state the app reads at build (a column width). Each
  step after `Update` that changes what the view shows sets the flag itself (listed in `endFrame`).
- `frametree_test.go` (app) runs a script of frames and asserts after each that the painted tree
  dumps the same as a fresh build. A path that fails it must set the flag; the test is never
  relaxed.

### Runtime rules

- **Events** (`view_test.go`): a left click reaches every `OnClick` under the pointer, innermost
  first, up through its ancestors; a right click only the innermost `OnRightClick`; the wheel the
  innermost scroller that uses it.
- **Clipping hides the pointer.** While the pointer is outside a clipping element (`Clip`,
  `Scroll`), its children cannot be hit; the click flag still passes, so fields can blur.
- **Modal owns the pointer.** While a `Modal` element is visible, nothing under it gets clicks,
  wheel or hover (`Ctx.HoverOff`), and a click outside runs `OnDismiss`. Toasts are not modal.
- **Layout writes no state** (`view.Layout` doc, `widget/layoutpure_test.go`); `Update` settles it.
- **Drawing culls** off-image children (`view.Draw` doc, `BenchmarkLongList`), so a `Behavior`
  keeps no state that only its `Draw` writes.
- **Measure once per pass**: an element is measured once per width in one `Layout` pass
  (`Ctx.LayoutGen`, `Ctx.Laying`).
- **Layout is a flexbox subset** (`flex.go`). When a row overflows, `Optional` children drop
  first, then `Shrink` children give up the rest, first to last, down to their minimum; a row that
  fits lays out as if none had it (why explicit: the `Shrink` doc).
- **Cursor.** Widgets ask for a shape in `Ctx.Cursor`, reset every frame and applied by the app.
  `App.showGrab` asks for `core.CursorMove` while a row, tab or tree row is dragged, unless a
  resize shape was asked for.
- **Focus.** Tab cycles inside the topmost `FocusScope`; `revealFocused` scrolls every `Scroll`
  holding the newly focused field so the field is in view.

### App work off the frame

App facts kept here for now; they belong with the app.

- **Background writes** (`app/asyncwrite.go`, `workspace/controller/serial`): autosave clones what
  it writes on the UI goroutine and runs the write as a Cmd. `serial.Queue` runs one write per file
  and keeps only the newest waiting one, so an older write never lands after a newer one; the
  outcome returns on the bus. Files the user names (saved body, Postman export, screenshot) go the
  same way through `port.Folders.Write`. `FlushWorkspace`, `SaveState` and `openWorkspaceAt` call
  `waitWrites`, then write synchronously. The App does no file I/O of its own
  (`TestTheAppDoesNoFileAccessOfItsOwn`; why one port: the `port.Folders` doc).
- **Socket sends** (`wsui.go`): messages and pings go through `App.sendInOrder`, a per-tab outbox
  run by a Cmd, so a slow network never stops a frame and messages keep their order.
- **Change keys** (`changekeys.go`): autosave and the window title compare JSON forms. `changeKeys`
  caches them and marks them stale only on input, a bus message, a toast change or a geometry
  change. Save, load and flush use `collectionKeyNow`; a test that edits the model directly calls
  `markStale`.
