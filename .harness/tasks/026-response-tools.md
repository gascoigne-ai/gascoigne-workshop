status: todo

# Response tools

## Goal
The response viewer shows text bodies well. Missing: other body kinds, long bodies, saving and history.

## Scope
- Image preview (PNG, JPEG, GIF, WebP where the standard library or `x/image` allows) and an HTML preview as source with a note (no browser engine).
- Line wrap toggle in the raw and pretty views; Shift+arrow selection in the code view.
- Very large bodies: stop reading at a limit, show a truncation notice with the size, offer to save the full body.
- Save the body to a file (needs the file-dialog decision in task 029).
- Response history per request (last N, with status and time), pick one to view; kept in the state folder, not in the collection files.
- Request history across the workspace: a searchable list of what was sent.

## Acceptance
- Tests for truncation, wrap geometry, history ring and persistence round trip.
- `make check` passes; architecture.md updated.

## Progress
- Done: bodies longer than the limit are cut and `Response.Truncated` says so (`REST.MaxBody`, default 32 MiB; the Timeline pane shows the note).
- Left: the rest.
- Done later: `Save body...` in the response menu (prompt for a path, `~/` allowed, name suggested from the content type); a `History` button lists the last 10 successful responses of the tab (newest first, picking one swaps it with the shown one).
- Left: image and HTML preview, line wrap, Shift+arrow selection, history across restarts and a workspace-wide request history.
