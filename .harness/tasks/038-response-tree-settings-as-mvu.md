status: todo

# Response, collection tree and settings move to Model, Msg and Update

## Goal
The rest of the `App` decisions follow the pattern of task 036, domain by domain, until `App` only owns models, the bus and the frame loop.

## Scope
- [done] Collection tree: rename, duplicate, delete, drag, new request/folder as `tree` messages over typed `Node`s, applied in place (tabs hold pointers into the collection). `model.Folder` operations still take `any`; retyping them is left for a later cleanup. Imports and keyboard selection stay App methods.
- [done] Response view: mode, find bar and active match are `viewer.Model` with `Update` (`reqTab.look`). The lines, the find text and the compact filter stay widget state because the controller cannot import the toolkit; the App passes match counts in.
- [open] Settings: the 25-method `Host` of `settingsui` becomes messages; drop the `pageHost` wrapper.
- [open] Overlay state (dialogs, menus, prompts, confirm; `overlay.go`) as a model with one Escape chain in place of the hand-ordered `switch` in `step`.
- [open] Environment chip and menu (`env.go`) move out of `App` (revisit decision 30).

## Acceptance
- Each item is its own commit; flow tests unchanged; `Update` tests added first.
- `App` struct is far below its 73 fields; `make check` green; `architecture.md` and `decisions.md` updated.

## Notes
- Needs task 036 and its review. Items may be reordered by dependency.
