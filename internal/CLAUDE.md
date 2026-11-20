# internal

Loaded when you work under `internal/`. Adds to `.harness/rules.md`.

- `internal/<domain>/<role>/<package>`; domains `request`, `collection`, `response`, `settings`, `environment`, `workspace`; roles `ui` (views and input), `controller` (state and decisions), `port` (interfaces to the outside and the types that cross them), `adapter` (implementations of ports: network, files, system), `model` (plain data and rules). Decision 11 records the scheme and the package map.
- Dependencies point inward and are enforced by `internal/layers_test.go`: model imports only model; port imports model and port; controller imports model, port and controller, and neither the toolkit, Ebitengine nor `net/http`; ui imports model, port and controller, never an adapter or another ui package; adapter imports model, port, controller and the adapters of its own domain, never a ui; another domain is reached only through its model or port (`workspace`, the composition root, may also import their ui and controller). A new package goes into a domain and a role; no `utils`, `common` or `helpers`.
- `workspace/ui/app` (the `App`) is the composition root; the layer test has no exceptions. UI-free state and decisions go to a `controller` package (decision 11); a controller never imports the toolkit.
- The layers are enforced by `internal/layers_test.go` and `uitoolkit/core/layers_test.go`, inside `go test` in `make check`.
