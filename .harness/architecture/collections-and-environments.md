## Collections and environments

-  **Workspace on disk** (`internal/collection/adapter/store`,
  `internal/workspace/ui/app/workspace.go`): `store.Files` `Save`/`Load` write and read a collection tree
  (format in decision 7; atomic writes, unchanged files are not rewritten). A gRPC request's proto
  files and import paths are `protoFiles` and `importPaths` in its file, left out when empty
  (reflection); in memory they are absolute, in the file relative to the collection's folder when
  inside it, written with `/` on every system and read back with the system's separators, and
  absolute otherwise (`ProtoSource.Stored`, `Loaded`, against the folder made absolute; decision
  19). Renaming a collection moves the paths that were inside its folder to the new one
  (`Folder.MoveProtoPaths`). At the start the app opens `-workspace DIR` alone, else the collections
  remembered in `settings.collections` in their order (an entry without a folder is the start
  collection), else, in a release build, the client config, else keeps the built-in sample in
  memory; a folder that fails to load is left out, named in the status line and never overwritten.
  `Ctrl+S` (`save` shortcut) saves the focused collection; the first save of a collection without a
  folder makes one named after it in `collections/` beside the default workspace (as New does) and
  remembers the set. Each saved collection is saved by itself (`workspaceTick`) after it stays
  unchanged for `saveDebounce` (1 s), with a debounce of its own (`App.saves`); saving clears the
  dirty marks of that collection's tabs, and so does a flush. `FlushWorkspace` writes every changed
  collection at exit and before a collection is renamed.
-  **Whole collections** (`internal/workspace/ui/app/collections.go`, rules in
  `collection/controller/collections`, decision 7): the collection menu (right-click the Collections
  chip for the focused collection, or a collection's root row in the tree for that one) has New
  collection, Open collection folder, Rename collection, Save collection and Delete collection. New
  makes a folder named after the collection in `collections/` beside the default workspace (never
  inside a collection: a `folder.json` below one is a folder of it) and opens it beside the open
  ones, as Open collection folder does (see Open collections); Rename changes the name and moves the
  folder to the new slug; Delete asks, removes only the files the store wrote and closes the
  collection; when it was the only one open, the next collection of that folder opens, or a new
  empty "My collection". Rename and Delete need a collection that is on disk (save first). Writes in
  flight are waited for before the folder moves or goes. Ctrl+S saves now, Ctrl+O opens a folder and
  Ctrl+Shift+W closes the focused collection; New has no shortcut.
-  **Open collections** (`internal/workspace/ui/app/openset.go`, rules in
  `collection/controller/collections`, task 071, decision 17): `App.open` is a `collections.Set`,
  the open collections in tree order, each with its folder ("" until its first save) and its own
  active environment. The tree lists every collection's rows, root after root, a root's name bold
  (`treeItem.Root`, style.md Sidebar); a tree message changes the collection that holds its node
  (`treeOwner`); a drop onto another collection moves the node there (Alt: copies), see Collection
  tree in the UI. Tabs, saving, rename, delete, find, sending, and captured values go to the
  collection that owns the request (`Set.Owner`, `App.entryFor`); a request of no collection, such
  as the blank tab, goes with the focused one, so with one open collection everything goes to it.
  The history files and the sent log key a request by its collection's folder and its path, so two
  open collections with the same path keep apart. The window title and the header show the focused
  collection. `addCollection` opens a folder beside the others (one already open is focused, its
  edits left pending and the remembered list untouched), with the environment remembered for that
  folder; New collection and Open collection folder (Ctrl+O, `openCollectionAt`) go through it, so
  they add instead of replacing. `closeCollection` saves, disconnects the sockets of its tabs,
  closes them and drops it; Close collection (the root row's menu, the chip's menu, and the
  `close-collection` command, Ctrl+Shift+W by default in the shortcut table, beside Ctrl+W for the
  tab; a binding matches its modifiers exactly, so the two never both fire) goes through
  `requestCloseCollection`, which first asks (`collections.ClosePrompt`, the confirm dialog that
  deleting uses) only for an edited collection without a folder, the one case where closing loses
  edits. Closing the last leaves an empty tree and creates nothing on disk; the sidebar then shows a
  muted note and Open collection folder / New collection buttons (`noCollection` in `sidebar.vml`),
  and the chip menu's Rename, Save, Close and Delete are disabled. The root row's menu
  (`rootMenuItems`) has Rename collection, Move up and Move down (disabled at the ends;
  `moveCollection` sends `collections.Shift`, which finds the place, and rewrites the remembered list), New, Open, Close and
  Delete; reordering is by menu, not by drag, because a drag in the tree moves or copies a node and
  a root row does not drag. Deleting the last open collection still opens the next one in
  `collections/` (or a new one), through `openWorkspaceAt`, which replaces the set. The screenshot
  flag `-close-all` closes every collection before the capture and `-add-collection DIR` opens one
  more; the golden scenes `tree-two-collections` and `tree-empty` draw both, and `dest-picker-move`
  and `dest-picker-copy` draw the destination picker over two collections. Every change of the set
  rewrites `settings.collections` (a list of `{folder, environment}`, General section, hidden; a
  remembered folder that failed to load stays in the list, after the open ones, so an unmounted
  drive is not forgotten), which clears the older `workspace` and `environment` keys; those are read
  only when `collections` is absent, as a set of one (`Settings.RememberedCollections`). An
  environment pick rewrites the list too when the open set came from it or the user changed the set
  since; when it did not (a folder given with `-workspace`), only the list's entries that name an
  open collection take the new environment, so such a run does not replace what the next ordinary
  start opens (`rememberEnvironments`, with the list built by `collections.Remembered` and `collections.WithEnvironments`).
-  **Folder variables and opening collections**: `model.Folder.Vars` (saved in each folder's
  `folder.json`) and `Collection.VarsFor(req, env)` give the resolution order collection <
  environment < folders (outermost first, nearest wins); every send, cURL copy, WebSocket connect,
  URL tooltip and variable highlight uses `App.varsFor(req)`. The folder menu's `Variables` entry
  (`App.editVars`) points the Vars pane at that folder. `Ctrl+O` (`openWorkspaceAt`) switches to
  another collection folder after saving the open ones; its environment is the one remembered for
  that folder, else the one of the same name active before.
-  **Environments** (`internal/workspace/ui/app/env.go`, `model.Environment`): `Collection.Envs` are
  named variable sets; each open collection has its own active one (`collections.Entry.Env`, -1
  none, remembered by name in its `settings.collections` entry). A request expands with its own
  collection's variables, overridden by that collection's environment (`App.varsFor`), never another
  collection's. The chip and the Vars pane show and set the environment of the focused collection:
  the one holding the active tab's request, else the tree selection's, else the first (`App.focus`).
  The header chip (template `envchip.vml`, slotted into the header; at most 200 px wide, a longer
  name cut with an ellipsis so the settings button stays in a 400 px window; in a narrower window
  the header's title and then the chip `shrink`, down to 48 and 24 px, so the settings button stays
  inside a 280 px window and "No environment" still shows in full there) opens the environment menu
  (switch, new, rename, delete), a toolkit popup menu built from `envs.Menu` rows. The Vars pane
  edits the active environment, else the collection variables, unless `App.varsTarget` points it
  elsewhere: a `varsTarget` (`varstarget.go`) is one kind (`varsActive`, the zero value;
  `varsFolder`; `varsCollection`, a collection's own variables although an environment is active)
  with its folder or collection, and says which rows to edit (`rows`), what to call them (`caption`)
  and whether closing a collection resets it (`refersTo`); choosing or adding an environment resets
  it. A variable marked secret (row menu) shows as bullets and its value is written to
  `.secrets.json`, not `environments/NN-name.env.json` (collection and folder variables likewise:
  not `folder.json`; an older `folder.json` that still holds the value loads as is and moves on the
  next save). `App.varSegments` colors defined `{{names}}` green and unresolved ones red in the URL
  field.
-  **OpenAPI import** (`internal/collection/model/openapi`, `App.importOpenAPI`): `openapi.Parse`
  reads an OpenAPI 3 document in JSON or YAML (YAML through `internal/collection/model/miniyaml`)
  into a folder (a subfolder per first tag, a request per operation), following `$ref` for
  parameters and schemas, turning `{id}` into `{{id}}`, and building a JSON body
  from the example or from `model.Skeleton` of the schema. Schemas (`components.schemas`, parameter
  and body schemas) are decoded into `model.JSONSchema` with the document; `resolveRefs` follows
  `$ref` into `#/components/schemas/` (and inside properties) and a reference outside the document
  is the zero schema. The importer defines the `baseUrl`
  collection variable from the first server when the collection has none. What cannot be read is
  listed in `Result.Skipped` instead of failing the import: an operation (`METHOD /path: why`, left
  out) or a path's shared parameters (`parameters of /path: why`, its operations imported without
  them). A path item that is not an object (an `x-` extension key excepted) is dropped and named in
  `Result.Warnings`; the importer shows warnings with the skipped parts in its toast.
-  **Postman import** (`internal/collection/model/postman`, `App.importPostman`): `postman.Parse`
  maps a v2.x collection to a folder tree and its variables; the importer adds the folder under the
  chosen one and the variables the collection lacks.
-  **Postman export**: `postman.Export(folder, vars)` writes a v2.1 collection (REST requests only;
  `postman.Skipped` counts the rest; the file is made to be shared, so literal tokens, passwords,
  API key values and the values of secret variables are left empty, while a `{{variable}}` reference
  stays); import reports what it cannot carry over (`Result.Skipped`: auth types and file or graphql
  bodies with no equivalent); `App.exportPostman` picks a file through `pickFile` and writes it in
  the background (`App.writeFile`).
-  **YAML subset** (`internal/collection/model/miniyaml`): a hand-written reader for what API
  descriptions use (block mappings and sequences, plain and quoted scalars with the core-schema
  types, one-line flow collections, `|` and `>` block scalars, comments); anchors, aliases, tags,
  multi-line flow collections and several documents are refused with a line number. The result has
  the shape `encoding/json` gives, so `openapi.Parse` marshals it and reads it like JSON. Decision:
  no YAML dependency, for the same reason as the other hand-written parsers.
-  **Feature packages** (task 018): code that does not need the `App` lives in a package of its own
  under `internal/`: `respview` (response body to colored lines: `CodeLines`, `CompactLines`,
  `RawLines`, `LimitedLines`, `FoldDeep`, `LooksBinary`, `Filter` for the compact search, `Decode`
  for pictures, `Timeline`, `SuggestedFileName`), `textfmt` (`Duration`, `Size`). `appearance` (the
  app's overlay themes in `themes/*.json`, the user's theme folder through the toolkit's catalog,
  `LayoutChoices`, `ThemeWith(settings)`, `MethodToken` and `CollectionToken`). `collection` (the
  collection tree: `Rows`, `FilteredRows`, `Row`, and the built-in `Sample`).
  `internal/workspace/ui/app` keeps only what shares window state.
