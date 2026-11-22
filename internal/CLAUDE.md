# internal

Loaded when you work under `internal/`. Adds to `.harness/rules.md`. This file is the one place the layout is written.

## Layout

- `internal/<domain>/`: domains are `request`, `collection`, `response`, `settings`, `workspace`.
- The root package `internal/<domain>` is named after the domain. It holds the domain types, their rules and the ports at its boundary.
- Sub-packages sit directly under the domain. A controller (state and decisions, no widget, no I/O) is such a package, for example `request/editor`, `collection/tree`, `workspace/tabs`.
- Only two directories are zones: `ui/` (views and input) and `adapter/` (implementations of ports: network, files, system).
- `internal/app` is the composition root: the `App`, its handlers and its window, in files prefixed by area (`request_`, `tree_`, `settings_`). It may import any package.
- `internal/exampleserver` is the local test server behind `cmd/example-server`.

## Imports

- Dependencies point inward. The domain root imports no other package of its domain.
- A controller imports its root and other controllers. It imports no toolkit, no Ebitengine and no `net/http`.
- `ui` imports its root and controllers. It imports no adapter and no other `ui` package.
- An adapter imports its root, controllers and adapters of its own domain. It never imports a `ui`.
- Another domain is reached through its root package. The shared leaf packages `settings/fileversion`, `settings/userdir`, `response/textfmt` and `response/jsonview` are the written exceptions. `workspace` and `app` may also import the controllers and `ui` of other domains.
- Written exception: `settings/theme` imports `uitoolkit/core` and `uitoolkit/theming`, because theme data is toolkit-bound. Its package doc says why.
- A new package goes into a domain, or into its `ui` or `adapter` zone. No `utils`, `common` or `helpers`.

No test checks these rules. The import-layer check was removed in commit 7d856abc and is not restored (the user's choice), so a review keeps them.

## Names

- Import a domain root without an alias: `request.Request`. A local that would shadow the package is renamed, never the import: a request value is `call`.
