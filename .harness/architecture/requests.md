## Requests

-  **Copying from tables**: right-clicking a row of a read-only table (the response Headers) opens a
  menu with Copy value, Copy name, Copy as `Name: Value` and Copy all headers; the editable tables
  lead their row menu with the first three (`copyRowItems`).
-  **Variable completion**: `Ctrl+Space` (`complete-variable` shortcut, `App.completeVariable`) in
  the URL field opens a context menu of the enabled variables that fit at the cursor
  (`variableCompletion` filters by what follows an open `{{`) and `insertVariable` writes `{{name}}`
  there. It is a key, not a popup while typing, so the field keeps the keyboard.
-  **Variables in the body editor** (`internal/workspace/ui/app/bodystyle.go`):
  `App.withVariables(base, req)` wraps the JSON or plain line styler so each `{{name}}` is split out
  and colored by whether `req` resolves it.
-  **Variables in table cells**: `widget.TableSpec.Styler` may cut a cell's text into colored
  `Span`s (an empty style keeps the cell's own color); `App.cellStyler(req)` colors `{{variables}}`
  green when the request resolves them and red when not, so Params, Headers, Vars and form tables
  match the URL field and the body editor.
-  **Path parameters**: a `:name` segment in the URL path (not a port, not a `{{variable}}`) has a
  `:name` row in the Params table, created and kept in step with the URL by `MergeParams`;
  `WithParams` leaves such rows out of the query and `Request.Expanded` fills the segments
  (`ApplyPathParams`).
-  **Assertions** (`internal/request/model/assert`, `internal/workspace/ui/app/assertpane.go`):
  `Request.Tests` is text, one assertion per line; `assert.Run(text, resp)` returns a `Result` per
  line (parse problems are failures with the reason, so a typo never passes). The request side has a
  `Tests` pane (help in small muted lines broken to fit the narrowest pane, so the editor keeps most
  of the height, above an `Area` editor, live results as you type; template `assertions.vml`) and
  the response side a `Tests` pane listing results (`results.vml`, or `results-none.vml` without
  assertions); `App.runTests` runs after each response and adds `Tests: n/m passed` to the status
  line.
-  **Request chaining and network settings**: `set name = json PATH | header Name | status` lines in
  a request's Tests text (`assert.Captures`) store values into the active environment, else the
  collection variables (`App.setVariable`), after every response. `proto.REST` also takes `CAFile`,
  `CertFile`, `KeyFile` (settings `caCert`, `clientCert`, `clientKey`) and a `socks5://` proxy; the
  tab menu `Cookies...` shows the jar's cookies for the URL.
-  **cURL import**: `model.ParseCurl` (the reverse of `proto.Curl`) splits the command like a POSIX
  shell (quotes, `$'..'`, line continuations) and maps the flags to a REST request:
  `-X -H -d/--data* -F -u --url -A -b -e -m -L -I`, bearer headers become Bearer auth, other flags
  are skipped. Redirects are followed only with `-L`, as in curl. The folder context menu entry
  `Import cURL from clipboard` adds the result named `METHOD /path`.
-  **Exit flush, URL tooltip and binary messages**: the window flushes pending collection edits on
  exit (`FlushWorkspace`); the URL field tooltip shows the resolved URL and undefined variables
  (`App.urlTip`); the WebSocket composer sends `hex: de ad be ef` as a binary message through
  `proto.BinarySender`, and subprotocols are offered by a `Sec-WebSocket-Protocol` header.
-  **Pure helpers outside the UI**: variable completion (`model.VarCompletion`), storing a captured
  value (`model.UpsertVar`), the Timeline text and the suggested file name of a saved body
  (`respview`). `internal/workspace/ui/app` keeps only the glue that reads or changes `App` state.
-  **Menu keys are swallowed** (`App.swallowMenuKeys`, which calls `core.Input.Consume`): while a
  context menu is open its keys are removed from the frame's input after the menu used them (Enter,
  Tab, Escape, Up, Down), so the text widget that still has the focus under it neither types a line
  break for the Enter that accepted a completion, nor moves its caret for the arrows that moved in
  the menu, nor loses the focus for the Escape that closed it.
-  **Trust** (`internal/workspace/ui/app/trust.go`, decision 21): before a send or connect,
  `App.allowed` lists what the request does on this machine (`model.Request.Accesses`: the command
  of an MCP `stdio:` server, a file a multipart field or file body reads outside the collection
  folder). Anything not in `a.trusted` opens a confirm dialog naming it; Allow adds it to
  `App.trusted`, a `model.AccessSet` (`accessset.go`; `Access.Kind` is a `model.AccessKind`, stored only inside the key: a set of accesses kept by `Access.Key`, usable
  as its zero value, `Keys()` sorted for the state file; `AccessSetOf` reads the list back) (saved
  as `trusted` in `state.json`) and runs the send or connect again.
-  **Captured values**: a variable set by a `set name = ...` assertion is `Literal`
  (`model.UpsertVar`, saved as `literal` in the collection files): `{{...}}` inside it is text and
  is never expanded, so a server reply cannot pull another variable, such as a secret, into the next
  request. Ordinary variables still expand each other in order.
