# internal

Loaded when you work under `internal/`. Adds to `.harness/rules.md`. This file is the one place the layout is written.

## Layout

Target layout (layout C, feature slices). Parts marked "not yet moved" still sit in the older domain layout; later tasks move them.

- `internal/kernel/<name>`: shared leaves and the model that several features use. Moved: `severity` (message level), `fileversion`, `userdir`, `textfmt`, `jsonview`, `openset` (the `OpenCollection` record), `request` (the request model and its ports: `Request`, `Folder`, `Protocols`, `Sender`, the MCP and JSON schema types, `ProtoSource`, `MessageSchema`). `internal/request` keeps the feature logic until task 006 moves it: `curlparse`, `mcpserver`, `graphqlschema`, `binarymessage`.
- Feature slices: each feature is a directory with its model (root package), `ui/`, `adapter/` and a `FEATURE.md` saying what it owns. Not yet moved: today the slices are the domains `request`, `collection`, `response`, `settings`, `workspace`, with controllers as sub-packages (`request/editor`, `collection/tree`, `workspace/tabs`) and no `FEATURE.md`.
- Platform: the OS adapters (window, clipboard, file dialog, fonts, system info). Not yet moved: they sit in the `adapter/` of `settings` and `workspace`.
- App: `internal/app` is the composition root: the `App`, its handlers and its window, in files prefixed by area (`request_`, `tree_`, `settings_`). It may import any package.
- `internal/exampleserver` is the local test server behind `cmd/example-server`.
- Only `ui/` and `adapter/` are zone directories inside a domain. The root package `internal/<domain>` is named after the domain and holds its types, rules and ports.

## Imports

- Dependencies point inward. The kernel imports no other `internal` package. The domain root imports no other package of its domain.
- A controller imports its root and other controllers. It imports no toolkit, no Ebitengine and no `net/http`.
- `ui` imports its root and controllers. It imports no adapter and no other `ui` package.
- An adapter imports its root, controllers and adapters of its own domain. It never imports a `ui`.
- Another domain is reached through the kernel or through the `request` model. Nothing else crosses between `request`, `collection`, `response`, `settings` and `workspace`; `go list` shows it. `app` may import anything.
- Written exception: `settings/theme` imports `uitoolkit/core` and `uitoolkit/theming`, because theme data is toolkit-bound. Its package doc says why.
- A new package goes into the kernel, a domain, or its `ui` or `adapter` zone. No `utils`, `common` or `helpers`.

No test checks these rules. The import-layer check was removed in commit 7d856abc and is not restored (the user's choice), so a review keeps them.

## Names

- Import a domain root without an alias: `request.Request`. A file that needs both `kernel/request` and `internal/request` aliases the kernel one as `kernelrequest`. A local that would shadow the package is renamed, never the import: a request value is `call`.
