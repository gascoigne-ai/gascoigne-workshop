status: todo

# The request editor works on a draft, not on the collection

## Goal
A tab owns a copy of its request. Editing never changes the collection until a commit message says so. Today `t.Req` points into `a.col` (`openRequest`), so 35 direct writes (`t.Req.X =` in `app.go`, `panes.go`, `graphql.go`, `grpcui.go`) edit the collection itself.

## Scope
- [open] Tab model: draft request, saved snapshot, dirty by comparison of revisions (replaces `Clone()` plus `reflect.DeepEqual` per frame, with task 040).
- [open] Edit messages for each pane (URL, params, headers, body, auth, tests, GraphQL, gRPC) handled by `Update`; the 35 direct writes go away, one pane per commit.
- [open] Explicit save message writes the draft into the collection; autosave debounce works from it.
- [open] One source for the URL text: drop the hand sync of `TextField` and the model in `urlEdited`.

## Acceptance
- Behavior seen by the user unchanged (tab dirty marker, save, autosave); regression tests state what happens when two tabs or the tree rename touch the same request.
- `make check` green; `architecture.md` updated.

## Notes
- Needs task 036. Ask the user before changing what "dirty" or "close tab with changes" does.
