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
  `App` implements, tested with a fake host: `calltrace`, `mcpform`, `sentlog`, `settingsui`,
  `colorpage`, `shortcutpage`, `systempage`, `destpicker`. Adapters reach those views through the
  host (`Host.SentStore`, `Host.FontStore`).

## Modules

One functional module per file in `architecture/`; read only the one you touch. A module is a
domain directory under `internal/` (root, controllers, `ui/`, `adapter/`) or the toolkit.

| File | Module | Owns |
|---|---|---|
| `architecture/request.md` | `internal/request` | model, ports, protocols (rest, websocket, sse, mcp, grpc), assert, editor, calltrace, mcpform |
| `architecture/collection.md` | `internal/collection` | tree, environments, importers, transfer, store, client config, example server |
| `architecture/response.md` | `internal/response` | viewer, json view, text formats, history, preview, sent log |
| `architecture/settings.md` | `internal/settings` | schema, prefs, theme, user dir, file versions, system adapters, settings dialog |
| `architecture/workspace.md` | `internal/workspace` | tabs, remembered state, bus, file browser, autosave, overlays, platform adapters |
| `architecture/app.md` | `internal/app` | composition root: the `App`, its handlers and window |
| `architecture/toolkit.md` | `uitoolkit` | core, view, widgets, markup API and drawing |
| `architecture/persistence.md` | shared | formats and versions of every persisted file |
| `architecture/limits.md` | shared | limits, fuzzing, known gaps |

## Module dependencies

```mermaid
flowchart LR
  app["app"] --> request & collection & response & settings & workspace
  app --> toolkit["uitoolkit"]
  collection --> request
  response --> request
  workspace --> request
  request -. "textfmt, workspace" .-> response
  settings -. "jsonview, workspace" .-> response
  collection -. "collections" .-> settings
  response -. "fileversion, userdir" .-> settings
  workspace -. "fileversion, userdir" .-> settings
  settings -. "prefs" .-> workspace
```

Solid edges are the normal direction (a domain root reached through its root). Dotted edges are
the written exceptions in `internal/CLAUDE.md`. Nothing checks them yet.

## Known layer gaps

- A domain root imports a sub-package of its own domain: `collection` imports `collection/clientconfig`,
  `workspace` imports `workspace/state`. Only `settings/fileversion` is a written exception.
- `settings/prefs` imports `workspace`, and `workspace` imports `settings` leaves: the two
  domains are cyclic at domain level.
- `internal/exampleserver` imports `request/adapter/proto/ws`; no rule covers it.
- No layer check runs in `make check` (task `004-layer-check`).
