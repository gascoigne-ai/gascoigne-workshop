## Responses

- **Views**: Compact (the default, `settings.ResponseView`), Pretty and Raw. Compact keeps a block on
  one line when it fits `respview.CompactWidth` columns and folds blocks from depth 2, so a long list
  shows one preview per item. While the find bar has text, Compact shows the matching lines, the
  inside of matching blocks and the openers and closers around them; clearing it restores the folds.
- **Wrap**: a per-tab choice for the response and the body editor, off by default and not saved.
  Rows come from the pure `layoutWrap` (`uitoolkit/widget/wraplayout.go`). Selection, search and
  copy still work by original line and column, so a copy has no line breaks the text did not have.
  A wrapped view never scrolls sideways.
- **Images**: PNG, JPEG and GIF are sniffed from the body whatever the Content-Type says and shown
  scaled down, never up.
- **History in a tab**: the last 10 successful responses a new one replaced, in the `History` menu.
- **History across restarts** (`response/adapter/history`): the last 10 successful responses of a
  request in `history/<hash of its key>.json` next to the state file, mode 0600 because responses
  can hold tokens. The key is the collection folder and the request path; the file holds the key,
  so a file of another request is never shown. Bodies over 256 KiB are not written. A request of the
  first open collection with no file under its key reads the bare path's file, written when one
  collection was open at a time.
- **Sent log** (`history/sent.jsonl`): one line per finished send through `dispatch` of a request in
  the tree (not WebSocket or MCP sessions). The newest 500 are kept; the file is cut back when it
  holds twice that. The sidebar's `Sent` list opens a row's request in its collection.
- **Call trace** (`request/ui/calltrace`): the last 500 sends, results and connection changes, in
  memory only, shown by the status bar's `Log` button. Socket messages are not traced; the Messages
  pane has them.
- **Open in browser** (`response/adapter/preview`): the app draws no HTML, so it serves the body on
  `127.0.0.1:0` at a random 128-bit path for 5 minutes, with the response's Content-Type. It
  refuses another Host header (DNS rebinding) and sends a sandboxed `Content-Security-Policy` with
  no scripts (`pageCSP`) and `nosniff`, because the body comes from a server the user does not control
  and is shown on a local origin.
- With no state path (tests, `-state ""`) no history or sent log is written.

## JSON view

`kernel/jsonview` turns a body into colored, foldable display lines (`Lines`, `Compact`).
Invalid JSON and other text come back as plain lines, so the response pane renders any body.
