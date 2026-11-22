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
- **History** and the **sent log**: `internal/feature/history/FEATURE.md`.
- **Call trace** (`request/ui/calltrace`): the last 500 sends, results and connection changes, in
  memory only, shown by the status bar's `Log` button. Socket messages are not traced; the Messages
  pane has them.
- **Open in browser** (`response/adapter/preview`): the app draws no HTML, so it serves the body on
  `127.0.0.1:0` at a random 128-bit path for 5 minutes, with the response's Content-Type. It
  refuses another Host header (DNS rebinding) and sends a sandboxed `Content-Security-Policy` with
  no scripts (`pageCSP`) and `nosniff`, because the body comes from a server the user does not control
  and is shown on a local origin.

## JSON view

`kernel/jsonview` turns a body into colored, foldable display lines (`Lines`, `Compact`).
Invalid JSON and other text come back as plain lines, so the response pane renders any body.
