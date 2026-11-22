# internal

Loaded when you work under `internal/`. Adds to `.harness/rules.md`.

- `internal/<domain>/<package>`; domains `request`, `collection`, `response`, `settings`, `workspace`; zone directories `ui` (views and input), `port` (interfaces to the outside and the types that cross them), `adapter` (implementations of ports: network, files, system), `model` (plain data and rules); a controller (state and decisions, no widget and no I/O) is a package sitting directly under its domain, not a directory.
- Dependencies point inward: model imports only model; port imports model and port; controller imports model, port and controller, and neither the toolkit, Ebitengine nor `net/http`; ui imports model, port and controller, never an adapter or another ui package; adapter imports model, port, controller and the adapters of its own domain, never a ui; another domain is reached only through its model or port (`workspace` may also import their ui and controller). A new package goes into a domain (or its `ui`, `port`, `adapter`, `model` zone); no `utils`, `common` or `helpers`.
- Under way (task 154): a domain's root package `internal/<domain>` holds its types and ports and replaces its `model` and `port` packages; `response`, `workspace`, `settings` and `collection` are done, so import `internal/<domain>` without an alias.
- `internal/app` (the `App`) is the composition root and may import any package. UI-free state and decisions go to a `controller` package; a controller never imports the toolkit.
- Exception, written: `internal/settings/theme` sits directly under its domain and imports `uitoolkit/core` and `uitoolkit/theming`, because theme data is toolkit-bound; its package doc says why.
