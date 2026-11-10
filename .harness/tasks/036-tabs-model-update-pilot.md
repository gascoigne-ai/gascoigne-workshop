status: todo

# Pilot: the tab list as Model, Msg and Update

## Goal
Prove the MVU pattern on one small domain. The tab list, the active index and tab dragging leave `App` and become a model changed only by `Update`; its tests need no window and no pixels.

## Scope
- [done] `workspace/controller/tabs/list.go`: `List` (tabs, active, drag), `Msg` types (`Open`, `Close`, `CloseOthers`, `CloseRightOf`, `Activate`, `Step`, `DragBegin`, `DragOver`, `DragEnd`) and `Update(list, msg) List`. Table-driven tests per message. Differences from the first draft, for the review item: `Update` returns no commands, because list operations have none (the `App` diffs the old and new list to cancel an inline edit and drop the widgets of closed tabs); send and connection state stay methods of `Tab` (`BeginSend`, `Accept`, `Begin`, ...), which are already UI-free and tested, so wrapping them in messages would add indirection and no behavior; a marker method on `Msg` makes only this package's messages compile.
- [open] `App` holds the model and calls `Update` through the bus (task 035); `tabs`, `active`, `dragTab` and `moveTab` leave `App`.
- [open] `requestTabs` (`view.go`) dispatches messages; its handlers are one-liners. Consider `view.Tabs` instead of the hand-built strip.
- [open] Review the pilot with the user: what read well, what did not. Adjust the pattern in `decisions.md` before tasks 037 and 038.

## Acceptance
- Existing flow tests stay green unchanged; new behavior has `Update` tests that failed first.
- `make check` green; `make shot` looked at; `architecture.md` updated.

## Notes
- Needs task 035.
