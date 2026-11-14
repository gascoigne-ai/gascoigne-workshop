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
| 11 | app `panes.go` `tableView` | 004 | default, request-post, request-tabs, request-params (added) | done (b4dae74 shot fix, eb6009d scene, 3eacdb7 pins, 456e15b swap) |
| 12 | app `panes.go` `codePane` | 005 | response | n/a: dead code since ec3c902, removed |
| 13 | app `panes.go` `responsePane` | 004, 005 | response, side-by-side | done (cc8641e scene, b9c2d81 pins, 416aee4 swap) |
| 14 | app `panes.go` `settingsPane` | 005 | request-* settings tab | done (61a7fa4 scene, f124dcd pins, ec8e4cb swap) |
| 14b | app `auth.go` auth pane (builds FormRow/CheckRow in Go; missing from the first manifest) | 005 | none: add a scene (`-pane Auth`) | done (a20665c scene, 2e95951 pins, 9e900b9 swap) |
| 15 | app `wsui.go` `messagesPane` | 004, 005 | request-websocket (idle only) | done (bfe231b pins, 40c6fa6 swap; no new scene, request-websocket+sse cover the layouts) |
| 15b | app `wsbar.go` `messagesToolbar`: inline `markup.View` parsed every frame, literal sizes; make it an embedded `.vml` with sizes via scope (wsbar_test.go compares with a Go view; keep it) | 003 | request-websocket | done (6e03ff3 pins, 0e10420 swap) |
| 15c | inline `markup.View` sources: app `view.go` `header` (:214) and `statusBar` (:456), settingsui `dialog.go` `navItem` (:197, two string-constant templates) -> embedded `.vml` via MustLoad, sizes via scope | 003 | default, settings-* | done (d99044b header, 14b3f5a+450442b status bar, af40b23+f481bc1 nav) |
| 16 | app `overlay.go` `dialogView`, `toastsView`, `dialogForm` | 004 | settings-*; toasts: add scene first | done (98bc7a3 scenes, 4067a7c pins, f8e5c85 swap; dialogForm stays: returns the mcpform view, row 19) |
| 17 | settingsui `dialog.go` frame, `raw.go` | 004, 005 | settings-menu | done (63c1a45 scene, 6a244af pins, 6668274 raw swap, 08d33d9 frame swap, 549694a resize pin) |
| 18 | `shortcutpage` | 004 | settings-shortcuts | done (a3df737 scene, 868ac45 pins, 7b44541 swap) |
| 19 | `request/ui/calltrace`, `mcpform` | 004, 005 | none: add scene first | done (0e7f17a+a43027e scenes, 117446c+933fd10 calltrace, f531f0b+10789ba mcpform) |
| 20 | `response/ui/respview` | 004, 005 | response | n/a: data only (CodeView lines and spans, Timeline text, filename helpers); its views are `responsePane`, row 13 |
| 21 | `workspace/ui/appicon` | none | default | n/a: draws the window icon bitmap, no view tree |
| 22 | app `filebody.go` (file body pane), `graphql.go` (GraphQL pane), `assertpane.go` (assertions pane) | 004, 005 | request-* panes (`-pane`); add scenes for uncovered | todo |
| 23 | app `panes.go` `messagePane`, `notePane`, body pane (~l.360 Format button) | 004 | default, request-post | todo |
| 24 | app `mcpexplorer.go` remaining Go views (explorer layout, list row, detail, ~l.104-121, :218) | 004 | request-mcp (+connected: MCP server on 8084) | todo |
| 25 | app `view.go` window root (~l.52), main column (~l.191), response status (~l.417-433) | 003, 004 | default, response | todo |

Rows 22-25 were found by the final grep (first manifest missed them); after row 25 re-run `grep -rEn "\b(Row|Col|Stack|Scroll|Table|Tabs|Field|Button)\(" internal --include=*.go | grep -v _test.go` and add rows for anything left except `Custom` behaviors and `Frag` glue.

Rule: a row with "none: add scene first" starts by adding the golden scene (and the shot flag it needs), test first, in its own commit.
