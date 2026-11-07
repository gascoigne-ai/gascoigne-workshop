status: todo

# Templates and further splits of internal/ui

## Goal
Finish what task 029 left: views written as templates where that fits, and the remaining features of `internal/ui` that can move behind a `Host`.

## Scope
- Templates: where a feature's UI is mostly static (values, lists, click handlers, slots), write it as a `component.View` template and keep the Go view as a twin test until they match, as the header and status bar already do. Candidates: the Log dialog's title bar, the sent list rows, the settings dialog's title bar and navigation. Keep Go views where logic dominates (decision 14). Ask before converting a view that is not clearly static.
- Further splits, one feature per commit with a `Host` interface (decision 17), if each still pays off: the font pickers are already in `settingsui`; candidates are the MCP tool form (`mcpform.go`), the GraphQL schema menu and the gRPC method picker, the environment chip and its dialogs, the colors page and the shortcut editor (they need the settings and the shortcut table as host methods). Features that read and write most of the `App` (tabs, panes, tree, workspace) stay.

## Acceptance
- No behavior change and no assertion changed; each split or conversion keeps `make check` green.
- `architecture.md` lists each new package in the diagram and the table.
