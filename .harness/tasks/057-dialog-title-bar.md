# 057 Dialogs have a separate title bar with a close button

After task 056 the title of a dialog is hard to use as a handle: the settings title row is filled by the search field (title, search, Save, JSON, close), the call log's by its filter and Clear, and message dialogs have a plain title text and no close button. Decided with the user: every dialog gets a separate title bar (title and close, the whole bar between them a drag area); the controls of the settings dialog and the call log go to their own toolbar row below it; message dialogs get the same bar with a close button (Escape and an outside click still cancel).

Scope
1. `DialogTitle` is a distinct bar (sunken background, bottom border, title, spacer, close) and `DialogToolbar(items...)` (`<dialog-toolbar>`) is the row for controls; tests, doc comments, widget doc index, template table.
2. (done) `MessageDialog` uses the title bar; its close button runs the dialog's own `OnDismiss` (the one an outside click uses), so the app needs no change; pinned dialog dumps updated and looked at.
3. Settings dialog: title bar, then a toolbar row (search, Save, JSON/Form), then the body; pinned settings dumps and `dialogdrag_test.go` updated.
4. Call log dialog: title bar, then a toolbar row (filter, Clear).
5. Docs: `architecture.md` Overlays, a decision in `decisions.md`.

Acceptance: every dialog shows a distinct title bar with a close button that closes it like Escape, and can be dragged anywhere on the bar between the title and the button; `make check` green; `make shot` of the settings, a confirm and the Log dialog looked at.
