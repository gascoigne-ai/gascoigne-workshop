status: todo

# Response, collection tree and settings move to Model, Msg and Update

## Goal
The rest of the `App` decisions follow the pattern of task 036, domain by domain, until `App` only owns models, the bus and the frame loop.

## Scope
- [done] Collection tree: rename, duplicate, delete, drag, new request/folder as `tree` messages over typed `Node`s, applied in place (tabs hold pointers into the collection). `model.Folder` operations still take `any`; retyping them is left for a later cleanup. Imports and keyboard selection stay App methods.
- [done] Response view: mode, find bar and active match are `viewer.Model` with `Update` (`reqTab.look`). The lines, the find text and the compact filter stay widget state because the controller cannot import the toolkit; the App passes match counts in.
- [done] Settings: the `Host` of `settingsui` keeps its reads (12 methods) and every command is a `prefs` message through `Send` (user chose reads + messages over a pure Update); `pageHost` is gone, the App is the host of the Colors and Shortcuts pages too.
- [done] Escape chain: one ordered `overlay.Layer` list and `Open.Top` replace the hand-ordered `switch` in `step`. The overlays' own state (`ctxMenu`, `dialog`, ...) stays on the App: each has a different owner and shape, so folding them into one model would be a rewrite with no behavior gain.
- [open] Environment chip and menu (`env.go`) move out of `App` (revisit decision 30).

## Acceptance
- Each item is its own commit; flow tests unchanged; `Update` tests added first.
- `App` struct is far below its 73 fields; `make check` green; `architecture.md` and `decisions.md` updated.

## Notes
- Needs task 036 and its review. Items may be reordered by dependency.
