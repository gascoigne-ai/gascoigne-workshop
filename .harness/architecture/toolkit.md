## core: the toolkit

`core` holds the primitives (`rules.md`, Toolkit API rule 11). Everything except the drawing calls
is tested without a GPU.

- **Logical pixels.** Widgets, layout and hit tests work in logical pixels. `Ctx` drawing calls
  multiply by `Scale`, text measures divide by it, and `Input.Scaled` maps the pointer back.
  `SetScale` and `SetTheme` rebuild the fonts and drop cached masks. This keeps HiDPI sharp.
- **Theme and tokens**: the look, the tokens and the theme pipeline are in `style.md`.
- **Input.** `Sampler.Read` samples the frame's pointer, wheel, runes and edit keys and counts
  clicks. `modifiedEdit` is the per-platform table that turns modifiers into word and line edits.
- **Clipboard.** `SystemClipboard` runs the platform's clipboard tools: Ebitengine has no clipboard
  API, and no dependency is added for it.
- **Text editors live in `widget`** (`uitoolkit/doc.go`): `TextField` (one line, undo), `TextArea` (a
  `TextField` in multiline mode plus view state) and `CodeView` (read-only spans, folds, selection).
  They share the word rule (`isWordRune`), the scrollbar math and drag (`widget/scrollbar.go`,
  `barDrag`) and the monospace geometry (`monoGrid`), so the three agree on words, caret and clicks.

## desktop: the OS the window runs on

`uitoolkit/desktop` holds what a window needs from the OS and Ebitengine does not give: the Dock
icon (`SetDockIcon`), the display scale (`displayscale`), the key repeat timing (`keyrepeat`) and the
system file chooser (`filedialog`) and the system report the settings show (`systeminfo`) and the installed fonts (`fonts`). Each is split per OS (build tags or a table keyed by OS) and its
parsing is pure and tested without a display. It imports nothing of the toolkit or the app.

- **Why in the toolkit.** Every program with a window needs the same few things, and they are about the
  window and the desktop, not about requests. An `internal/platform` layer once held them
  behind ports owned by `settings` and `workspace`, but no feature code used those ports: only
  `app` and a view did, so the layer is gone. The ports live where they are used: `internal/app/ports.go`
  and `ui/overlays/settingsdialog/formpage` (fonts), next to the fakes of the tests.
- **Sub-packages.** `displayscale`, `keyrepeat`, `filedialog`, `systeminfo` and `fonts` each have their own `source`,
  `System` and `run`; one flat package would need prefixes on all of them.
- **Rejected.** A D-Bus file chooser portal (see the `filedialog` package doc). Revisit when a
  client library is already a dependency.

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

- **Background writes** (`feature/workspace/serial`, `serial.Writer`): autosave clones what
  it writes on the UI goroutine and runs the write as a Cmd. `serial.Writer` (its `Queue` is the rule) runs one write per file
  and keeps only the newest waiting one, so an older write never lands after a newer one; the
  outcome returns on the bus. Files the user names (saved body, Postman export, screenshot) go the
  same way through `workspace.Folders.Write`. `FlushWorkspace`, `SaveState` and `openWorkspaceAt` call
  `Writer.Wait`, then write synchronously. The App does no file I/O of its own
  (`TestTheAppDoesNoFileAccessOfItsOwn`; why one port: the `workspace.Folders` doc).
- **Socket sends** (`request_send.go`): messages and pings go through `App.sendInOrder`, a per-tab `outbox.Outbox`
  (`internal/feature/send/outbox`: the queue and the in-flight flag, `Enqueue` and `Done`) run by a Cmd, so a slow network never stops a frame and messages keep their order.
- **Change keys** (`workspace/autosave.Keys`, held by `App.keys`): autosave and the window title compare JSON forms.
  `Keys` caches them and marks them stale only on input, a bus message, a toast change or a geometry
  change. Save, load and flush use `CollectionNow`; a test that edits the model directly calls
  `keys.MarkStale`.

## Toolkit: API and templates

The package map is `uitoolkit/doc.go`; import and naming rules are `rules.md` (Layers, Toolkit API
rules). `uitoolkit/examples` are tested examples that use the exported API only.

- **Families stay whole.** An option family (`MinW`, `AlignStart`, `JustifyBetween`, `PadL`...) is
  exported even where the app uses part of it, since a half family breaks rule 1; helpers only the
  toolkit uses are unexported. Borders are one-pixel hairlines in a token; `EdgeT/L/R/B` are the
  separate two-pixel open-tab outline. The `Round` family is only the corner masks a panel needed
  (`RoundT`, `RoundB`, `RoundBL`; see Exceptions).
- **Mistakes fail loudly.** A component option on the wrong kind of element panics, naming the
  option and the element's control (`behaviorOf`). A template error names the file and line,
  suggests the nearest name and lists the valid ones.
- **Theme pipeline**: in `style.md`, Theming pipeline. `core`'s tests build a small `testTheme`,
  since `theming` imports `core` and never the reverse.
- **Templates** (`markup`; format and why in its package doc). The app writes each view as a
  template in a `const` at the end of its Go file, parsed once with `markup.Must`. A template has
  one root, so a view that shows one of two things is two templates and Go picks. Parse does all
  the work that does not depend on the scope (element specs, `for`, params, `{{path}}` splits, the
  options of static attributes, shared by every build); a build reads only handlers, `if`, `for`,
  params and `{{...}}` from the scope. A path reads map keys and exported fields only: no methods,
  comparisons or filters, so anything computed is a field of a view model. A wrong scope type, a
  nil state pointer or a widget-only option on another element is a parse or build error, never a
  panic at layout. `Load`/`MustLoad` (templates in an `fs.FS`) are for other programs.
- **Slots.** `<slot name="path"/>` puts a scope `view.Node` into a template: another template's
  tree or a `view.Custom` component, whose measure, layout and input run as in a Go tree
  (`custom_test.go`).
- **Dumps pin views.** `view.Dump` prints a laid-out tree with every non-zero attribute (one
  name-and-getter table, `dumpedAttributes`). Pinned dumps keep a view from changing, and tests
  compare a template with its Go form this way.

### Exceptions to the toolkit API rules

Each is also said in the doc comment of what breaks the rule; a new exception adds a row here.

- Rule 1: `Round` is not a per-side family but the corner masks a panel needed (`RoundT`, `RoundB`,
  `RoundBL`); a mask is added with the panel that needs it, not ahead.
- Rule 7: state types are values (`NewTextField`, `NewTextArea`, `NewMenu`); `OpenMenu` stores a
  pointer to its own copy.
- Rule 11: `core.Ctx.Menu` is `any`, because `widget` owns `MenuState` and `core` cannot name it;
  that is the price of any element opening a menu without a host. `core.Ctx.FocusOwner` is typed,
  since its interface (`core.Focuser`, aliased by `view.Focuser`) uses no widget type.
- Exported mutable tables are copies (`theming.CornerChoices` and the other choice lists);
  `engine.keyNames` is unexported, since nothing outside `engine` reads it.

## Toolkit: widgets and drawing

- **Sideways strips.** `widget.HScroll` shows a row at natural width in a clipped box, with an
  arrow only at an end that can still scroll, the wheel, a touchpad swipe (`Input.WheelX`) and
  `KeepInView()` for one child. A press dragged a few pixels pans it, following `Ctx.PointerX`
  outside the strip. So the request strip (`requesttabs.go`, `App.tabsScroll`) reorders by left
  drag only while it fits (`ScrollState.Overflowing`); the middle button always reorders. The pane
  tab strips (`ScrollTabs`) use it too.
- **Tooltips.** `view.Tooltip(text)` on any element; `view.Draw` records the innermost hovered tip
  in `Ctx.TipText`, and `widget.TooltipLayer`, mounted last in the window's Stack, shows it after a
  rest. While a tip waits the app keeps drawing (`TooltipState.Waiting`). Their look is in `style.md`.
- **Icons** (look and sources: `style.md`, Icons). The SVG subset is in `svg.go`; `RegisterIcons(fs)`
  adds or replaces icons from a folder; one-color icons are cached as white masks and tinted
  (`iconMask`).
- **Sharp edges.** `Ctx.Fill`, `Outline`, `HLine` and `VLine` snap each logical edge to device
  pixels (`Ctx.snap`), so neighbours share an edge, and draw lines `Ctx.LineWidth()` thick. Rounded
  fills and outlines (`FillStyled`, `OutlineStyled`) snap the same way and draw from cached corner
  masks in one `DrawTriangles` (`core/cornermask.go`). `Ctx.Stats` counts mask builds and `Width`
  calls; `BenchmarkDrawStats` in `internal/app` reports them.
- **Caret blink** follows the clock (`Ctx.CaretVisible`), not ticks, so its rate holds when the idle
  app drops from 60 to 30 updates. `engine.MinFPS` (10) draws at least every 100 ms, so a phase
  change shows.
- **Monospace columns.** `CodeView` and `TextArea` share `monoGrid`, which places columns by the
  exact advance so the caret, selection, matches and clicks stay on the text. A tab advances to
  the next stop of `core.TabStop` columns along its line (`core.ColumnAfter`); the wrap layout keeps
  each row's start column, so drawing, the caret, selection, matches, clicks, wrapping and the
  horizontal extent all count it the same way. The text keeps the real tab (display only).
