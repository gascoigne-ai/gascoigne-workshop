# internal

Loaded when you work under `internal/`. Adds to `.harness/rules.md`. This file is the one place the layout is written.

## Layout

Layout C, feature slices. Every domain has moved; `internal/workspace` is gone.

- `internal/kernel/<name>`: shared leaves and the model that several features use. Moved: `textfmt`, `jsonview`.
- `internal/storage` (zone `base`): user-file location and versioning, merged from `kernel/userdir` and `kernel/fileversion`; persistence plumbing for the adapters of four features.
- `internal/feature/request`: the base feature every other may import (zone `base`): the request model and its ports (`Request`, `Folder`, `Protocols`, `Sender`, the MCP and JSON schema types including `MCPServer` and `ParseServerInfo`, `ProtoSource`, `MessageSchema`) and the `severity` leaf (message level, `feature/request/severity`) and the `OpenCollection` record the settings schema persists and the collection set builds.
- `internal/feature/<name>`: a feature slice. Moved: `collections` (root package `collection`: model and sample, `collections`, `tree`, `transfer`, `clientconfig`, `importers/{postman,openapi,miniyaml}`, `adapter/{store,clientfile}`, `ui/destpicker`, `FEATURE.md`), `environments` (package `envs`, `FEATURE.md`), `history` (model in the root package, `adapter/`, `ui/sentlog`, `FEATURE.md`) and `settings` (model, `prefs`, `theme`, `form`, `adapter/settingsfile`, `ui/{settingsui,formpage,colorpage,shortcutpage,systempage}`, `FEATURE.md`) `send` (root package `send`: `curlparse`, `mcpserver`, `graphqlschema`, `binarymessage` and the `BodyPreview` port; `proto/` with `proto/ws`, the protocol clients keyed by name; `assert`, `editor`, `mcpexplorer`, `viewer`; `adapter/preview`; `ui/{respview,calltrace,mcpform}`; `FEATURE.md`) and `workspace` (model with the `Folders` and `StateStore` ports; `tabs`, `state`, `filebrowser`, `autosave`, `bus`, `overlay`, `serial`; `adapter/{folders,statefile}`; `ui/appicon`; `FEATURE.md`). A feature imports the kernel only.
- Feature slices: each feature is a directory with its model (root package), `ui/`, `adapter/` and a `FEATURE.md` saying what it owns. A feature's controllers are sub-packages of it (`send/editor`, `send/viewer`, `send/outbox`).
- OS access: what any windowed program needs (Dock icon, display scale, key repeat, file chooser, installed fonts, system report) is in `uitoolkit/desktop`. Its ports live in `internal/app/ports.go` (the fonts one in `settings/ui/formpage`) with their test stand-ins. An OS adapter for a port a feature owns sits in that feature's `adapter/` zone (`workspace/adapter/folders`). There is no `internal/platform`.
- App: `internal/app` is the composition root: the `App`, its handlers and its window, in files prefixed by area (`request_`, `tree_`, `settings_`). It may import any package.
- `internal/exampleserver` is the local test server behind `cmd/example-server`.
- Only `ui/` and `adapter/` are zone directories inside a domain. The root package `internal/<domain>` is named after the domain and holds its types, rules and ports.

## Imports

- Dependencies point inward. The kernel imports no other `internal` package. The domain root imports no other package of its domain.
- A controller imports its root and other controllers. It imports no toolkit, no Ebitengine and no `net/http`.
- `ui` imports its root and controllers. It imports no adapter and no other `ui` package.
- An adapter imports its root, controllers and adapters of its own domain. It never imports a `ui`.
- Another domain is reached through the kernel or through the `request` model. Nothing else crosses between features; `go list` shows it. `app` may import anything.
- Written exception: `internal/exampleserver` (a dev tool, not a feature) imports `feature/send/proto/ws`, the WebSocket framing it serves.
- Written exception: `feature/settings/theme` imports `uitoolkit/core` and `uitoolkit/theming`, because theme data is toolkit-bound. Its package doc says why.
- A new package goes into the kernel, a domain, or its `ui` or `adapter` zone. No `utils`, `common` or `helpers`.

`make check` enforces these rules: `e2e/layers_test.go` reads the compiler's direct imports (`go list`, both tag sets, test files included) and checks them against the default-deny rule table in `e2e/layers_rules_test.go`. Every written exception is a row there with its reason. Six test-only rows let a test file import a real adapter or package as a fixture (`exampleserver` tests, `history/ui/sentlog`, `settings/ui/formpage`, `settings/ui/settingsui`); a production import never matches them. A new exception is a new row, with the user's approval.

During the reorganization (`.harness/reorg.md`) the table accepts both layouts: `currentLayoutRules` for the tree as it is, `targetLayoutRules` beside them for the target zones (`base`, `uilogic`, `ui`, the `Below` flag for a parent importing its children). A target row may match no import until its packages move. The last reorganization task drops the old rows and zones.

## Names

- Import a domain root without an alias (`send.ParseCurl`) and the kernel request model as `request` (`request.Request`); an older file may still alias it `kernelrequest`. A local that would shadow the package is renamed, never the import: a request value is `call`.
