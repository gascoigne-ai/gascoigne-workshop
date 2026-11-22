# Architecture

Start with `overview.md`; this file is the index of the detail. The layout follows the shared
`common/architecture-style.md`: each domain's root package (types and ports) is its core, `adapter` and `ui` its adapters,
the `App` its composition root. Known gaps are under Limits.

```mermaid
flowchart LR
  cmd["cmd/workshop"] --> app["internal/app"]
  app --> ui["domain ui"] & ctl["domain controllers"] & adp["domain adapter"]
  ui --> ctl --> root["domain root"]
  adp -. implements .-> root
  subgraph toolkit["uitoolkit (knows nothing of the app)"]
    markup["markup"] --> widget["widget"]
    markup --> view["view"]
    widget --> view
    view --> core["core"]
    engine["engine"] --> core
    theming["theming"] --> core
  end
  app & ui --> markup & widget & view
  app --> engine & theming
```

## Package layout

- Packages are `internal/<domain>/<package>`; only `ui/` and `adapter/` are zone directories. The import
  rules and their written exceptions are in `internal/CLAUDE.md`; no test checks them.
- `internal/app` is one package: Go cannot declare a method in another package, and everything
  that reads or writes the one `App` struct needs its methods. Adapters come in through `Options`.
- A feature that needs little of the `App` is its own `ui` package behind a `Host` interface the
  `App` implements, tested with a fake host: `calltrace` (`feature/send/ui/calltrace`), `mcpform` (`feature/send/ui/mcpform`), `sentlog` (`feature/history/ui/sentlog`), `settingsui`,
  `formpage`, `colorpage`, `shortcutpage`, `systempage`, `destpicker`. Adapters reach those views through the
  host (`Host.SentStore`, `Host.FontStore`).

## Modules

One functional module per file in `architecture/`; read only the one you touch. A module is a
domain directory under `internal/` (root, controllers, `ui/`, `adapter/`) or the toolkit.

| File | Module | Owns |
|---|---|---|
| `internal/feature/send/FEATURE.md` | `internal/feature/send`, `internal/feature/request` | the request model and ports (kernel), protocols (rest, websocket, sse, mcp, grpc), assert, editor, mcpexplorer, response viewer and views, calltrace, mcpform, browser preview |
| `internal/feature/collections/FEATURE.md` | `internal/feature/collections` | open set, tree, importers, transfer, store, client config, sample and example server |
| `internal/feature/environments/FEATURE.md` | `internal/feature/environments` | environments, variables resolution, the chip's menu |
| `internal/feature/history/FEATURE.md` | `internal/feature/history` | responses kept per request, sent log |
| `internal/feature/settings/FEATURE.md` | `internal/feature/settings` | schema, prefs, theme, settings dialog and its pages |
| `internal/feature/workspace/FEATURE.md` | `internal/feature/workspace` | tabs, remembered state, bus, file browser, autosave, overlays, app icon, user's files |
| `internal/ui/format` (see `internal/CLAUDE.md`) | `internal/ui/format` | display formatting (was `kernel/textfmt` and `kernel/jsonview`); documented in its package doc and in the module files that use it |
| `architecture/app.md` | `internal/app` | composition root: the `App`, its handlers and window |
| `architecture/toolkit.md` | `uitoolkit` | core, view, widgets, markup API and drawing |
| `architecture/persistence.md` | shared | formats and versions of every persisted file |
| `architecture/limits.md` | shared | limits, fuzzing, known gaps |

## Module dependencies

```mermaid
flowchart LR
  app["app"] --> send & collections & environments & settings & workspace
  app --> toolkit["uitoolkit"]
  collections & environments & send & settings & workspace --> base["base (request model, storage)"]
```

Every edge between features is a base package (the request model included); `go list` shows no
other one, except the `exampleserver` dev tool. `e2e/layers_test.go` checks it in `make check`.

## Known layer gaps

- A domain root imports a sub-package of its own domain: `feature/workspace` imports `feature/workspace/state`.
- `internal/exampleserver` imports `feature/send/proto/ws`; the one written exception in `internal/CLAUDE.md`.
- Six test-only rows in `e2e/layers_rules_test.go` let tests import real adapters as fixtures:
  `exampleserver` (`feature/collections`, `feature/send/proto`), `history/ui/sentlog` (`history/adapter`),
  `settings/ui/formpage` (`uitoolkit/desktop/fonts`, `settings/adapter/settingsfile`), `settings/ui/settingsui`
  (`settings/adapter/settingsfile`). Follow-up: replace the real adapters with fakes and delete the rows.
