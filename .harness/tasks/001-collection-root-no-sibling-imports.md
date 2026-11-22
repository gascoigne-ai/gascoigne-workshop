status: todo
kind: refactor
commit: refactor(layout): domain roots import no sibling package
scope: internal/collection/**, internal/workspace/**, internal/settings/**
not_before: 2026-01-01
depends_on:

# Domain roots import no sibling package

Goal: `collection` imports `collection/clientconfig` and `workspace` imports `workspace/state`; the
rule in `internal/CLAUDE.md` says a domain root imports no other package of its domain
(only `settings/fileversion` is excepted). Move the shared types into the root, or invert the
import, so the root stands alone. Update `internal/CLAUDE.md` and the Known layer gaps in
`.harness/architecture.md` in the same commit.

## Model
Decide per package which types move into the root (a refactor: names and behavior unchanged).

## Acceptance
- A refactor: existing tests stay green and none changes.
- `go list -deps` of each root shows no sibling package of its domain (except the written exception).
- The repo check command passes.
