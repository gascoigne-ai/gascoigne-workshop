status: done

# UI package

## Goal
Make the UI toolkit (`uitoolkit/core`, `uitoolkit/vml`, `uitoolkit/engine`) a self-contained package with a stable public API, independent of workshop-specific code.

## Scope
- Decide the package boundary and name (single `ui` module tree vs. separate module); list the exported API and hide the rest.
- No imports from `internal/*` inside the toolkit; app-specific theme overlays stay in the app.
- Document usage with a minimal example app and tests.

## Acceptance
- Toolkit builds and tests without the app packages.
- `make check` passes; architecture.md updated.

## Progress
- Decision: the toolkit stays in this module under the root `uitoolkit/` directory (`uitoolkit/core`, `uitoolkit/vml`, `uitoolkit/engine`); it is not a separate module yet. Import boundary enforced by tests (`uitoolkit/boundary_test.go`, `widget/layers_test.go`).
- Done: moved from `ui/`, `uitoolkit/doc.go` documents the layers.
- Done later: a runnable example, `uitoolkit/examples/hello` (Go DSL and a template side by side, tested).
- Left: trimming exported names that only the workshop needs.
- Done: unused exports removed, naming rule applied (PadX/PadY/PadB, MarginX/MarginY), package docs corrected, `Example*` functions in vml, component, theming and widget, overview in `uitoolkit/doc.go`, theming mechanics moved in (`uitoolkit/theming`).
