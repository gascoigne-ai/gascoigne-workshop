# 061 Merge VML templates into Go source

Asked for by the user ("merge vml into code"); the reading, chosen by the user: inline the templates into the Go files. Reverses the file-per-template part of decision 4.

**Goal:** one place per view: the Go file holds the scope, the handlers and the template.

**Objectives**
1. No `.vml` file is left under `internal/`.
2. Each template is an inline string parsed once at package level (`markup.MustLoad` or `markup.Must`), placed in a `const` block at the end of its file.
3. Pixels do not change: golden scenes and `view.Dump` pins pass with a zero-line diff in tests and PNGs.
4. The markup loader's folder reading (`markup.Load`, `MustLoad` over an `fs.FS`) stays in the toolkit as public API (user's answer); the app no longer uses it.

Scope
1. (decided) Keep `markup.Load` in the toolkit; the app stops using it and `go:embed`. Tests of `Load` stay.
2. Move the templates one package per commit (colorpage, sentlog, settingsui, app, calltrace, mcpform, shortcutpage...); test files and PNGs untouched.
3. Update the markup package doc, rules.md (VML rules), architecture.md and a new decision that supersedes decision 4's "embedded `.vml` next to its Go file".

Acceptance: `find internal -name '*.vml'` is empty, `make check` green with golden scenes running (not skipped).
