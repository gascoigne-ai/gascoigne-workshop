status: todo
kind: refactor
commit: refactor(send): sending a request is one feature
scope: internal/request/**, internal/response/**, internal/feature/send/**, internal/exampleserver/**
not_before: 2026-01-01
depends_on: 002-kernel-model, 004-feature-settings

# Feature: send

Goal: the largest feature. `request/{assert,editor,mcpexplorer,ui/calltrace,ui/mcpform}`,
`request/adapter/proto/*` (REST, WebSocket, SSE, MCP, gRPC: a registry keyed by name, no switch on
protocol) and the response side (`response/{viewer,adapter/preview,ui/respview}`) become
`internal/feature/send/` with `proto/` below it. The feature logic that task 002 took out of the
request root lands here. `internal/exampleserver` stays a dev tool: it may import
`feature/send/proto/ws`, which is written as the one exception in `internal/CLAUDE.md`.

## Model
Model: unchanged.

## Acceptance
- A refactor: tests stay green and none changes; the protocol fuzz targets still run.
- `FEATURE.md` replaces `architecture/request.md` and `architecture/response.md`.
- The repo check command passes.
