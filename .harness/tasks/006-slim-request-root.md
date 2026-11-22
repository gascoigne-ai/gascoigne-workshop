status: todo
kind: refactor
commit: refactor(request): keep only types and ports in the request root
scope: internal/request/**
not_before: 2026-01-01
depends_on: 001-collection-root-no-sibling-imports

# Slim the request root

Goal: the `request` root is 2977 lines in 47 files, though a root holds types and ports only.
It also holds the tree operations (`tree.go`) and other logic. Move logic that is not a type, a
port or a pure invariant into controller packages under `internal/request/` (for example `tree`),
so the root reads as the domain model. `collection` and `response` import it as a root, so the
public names the other domains use stay.

## Model
List what stays in the root (types, ports, invariants) and what moves; one owner per fact.

## Acceptance
- A refactor: tests stay green and none changes (test files move with their code).
- The root has no file with logic that needs a package of its own.
- The repo check command passes.
