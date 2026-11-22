status: todo
kind: refactor
commit: refactor(kernel): request model moves to kernel, sheds feature logic
scope: internal/**, cmd/**, e2e/**, .harness/architecture.md, .harness/architecture/**, .harness/overview.md, uitoolkit/CLAUDE.md, Makefile
not_before: 2026-01-01
depends_on: 001-kernel-shared-leaves

# Kernel: the model

Goal: `internal/request` (2977 lines, 47 files) is already the kernel: every domain imports it. It
mixes the model with feature logic. Model and ports stay: `model.go`, `node.go`, `tree.go`, `vars.go`,
`rows.go`, `url.go`, `port.go`, `protocol.go`, `access*.go`, `errors.go`, `transfer.go`. Feature logic
leaves for the feature that owns it: `curlparse`, `shellsplit` (import curl), `mcp*` and
`mcpserver` (MCP), `grpcschema`, `protosource`, `graphqlschema`, `jsonschema`, `binarymessage`,
`formfile` (protocol forms). The task designer confirms each file by its importers.

## Model
List per file: stays, or the feature that takes it. A file used by two features stays in kernel.

## Acceptance
- A refactor: tests stay green and none changes.
- `kernel/model` (name decided by the designer) imports no feature.
- The repo check command passes.
