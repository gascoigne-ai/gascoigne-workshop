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

## Design (Opus, 2026-11-23; read before building)
- `tabs.Set` in `internal/feature/workspace/tabs/set.go`, one unexported field `list List`; `Current()` is computed from
  `list.Active` (nil only before the first Open and after Reset; closing the LAST tab leaves a blank "New Request" tab,
  pinned by `TestUpdateClosingTheLastTabLeavesABlankRequest`: the task text "close last leaves nil" is wrong, keep the
  blank tab). `List`, `Update`, Msg types stay pure and exported; `Set.Apply` wraps `Update`.
- API: `NewSet()` (Drag -1), `Tabs() []*Tab`, `Current() *Tab`, `Active() int` (0 when empty), `Drag() int`,
  `Open(req)`, `Close(i)`, `Apply(msg)`. `Active` and `Drag` are needed so test rewrites stay access-only.
- Widget sync stays in `App.updateTabs` (the only caller of Set mutators in app): `was := a.activeRequest()` before
  Apply; `a.tabs.Apply(msg)`; make missing widgets FIRST (`restore` calls `cur()` right after Open); prune widgets of
  tabs no longer in `Tabs()` (cancelEdit, cancelSend, disconnect, delete); reveal if the current request changed.
  Verify first that `cancelSend` and `disconnect` do not read `a.list`, `a.cur()` or `openTabs()`; if they do, have Apply
  return the closed tabs and prune before the list changes. Rejected: change callback, diff-then-Apply (mints different
  `*Tab` pointers), per-frame pruning (one frame late).
- App field `a.tabs *tabs.Set` (a field does not shadow the package; check no local named `tabs`). About 21 production
  rewrites; sed `\.list\.Tabs\b` -> `.tabs.Tabs()`, `\.list\.Active\b` -> `.tabs.Active()`, `\.list\.Drag\b` ->
  `.tabs.Drag()` over all receivers (`a.`, `app.`, `d.app.`); by hand: `updateTabs`, `NewApp`, `app.go` field, `cur`,
  `activeRequest`. Run `rg -n '\.list\b' internal/app` first for the full list. `settings_shortcuts.go` uses `app.list`.
- `sidebar` Requests.Show keeps going through `App.openRequest` (it does the widget sync and tree select).
- Layers: controller imports only base: no new row. appsize: no change.
- Tests first (stubs so they fail on assertions): TestSetOpenAddsATabAndMakesItCurrent, TestSetOpenOfAnOpenRequest
  FocusesItsTab, TestSetCloseCurrentPicksItsNeighbour (abcd/1/close 1: c; abc/2/close 2: b; abcd/2/close 0: c),
  TestSetClosingTheLastTabLeavesABlankCurrentTab, TestNewSetDragsNothing, TestSetApplyMatchesUpdate (table),
  TestSetCurrentIsAlwaysAMember. Then access-only rewrites of the app tests (request_tabs_flow_test.go reads
  a.list.Tabs/Active/Drag about 30 times).
- Docs: architecture/app.md "Tab list", workspace/FEATURE.md.
