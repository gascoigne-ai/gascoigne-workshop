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
- [done] Domain list and role names agreed with the user (decisions 19, 20).
- [done] Layer test `internal/layers_test.go`: pure rules function, a table that proves each violation kind is caught, the real tree checked, and an `exceptions` list.
- [done] Every package moved into `domain/role` (decision 20); cross-domain edges other than the `App` removed (`Skeleton` and MCP helpers to `request/model`, `SentStore` and `FontStore` ports, `fontfamily`, `jsonview.Kind.Style`).
- [done] `architecture.md` has the tree rules, the Mermaid diagram (checked with `mermaid.parse` under jsdom) and the package table.
- [open] Split the `App` (`workspace/ui/app`, about 14k lines) into controller and views behind ports until the 4 remaining `exceptions` in the layer test are gone: `request/adapter/proto`, and the other domains' views (`calltrace`, `respview`, `sentlog`, `settingsui`). Order: ports and fakes for the file and network adapters, then the controller, then the views. Blocked by size, not by a decision; unblocked by a session per feature.
- [open] Features that can leave the `App` behind a `Host` (from task 032), each into its domain's `ui` role: MCP tool form (`mcpform.go`), GraphQL schema menu, gRPC method picker, environment chip and its dialogs (this creates the `environment` domain), colors page, shortcut editor. Tabs, panes, tree and workspace stay in the controller.
- [done] Test seams `openBrowser`, `chooseFile`, `hasFileDialog` are ports (decision 21).
- [open] The `now` clock seam into a port.
- [open] `settings/model/settings` reads and writes the settings file: split the file part into a port and an adapter.

## Acceptance
- Every step keeps `make check` green; no behavior change and no test assertion changed (imports and names only).
- The layer test fails when a role rule is broken (add a case that proves it fails for a deliberate violation).
- `architecture.md` and `decisions.md` describe the new layout and the rule.

## Notes
- Task 032 was folded into this one (user decision, 2026-11-07); its template conversions are done.
- Large: plan the order by dependency (model first, then ports and adapters, then controllers, then ui) so each step compiles.
