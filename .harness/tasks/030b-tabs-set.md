status: todo
kind: refactor
commit: refactor(tabs): tabs.Set owns the tab list and the current tab
scope: internal/**, cmd/**, e2e/**, Makefile, .harness/**
not_before: 2026-01-01
depends_on:

# feature/workspace/tabs.Set

Goal: second half of task 030 (split by the orchestrator after an Opus design; the sidebar part is done).
`a.list` (tabs.List) is read in many places in `internal/app`: `cur`, `activeRequest`, `openTabs`, `requesttabs`,
`tabMenuItems`, `focus`, `closeTabsIn`, `updateTabs` and many tests (`len(a.list.Tabs)`). Add `tabs.Set` that owns
`List` and keeps `tabs.List` and `Update`: `Tabs()`, `Current() *Tab` (nil when empty), `Open(req)`, `Close(i)`,
`Apply(Msg)`. A `Set` called directly would skip the `widgets` sync in `updateTabs` and `cur()` would hit a nil tab:
keep that sync in one place (the designer decides where: a Set callback or the App wrapper) and say so. Then
`sidebar.Deps.Requests.Show` can call the Set. Follow `.harness/reorg.md`.

## Model
`Set`: owner of the open tabs and the current one; invariant: current is nil or a member; a slice (order is the
strip order, sizes are tens), no map. Rejected: more fields on `App`.

## Acceptance
- Test first: `Set` unit tests (open, close current, close last, current stays a member, apply messages).
- Access-only rewrites of the app tests that read `a.list`.
- Common acceptance of `.harness/reorg.md`; cap lowered if files leave `internal/app`.
