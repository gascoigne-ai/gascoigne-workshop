status: todo
kind: refactor
commit: refactor(app): one importer flow and one request lookup
scope: internal/workspace/*,internal/collection/*
depends_on: 112-mono-grid

# Remove the duplicated flows in the app

Origin: the repo audit of 2026-11-19 (three read-only reviews against the harness rules). The user approved all phases and, with that, the deletions and the gate change named here. Behavior stays the same unless the task says otherwise; tests stay green and none is loosened. Update `architecture.md` in the same commit where it describes what changes.

- The three clipboard importers in `treeops.go` (cURL, OpenAPI, Postman) become one `importFolder(dir, kind, parse)` with a three-row table.
- "Find a request by path or name" exists four times (`statesync.go` `requestPath` and `requestAt`, `ctxmenus.go` `OpenRequest`, `shot.go` `findRequestByName`): `tree.PathOf(root, req)` and `tree.At(root, path)` in the collection controller, with `walkFolders` / `walkRequests`; `snapshot` stops looking each owner up twice.
- Two "open a collection" paths (`workspace.go` `openWorkspaceAt`, `openset.go`): `openWorkspaceAt` closes all then calls `addCollection`; `startOpen` is `openAt("")`; name the sample's start request in the collection model instead of `Folders[1].Requests[0]`.
- Composer reset written five times in `wsui.go`: one `setComposer`; clone-and-rename twice in `ctxmenus.go`; the duplicated `Disabled` rule; derive `overlayOpen` from `openLayers`.
Acceptance: new tests for `PathOf`/`At` first; `make check` green.
