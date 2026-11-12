# Migration manifest: views to markup

Status: todo, doing, done, blocked (reason). The loop (`loop.md`) takes the first `todo` row whose toolkit task is done. Sizes are file lines.

| # | View (file, function) | Needs | Golden scenes that show it | Status |
|---|---|---|---|---|
| 1 | `settings/ui/colorpage` (whole page) | 002 | settings-colors | done (8cc3cfc pins, aef8636 swap) |
| 2 | response `sentlog` list | 002 | none (log panel): add scene first | todo |
| 3 | settingsui `fontlist`, `fontRow` | 002, 003 | settings-menu | todo |
| 4 | settingsui `settingRow` / `form.go` | 002, 003, 005 | settings-menu, settings-requests, settings-general | todo |
| 5 | app `mcpexplorer.go` `explorerList` | 002, 003 | request-mcp (not connected only) | todo |
| 6 | app `env.go` `envChip` | 002, 003 | default | todo |
| 7 | app `panes.go` `findBar` | 002, 005 | none: add scene first | todo |
| 8 | app `view.go` `sidebar`, `treeRow`, `rowName` | 003, 004 | default | todo |
| 9 | app `view.go` `urlBar` | 003, 005, 006 | default, request-* | todo |
| 10 | app `view.go` `requestTabs`, `requestTab` | 003, 004 | request-* | todo |
| 11 | app `panes.go` `tableView` | 004 | default, request-post | todo |
| 12 | app `panes.go` `codePane` | 005 | response | todo |
| 13 | app `panes.go` `responsePane` | 004, 005 | response, side-by-side | todo |
| 14 | app `panes.go` `settingsPane` | 005 | request-* settings tab | todo |
| 15 | app `wsui.go` `messagesPane` | 004, 005 | request-websocket (idle only) | todo |
| 16 | app `overlay.go` `dialogView`, `toastsView`, `dialogForm` | 004 | settings-*; toasts: add scene first | todo |
| 17 | settingsui `dialog.go` frame, `raw.go` | 004, 005 | settings-menu | todo |
| 18 | `shortcutpage` | 004 | settings-shortcuts | todo |
| 19 | `request/ui/calltrace`, `mcpform` | 004, 005 | none: add scene first | todo |
| 20 | `response/ui/respview` | 004, 005 | response | todo |
| 21 | `workspace/ui/appicon` | none | default | todo |

Rule: a row with "none: add scene first" starts by adding the golden scene (and the shot flag it needs), test first, in its own commit.
