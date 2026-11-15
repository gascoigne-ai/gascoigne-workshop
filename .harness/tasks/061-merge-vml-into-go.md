# 061 Merge VML templates into Go source

Asked for by the user ("merge vml into code"); the reading, chosen by the user: inline the templates into the Go files. Reverses the file-per-template part of decision 4.

**Goal:** one place per view: the Go file holds the scope, the handlers and the template.

**Objectives**
1. No `.vml` file is left under `internal/`.
2. Each template is an inline string parsed once at package level (`markup.MustLoad` or `markup.Must`), placed in a `const` block at the end of its file.
3. Pixels do not change: golden scenes and `view.Dump` pins pass with a zero-line diff in tests and PNGs.
4. The markup loader's folder reading (decision 4) is kept or dropped on purpose, recorded in a decision.

Scope
1. Ask the user: keep folder-read templates (hot reload) in the toolkit, or drop them? Record the answer.
2. Move the templates one package per commit (colorpage, sentlog, settingsui, app, calltrace, mcpform, shortcutpage...); test files and PNGs untouched.
3. Update the markup package doc, rules.md (VML rules), architecture.md and a new decision that supersedes 50's "embedded `.vml` next to its Go file".

Acceptance: `find internal -name '*.vml'` is empty, `make check` green with golden scenes running (not skipped).
