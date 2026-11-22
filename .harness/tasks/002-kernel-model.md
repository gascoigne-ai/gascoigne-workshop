status: todo
kind: refactor
commit: refactor(kernel): request model moves to kernel, sheds feature logic
scope: internal/**, cmd/**, e2e/**, .harness/architecture.md, .harness/architecture/**, .harness/overview.md, uitoolkit/CLAUDE.md, Makefile
not_before: 2026-01-01
depends_on: 001-kernel-shared-leaves

# Kernel: the model

Goal: `internal/request` (2977 lines, 47 files) is already the kernel: every domain imports it. It
mixes the model with feature logic. Model and ports move to `internal/kernel/request` (same package
name `request`, so importers change a path and not identifiers): `model.go`, `node.go`, `tree.go`,
`vars.go`, `rows.go`, `url.go`, `port.go`, `protocol.go`, `access*.go`, `errors.go`, `transfer.go`.
Feature logic stays in a slimmed `internal/request` until task 006 moves it under `feature/send`
(user's answer: keep in request, rename later): `curlparse`, `mcp*` and `mcpserver`, `grpcschema`,
`protosource`, `graphqlschema`, `jsonschema`, `binarymessage`, `formfile`. A file used by two
features, or by the model itself, goes to kernel. The implementer confirms each file by its importers.

## Model
List per file: moves to kernel, or stays in `internal/request` for task 006. The model files import no feature.

## Acceptance
- A refactor: tests stay green and none changes.
- `internal/kernel/request` imports no other `internal` package; the slimmed `internal/request` imports it.
- The repo check command passes.
