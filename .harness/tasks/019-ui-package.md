status: todo

# UI package

## Goal
Make the UI toolkit (`ui/widget`, `ui/vml`, `ui/engine`) a self-contained package with a stable public API, independent of workshop-specific code.

## Scope
- Decide the package boundary and name (single `ui` module tree vs. separate module); list the exported API and hide the rest.
- No imports from `internal/*` inside the toolkit; app-specific theme overlays stay in the app.
- Document usage with a minimal example app and tests.

## Acceptance
- Toolkit builds and tests without the app packages.
- `make check` passes; architecture.md updated.
