status: todo
kind: refactor
commit: refactor(send): the protocol clients become an adapter
scope: internal/**, cmd/**, e2e/**, Makefile, .harness/**
not_before: 2026-01-01
depends_on: 023-ui-format-package

# send/proto becomes send/adapter/protocol

Goal: `send/proto` and `proto/ws` do network and process I/O (net/http, tls, exec) and implement
`request.Sender`: they are an adapter, not domain. Move to `internal/feature/send/adapter/protocol{,/ws}`
(7 fuzz target lines; wiring in `cmd/workshop` and `e2e/cmd/shot`; the `exampleserver` exception row follows the
path). Follow the move recipe in `.harness/reorg.md`.

## Model
Model: unchanged.

## Acceptance
- Common acceptance of `.harness/reorg.md`.
