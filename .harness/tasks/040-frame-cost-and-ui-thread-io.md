status: todo

# Cut per-frame work and move file I/O off the UI goroutine

## Goal
A frame does no work proportional to the collection size and no blocking I/O.

## Scope
- [done] `workspaceTick`, `stateTick` and the window title no longer marshal the collection and the state every frame: `changeKeys` (`changekeys.go`) rebuilds the JSON forms only on frames where input, a message, a toast change or a window change could have changed something (user chose this over a revision counter, which needs a bump at every mutation site and loses an edit if one is missed). Saves, loads and flushes build a fresh key. Idle frames cost nothing; a frame with input still pays the marshal, so a counter stays a follow-up if that shows in a profile.
- [done] Tab `Dirty()` is compared once per stale frame, not per tab per frame: `App.tabDirty` caches the answer in `changeKeys` and the same staleness rule rebuilds it. Task 037 settled on edits as messages with no draft, so there is no draft revision to use; the comparison itself (`reflect.DeepEqual` on a clone) is unchanged and runs only on frames with input.
- [open] Save of the collection, history and UI state, settings `Persist` and image decode in `refreshView` run as commands (task 035), not on the UI goroutine.
- [open] `Sender.Send` and `Ping` of WebSocket sessions (`wsui.go`) run as commands.
- [open] Build and layout the tree once per frame where possible (today `step` and `Draw` each build it); measure first and say in the commit what it saved.

## Acceptance
- A benchmark or test with a large collection shows the frame cost no longer grows with its size; tests for the revision counter written first.
- `make check` green.

## Notes
- Needs tasks 035 and 037 for the commands and the draft revision.
