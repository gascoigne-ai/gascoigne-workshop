status: todo
kind: refactor
commit: docs: describe the new internal layout
scope: internal/CLAUDE.md,.harness/**,README.md
not_before: 2026-01-01
depends_on: 156-app-file-prefixes

# Docs for the new layout

Part of the package re-layout (domain first, two levels). Decided by the user: layout C; the import-layer check does NOT come back; `settings/model/appearance` loses its toolkit import by moving the theme types out. Out of scope: `uitoolkit` renames/splits and code changes that shrink `App`.
Target shape: `internal/<domain>/` holds the domain root package (types and ports, named after the domain, so no `model` or `port` packages and no import aliases); sub-packages sit directly under it; only `adapter/` and `ui/` are zone directories; the composition root is `internal/app`.
A refactor keeps tests green and changes none of them except import paths and renamed identifiers. `make check` is green after the task. Docs that name a moved path change in the same commit (`internal/CLAUDE.md`, `.harness/overview.md`, `.harness/architecture*`, Makefile `FUZZ_TARGETS` and `RELEASE_BASE`).

Rewrite `internal/CLAUDE.md` (layout, what may import what, written once), `.harness/overview.md` (tree and diagram) and `.harness/architecture.md`. Remove every claim of a layer test: none runs since 7d856abc and the user chose not to restore it; say so plainly. Check mermaid diagrams with mermaid.parse.
