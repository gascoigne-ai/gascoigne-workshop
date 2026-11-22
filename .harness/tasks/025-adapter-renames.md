status: todo
kind: refactor
commit: refactor(adapters): adapter packages are named for what they hold
scope: internal/**, cmd/**, e2e/**, Makefile, .harness/**
not_before: 2026-01-01
depends_on: 024-protocol-adapter

# history/adapter becomes historyfile, collections/adapter/store becomes collectionstore

Goal: `history/adapter` is package `history` today and `store` says nothing at an import site. Rename the
packages (clause and path) to `historyfile` and `collectionstore`; update `cmd`, `shot`, tests, fuzz lines and
docs.

## Model
Model: unchanged.

## Acceptance
- Common acceptance of `.harness/reorg.md`.
