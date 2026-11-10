status: todo

# Cut per-frame work and move file I/O off the UI goroutine

## Goal
A frame does no work proportional to the collection size and no blocking I/O.

## Scope
- [done] `workspaceTick`, `stateTick` and the window title no longer marshal the collection and the state every frame: `changeKeys` (`changekeys.go`) rebuilds the JSON forms only on frames where input, a message, a toast change or a window change could have changed something (user chose this over a revision counter, which needs a bump at every mutation site and loses an edit if one is missed). Saves, loads and flushes build a fresh key. Idle frames cost nothing; a frame with input still pays the marshal, so a counter stays a follow-up if that shows in a profile.
- [done] Tab `Dirty()` is compared once per stale frame, not per tab per frame: `App.tabDirty` caches the answer in `changeKeys` and the same staleness rule rebuilds it. Task 037 settled on edits as messages with no draft, so there is no draft revision to use; the comparison itself (`reflect.DeepEqual` on a clone) is unchanged and runs only on frames with input.
- [done] Save of the collection and of the UI state run as background writes (`App.write`, `serial.Queue`): the autosave clones the data on the UI goroutine, marks it saved and a Cmd writes it, one write per file at a time with the newest waiting write replacing older ones; the outcome comes back as a bus message (error toast, 'Saved', workspace remembered). Quitting and opening another collection call `waitWrites` first and then write synchronously. User chose this design.
- [done] The history save of a response is a background write too (same mechanism).
- [open] Settings `Persist` and the image decode in `refreshView` still run on the UI goroutine. Reasons: `Settings` holds maps (`Colors`, `Shortcuts`) with no deep copy, so a background write needs a `Clone` first, and the file is small and written only on a change; the image decode must hand a decoded image back to the UI goroutine for `ebiten.NewImageFromImage` and `refreshView` has callers that expect the view lines at once (find, mode switch). Both need a decision: do them (a `Settings.Clone` and a 'decoding' state of the response pane) or leave them.
- [done] WebSocket and MCP sends and pings run as commands: `App.sendInOrder` keeps a per-tab outbox so messages leave in the order they were written (a coalescing queue would drop messages), and a failure is logged in the message log. A bad hex message is still refused at once, before anything is sent. The composer is emptied when the send is queued, not when it succeeded.
- [open] Build and layout the tree once per frame where possible (today `step` and `Draw` each build it); measure first and say in the commit what it saved.

## Acceptance
- A benchmark or test with a large collection shows the frame cost no longer grows with its size; tests for the revision counter written first.
- `make check` green.

## Notes
- Needs tasks 035 and 037 for the commands and the draft revision.
