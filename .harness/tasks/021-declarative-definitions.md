status: in progress (format, loader, hot reload and the status bar done; more views and handlers pending)

# Non-code definitions (Vue-style)

## Goal
Views are built with Go code (`vml` DSL). Allow describing components declaratively, like Vue single-file components, so UI can be edited without recompiling.

## Scope
- File format for a component: template (element tree, bindings, conditionals, loops), style (tokens/control styles from theme files) and optional script hooks bound to named Go handlers.
- Loader that turns a definition into the same `vml` element tree; Go DSL stays supported and shares the runtime.
- Binding syntax for state and events, validated at load time with clear errors (file, line).
- Hot reload in development; embedded definitions in release builds.
- Start by converting one small view (e.g. the status bar or a settings page) as proof.

## Acceptance
- Definition for the proof view renders identically to the Go version (screenshot test).
- Malformed definitions produce errors, not panics.
- Decision recorded in `decisions.md`; `make check` passes.

## Progress
- Decision: HTML-style XML templates, stdlib parser, same runtime (decisions 11).
- Done: `uitoolkit/component` (elements and attributes as tables, `{{path}}`, `if`, `for`, `onclick`, errors with file and line, `Loader` with reload on change), `vml.Dump`, `-views DIR`, the status bar as `views/statusbar.vml` with a test that it matches the Go view.
- Left: tables, text fields, scroll and tabs as elements (they need state objects in the scope), event handlers other than click, move the header and the tab strips, a style block for control styles, an `include` for sub-templates.
- Renamed `tmpl` to `component`; templates can also be inline in Go (`component.Must`, `MustBuild`).
- Inline form chosen: `component.View(src, scope)` in the method that uses it; the status bar is inline, the `-views` flag and embedded `.vml` files were removed (the `Loader` stays in the toolkit).
- Done later: `<slot name="x"/>` inserts a `vml.Node` from the scope (Vue's slot), so a template can contain a hand-built part; `<icon name=.../>`; the header is now an inline template with the environment chip as a slot, and a test checks it against the Go view it replaced. Tables, fields and scroll stay Go because they need state objects; the next candidates are the tab strips and the Messages toolbar.
