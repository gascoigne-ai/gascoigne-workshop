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
  `App` implements, tested with a fake host: `calltrace`, `mcpform`, `sentlog` (`feature/history/ui/sentlog`), `settingsui`,
  `colorpage`, `shortcutpage`, `systempage`, `destpicker`. Adapters reach those views through the
  host (`Host.SentStore`, `Host.FontStore`).

## Modules

One functional module per file in `architecture/`; read only the one you touch. A module is a
domain directory under `internal/` (root, controllers, `ui/`, `adapter/`) or the toolkit.

| File | Module | Owns |
|---|---|---|
| `architecture/request.md` | `internal/kernel/request`, `internal/request` | model, ports, protocols (rest, websocket, sse, mcp, grpc), assert, editor, calltrace, mcpform |
| `internal/feature/collections/FEATURE.md` | `internal/feature/collections` | open set, tree, importers, transfer, store, client config, sample and example server |
| `internal/feature/environments/FEATURE.md` | `internal/feature/environments` | environments, variables resolution, the chip's menu |
| `architecture/response.md` | `internal/response` | viewer, preview |
| `internal/feature/history/FEATURE.md` | `internal/feature/history` | responses kept per request, sent log |
| `internal/feature/settings/FEATURE.md` | `internal/feature/settings`, `internal/platform` | schema, prefs, theme, settings dialog and its pages; the OS adapters for display scale, fonts, key repeat and system info |
| `architecture/workspace.md` | `internal/workspace` | tabs, remembered state, bus, file browser, autosave, overlays, platform adapters |
| `internal/kernel` (see `internal/CLAUDE.md`) | `internal/kernel` | shared leaves: `severity`, `fileversion`, `userdir`, `textfmt`, `jsonview`, `openset`; documented in their package docs and in the module files that use them |
| `architecture/app.md` | `internal/app` | composition root: the `App`, its handlers and window |
| `architecture/toolkit.md` | `uitoolkit` | core, view, widgets, markup API and drawing |
| `architecture/persistence.md` | shared | formats and versions of every persisted file |
| `architecture/limits.md` | shared | limits, fuzzing, known gaps |

## Module dependencies

```mermaid
flowchart LR
  app["app"] --> request & collections & environments & response & settings & workspace
  app --> toolkit["uitoolkit"]
  collections & environments --> request
  response --> request
  workspace --> request
  collections & environments & response & settings & workspace --> kernel["kernel (shared leaves)"]
  request --> kernel
```

Every edge between the five domains is either the `request` model or a kernel leaf; `go list` shows no
other one. Nothing checks it in `make check` yet.

## Known layer gaps

- A domain root imports a sub-package of its own domain: `workspace` imports `workspace/state`. 
- `internal/exampleserver` imports `request/adapter/proto/ws`; no rule covers it.
- No layer check runs in `make check`; `common/scripts/check-layers.sh` exists and is not wired
  (it needs a `.harness/layers.txt`).
