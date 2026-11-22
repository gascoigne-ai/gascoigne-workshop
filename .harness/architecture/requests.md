## Requests

- **Variables**: `{{name}}` is colored green when the request resolves it and red when not, the same
  in the URL field, the body editor and every table (`App.withVariables`, `App.cellStyler`).
  Completion is a key (`Ctrl+Space`, `complete-variable`), not a popup while typing, so the field
  keeps the keyboard; the pure part is `request.VarCompletion`.
- **Path parameters**: a `:name` path segment has a `:name` row in Params, kept in step by
  `MergeParams`; `Request.Expanded` fills it (`ApplyPathParams`) and it never reaches the query.
- **Assertions** (`request/assert`): `Request.Tests` holds one assertion per line, run after
  each response. A line that does not parse is a failure with the reason, so a typo never passes.
- **Chaining**: a `set name = json PATH | header Name | status` line stores the value into the
  active environment, else the collection variables. The value is `Literal` (saved as `literal`):
  `{{...}}` inside it is never expanded, so a server reply cannot pull another variable, such as a
  secret, into the next request.
- **cURL import** (`request.ParseCurl`, the reverse of `proto.Curl`): POSIX-shell splitting, the
  common flags mapped to a REST request, others skipped. Redirects are followed only with `-L`, as
  in curl.
- **Trust**: before a send or connect, `App.allowed` lists what the request does on
  this machine (`request.Request.Accesses`: an MCP `stdio:` command, a file read outside the
  collection folder). Anything not yet trusted opens a confirm dialog; Allow adds it to a
  `request.AccessSet`, saved as `trusted` in `state.json`, and retries.
- **Exit**: the window flushes pending collection edits on exit (`FlushWorkspace`).
