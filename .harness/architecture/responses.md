## Responses

-  **Response views**: a response opens **Compact** by default (`settings.ResponseView`: compact,
  pretty, raw; Appearance has a Response view row that applies to the open tabs too) and the toolbar
  lists Compact, Pretty, Raw in that order. The filtered find view keeps each line's real number
  (`Line.Num`, the gutter widens for it). Find is tested in all three views.
-  **Compact JSON view** (`internal/response/model/jsonview/compact.go`,
  `internal/response/ui/respview/filter.go`): the response toolbar has Pretty, Raw and Compact.
  `jsonview.Compact(body, width)` parses with key order and number text kept, keeps every object or
  array whose one-line form fits (90 columns with indent and key) on one line and breaks only larger
  ones; block openers are foldable, carry their one-line preview (`{"id": 1, "name": "…", …}`) in
  `Line.Folded` (drawn by `CodeView` while folded) and every line has its `Depth`. Blocks from depth
  2 start folded, so a long list shows one preview per item, like a browser's JSON viewer. While the
  find bar has text the compact view is filtered (`filterCompact`): matching lines, everything
  inside a matching block (its children) and the openers and closers of the blocks around them stay;
  clearing the text or closing the bar restores the full compact view with its folds. Line numbers
  in the filtered view count the shown lines.
-  **Line wrap** (`widget.Wrap` on `Code` and `Area`, template attribute `wrap`;
  `widget.CodeView.Wrap`, `widget.TextArea.Wrap`; `App.toggleWrap` and the `Wrap` chip of the
  response toolbar, `App.toggleBodyWrap` and the `Wrap` chip of the JSON and Text body editors): the
  option sets the widget's `Wrap` every frame, so the app keeps each choice per tab
  (`reqTab.view.Wrap`, `reqTab.body.Wrap`) and the template passes it as `wrap="{{wrap}}"`; it stays
  across new responses and is off by default. The GraphQL query and variables editors do not wrap. A
  wrapped view lays its visible lines out with `layoutWrap` (`uitoolkit/widget/wraplayout.go`),
  which is pure: it takes the visible lines (rune text, a `whole` flag for folded ones), a width in
  pixels and a per-rune measure, and returns `visualRow`s (line, runes `from..to`, `x` = width of
  the line before `from`). A row ends after spaces or where a word starts after punctuation (the
  word rule of word-wise motion), spaces at a break hang at the row's end, a word wider than the
  view is cut at the width, every row holds at least one rune, and a width of 0 means one row per
  line, which is how an unwrapped view lays out too. Its `wrapLayout` maps a position to its row and
  pixel offset (`rowOf`, `xOf`: a column at a break belongs to the row it starts), a pixel on a row
  back to a position (`posAt`, which stops one rune before a break so a hit stays on its row) and
  rows to lines for scrolling (`rowStart`). The code view measures every rune as one
  advance, as drawing places text by column, and leaves room for the vertical bar, so the rows do
  not change when the bar appears. Everything that counted lines counts rows: content height and
  scrollbars, hit testing, reveal (`RevealMatch` scrolls to the row of a find hit, which on a
  minified body can be far below the line's first row), Shift+Up/Down (one visual row, at the same
  offset), and the drawing of text, matches and selection per row; a continuation row has no number
  and no fold arrow, a folded line is one row drawn whole and shows no hits. Switching wrap keeps
  the line at the top of the view in place. Selection, search and copy still work by original line
  and column, so a copied selection has no line breaks the text did not have. A wrapped view never
  scrolls sideways. The body editor (`TextArea`) lays out the same way, unwrapped as one row per
  line: drawing (styled once per line, cut per row, the number on the first row), hit testing, the
  selection (a line's last row also fills its line break), the caret and `CaretAt` (on the caret's
  row, where a completion popup opens) and reveal go by row; wrapped, Up and Down move one visual
  row keeping the goal column within the row (Home and End stay on the line), and switching wrap
  keeps the top line in place. Tabs and wide runes are as wide as the measure says; nothing expands
  tab stops, since Tab inserts spaces.
-  **Image responses**: `respview.Decode` sniffs a body for PNG, JPEG or GIF (whatever the
  Content-Type says); `refreshView` then builds `reqTab.picture` and the response pane shows
  `widget.Image` (scaled down to fit, centered, never up) with the format, size and a Save button
  instead of the text viewer.
-  **Response history and saving**: `reqTab.respHist` keeps the last `respHistoryLimit` (10)
  successful responses a new one replaced; the response toolbar's `History` button (`historyMenu`,
  `showHistory`) swaps one back in. `Save body...` (`saveBody`, `writeBody`) writes the body to a
  typed path (a leading `~/` is `Folders.Home`) in the background; "Saved 7B to out.json" or "Saving
  failed: ..." comes when the write is done. `App.Title` is the window title with a dot for unsaved
  work, set from `Update`.
-  **Response history across restarts** (`internal/response/adapter/history`, `App.saveHistory` /
  `loadHistory`): after every response the tab's request keeps its last 10 successful responses (the
  shown one first) in `history/<hash of its key>.json` next to the state file; the key is the
  collection's folder and the request's path (`historyKey`), the bare path for a collection not
  saved yet, and a request of the first open collection with no file under its key reads the bare
  path's, where a version with one collection at a time wrote it, written whole and renamed into
  place with mode 0600 (responses can hold tokens). Bodies over 256 KiB are not written. The file
  holds the key, so a file that belongs to another request is never shown. A new tab for the request
  loads them into its History menu; the shown response itself starts empty. No state path (tests,
  `-state ""`), no files. Failed writes and unreadable files go to the status line.
-  **Sent log** (`history.SentLog` `Append` / `Load`, `internal/response/ui/sentlog`): every finished
  send of a request that is in the collection tree appends one line (time, the folder of the
  request's collection, request path, protocol, method, URL, status; 0 = failed) to
  `history/sent.jsonl` next to the state file. The newest 500 are kept; the file is cut back when it
  holds twice that, and a damaged line, or one whose protocol is missing or unknown, is skipped on
  load. `sentlog.Log` (owned by `App.sent`) holds the rows and the list's scroll state, and its list
  is markup templates fed a `sentRow` view model per row; it reaches the App through `sentlog.Host`
  (`HistoryDir`, `Toast`, `SetStatus`, `OpenRequest`, `Badge`). The sidebar's header chip switches
  between `Collections` and `Sent`; in `Sent` the list is newest first, the sidebar filter matches
  path, method and URL, and a click opens the request's tab in the collection of the row (a row with
  no folder, from a collection not saved or written before rows had one, looks in the unsaved
  collections, then the first; a request that has been deleted or renamed gives a status note). Only
  sends through `dispatch` are logged, not WebSocket or MCP sessions. No state path, no file.
-  **Call trace** (`internal/request/ui/calltrace`): `calltrace.Log` holds the last 500 `Entry`s
  (time, text, failed) and the state of its dialog; the App owns one (`App.calls`) and adds entries:
  `dispatch` adds `→ METHOD URL` with the expanded URL, `apply` adds `← status text in time, size`
  or `✗ ... failed: reason`, and `applyWS` adds connected, disconnected and refused connections. The
  status bar's `Log` button calls `Open`; `Log.View(host)` is a modal dialog (Escape closes it),
  built from the markup templates `dialog.vml` and `title.vml`, with a filter, Clear and a
  `CodeView` that follows the end, rebuilt only when the entries or the filter change so a selection
  survives. The package knows the App only through `calltrace.Host` (`Size`, `TextMenu`), which
  `App` implements. Nothing is written to disk; socket messages are not traced (the Messages pane
  has them).
-  **Open in browser** (`internal/response/adapter/preview`, `App.openInBrowser`, response context
  menu): the app draws no HTML, so the menu item serves the response body to the user's browser
  instead. `preview.Serve` listens on `127.0.0.1:0`, answers only a random 128-bit path with the
  response's own Content-Type, refuses a Host header other than the listener's (DNS rebinding),
  sends `Content-Security-Policy: sandbox; script-src 'none'` and `nosniff` (the body comes from a
  server the user does not control and is shown on a local origin) and stops after 5 minutes.
  `preview.OpenInBrowser(ctx, address)` runs `xdg-open`, `open` or `rundll32` by GOOS under
  `exec.CommandContext` and reaps it when it ends; ending the context (`BodyPreview.Show` takes it
  first) stops an opener still running, not the page, which keeps its own lifetime (`Browser.Open`
  is the test seam).
