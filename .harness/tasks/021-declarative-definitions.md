status: todo

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
