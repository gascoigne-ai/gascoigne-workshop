## Collections and environments

### On disk

- `collection/adapter/store` writes a collection as a folder of small JSON files (format and why: its package
  doc). Writes are atomic; unchanged files are not rewritten.
- A gRPC request's proto paths are relative to the collection's folder when inside it, written with
  `/`, absolute otherwise (`ProtoSource.Stored`, `Loaded`), so the folder can move. Renaming a collection moves
  the paths that were inside its folder.
- Secret values (credentials, variables marked secret) go to `.secrets.json`, never to a request,
  `folder.json` or environment file. An older file that still holds one loads as is and moves on
  the next save.

### Open collections

Several collections are open at once (the user's decisions), each a root of the tree with its own
folder (`""` until the first save) and active environment, so nothing leaks between them; a
collection is known by its folder. `App.open` is a `collections.Set`
(`collection/collections`).

- **Ownership**: tabs, saving, rename, delete, find, sending and captured values go to the collection
  that owns the request (`Set.Owner`). A request of none, such as the blank tab, goes with the
  focused collection: the active tab's, else the tree selection's, else the first (`App.focus`).
- History and the sent log key a request by its collection's folder and its path, so equal paths in
  two collections stay apart.
- **Start**: `-workspace DIR` alone, else the remembered set, else (release build) the client
  config, else the built-in sample in memory. A folder that fails to load is left out, named in the
  status line and never overwritten.
- **Remembered set**: every change of the set rewrites `settings.collections` (`{folder,
  environment}`, hidden, General). A remembered folder that failed to load stays in the list, so an
  unmounted drive is not forgotten. A `-workspace` run does not replace the list; an environment
  pick there only updates entries that name an open collection (`App.rememberEnvironments`). The
  older `workspace` and `environment` keys are read only when `collections` is absent.
- **Saving**: each collection saves itself after it stays unchanged for `saveDebounce`, with its
  own debounce. Everything changed is flushed at exit and before a rename. A first save of an
  unsaved collection makes a folder in `collections/` beside the default workspace.
- **New, rename, delete**: New makes its folder in `collections/`, never inside a collection (a
  `folder.json` below one would be a folder of it). Delete removes only the files the store wrote.
  Writes in flight finish before a folder moves or goes.
- **Close** asks only for an edited collection without a folder, the one case where closing loses
  edits. Closing the last leaves an empty tree and creates nothing on disk.
- Roots reorder by menu, not by drag: a drag in the tree moves or copies a node, and a root does not
  drag.

### Variables and environments

- Resolution order (`Collection.VarsFor`): collection < active environment < folders, outermost
  first, nearest wins. A request expands only with its own collection's variables. Every send,
  cURL copy, connect, URL tooltip and highlight uses `App.varsFor`.
- Each open collection has its own active environment, remembered by name in its
  `settings.collections` entry. The header chip and the Vars pane act on the focused collection.
- The Vars pane edits the active environment, else the collection variables, unless a `varsTarget`
  (`env_varstarget.go`) points it at a folder or at the collection's own variables.

### Import and export

All importers are pure leaf packages of `collection`; the App only picks the file and adds the result.

- **OpenAPI 3** (`openapi`, JSON or YAML): a subfolder per first tag, a request per operation,
  `{id}` becomes `{{id}}`, bodies from the example or a schema skeleton. What cannot be read is
  listed (`Result.Skipped`, `Result.Warnings`) instead of failing the import. `baseUrl` comes from
  the first server when the collection has none.
- **Postman** (`postman`): v2.x import adds folders and the missing variables. Export writes v2.1,
  REST requests only. The file is made to be shared, so literal tokens, passwords, API keys and
  secret values are left empty; `{{variable}}` references stay.
- **YAML** (`miniyaml`): a hand-written subset reader, no YAML dependency, for the same reason as
  the other hand-written parsers. What it refuses fails with a line number.

## Collection tree

- `request.Folder` holds pointers (`[]*Folder`, `[]*Request`), so a request keeps its identity (open
  tabs point at it) however its siblings change. The tree rows (`collection.Rows`) key the
  expanded set by folder pointer, so a renamed or moved folder stays open.
- The tree operations are pure functions in `internal/kernel/request/tree.go`; a move or copy is
  `collection/transfer`, which validates before it changes anything.

## Example server and client config

The default build carries example endpoints and a release build reads client configs instead
(the user's choice). `make example-server` runs `cmd/example-server`, one port per service on `127.0.0.1`
(flags change host and ports; port 0 picks a free one):

| Port | Service |
|---|---|
| 8081 | REST: `/echo`, `/status/{code}`, `/delay/{ms}`, `/auth` |
| 8082 | WebSocket echo |
| 8083 | SSE: `/events?count=&interval=` |
| 8084 | MCP: `/mcp`, tools `echo` and `add` |
| 8085 | gRPC health and reflection |
| 8086 | gRPC health without reflection, reached only through its `.proto` file |

Builds:

- Default build: `collection.Sample()` is the `Example Server` collection, one request per endpoint
  (tests in `exampleserver` fail when one is missing). It reads no client config and keeps its data
  in its own directory (`userdir`; why in its package doc), so a run from an IDE never touches the release
  build's data. Screenshot runs and golden scenes keep settings and state in memory.
- Release build (`make release`, `-tags release`, installed by `make install`): no example server,
  `Sample()` is empty, and with no workspace the App opens the client config's tree. An unreadable
  config is named in the status line and the sample stays.

Client config:

- `.workshop.yaml` (or `.yml`, `.json`, `.toml`) in the settings directory (global) and in the
  working directory (local). A local entry replaces a global one of the same name; two files in one
  directory are an error.
- TOML is read with `github.com/BurntSushi/toml` (approved by the user), YAML with the `miniyaml`
  subset reader. All three formats become one generic tree, decoded strictly: a misspelled key is an error.
- It opens as one collection: each configured collection is a folder (variables on the folder,
  environments on the root), servers are requests in a `Servers` folder.

```yaml
servers:
  - name: Local REST
    protocol: rest
    url: http://127.0.0.1:8081/echo
    auth: {type: bearer, token: secret}
collections:
  - name: Demo
    vars: {baseUrl: http://127.0.0.1:8081}
    envs:
      - name: Dev
        vars: {token: abc}
    requests:
      - name: Echo
        protocol: rest
        method: POST
        url: "{{baseUrl}}/echo"
        headers: {X-Trace: "1"}
        body_type: json
        body: '{"hello": "world"}'
    folders:
      - name: More
        requests: []
```

Request keys: `name`, `protocol`, `method`, `url`, `params` and `headers` (maps), `body`,
`body_type`, `timeout` (seconds), `auth` (`type`, `token`, `user`, `password`, `key_name`,
`key_value`, `key_in`). A missing or unknown `protocol`, or an unknown `body_type` or auth `type`, is
an error that names the server or request.
