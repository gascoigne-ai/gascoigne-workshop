status: todo

# Restructure uitoolkit and the app

## Goal
Names and directories say what things are, so a new reader finds a feature or a widget without a guide.

## Scope
uitoolkit:
- Find a more meaningful name for the `vml` package. It holds views, inputs and the markup-free element API, so the name should say that. Pick the name, record it in `decisions.md`, and rename in one mechanical commit: imports, the layer test in `uitoolkit/core/layers_test.go`, docs, and the `vml` entry in `architecture.md`.
- Review the toolkit tests so each reads on its own. A test name states the behavior, there are table-driven cases where several inputs exist, and shared helpers such as `viewport`, `press` and `colX` live in one test-helper file with a comment. Fix names and structure only; no assertion is loosened.

app:
- Move `internal/ui` from one flat package to feature directories, such as request editor, response viewer, sidebar and collections, protocols (grpc, mcp, websocket), settings, history and dialogs. Nested directories are allowed. Each directory has a package doc and one clear job. Keep the layer rules: `internal/model` and `internal/proto` stay free of UI.
- Where a feature's UI is a template, write it as a component definition (`component.View` templates) instead of hand-built Go views. Keep Go views where logic dominates. The existing twin tests (template against Go view) show a template is equivalent before the Go view is removed.

## Acceptance
- A rename commit and a layout commit each keep `make check` green, with no behavior change and no test changed except imports and names.
- `architecture.md` describes the new tree, with a Mermaid diagram of the packages.
- Decision recorded for the new `vml` name and for the directory scheme.

## Notes
- Do this after 028 (done) and after 030 (settings view) is settled, since both touch `internal/ui`. Moving files while other work is in flight makes both harder to review.
- One step per commit: rename first, then the moves one feature at a time, then template migrations.
- The new `vml` name and the directory scheme are real forks: ask the user with options before renaming or moving (see the Ambiguity rule).
