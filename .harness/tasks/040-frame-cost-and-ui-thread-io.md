status: todo

# Cut per-frame work and move file I/O off the UI goroutine

## Goal
A frame does no work proportional to the collection size and no blocking I/O.

## Scope
- [open] `workspaceTick` marshals the whole collection to JSON every frame (`tree.Key`, `workspace.go`); `stateTick` does the same with `stateKey(a.snapshot())` (`statesync.go`). Replace both with a revision counter bumped by `Update`.
- [open] Tab `Dirty()` runs `reflect.DeepEqual` on a `Clone()` per tab per frame; use the revision of the draft (task 037).
- [open] Save of the collection, history and UI state, settings `Persist` and image decode in `refreshView` run as commands (task 035), not on the UI goroutine.
- [open] `Sender.Send` and `Ping` of WebSocket sessions (`wsui.go`) run as commands.
- [open] Build and layout the tree once per frame where possible (today `step` and `Draw` each build it); measure first and say in the commit what it saved.

## Acceptance
- A benchmark or test with a large collection shows the frame cost no longer grows with its size; tests for the revision counter written first.
- `make check` green.

## Notes
- Needs tasks 035 and 037 for the commands and the draft revision.
