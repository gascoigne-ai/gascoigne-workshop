# internal

Loaded when you work under `internal/`. Adds to `.harness/rules.md`. This file is the one place the layout is written.

## Layout

Four parts: `feature`, `storage`, `ui`, `app`. The composition, the layer table and the reasons are in `.harness/architecture.md`.

- `internal/storage` (zone `base`): user-file location and versioning; persistence plumbing for the adapters of four features.
- `internal/feature/request`: the base feature every other may import (zone `base`): the request model and its ports (`Request`, `Folder`, `Protocols`, `Sender`, the MCP and JSON schema types including `MCPServer` and `ParseServerInfo`, `ProtoSource`, `MessageSchema`), the `severity` leaf (`feature/request/severity`) and the `OpenCollection` record the settings schema persists and the collection set builds.
- `internal/feature/<name>`: a feature slice with no UI: its model (root package), controllers (sub-packages: `send/requestedit`, `send/viewer`, `send/outbox`, `send/sends`, `settings/keymap`, `workspace/autosave` with its `StateFile`), `adapter/` and a `FEATURE.md` saying what it owns. Features: `collections` (root package `collection`), `environments` (package `envs`), `history`, `settings`, `send` (the protocol clients keyed by name are `adapter/protocol` with `adapter/protocol/ws`) and `workspace`. A feature imports the base feature and `storage` only.
- `internal/ui`: the screen, one directory per component, mirroring the window: `theme`, `format`, `appicon`, `sizes` (fixed widths, `StatusH`, `HeaderH`, the badge slots, the template scope), `statusbar`, `sidebar` (with `sidebar/logic` and `sidebar/sentlog`), `tabstrip`, `header` (the banner, with `header/envchip`, the environment chip), `response` (a composite with `response/respview`), `editor` (the composite of the request pane's components: `New(Deps)`, exported children), `editor/kvtable` (with `editor/kvtable/logic`), `editor/urlbar`, `editor/body`, `editor/auth`, `editor/assertions`, `editor/ws`, `editor/graphql`, `editor/grpc`, `editor/mcp` with `editor/mcp/mcpform` below it, `overlays` (the composite: the one dialog, the destination picker, the call trace log and its children; `New(Deps)`) with `overlays/destpicker`, `overlays/calltrace` (the call trace, its host is the composite), `overlays/filebrowser` (asks for a file: the desktop dialog or the in-app browser), `overlays/welcome`, `overlays/toasts`, `overlays/menus` (functions over the context menu, no component), `overlays/settingsdialog` (a component: `New(Deps)`, the `Host` of its pages) with `formpage`, `colorpage`, `shortcutpage` and `systempage` below it. The intermediate directory `editor` gets a Go file only when the area becomes a composite component; `response`, `overlays` and `editor` have one. Moved packages keep their names; the dialog is `settingsdialog` (it was `settingsui`).
- OS access: what any windowed program needs (Dock icon, display scale, key repeat, file chooser, installed fonts, system report) is in `uitoolkit/desktop`. Its ports live in `internal/app/ports.go` and, for the two the window calls, `ui/window/stores.go` (the fonts one in `ui/overlays/settingsdialog/formpage`) with their test stand-ins. An OS adapter for a port a feature owns sits in that feature's `adapter/` zone (`workspace/adapter/folders`). There is no `internal/platform`.
- Window: `internal/ui/window` is the root composite of the screen: `window.New(Stores)` builds the screen areas (sidebar, header, tab strip, editor, response, status bar, overlays), owns the frame state and the handlers that reach across them, and routes the keys of a frame (`keys.go`). It is the one `ui` package that imports the other screen areas (a written row in the layer table); its tests start the whole window on the real adapters (test-only rows).
- App: `internal/app` is the composition root, at most five files (`options`, `ports`, `app`, `game`, `frame`): it builds the three shared stores, maps `Options` to `window.Stores`, runs the frame loop (`ebiten.Game`) and forwards what the program, `e2e/cmd/shot` and `apptest` reach the window through. It holds no handler. It imports base, features, views and the toolkit but no adapter: `cmd/workshop` and the e2e harness (`e2e/apptest_adapters_test.go`) wire them, and `apptest.Start` takes them from its caller.
- `internal/exampleserver` is the local test server behind `cmd/example-server`.
- Only `adapter/` is a zone directory inside a feature. The root package `internal/feature/<name>` is named after the feature and holds its types, rules and ports.

### Component anatomy

`ui/<area>/<name>` is one view package (anatomy and why in `.harness/architecture.md`): `<name>.go` (the component, `New(Deps)`), `deps.go` (narrow interfaces it declares itself), `view.go` (the VML template and click closures), `keys.go`, `commands.go`, `tick.go`, and an optional nested `logic/` package (toolkit-free state and decisions).

## Imports

- Dependencies point inward. The base zone (`feature/request`, `storage`) imports no other `internal` package. A feature root imports no other package of its feature except its own controllers.
- A controller imports its root and other controllers of its feature. It imports no toolkit, no Ebitengine and no `net/http`.
- `ui` imports base, feature roots and controllers, the toolkit, `ui/format`, `ui/theme`, `ui/sizes` and its own sub-tree (a parent imports its children); `ui/window` also imports every screen area and their children. It imports no adapter, no sibling `ui` package and never its parent. A `logic` package imports base, feature roots and controllers only, no toolkit.
- An adapter imports its root, controllers and adapters of its own feature. It never imports a `ui`.
- Another feature is reached through the base feature, that is the `request` model. Nothing else crosses between features; `go list` shows it. `app` imports no adapter.
- Written exception: `internal/exampleserver` (a dev tool, not a feature) imports `feature/send/adapter/protocol/ws`, the WebSocket framing it serves.
- A new package goes into the base feature, `storage`, a feature, `ui` or a feature's `adapter` zone. No `utils`, `common` or `helpers`.

`make check` enforces rules: `e2e/layers_test.go` reads compiler's direct imports (`go list`, tag sets, test included) and checks against default-deny rule table in `e2e/layers_rules_test.go`. Every written exception row has a reason. Test-only rows let a test file import a real adapter or package as fixture (`exampleserver` tests (the sample collection; its client tests run from `e2e`), `ui/overlays/settingsdialog`, `ui/overlays/filebrowser`, which lists a real temp folder); a production import never matches them. `ui/window` has nine such rows (`collectionstore`, `historyfile`, `protocol`, `protocol/ws`, `settingsfile`, `statefile`, `folders`, `preview` and `exampleserver`). A new exception is a new row, with approval.

The table holds `currentLayoutRules` (the written exceptions and test-only rows) beside `targetLayoutRules` (`base`, `uilogic`, `ui`, the `Below` flag for a parent importing its children, the `ui/window` row); a later task folds the two lists into one.

## Tests

- A view package is tested against a fake `Host` or `Deps` and, for its look, a `view.Dump` golden in its `testdata/`; goldens and fuzz corpora move with the package and are never regenerated by a move.
- Logic in `logic/` and controllers are tested without a display; a test that needs a real adapter as fixture has a test-only row.

## Names

- Import a domain root without an alias (`send.ParseCurl`) and the base request model as `request` (`request.Request`) and never under another alias. A local that would shadow the package is renamed, never the import: a request value is `call`.
