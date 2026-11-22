# internal

Loaded when you work under `internal/`. Adds to `.harness/rules.md`. This file is the one place the layout is written.

## Layout

Layout C, feature slices. Every domain has moved; `internal/workspace` is gone.

- `internal/kernel/<name>`: shared leaves and the model that several features use. Moved: `severity` (message level), `fileversion`, `userdir`, `textfmt`, `jsonview`, `openset` (the `OpenCollection` record), `request` (the request model and its ports: `Request`, `Folder`, `Protocols`, `Sender`, the MCP and JSON schema types including `MCPServer` and `ParseServerInfo`, `ProtoSource`, `MessageSchema`).
- `internal/feature/<name>`: a feature slice. Moved: `collections` (root package `collection`: model and sample, `collections`, `tree`, `transfer`, `clientconfig`, `importers/{postman,openapi,miniyaml}`, `adapter/{store,clientfile}`, `ui/destpicker`, `FEATURE.md`), `environments` (package `envs`, `FEATURE.md`), `history` (model in the root package, `adapter/`, `ui/sentlog`, `FEATURE.md`) and `settings` (model, `prefs`, `theme`, `adapter/settingsfile`, `ui/{settingsui,colorpage,shortcutpage,systempage}`, `FEATURE.md`) `send` (root package `send`: `curlparse`, `mcpserver`, `graphqlschema`, `binarymessage` and the `BodyPreview` port; `proto/` with `proto/ws`, the protocol clients keyed by name; `assert`, `editor`, `mcpexplorer`, `viewer`; `adapter/preview`; `ui/{respview,calltrace,mcpform}`; `FEATURE.md`) and `workspace` (model with the `Clock`, `FileDialog`, `Folders` and `StateStore` ports; `tabs`, `state`, `filebrowser`, `autosave`, `bus`, `overlay`, `serial`; `adapter/statefile`; `ui/appicon`; `FEATURE.md`). A feature imports the kernel only.
- Feature slices: each feature is a directory with its model (root package), `ui/`, `adapter/` and a `FEATURE.md` saying what it owns. A feature's controllers are sub-packages of it (`send/editor`, `send/viewer`).
- `internal/platform/<name>`: the OS adapters. `displayscale`, `fonts`, `keyrepeat`, `systeminfo`, `systemclock`, `dockicon`, `filedialog`, `folders`. They implement ports a feature owns (`DisplayScale`, `FontStore`, `KeyRepeat`, `SystemInfo` of settings; `FileDialog` and `Folders` of workspace) and so import that feature for the port types, the one written exception for platform. A feature never imports platform.
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

No test checks these rules. The import-layer check was removed in commit 7d856abc and is not restored (the user's choice), so a review keeps them.

## Names

- Import a domain root without an alias (`send.ParseCurl`) and the kernel request model as `request` (`request.Request`); an older file may still alias it `kernelrequest`. A local that would shadow the package is renamed, never the import: a request value is `call`.
