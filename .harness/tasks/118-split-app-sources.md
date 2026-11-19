status: todo
kind: refactor
commit: refactor(app): split app, panes, treeops and ctxmenus by responsibility
scope: internal/workspace/ui/app/*
depends_on: 117-svg-tables

# Split the large app files

Origin: the repo audit of 2026-11-19 (three read-only reviews against the harness rules). The user approved all phases and, with that, the deletions and the gate change named here. Behavior stays the same unless the task says otherwise; tests stay green and none is loosened. Update `architecture.md` in the same commit where it describes what changes.

Pure moves, no behavior change.
- `app.go` (724 lines): `tab.go` (reqTab, makeTab, updateTabs, newReqTab), `options.go` (Options, NewApp), `send.go` (send, dispatch, cancelSend, timeoutFor, handle, apply), `game.go` (Layout, Update, step, Draw); chip handlers beside the URL bar.
- `panes.go` (877): keep the registry; `tablepane.go`, `bodypane.go`, `responsepane.go`, `requestsettings.go`; move `requestPanesFor` from `wsui.go`.
- `treeops.go`: `treemenu.go`, `treedrag.go`, `import.go`; `textFocused` and `sidebarWidth` go where they belong. `ctxmenus.go`: `hosts.go`, `cookies.go`.
- `view.go` (651): split to match its test files (`sidebar`, `urlbar`, `requesttabs`, `window`).
- Update `architecture.md` and `packagemap.md` if file names are listed.
Acceptance: `git diff -M` shows moves only; `make check` green.
