status: todo
kind: docs
commit: docs(arch): architecture docs describe app, window and components
scope: .harness/**, internal/CLAUDE.md, e2e/appsize_test.go, internal/**/doc.go
not_before: 2026-01-01
depends_on: 045-window-composite

# Final docs pass of the reorganization

Goal: reconcile `.harness/architecture.md`, `architecture/app.md`, `overview.md`, `rules.md`,
`internal/CLAUDE.md`, every `FEATURE.md` and the package docs with the final tree. Fold what stays true of
`.harness/reorg.md` into them (composition and sequence diagrams, layer table, component anatomy), then delete
`reorg.md` (it is the working note of tasks 015-048; ask first, per the ask-before-delete rule). The reasons of
each choice (no hub, `logic/` as a nested package, no event bus between components, rejected alternatives) go in
the doc comment of the owning package. The ratchet cap is set to the final file count of `internal/app`.

## Model
Model: unchanged. Docs only.

## Acceptance
- Each Mermaid diagram passes `mermaid.parse` (jsdom in node), as `common/code-style.md` asks.
- No doc mentions `kernel`, `feature/*/ui`, `routeKeys`, `textFocused` or `appstate`.
- The repo check command passes.
