status: todo

# Organize internal packages by domain, then by role

## Goal
`internal/` is grouped first by domain (what the code is about), and inside a domain by logical role (what the code does), so a reader finds a feature in one place and sees which parts draw, which decide and which talk to the outside.

## Rule (also in `.harness/rules.md`)
- Top level of `internal/`: one directory per domain, for example `request` (model, send, protocols), `collection` (tree, store, import/export), `response` (views, history), `settings`, `workspace` (tabs, state), `environment`.
- Inside a domain, directories by role:
  - `ui`: views, widgets and layout of the domain (drawing and input only, no I/O).
  - `controller` (or `app`): state, handlers and decisions; turns user actions into calls on ports.
  - `port`: the interfaces the domain needs from the outside (file system, network, clipboard, dialogs, clock) and the types that cross them.
  - `adapter` (communication): implementations of ports (HTTP, gRPC, WebSocket, MCP, files, system dialogs).
  - `model`: plain data and rules with no I/O and no UI.
- Dependencies point inward: `ui` and `adapter` depend on `controller`/`port`/`model`, never the other way; `model` imports nothing of the module; domains talk through ports or small public APIs, not through each other's internals. `uitoolkit` stays outside `internal` and knows nothing of any domain.
- A package has one role; a package named after a role alone (`utils`, `common`, `helpers`) is not allowed.

## Scope
- Decide the domain list and the role directory names with the user (the names above are a proposal; ask before moving anything, see the Ambiguity rule). Record the decision in `decisions.md` and replace decision 17's scheme where it overlaps.
- Write the dependency rules as a test next to `uitoolkit/core/layers_test.go` (for example `internal/layers_test.go`): `model` imports no `ui`/`adapter`, `ui` imports no `adapter`, no domain imports another domain's `ui` or `adapter`.
- Move one domain per commit, mechanical first (move and rename, fix imports, no behavior change, no assertion changed), then split packages that mix roles. Candidates today: `internal/ui` mixes ui, controller and ports (clipboard, file dialog, browser, clock seams); `internal/proto` is adapters plus the protocol interface (port); `internal/store`, `internal/history`, `internal/state` are file adapters of ports that do not exist yet; `internal/respview`, `internal/jsonview`, `internal/textfmt` are response ui and model.
- Folded in from task 032 (user decision 2026-11-07): the remaining `internal/ui` features that can leave the `App` behind a `Host` interface, each placed in its domain's `ui` role while it moves: MCP tool form (`mcpform.go`), GraphQL schema menu, gRPC method picker, environment chip and its dialogs, colors page, shortcut editor (settings and the shortcut table become host methods). Tabs, panes, tree and workspace stay in the `App`. Template conversions of 032 are done (Log title bar, sent rows, settings title bar and navigation).
- Turn the test seams that are now fields (`App.openBrowser`, `chooseFile`, `hasFileDialog`, `now`) into explicit ports with adapters, so the controller is tested with fakes.
- Update `architecture.md`: the tree, a Mermaid diagram of domains and roles with the allowed dependency directions (checked with `mermaid.parse`), and the package table.

## Acceptance
- Every step keeps `make check` green; no behavior change and no test assertion changed (imports and names only).
- The layer test fails when a role rule is broken (add a case that proves it fails for a deliberate violation).
- `architecture.md` and `decisions.md` describe the new layout and the rule.

## Notes
- Do this after task 032 (templates and further splits) or fold 032's splits into it: both move code out of `internal/ui`, and doing them apart doubles the churn.
- Large: plan the order by dependency (model first, then ports and adapters, then controllers, then ui) so each step compiles.
