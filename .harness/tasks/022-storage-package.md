status: todo
kind: refactor
commit: refactor(storage): user-file location and versioning share a package
scope: internal/**, cmd/**, e2e/**, Makefile, .harness/**
not_before: 2026-01-01
depends_on: 021-severity-to-notice

# kernel/userdir and kernel/fileversion become internal/storage

Goal: both are persistence plumbing used by the adapters of four features (187 lines together). Merge them into
`internal/storage` (zone `base`); the `RELEASE_BASE` entry for `userdir` follows. Keep each exported name; if two
names clash, ask. Follow the move recipe in `.harness/reorg.md`.

## Model
Model: unchanged. The `"version"` key rules of `persistence.md` are untouched.

## Acceptance
- Common acceptance of `.harness/reorg.md`.
