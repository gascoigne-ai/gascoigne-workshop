# Migration manifest: views to markup

Status: todo, doing, done, blocked (reason). The loop (`loop.md`) takes the first `todo` row whose toolkit task is done. Sizes are file lines.

| # | View (file, function) | Needs | Golden scenes that show it | Status |
|---|---|---|---|---|
| 1 | `settings/ui/colorpage` (whole page) | 002 | settings-colors | done (8cc3cfc pins, aef8636 swap) |
| 2 | response `sentlog` list | 002 | sent-log (added) | done (bd1b7c4 scene, 3da5868 pins, 979d6f3 swap) |
| 3 | settingsui open font list (`fontlist.go`, `fontrow*.vml`) | 002, 003 | none reachable (pinned by tests only) | done (a58d9bc pins, acad305 swap) |
| 4 | settingsui `settingRow` / `form.go`, plus `fontControl` and `fontLabel` from row 3 | 002, 003, 005 | settings-menu, settings-requests, settings-general | done (268fdf1 pins, ed2f230..75760a3 swaps) |
| 5 | app `mcpexplorer.go` `explorerList`, open entry, result (`serverInfo` stays Go: two roots, `Lines`) | 007 | request-mcp (not connected only) | done (0b0b592 pins, 9c88441 swap) |
| 6 | app `env.go` `envChip` (menu stays Go: data only) | 002, 003 | default (No environment only) | done (b42315d pins, 0fcfddd swap) |
| 7 | app `panes.go` `findBar` | 002, 005 | response-find (added) | done (07d023c scene, 6b3153d pins, 88b3b43 swap) |
| 8 | app `view.go` `sidebar`, `treeRow`, `rowName` | 004, 007 | default, tree-filter (added) | done (6d6b256 scene, beb68d7 pins, 8c20a14 swap) |
| 9 | app `view.go` `urlBar` | 003, 005, 006 | default, request-*, request-sending (added) | done (5166f7a scene, 572910c pins, 9381dd7 swap) |
| 10 | app `view.go` `requestTabs`, `requestTab` | 003, 004 | request-*, request-tabs (added) | done (fc7912e scene, 0b32e8c pins, aaabb32 swap) |
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
