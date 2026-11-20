## workspace/ui/app: the app

**State changes** (decision 13): the domains with a controller change their state through Model,
Msg, Update and Cmd behind one message bus; the handlers below change the rest of the `App`
directly.

-  **Message bus** (`internal/workspace/controller/bus`, decision 13): `Bus.Dispatch(msg)` queues a
  message from any goroutine without blocking; `Bus.Drain(handle)` delivers them in order once per
  frame (`App.drain`, which dispatches on the message type in `App.handle`) and says whether the
  frame is dirty. A message dispatched while draining waits for the next frame. `Bus.Run(ctx, cmd)`
  starts a `Cmd`, a function built on the UI goroutine with what it needs captured, on its own
  goroutine and dispatches the message it returns. One-shot send results travel this way (`dispatch`
  builds the `Cmd`, `sendResult` is the message). A connection emits many events, so its goroutine
  dispatches `wsOpened` or `wsFailed`, then one `wsReceived` per event, itself and returns `wsClosed`
  (a `Cmd` returns one message); retries dispatch a `wsRetry` from a timer. A
  dispatched `func()` runs on the UI goroutine (`App.handle`): file dialog answers, the gRPC method
  menu, MCP sampling dialogs and `settingsui.Host.Later` use it. The window has no other channel
  from goroutines to the UI; goroutines read only fields that never change after `NewApp` (the bus,
  the ports).
-  **Request edits** (`internal/request/controller/editor`, decision 13, task 037): an edit of the
  request in a tab is a typed message and `editor.Update(req, msg)` returns the request after it,
  never changing the one it is given. The App writes the result back into the collection's own
  request (`editRequest` in `tab.go`), so the sidebar, the autosave and the dirty dot see it at
  once. Edits: URL typed (the Params table follows), Params changed (the URL's query follows),
  method, protocol (a REST tab never keeps an empty or gRPC method), timeout, the body text (also a
  file path or a GraphQL query), the GraphQL variables and starter operation, and the body type
  (which keeps one `Body` for text and file paths, and turns a GET into a POST for GraphQL), the
  auth type and credential fields (`editor.AuthValue` reads one text field by `editor.AuthField`; the API key location is its own message, `editor.AuthKeyInChosen`, the `authTypes` table in
  `auth.go` keeps labels and secrecy), the assertions text, and the redirect and certificate
  toggles. Table rows change in `App.applyRows` with pure row functions and the undo history of the
  `tabs` controller. No code outside `editor` assigns to a field of a tab's request; the gRPC stream
  flag is tab state (`reqTab.grpcStream`), not part of the request. Renames and moves of a request
  in the tree are collection operations (`tree.Update`, applied by `treeops.go`), not tab edits.

`app.App` implements `ebiten.Game` and owns the state: the tab list (`App.list`, a `tabs.List`: open
tabs, the active one, the one being dragged) with a `reqTab` of widget state for each
(`App.widgets`; `App.openTabs()` gives them in strip order), each pointing at the collection's
`model.Request` (plus URL `TextField`, active pane tabs, last `Response`, two `CodeView`s), the
expanded tree folders, settings and the status line. The `App` fields are grouped by concern in
structs it embeds (`appstate.go`): `ports` (the adapters), `treeState` (sidebar selection, filter,
drag, expanded folders, scroll), `tableState` (cell edit, row drag, bulk edit, selected row, column
widths), `frameState` (size, redraw flag, pacing, window geometry), `persistState` (state file,
autosave, background writes, trust) and `settingsState` (settings, their file, the dialog); a
`reqTab` embeds `socketBlock`, `grpcBlock`, `mcpBlock` and `responseBlock` for the state only that
protocol or the response pane uses (`tab.go`). A frame is `beginFrame`, `routeKeys`, `endFrame`
(`game.go`). `window.go`, `sidebar.go`, `requesttabs.go`, `urlbar.go` and `statusbar.go` are the window as VML views
(`sidebar`, `header`, `requestTabs`, `urlBar`, `pane`, `statusBar`); handlers are plain closures
that change `App`.

-  **Panes** are two tables, `requestPanes` and `responsePanes`: label, kind (`paneTable`,
  `paneNote`, `paneBody`, `paneSettings`, `paneAuth`, `paneResponse`, `paneMessages`,
  `paneAssertions`, `paneResults`, `paneExplorer`, `paneProto`, `paneSchema`), row source, edit hook
  and an `empty` hook that supplies "Sending...", errors and "no response yet", shown in place of
  the content by the template `messagepane.vml` (Go hands it the text and its color: muted, or red
  for an error). A note pane (the response's Timeline) is `notepane.vml`, its text in a scroll. A
  map from kind to builder turns a pane into nodes; adding a pane means adding a row. A table pane
  (Params, Headers, Vars, form fields, response headers) is the template `tableview.vml`: Go builds
  the `TableSpec` (columns, rows, handlers, the edited cell) and hands it over with the scroll
  state, the padding and the caption row's title, label and toggle; in bulk mode `bulkview.vml`
  shows the rows as text under the same caption row. The request side depends on the protocol:
  `requestPanesFor(tab)` returns `requestPanes` (REST), `grpcRequestPanes` (gRPC) or, for WebSocket,
  SSE and MCP, `connectionRequestPanes` (Headers, Auth, Vars, Settings; what they send is composed
  in the Messages pane). What the app shows per protocol is a row of `protocolViews` (`panes.go`),
  one per row of `model.Protocols` in the same order (a test checks it): request panes, the panes of
  the response side while connected (`streamPanes`: `wsPanes`, or `mcpPanes` for MCP), whether the
  Messages pane has a composer, whether it offers `{ }`, the composer hint, and whether the protocol
  chip offers it (`offered`, from settings; nil is always). `viewOf` finds the row; a request with
  no known protocol, only one built in code, is shown as the first. `paneTab` indexes that list, so
  choosing another protocol resets it to 0.
-  **URL bar** (template `urlbar.vml`; Go hands it the tab's URL `TextField`, the chip labels and
  colors, flags from the protocol's row (`httpMethod`, `rpcMethod`, `streamable` for
  `Connection == OnStream`), three flags that pick the button and the handlers): protocol chip (the
  `Label` of each row: REST / WS / MCP, plus SSE and gRPC once the `experimental` settings `sse` and
  `grpc` are on; click cycles the rows of `protocolViews` whose `offered` allows them,
  `App.protocolsFor`, and a request already on an experimental protocol keeps it in the cycle),
  method chip (a row with `MethodHTTP` only; one with `MethodRPC`, gRPC, shows its method, which
  lists the server's on click, and a Stream chip), the URL field with `{{vars}}` colored and a
  tooltip of what they resolve to, and one button: Send, Connect / Disconnect on a socket, Cancel
  while a request runs. Editing the URL re-derives the Params table with `model.MergeParams`;
  toggling or deleting a param rewrites the URL with `model.WithParams`.
-  **Clipboard status**: `app` wraps the system clipboard with `core.ReportClipboard` so the status
  bar shows "Copied N characters" or why the clipboard failed.
-  **Response pane**: a toolbar (Compact / Pretty / Raw, Wrap, History, Find, Copy, cURL) above the
  code view, the template `responsepane.vml` (Go hands it the mode flags, the tab's `CodeView`, the
  find bar when it is open and the handlers); an image body is `responsepicture.vml` instead. Pretty
  is the colored, foldable JSON view; Raw shows the body exactly as received. Copy puts the received
  body on the clipboard; cURL puts `proto.Curl` of the request (variables expanded) there. A body
  that is not text (`looksBinary`: a NUL byte or invalid UTF-8 in the first 8 KiB) is replaced by a
  size note, while the Headers and Timeline panes still work. Ctrl+F (or the Find button) opens the
  find bar (template `findbar.vml`; Go hands it the tab's `TextField`, the counter text and the step
  and close handlers): typing searches case-insensitively, the code view highlights every hit and
  the active one and scrolls to it (opening a fold that hides it), Enter / Shift+Enter or the < >
  buttons step through the hits with wrap-around, and Escape closes the bar before it cancels
  anything else. A new response redoes an open search.
-  **Auth pane**: `authTypes` is the table of authentication types (None, Bearer, Basic, API key)
  with their fields; the pane, the stored `model.Auth` values and the masking all follow it, and
  each type keeps its own values when you switch. The pane is the template `authpane.vml`: Go hands
  it the type chips and one `authRow` per field (a chooser's chips, or the field's `TextField` and
  its handler); each chip is an `authChip` the template shows as a `<segment-chip>`, lit when chosen, and
  the fields are `<form-row>`s with a `<check-row>` for showing secrets. Secret fields are masked
  (`TextField.Masked`: bullets on screen, and copy and cut are refused) until the Show secrets row
  is ticked. `proto.REST` applies credentials from a table keyed by type after the headers, so they
  win over an `Authorization` header typed by hand; an API key goes in a header or the query.
  Credentials never appear in the Headers table. At send time `Request.Expanded` replaces `{{vars}}`
  in the URL, headers, bodies, form fields and credentials, without changing the stored request.
-  **Body pane**: a row of type chips (`bodyTypes`: None, JSON, Text, Form, Multipart, GraphQL,
  File) above an editor that fits the type: a `TextArea` for JSON (colored line by line with
  `jsonview.Tokenize`) and Text, the editable fields table for Form and Multipart, the GraphQL
  editor (GraphQL below), the file chooser for File, a note for None. The pane is the template
  `bodypane.vml`: Go hands it one `bodyChip` per type (label, width, chosen, handler; the chosen
  chip is kept in view as the row scrolls sideways), the JSON status, the Format handler, the tab's
  body wrap and its toggle, and the editor it picks, which enters through a slot (`bodyarea.vml` for
  JSON and Text, `bodynone.vml` for None, the table, file and GraphQL editors for the rest). The
  text and the form rows are kept apart, so switching types loses nothing. JSON and Text bodies have
  a Wrap chip before Format (`App.toggleBodyWrap`, see Line wrap). JSON bodies show "Valid JSON" or
  the parse error and a Format button (`json.Indent`, applied as an undoable edit; invalid JSON is
  left alone and reported in the status bar). Requests from before body types get a type inferred
  when their tab opens.
-  **Collection tree in the UI** (`internal/workspace/ui/app/treeops.go`, with `treemenu.go`, `treedrag.go` and `import.go`): right-clicking a row
  opens its context menu: a request gets Rename, Duplicate, Move to..., Copy to..., Copy URL and
  Delete; a folder New request, New folder, Rename, Duplicate, Move to..., Copy to... and Delete
  (Move to... and Copy to... open the destination picker, `App.openTransferPicker`); the collection
  root only the first three. Rename and Delete use the prompt and confirm dialogs (a refused name or
  move is an error toast, and nothing changes); deleting closes the tabs of everything removed. The
  "+" button and Ctrl+T add a request to the selected folder (`App.selected`, the node last clicked
  or moved to). Pressing a row starts a drag: dropping on a folder or a collection root moves into
  it, on a request inserts before it, in this or another open collection, and the row under the
  pointer gets an accent outline (green while Alt is held: releasing then copies; decision 25).
  `finishDrag` builds a `transfer.Drop` with the destination collection's root and
  `App.transferNode` applies it: it commits a pending cell edit first, a name clash renames ("x 2",
  an info toast, also inside one collection), the moved or copied node becomes the selection and its
  folder opens, an open tab keeps its request (a move keeps the pointer, a copy is a new request
  that has no tab), the Vars pane is reset when it edited a moved folder, and an info toast names up
  to five variables the node uses that its new collection does not define and "and N more"
  (`transfer.MissingText`; variables are never rewritten). A move between collections writes the target
  first and then the source (`flushMoved`); a failed target write leaves the source on disk and
  shows an error toast; a target with no folder yet is allowed, then nothing is written for either
  until it is saved; a request opens on press, while a folder toggles when the button is released on
  the same row so an open folder can still be dragged. The filter box in the sidebar header narrows
  the tree to requests whose name or URL contains the text and the folders that hold them, and a
  row's name shows the first match in the accent line color (`core.AccentLine`, readable on the
  sunken sidebar; `AccentText` is for text on an accent fill) (`collection.SplitMatch`). The sidebar
  is markup (`sidebar.go`): `sidebar.vml` holds the header and the tree's scroll, and each row is built
  from `treefolder.vml` or `treerequest.vml` with a `treeItem` view model and walked through
  `<slot for>`; the drop outline is the row's `DropBorder` token, empty when the row is not the
  target (decision 4). While the tree has the keyboard (after clicking a row, and no text widget
  focused) Up/Down move the selection, Enter opens or toggles, F2 renames and Delete deletes.
-  **Right-click menus** (`internal/workspace/ui/app/ctxmenus.go`, the tree's is in `treemenu.go`): a
  request tab (Close, Close others, Close tabs to the right, Duplicate, Copy URL), a table row
  (Insert row above / below, Duplicate, Move up / down, Enable all / Disable all, Delete, all
  through `applyRows` so they are undoable; the placeholder row has none), every text field and text
  area (the toolkit's `widget.TextMenu(field)` option: Undo, Redo, Cut, Copy, Paste, Select all with
  shortcut text and enabled states; Cut and Copy are off for masked secrets, and the model follows
  via the element's own `OnChange`; table cell editors get it too) and the response (Copy the
  selection, Copy body, Copy as cURL, Select all, Find). A right click never clears a selection or
  moves focus. The innermost element with a right-click handler wins: a right click is consumed by
  the first handler that takes it, so the cell editor shows the text menu and not the row menu under
  it. A row has an icon (`menuRow`, `internal/workspace/ui/app/menuicons.go`): the item's own
  `icon`, else the `menuIcons` table keyed by the first word of its label (two words win,
  `Move up`). The icon column exists only in a menu where some row has an icon, then rows without
  one keep an empty column of the same width so labels line up. A menu is as wide as its widest row
  (`menuWidth`, 120 to 420 px), and an item may carry a `hint`: muted lines under its label (the
  settings.json completion shows what the setting is and what it expects, `settings.Expectation`).
-  **Request tabs**: each tab has a × (the label and the × are separate click targets, so closing
  never also selects), dragging with the left or the middle button reorders (a middle click does not
  close), Ctrl+W and the context menu close; closing the last tab leaves a blank request. A tab
  shows a dot when its request differs from the snapshot taken when it was opened (`reqTab.saved`,
  compared with `model.Request.Clone`, which turns empty slices into nil so deleting rows again is
  clean). The tab list changes only through `tabs.Update(list, msg)` (`App.updateTabs`; messages
  `Open`, `Close`, `CloseOthers`, `CloseRightOf`, `Activate`, `Step`, `DragBegin`, `DragOver`,
  `DragEnd`, `Reset`), which is pure and tested without a window (`controller/tabs/list_test.go`);
  `updateTabs` then ends an inline edit in a closed tab and creates or drops the widget state of
  tabs that appeared or went. Pressing a tab starts a drag (`DragBegin`); `OnDragOver` on the tab
  under the pointer sends `DragOver`, which moves the dragged tab there, and the active tab stays
  the same tab. Restoring the saved state and loading another workspace send `Reset` and then `Open`
  for each tab. Each open collection has a color (`appearance.CollectionToken(index)`, the tokens
  `collection.0` to `collection.5` in both app themes, repeating after six, by the collection's
  place in the tree): a 3 px stripe at the left of its tabs and before its root row in the tree.
  Changing the active tab (any `tabs` message) selects that request in the tree and opens the
  folders above it (`App.revealRequest`).
-  **Table rows** (`internal/workspace/ui/app/tablerows.go`): every change of a table's rows goes
  through `App.applyRows(pane, tab, change)`: it runs the change on a copy, ignores it when nothing
  differs, records the old rows for undo (100 steps per table and tab, redo cleared), stores the new
  rows and runs the pane's `changed` hook (the URL rewrite for Params), then drops an edit or
  selection that points past the rows. Toggle, delete, cell commit, the header "enable all"
  checkbox, bulk edit and the keyboard operations all use it. The keyboard acts on the edited row,
  or on the selected row (the last one edited) while its table shows: Ctrl+Enter while editing
  inserts an empty row below and edits it (outside editing it still sends), Ctrl+D duplicates,
  Alt+Up / Alt+Down move it (an edit, even uncommitted, travels with it), Delete removes the
  selected row when nothing is typed. Editable tables also have a drag handle (`GripIcon` in a 20 px
  first column, `TableSpec.OnGrabRow` / `OnDragOverRow`): pressing it selects the row
  (`App.grabRow`) and while the button is held the row moves to the row under the pointer
  (`dragRowOver`, one undo step per row passed); a release ends the drag (`App.dragRow`). Ctrl+Z /
  Ctrl+Y undo and redo table changes when no text widget has the keyboard; inside a cell they belong
  to the text. The Bulk edit button shows a table as `name: value` lines (`# ` for disabled, one
  space after the first colon dropped, descriptions carried over by name) in a text area and applies
  it as a single undoable change when you switch back, change pane or tab. Duplicate enabled header
  names are drawn in the error color; rows with an empty name stay in the table but are never sent.
-  **Resizable table columns**: a `TableSpec` with a `State *widget.TableState` (owned by the
  app, like a scroll or split state) gets draggable dividers over its header: dragging the boundary
  between two columns moves weight between those two only (`resizePair` (in `widget`), neither
  narrower than `MinColumnPx` = 56, the total unchanged, evenly split when the pair is too small for
  two minimums), double click on a divider clears the sizes back to the spec's defaults, and the
  resize cursor shows over a divider. Sizes are weights, so they keep their proportions when the
  window resizes. The app keeps one `TableState` per table (`widthsFor`: request tables by pane
  name, response tables prefixed), so Params, Headers, Vars, form fields and the response headers
  are independent. The weights are remembered in `state.json` (Remembered state).
-  **Cell editing**: clicking a cell of an editable table starts an edit (`App.beginEdit`): the
  table shows a `Field` in that cell with the old text selected. Enter or a click elsewhere commits
  (`commitEdit`), Escape cancels, Tab / Shift+Tab commit and move to the next / previous cell,
  wrapping across rows. Text typed into the placeholder row creates a row (enabled); an empty
  placeholder adds nothing. Params edits rewrite the URL through the pane's `changed` hook.
-  **Layout**: the window is a `Split` of the sidebar and the main column (default 259 px, minimum
  160, the main column keeps at least 420), and the request and response panes are a ratio `Split`
  (default one half, minimum 220 each). Both splits, the frame, the main column and the overlays
  stacked over them (settings, popup menu, dialog, call log, toasts, tooltip) are the template
  `window.vml`; Go builds each part and slots it in, the popup menu and tooltip layers included,
  which have no element. The response status beside the response tabs (pill and timing, dropped when
  the tabs need the room) is `responsestatus.vml`, its colors from `statusPill`. Each pane
  (`App.pane`) is `pane.vml`: a `<tab-bar>` of `<scroll-tabs>` with the response status slotted
  beside them when there is one, above the active pane's body; Go clamps the active tab to the
  protocol's panes and selecting a tab scrolls the pane to its top. The Appearance page has a
  Request and response choice, Stacked (the default, listed first) or Side by side, saved as
  `layout` in the settings file when it is not the default. Tests that click panes use `newTestApp`,
  which pins `side`; `newSettingsTestApp` keeps the defaults.
-  **Drawing only when something changed** (`uitoolkit/engine/pacing.go`, wired in
  `internal/workspace/ui/app/redraw.go`): `main` turns off screen clearing
  (`SetScreenClearedEveryFrame(false)`), and `App.Draw` returns at once unless the `dirty` flag is
  set, so Ebitengine skips the frame's GPU work. A frame is dirty after any input that can change
  the screen (a moved pointer, a click or held or just-released button, the wheel, typed text or any
  key), when a response or an expired toast was applied, when a toast is shown, when the window size
  or scale changes, always while a screenshot is taken, and otherwise at a floor of `engine.MinFPS`
  = 10 frames a second (every `tps/10`-th tick, `engine.MustDraw`), focused or not: the caret keeps
  blinking on its tick clock, and a picture the window system lost comes back within 100 ms. After
  60 quiet ticks the update rate drops from 60 to 30 and returns on the next activity.
  `ebiten.ScheduleFrame`/`SetFPSMode` are deprecated, so this follows Ebitengine's `skipdraw`
  pattern.
-  **Cancel and timeout**: while a request runs the Send button reads Cancel and Escape cancels it,
  in this order of priority: close the menu, cancel a cell edit, cancel the request (`cancelSend`).
  Each tab numbers its sends (`reqTab.seq`); a result whose number is no longer current (cancelled
  or replaced) is dropped. The timeout is the request's `TimeoutSec`, else `settings.TimeoutSec`
  (default 60), else a minute. `proto.REST` reports a stopped request as `Cancelled` or `Timed out`,
  and `Request.NoRedirects` stops it at the first redirect. Both options live in the request's
  Settings pane, with the certificate check: the template `settingspane.vml` (Go hands it the tab's
  timeout `TextField`, the hint naming the default, whether each check is ticked and the handlers).
-  **Sending**: `App.dispatch` expands `{{vars}}` from the request's collection and runs
  `proto.Registry.Send` as a `Cmd` with the timeout above; the result (`sendResult`) comes back
  through the bus and is applied in `Update`, so UI state is only touched from the game goroutine.
-  **Focus and shortcuts**: text fields and code views are the focusable elements.
  `core.Ctx.FocusOwner` (a `core.Focuser`, the interface `view.Focuser` aliases) is the one state that has the keyboard: the app asks for focus by setting a
  state's `Focused`, and `view.Update` (`settleFocus`) makes the last asker the owner, blurs every
  other state and any owner that left the tree (a closed find bar), so code never clears the old
  owner by hand. Tab / Shift+Tab (`view.Update`) move focus through them in tree order, wrapping; a
  focused field that handles Tab itself (a cell editor) keeps it. The focused element draws an
  accent outline. Global key bindings are the `shortcuts` table in
  `internal/workspace/ui/app/shortcuts.go` (name, key, modifiers, action): Ctrl+Enter send, Ctrl+L
  focus URL, Ctrl+T new request, Ctrl+W close tab, Ctrl+Tab / Ctrl+Shift+Tab next / previous tab,
  Ctrl+, settings. A binding fires only when its modifiers match exactly, so plain keys never
  trigger anything; the Shortcuts page of the settings lists the table.
-  **Overlays**: the window's Stack ends with, from bottom to top, the settings dialog, the in-app
  file browser, a context menu, a dialog (with a dimmed scrim) and the toasts.
  `App.openMenu(x, y, items)` shows a context menu, which is the toolkit's popup menu: its state is
  `core.Ctx.Menu` (an `any` holding the `*widget.MenuState` that `widget.OpenMenu` makes from the `widget.NewMenu` value; decision 27), any element opens one with `widget.OpenMenu(c, ...)`, the
  app mounts `widget.PopupMenuLayer(c)` in its Stack once and calls `widget.PopupMenuKeys` each
  frame (rows `widget.MenuItem`, state rows `widget.MenuItem`: label, shortcut text, hint, icon,
  disabled, separator, action; `widget.NextSelectable` and `MenuState.Keys` do the highlight and
  Enter; the width is measured with `widget.MenuWidth`; the app only fills a missing icon from the
  label, `menuIcons`). It closes on a pick, an outside click or Escape and is driven by Up / Down
  (skipping separators and disabled rows, wrapping) and Enter; the action runs after the menu closes
  so it can open another overlay. `App.confirm` and `App.prompt` open a modal dialog (the toolkit's
  `widget.MessageDialog` over a `widget.DialogState` of title, body, input and `DialogButton`s; the
  app's `dialog` adds the MCP tool form drawn between body and input, or a `content` builder, a
  `scroll` and a `width` for the welcome dialog; `App.dialogView` builds it from `dialog.vml`, the
  form entering through a slot) whose Enter runs the primary button, Escape or a click outside
  cancels, and whose text field (prompt) starts focused with its text selected; Tab stays inside it.
  While a menu, dialog or the file browser is open the global shortcuts are off.
  `App.toast(text, level)` (a `severity.Level` of `workspace/model/severity`, mapped once to `widget.ToastKind`; prefs `Toast`, `sentlog.Host.Toast` and `calltrace.Entry` carry it too) shows a message above the status bar: info toasts fade after 3 s (the
  clock is `App.now`, replaced in tests), errors stay until clicked, at most five are kept;
  `App.toastsView` is `toasts.vml`. Clipboard failures and settings save failures are error toasts;
  a successful copy is still a status line. The settings dialog has a read-only **System** page
  (`settings/ui/systempage`, decision 16): the report of `port.SystemInfo` (`model/sysinfo`:
  sections of label/value rows; adapter `settings/adapter/systeminfo` reads the Go runtime, the
  build info and Ebitengine's graphics library, GPU memory, vsync, FPS, monitor and window) and a
  Copy report button; the numbers are read when the page is built, so they are as fresh as the last
  redraw. Every dialog has a title bar of its own (`widget.DialogTitle`: title, drag area, close
  button) and, where it has controls, a `widget.DialogToolbar` row under it (settings: search, Save,
  JSON/Form; Log: filter, Clear; decision 15); the message dialog's close button runs its
  `OnDismiss`. Dialogs move: each keeps a `view.MoveState` (the message dialog in `dialog.move`, the
  settings dialog's `Move`, the call log's `move`) passed as `moved="move"`, and the user drags the
  title (`DragHandle`; decision 15). Escape closes, in order: context menu, dialog, file browser,
  settings dialog, cell edit, find bar, running request.
-  **Configurable shortcuts** (`internal/workspace/ui/app/shortcuts.go`, `keybindings.go` and
  `internal/settings/ui/shortcutpage`): every entry of the `shortcuts` table has a stable `id`; its
  default is the table row, a user binding is `settings.Shortcuts[id]` as a key string
  (`Ctrl+Shift+Tab`: modifiers in any order when parsing, written Ctrl, Alt, Shift; `""` is unbound;
  unknown ids and unparsable values are ignored). `App.bindingOf` is the one place that resolves a
  binding, and `runShortcuts`, the settings list, the context menus' shortcut text (`shortcutHint`)
  and the key handling all read it. On the Shortcuts page each action shows its key as a button:
  click it and the next combination is recorded (`recordKey`); modifier keys alone are waited out,
  Escape cancels, Backspace unbinds, and anything else goes through `checkBinding`, which refuses
  what the text widgets own (Ctrl+A/C/V/X/Z/Y, Ctrl+Shift+Z), plain Escape and a combination another
  action has (naming it). A refusal shows a note under the row and recording continues. A plain key
  (no Ctrl or Alt, not F1-F12 or Delete) is allowed: its row carries the warning from
  `bindingWarning` (an unbound action has no key and no warning), and `runShortcuts` holds it back
  while a text field has focus (`Binding.TypesText`, `textFocused`). Below the shortcuts the page
  lists the text widgets' own keys read-only (`textKeys`, kept in step with `reservedForText` by a
  test). A binding equal to the default is not stored, Reset removes an override, and while
  recording no shortcut or tree key fires.
-  **Colors and contrast** (`internal/settings/model/appearance/theme.go`, `uitoolkit/theming`,
  `internal/settings/ui/colorpage`): `ThemeWith(settings)` builds the theme: the chosen theme and
  text size, then, when Contrast is High, the selection, find-match, active-match and accent colors
  moved (by `core.EnsureContrast`, toward black or white) until their text reaches a 7:1 ratio, then
  the user's overrides for that theme. An override is ignored unless it parses as `#rrggbb`/`#rgb`,
  names one of the four customizable tokens (`theming.CustomColors()`: token, label, the text drawn on
  it) and reaches the required ratio (4.5:1, 7:1 in High); each theme keeps its own overrides in
  `settings.Colors[theme][token]`. The Appearance page has the Contrast chooser and the Colors page,
  per color, a swatch (it paints with the token, so it always shows the color in effect), a hex
  field and a Reset button, plus Reset colors; typing applies live and saves, a refused color shows
  why (`Contrast 2.1:1 is below 4.5:1` or `Use #rrggbb`) and changes nothing. Switching theme or
  contrast rebuilds the fields from the colors then in effect. The helpers `ParseHex`, `ParseHexStrict`, `ParseHexAlpha`, `FormatHex`, `Mix`,
  `ContrastRatio` and `EnsureContrast` are in `uitoolkit/core/color.go`.
-  **Settings button**: the gear at the right of the header (and Ctrl+,) opens and closes the
  settings dialog (Settings below). Theme, text size and the other choices are tables
  (`theming.SizeChoices()`, `CornerChoices()`, `ContrastChoices()`, the catalog's themes), so a new theme
  is one row. A change calls `Ctx.SetTheme` and saves the settings through `settingsfile` to
  `settings.json` in the user directory (`userdir.Name`); `-settings PATH` overrides the file and an
  empty path keeps settings in memory. Unknown theme or text size ids in the file fall back to the
  first built-in theme and the medium size (`theming.Catalog.ThemeID`).
-  **Settings schema** (`internal/settings/model/settings/rows.go`): every key of `settings.json` is one
  `row` (name, kind, limits, text, plus a getter and a setter), built by one constructor per kind
  (`textRow`, `fileRow`, `enumRow`, `flagRow`, `intRow`, `shortcutsRow`, `colorsRow`,
  `collectionsRow`) from the one accessor of its `Settings` field, so the kind and the type of the
  value cannot disagree. `rows` is the order of the form, of `Schema` and of the keys in the file.
  A value is a `settings.Value`: `Text`, `Flag`, `Number`, `ShortcutTable`, `ColorTable` or
  `Collections` (sealed; `Same` compares, and an empty table is the same as a missing one, as in
  the file). `Get(s, key)`, `Set(&s, key, v)`, `With(s, key, v)`, `Changed` and `Reset` read and write
  one field and marshal nothing; an unknown key is `ErrUnknownKey` and a value of another type
  `ErrWrongValue`, and a refused `Set` leaves the settings as they were. The form shows a refusal
  as an error toast. `Minimal` keeps the rows that differ from `Default()`, in row order, and
  `settings.json` (version 1) stays byte for byte what it was (`testdata/encode-*.golden`;
  `Collections` writes `environment` before `folder`).
-  **Welcome dialog**: `welcome.go` holds the release notes of the version as a table
  (`welcomeNotes`: sections of a title and points) and `App.openWelcome`, which opens a wide message
  dialog (`welcomeW`) whose body is `welcome.vml`, a scrolling list of bulleted points that
  `wrapped` breaks into lines first (a `Lines` column does not wrap). `state.State.Welcomed` is the
  version whose notes were shown; `NewApp` opens the dialog when `Options.Welcome` is set and the
  stored version differs from `version`, and remembers the version at once, so it opens by itself
  once per version (`Options.WelcomeEvery`, set by the default dev build through
  `welcomeEveryStart`, opens it at every start). Clicking the version in the status bar opens it
  again. `cmd/workshop` sets `Welcome` for a normal run; a screenshot run opens it only with
  `-welcome`, so the other scenes show the window; the `welcome` golden scene draws the dialog this
  way. The notes are kept in step with `CHANGELOG.md` by hand.
-  **Screenshot mode**:
  `workshop -screenshot out.png [-send] [-menu] [-window WxH] [-request NAME] [-page ID] [-raw-settings] [-sent] [-filter TEXT] [-find TEXT] [-sending] [-open-all] [-type-url TEXT] [-pane LABEL] [-response-pane LABEL] [-body TYPE] [-type-body TEXT] [-wrap] [-toast TEXT] [-dialog KIND] [-record ACTION[=KEYS]] [-log] [-welcome] [-tool NAME] [-add-collection DIR] [-close-all] [-browse DIR] [-settings FILE]`
  renders after a short settle, writes the PNG and exits (`make shot`). The capture moves through
  `shotPhase`: waiting for its frame, saving (`App.saveShot` encodes and writes the PNG through
  `Folders.Write` in the background; nothing more is set up and no frame is taken again), over
  (written or failed: the next update ends the run, with the error from `App.Err`). The scene is a
  list of steps (`app.Step`, `internal/workspace/ui/app/shotsteps.go`, task 086): one small type per
  flag below with an exported constructor (`OpenRequest`, `ShowPane`, `Set`, ...), run in list order
  at frame `setFrame` through `Options.Scene`; the first step that fails ends the run with its
  error, and a repeated flag (`-toast`, `-record`, `-add-collection`, `-set`) is one step each.
  `cmd/workshop/scene.go` builds the list from a table of flag name to constructor
  (`sceneFlagTable`), whose order is the execution order whatever the order on the command line:
  settings, collections, `-open-all`, `-request`, the panes, body, wrap, dialog, toasts, settings
  page, records, logs, browser, `-close-all`, filter, URL, then the steps that wait for a response.
  Those (`-sending`, `-find`, `-tool`) are `deferred` steps: they run once the sends are answered,
  are tried every frame until `ready`, and the capture waits for them until `shotTimeout`. `Options`
  keeps only `ScreenshotPath`, `SendOnStart` and `Scene`. It is how UI changes are checked without a
  person at the window. A screenshot run keeps settings in memory unless `-settings FILE` is given,
  as it keeps the state unless `-state` is given, so `-set` and `-record` never write the user's
  settings.json. `-raw-settings` opens the settings dialog on the text of settings.json, as its JSON
  button does; `-sent` shows the sidebar's log of sent requests; `-filter TEXT` types TEXT into the
  sidebar's filter; `-open-all` opens every request of the collection in its own tab, so the tab
  strip overflows; `-type-url TEXT` types TEXT into the open request's URL, so its tab shows the
  unsaved mark; `-pane LABEL` shows the open request's pane with that label (after `-request`), and
  a label the request does not have ends the run with an error; `-response-pane LABEL` does the same
  for the response side, whose Tests pane `-pane` cannot reach; `-body TYPE` chooses that body type
  (`file`, `graphql`, ...) on the open request, as its chip does, so a scene shows a body editor no
  sample request has, and an unknown type ends the run with an error; `-type-body TEXT` replaces the
  open request's body text (after `-body`), as selecting all and typing would, and a body type
  without the text editor (anything but JSON and Text) ends the run with an error; `-wrap` turns on
  Wrap for the open request's response and body, as their two Wrap chips do, so `response-wrap` and
  `request-body-wrap` show lines broken before a word; `-toast TEXT` (repeatable) shows TEXT as an
  error toast, which stays until clicked, so a slow frame cannot let it fade before the capture;
  `-dialog KIND` opens the `rename` or `delete` dialog, or the `move` or `copy` destination picker,
  on the open request through its tree menu path, and an unknown kind ends the run with an error;
  `-record ACTION=KEYS` (repeatable) starts recording on that action of the Shortcuts page and
  delivers KEYS as a key press would, so the page shows a conflict, a refusal note or a plain-key
  warning, and a bare `ACTION` is left recording; an unknown action or keys that do not parse end
  the run with an error; `-log` opens the Log dialog of calls, as the status bar's Log button does;
  `-welcome` opens the welcome dialog; `-add-collection DIR` (repeatable) opens the collection saved
  in DIR beside the start one, in order, as Open collection folder does, before `-request` looks for
  its request, and a folder that holds no collection ends the run with an error (the prompt would
  start an empty one there); `-close-all` closes every collection, so the tree shows its empty
  prompt; `-browse DIR` opens the in-app file browser on DIR (made absolute) with the title of the
  request body's file chooser, as choosing a file without a dialog program does, so its footer shows
  the hint for this desktop; `-tool NAME` (with `-request` of an MCP request and `-send`) opens the
  call form of that tool once the server announced it, as the composer's tool menu does, and the
  capture waits for it (a tool missing from the announced list, or `-tool` without `-send`, ends the
  run with an error); with `-send` and `-request` the first request is sent at start-up and the
  named one once that is answered, so the log has two rows in a fixed order, and the capture waits
  for every send. `-find TEXT` opens the find bar once every send is answered and types TEXT into
  it, as Ctrl+F and typing would. `-sending` sends the open request once it is shown and captures
  while it is in flight, so the URL bar shows Cancel; the golden stub holds the sample's `/delay/`
  request until the client goes away. With `-send` on a socket tab (WebSocket, SSE, MCP, gRPC
  stream) the capture waits for the connection instead of a response (`answered`): an open session,
  for MCP also the server info, or a failed connect with its error in the log. With `-pane Proto` on
  a gRPC request with proto files the capture also waits until the pane has read them
  (`protoReading`), so a problem at its line is in the picture.
-  **Software-rendered shots in tests** (`cmd/workshop/softshot_test.go`, default build only):
  `renderShot` builds the program once, runs it under `xvfb-run` with
  `LIBGL_ALWAYS_SOFTWARE=1 GALLIUM_DRIVER=llvmpipe`, `HOME` and every `XDG_*` directory in a temp
  sandbox and `-settings "" -state "" -themes ""`, and returns the PNG. No Xvfb: the test is
  skipped, but fails when `CI` is set. Two runs give byte-identical files here, so goldens can be
  compared. `TestGoldenScenes` draws each row of `goldenScenes` (flags, virtual screen size,
  optional stub for the sample request's `127.0.0.1:8081`; the example server's MCP endpoint answers
  at `127.0.0.1:8084` and its gRPC port without reflection at `127.0.0.1:8086`
  (`serveExampleServer`), so `mcp-connected` shows a connected server's explorer, `mcp-tool-form`
  its tool and `grpc-proto-files` a call made from a `.proto` file) and compares it with
  `cmd/workshop/testdata/golden/<name>.png` through `assertGolden`: 4 per channel and 100 pixels of
  slack, `.got.png` and `.diff.png` written beside the golden on a mismatch (git-ignored),
  `go test ./cmd/workshop -update` rewrites goldens (look at the PNG before committing).
  `-window WxH` sets the window size, which is how the narrow scene is drawn; `-request NAME` opens
  that request (an unknown name ends the run with an error, so a golden never passes for the wrong
  scene) and `-page ID` opens the settings dialog on a page. Scenes run in parallel on their own X
  displays, sharing one stub for the sample request; the stub also answers `/users/octocat` with a
  captured GitHub user (`cmd/workshop/testdata/octocat.json`, one long line of JSON) for
  `response-wrap`. A scene lists rectangles to black out on both sides (`mask`) where a measured
  duration is printed, and folders under `cmd/workshop/testdata` to open beside the sample
  (`collections`), each copied to a temp dir and passed as `-add-collection`, so nothing the program
  writes lands in testdata: `tree-two-collections` opens `github-collection` (what the store wrote
  for the GitHub sample) and `tree-empty` draws `-close-all`. A scene's `browse` folder is copied
  the same way and passed as `-browse`: `file-browser` lists a copy of `github-collection`, with the
  path field masked (`browserPath`) since it shows the temp folder; the child's environment has no
  desktop variable, so the footer's hint reads "For native dialogs install zenity or kdialog".
  `grpc-proto-files` and `grpc-proto-error` open `grpc-proto-collection` (written by the store,
  proto paths relative to it: the real `health.proto` under an import path, and a copy cut off after
  line 33) on the Proto pane: the first sent, answered by the port without reflection, the second
  with its parse error at `broken/health.proto:34:1`.
