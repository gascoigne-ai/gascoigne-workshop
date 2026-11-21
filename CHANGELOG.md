# Changelog

## 0.1.0 alpha (unreleased)

First release, an alpha: ready for daily use and still growing.

- Requests: REST, WebSocket, SSE, MCP (tools explorer, streamable HTTP and stdio) and gRPC (reflection or `.proto` files with import paths, chosen on the Proto tab; unary calls and streams), with params, headers, bodies (JSON, text, form, multipart, file), auth, assertions and variables from the collection, its folders and environments.
- Collections are folders of JSON files; several open side by side; create, open, rename, save, reorder, close (Ctrl+Shift+W) and delete them from the sidebar, each a root of the tree with a bold name; tree with rename, duplicate, move and delete; secrets kept out of request files.
- Response viewer: pretty, raw and compact views, foldable and searchable JSON, history, timeline, call log, sent-request log.
- Import cURL, OpenAPI and Postman; export Postman.
- Settings: one `settings.json` with a form and a raw editor, shortcut editor, colors page, System page (GPU rendering), dark and light themes, JSON theme files.
- Client configs (`.workshop.yaml`, `.json`, `.toml`) in release builds; example server for every protocol in the default build.
- The default (dev) build keeps its data in `gascoigne-workshop-dev`, apart from the release build's `gascoigne-workshop`, and reopens the collections open last.
- In-app movable and resizable dialogs; a gear icon for the settings button.
- Native file dialogs on macOS, Windows and Linux (zenity, kdialog, yad, qarma or matedialog, ordered by the desktop); without a dialog program, an in-app file browser for opening and saving.
- A welcome dialog with these notes opens the first time a version runs and from the version in the status bar.
- Tabs and the root rows of the tree show their collection's color.
- Drag a request or folder onto another open collection to move it, or hold Alt to copy it; a name already taken gets a number, and a note lists the variables the new collection does not define. The row menu has Move to... and Copy to... for the same, with a picker of every open collection and folder.
- Every saved file now starts with a `version`; the first save after upgrading rewrites each collection file once (and wraps `.secrets.json`), so a collection in git shows a one-time diff. Files from a newer build are not overwritten.
- Text size is now a number of pixels (10 to 24, default 14) with a minus and a plus button; the old small, medium and large become 13, 14 and 16.
- `make install` installs the release build, with its desktop entry and icon, under `~/.local`.
