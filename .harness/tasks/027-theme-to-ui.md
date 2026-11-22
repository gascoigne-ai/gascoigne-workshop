status: todo
kind: refactor
commit: refactor(ui): the theme moves to ui/theme
scope: internal/**, cmd/**, e2e/**, Makefile, .harness/**
not_before: 2026-01-01
depends_on: 026-requestedit-rename

# settings/theme becomes ui/theme

Goal: `settings/theme` imports `uitoolkit/core` and `uitoolkit/theming`, a written exception inside a feature.
In `ui/theme` it needs none: delete the exception row and update the package doc. The `ui` zone may import
`ui/theme`. Follow the move recipe in `.harness/reorg.md`.

## Model
Model: unchanged. Theme file format and versions are untouched.

## Acceptance
- The exception row for `settings/theme` is gone; `TestEveryRuleMatchesSomething` stays green.
- Common acceptance of `.harness/reorg.md`.
