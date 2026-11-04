# Backlog

Ideas and gaps that have no task yet, plus the map of what a basic working client needs. Work for
the autonomous loop is in `tasks/` (one commit each, in numeric order); read this before picking
work that has no task. Finished tasks are deleted; what they left open is under "Deferred from finished tasks".

## What exists (milestone 1)
Window with collections tree, request tabs, URL bar (type, select, copy, paste), Params / Headers /
Vars tables (toggle and delete only), response viewer (colored, foldable, line selection, copy),
REST send, dark and light themes, text size, settings menu, HiDPI, copy and paste through system
tools.

## Needed for a basic working client, by area
Task numbers refer to `tasks/`; "-" means no task yet.

### Editing and selection
- Editable table cells (001). Text field word movement, double / triple click, horizontal scroll, undo (002).
- Multi-line editor for bodies (005), body types and content type (006).
- Character-level selection in the response, find in response (010).
- Context menu on text: cut, copy, paste, select all (012).
- IME and dead keys, drag and drop of text and files (-).

### Focus, keyboard, mouse
- One focus owner, Tab order, focus outline, global shortcuts (003).
- Tree keyboard navigation and F2 rename (013). Cursor shapes (text, resize, pointer) (011).
- Tooltips, press and drag feedback on buttons (-).

### Layout and scrolling
- Scroll containers and scrollbars everywhere, horizontal code scroll (004).
- Draggable splitters, vertical request / response layout (011).
- Window size and position remembered, minimum size, app icon, title with dirty marker (-).

### Requests
- Cancel and timeout, redirects option (008). Auth: bearer, basic, API key (009).
- Body: JSON, text, form, multipart (006); binary file body and file picker (-).
- Path variables `:id`, cookies jar and viewer, proxy and TLS settings, client certificates (-).
- Pre-request scripts and response assertions (-).

### Responses
- Pretty, raw, preview, wrap, copy, save, copy as cURL (010).
- Timeline with DNS, connect, TLS, first byte, transfer from `httptrace` (-).
- Image and HTML preview, very large bodies (streaming, truncation notice), response history per request (-).

### Collections and data
- Create, rename, delete, duplicate, move, filter (013). Tabs: close, reorder, dirty (007).
- Save and load on disk, autosave, import cURL and OpenAPI JSON (done). Import YAML OpenAPI, Postman and other collection formats, export (-).
- Environments, secrets, folder variables, Ctrl+Space completion (done). Request history (-).

### Protocols
- WebSocket and MCP (done). SSE, GraphQL (query editor, variables, schema), gRPC (-).

### App quality
- Dialogs, toasts (012). Error states for every failure the user can see (-).
- Redraw on demand instead of 60 fps when idle (-).
- Golden screenshot tests for both themes (`make shot` compare), CI workflow, release packaging and desktop entry (-).
- Accessibility pass: contrast already tested; focus order and screen reader hints (-).

### Deferred from finished tasks
- Tab strip overflow: horizontal scrolling with the active tab kept in view (needs a horizontal `vml.Scroll`).
- Shortcuts: the text widgets' own keys (Ctrl+A/C/V/X/Z/Y) are fixed and not listed in the settings yet; plain-key bindings are refused rather than allowed with a warning.
- Table columns: resizing from the keyboard (Alt+Left/Right on a header cell needs a header focus concept); the widths are saved in the state file.
- Table rows: a drag handle to reorder rows with the mouse (keyboard and menu moves exist), bulk edit for form fields.
- Tree: match highlighting in the filter, expanding a folder while a drag hovers over it.
- Scroll-into-view of a focused child inside a `vml.Scroll`.
- Response: line wrap, image and HTML preview, save body to a file (needs the dialogs of task 012), Shift+arrow selection in the code view.
- Collections: a YAML OpenAPI reader (stdlib has no YAML; convert to JSON first today).
- Environments: `{{variable}}` highlighting in header cells and the body editor (only the URL field has it); a popup while typing (the URL field has Ctrl+Space instead, so it keeps the keyboard).
- WebSocket: a JSON-formatting composer (the composer is one line); reconnect and the hex binary composer exist.
- MCP: a typed form with one input per tool property (today a schema-filled `call` line is edited as JSON), server-initiated requests (sampling, roots), the long-lived notification stream.
- Toolkit: templates for tables, fields, scroll and tabs (they need state objects in the scope) and moving the tab strips and the Messages toolbar to templates; an icon mask cache if profiling asks for it and an open icon set (check the license, vendor only what is used); `core` could be freed of the `Theme` type by making `Ctx` take a colors interface; pointer positions are whole logical pixels (accepted); no automated golden screenshots at 150%; trim exported names only the workshop uses.
- App state: panes, environments, the WebSocket/MCP tab, workspace saving, overlays and shortcuts stay in `internal/ui` because they share `App` state; split them when one gets a second user (needs an app-state interface).


### Added later
- Right-click menus everywhere (018), table row operations and change handling (019). Task 018 depends on 012 (menu primitive) and uses 013 and 019; 

- Added after review: resizable table columns (021), configurable colors and selection contrast (022), configurable shortcuts (023). 021 follows 019; 022 and 023 extend the settings menu and are independent of the other tasks. Layout resizing is part of 011.

## Suggested order
Editing foundation first, because everything else needs it: 001, 002, 003, 004, 005. Then the
request side (006, 008, 009, 007), then response tools (010), layout (011), overlays and CRUD
(012, 013), table rows and right-click menus (019, 018), persistence, remembered state and environments (014, 020, 015), then protocols (016, 017).

## Open questions
- Native file dialogs: shell out to zenity / kdialog, or build a file picker widget.
