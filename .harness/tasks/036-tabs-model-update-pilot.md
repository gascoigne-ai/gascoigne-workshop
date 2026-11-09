status: todo

# Pilot: the tab list as Model, Msg and Update

## Goal
Prove the MVU pattern on one small domain. The tab list, the active index and tab dragging leave `App` and become a model changed only by `Update`; its tests need no window and no pixels.

## Scope
- [open] `workspace/controller/tabs`: `Model` (tabs, active, drag), `Msg` types (open, close, activate, move, begin and end drag, send started and finished, connection events) and `Update` returning the new model and commands. Table-driven tests per message.
- [open] `App` holds the model and calls `Update` through the bus (task 035); `tabs`, `active`, `dragTab` and `moveTab` leave `App`.
- [open] `requestTabs` (`view.go`) dispatches messages; its handlers are one-liners. Consider `view.Tabs` instead of the hand-built strip.
- [open] Review the pilot with the user: what read well, what did not. Adjust the pattern in `decisions.md` before tasks 037 and 038.

## Acceptance
- Existing flow tests stay green unchanged; new behavior has `Update` tests that failed first.
- `make check` green; `make shot` looked at; `architecture.md` updated.

## Notes
- Needs task 035.
