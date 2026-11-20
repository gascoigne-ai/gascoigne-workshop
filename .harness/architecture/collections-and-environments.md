## Collections and environments

### On disk

- `collection/adapter/store` writes a collection as a folder of small JSON files (format: decision 7
  and the package doc). Writes are atomic; unchanged files are not rewritten.
- A gRPC request's proto paths are relative to the collection's folder when inside it, written with
  `/`, absolute otherwise (`ProtoSource.Stored`, `Loaded`; decision 19). Renaming a collection moves
  the paths that were inside its folder.
- Secret values (credentials, variables marked secret) go to `.secrets.json`, never to a request,
  `folder.json` or environment file. An older file that still holds one loads as is and moves on
  the next save.

### Open collections

Decision 17: several collections are open at once, each a root of the tree with its own folder
(`""` until the first save) and active environment. `App.open` is a `collections.Set`
(`collection/controller/collections`).

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
  (`varstarget.go`) points it at a folder or at the collection's own variables.

### Import and export

All importers are pure `collection/model` packages; the App only picks the file and adds the result.

- **OpenAPI 3** (`openapi`, JSON or YAML): a subfolder per first tag, a request per operation,
  `{id}` becomes `{{id}}`, bodies from the example or a schema skeleton. What cannot be read is
  listed (`Result.Skipped`, `Result.Warnings`) instead of failing the import. `baseUrl` comes from
  the first server when the collection has none.
- **Postman** (`postman`): v2.x import adds folders and the missing variables. Export writes v2.1,
  REST requests only. The file is made to be shared, so literal tokens, passwords, API keys and
  secret values are left empty; `{{variable}}` references stay.
- **YAML** (`miniyaml`): a hand-written subset reader, no YAML dependency, for the same reason as
  the other hand-written parsers. What it refuses fails with a line number.
