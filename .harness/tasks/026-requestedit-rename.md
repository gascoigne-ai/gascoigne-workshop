status: todo
kind: refactor
commit: refactor(send): editor is renamed requestedit
scope: internal/**, e2e/**, Makefile, .harness/**
not_before: 2026-01-01
depends_on: 025-adapter-renames

# send/editor becomes send/requestedit

Goal: frees the name `editor` for the request-editor UI composite (`ui/editor`). Path and package clause only.

## Model
Model: unchanged.

## Acceptance
- Common acceptance of `.harness/reorg.md`.
