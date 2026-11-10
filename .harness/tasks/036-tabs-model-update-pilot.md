status: todo

# Pilot: the tab list as Model, Msg and Update

## Goal
Prove the MVU pattern on one small domain. The tab list, the active index and tab dragging leave `App` and become a model changed only by `Update`; its tests need no window and no pixels.

## Scope
- [done] `workspace/controller/tabs/list.go`: `List` (tabs, active, drag), `Msg` types (`Open`, `Close`, `CloseOthers`, `CloseRightOf`, `Activate`, `Step`, `DragBegin`, `DragOver`, `DragEnd`) and `Update(list, msg) List`. Table-driven tests per message. Differences from the first draft, for the review item: `Update` returns no commands, because list operations have none (the `App` diffs the old and new list to cancel an inline edit and drop the widgets of closed tabs); send and connection state stay methods of `Tab` (`BeginSend`, `Accept`, `Begin`, ...), which are already UI-free and tested, so wrapping them in messages would add indirection and no behavior; a marker method on `Msg` makes only this package's messages compile.
- [done] `App` holds the model (`App.list`) and calls `Update` through `App.updateTabs`; `tabs`, `active`, `dragTab` and `moveTab` left `App`. Not through the bus: every caller is on the UI goroutine and needs the new state at once (`openRequest` is followed by `a.cur()`), so a queued message would only delay it; the bus stays for what goroutines report. Added `Reset` (restore and workspace switch) and a `Step` guard for an empty list. `restore` now opens each saved tab with `Open`, so a state file naming the same request twice gets one tab (the UI cannot produce that).
- [done] `requestTabs` (`view.go`) dispatches messages; each handler is a call (`pressTab` is the one with two). `view.Tabs` was considered and not used: it draws label and badge only, with no cross, drag, middle or right click, dirty mark or colored method badge, so using it means growing the toolkit for this one strip. The tab is its own function, `requestTab`.
- [open] Review the pilot with the user: what read well, what did not. Adjust the pattern in `decisions.md` before tasks 037 and 038.

## Acceptance
- Existing flow tests stay green unchanged; new behavior has `Update` tests that failed first.
- `make check` green; `make shot` looked at; `architecture.md` updated.

## Notes
- Needs task 035.
