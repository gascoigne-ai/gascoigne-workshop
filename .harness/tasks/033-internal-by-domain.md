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
- [open] Split the `App` (`workspace/ui/app`) into a `controller` and views (decision 31). Done: `workspace/controller/tabs` (`Tab`, tab strip rules, `Connection`, `RowHistory`, earlier responses), `autosave` (`Debounce`) and `toasts` (`Stack`), and the layer test rejects a toolkit import from a controller. Left, each a handler that calls `prompt`, `confirm`, `openMenu`, `toast` or reads widget state: tree and workspace operations (`treeops.go`, `workspace.go`: rename, duplicate, delete, drag, open), the find bar and response view mode in `panes.go` (they drive a `CodeView`), and the dialog and menu state in `overlay.go`. Needs the user to say whether these also move or the item closes here.

## Acceptance
- Every step keeps `make check` green; no behavior change and no test assertion changed (imports and names only).
- The layer test fails when a role rule is broken (add a case that proves it fails for a deliberate violation).
- `architecture.md` and `decisions.md` describe the new layout and the rule.

## Notes
- Task 032 was folded into this one (user decision, 2026-11-07); its template conversions are done.
- Large: plan the order by dependency (model first, then ports and adapters, then controllers, then ui) so each step compiles.
