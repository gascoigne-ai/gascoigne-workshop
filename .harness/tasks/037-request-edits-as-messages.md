status: todo

# Request edits are messages handled by a pure Update

## Goal
Every edit of the request in a tab is a typed message handled by one pure `Update`, instead of 35 scattered `t.Req.X =` writes in `app.go`, `panes.go`, `graphql.go`, `grpcui.go` and others. The tab keeps pointing into the collection: edits show in the sidebar and the autosave at once, and the dirty dot keeps its meaning (decision by the user, 2026-11-10: no draft copy, it solves no bug today and would change what the user sees).

## Shape
`internal/request/controller/editor`: `Update(req model.Request, msg Msg) model.Request` returns the request after the edit and never changes the one it is given (a slice it changes is copied). The App writes the result back, `*t.Req = editor.Update(*t.Req, msg)`, in one function (`App.edit`-style helper, named in the first item) that also does what the edit means for the window (re-derive the Params table from the URL, refresh the view). Messages are typed structs, one per kind of edit (`URLChanged`, `MethodChosen`, `BodyTypeChosen`, ...), not one per widget.

## Scope
- [done] Package `editor` (`request/controller/editor`; helper `editRequest` in `app.go`, a function not a method so `paramsChanged` can use it) with the URL, method, protocol and timeout edits; `urlEdited` and `paramsChanged` rules (`model.MergeParams`, `model.WithParams`) move into it. Table tests; the helper in `App`; the first writes in `app.go` go through it.
- [done] Params, Headers, Vars and form-field table edits: no code change needed. They already follow the pattern: `App.applyRows` is the one place a table's rows change, it runs a pure `model.*` row function (`InsertRow`, `DuplicateRow`, `DeleteRow`, `SetAll`, `ParseBulk`) on a copy, undo and redo live in `tabs.RowHistory`, and no write to `Params`, `Headers`, `Vars` or a form row exists outside it. Turning each row operation into a message would only rename the call.
- [done] Body: type, text, file path; GraphQL query, variables and starter operation (`BodyTyped`, `VarsTyped`, `BodyTypeChosen`, `OperationChosen`). The form fields are table rows and were done by the previous item.
- [done] Auth type and fields and the assertions text (`AuthTypeChosen`, `AuthFieldSet`, `TestsTyped`). The id-to-field table of the credentials moved from the App's `authField.value` accessors to `editor` (`AuthValue` reads), so the mapping exists once; an unknown field id panics, since the ids come from a table.
- [open] gRPC method and stream flag (`grpcui.go`), and whatever write to `t.Req.` is left (grep `t\.Req\.[A-Za-z.]* =`); none remain outside `editor`'s helper.

## Acceptance
- No behavior change and no assertion changed in existing tests (names and imports only); each message has an `Update` test that failed first.
- `grep -n 't\.Req\.[A-Za-z]* = ' internal/workspace/ui/app/*.go` finds nothing outside tests.
- `make check` green; `architecture.md` describes the editor; layer test passes (`request/controller` imports model only).

## Notes
- Needs nothing from task 036 beyond decision 35 (UI handlers apply their message at once; no Cmds here, edits have no effects).
- The `TextField` holding the typed URL stays widget state; the message carries its text.
