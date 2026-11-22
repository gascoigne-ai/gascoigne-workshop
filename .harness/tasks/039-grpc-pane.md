status: todo
kind: refactor
commit: refactor(ui): the gRPC pane is its own component
scope: internal/**, cmd/**, e2e/**, Makefile, .harness/**
not_before: 2026-01-01
depends_on: 038-ws-pane

# ui/editor/grpc

Goal: `protocol_grpcui` and `protocol_protoui` move to `ui/editor/grpc` (method picker, proto source, strip).
Follow the component recipe of `.harness/reorg.md`.

## Model
Model: unchanged.

## Acceptance
- Dump tests against a fake `Deps`; gRPC pane and stream tests move with it.
- Common acceptance of `.harness/reorg.md`.
