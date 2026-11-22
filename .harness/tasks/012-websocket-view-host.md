status: todo
kind: refactor
commit: refactor(send): the websocket view lives in the send feature
scope: internal/**, cmd/**, e2e/**, .harness/architecture.md, .harness/architecture/**, .harness/overview.md, uitoolkit/CLAUDE.md, Makefile
not_before: 2026-01-01
depends_on: 011-outbox-and-message-target

# Websocket view behind a Host

Goal: with the send queue a pure outbox model (task 011; the message targets keep `*reqTab`, user's answer), move `internal/app/protocol_wsui.go`
(477 lines) into `internal/feature/send/ui/<package>` behind a `Host` of at most about 5 methods,
tested with a fake host. Measured needs of the view today: policy (`allowed`, `varsFor`,
`timeoutFor`), the protocol registry, the bus, the status line, the call log and clock, retry delay,
window context (`scope`, row size, `openMenu`), the protocol views table and the MCP form opener.
Group them by capability; if it still needs more than about 5, say which and ask.

## Model
Model: unchanged (the connection (`tabs.Connection`) and the outbox model from task 011 stay the owners).

## Acceptance
- Test first: a fake-host test states what the view needs from its host.
- Existing app tests for the area move with it and stay green.
- The repo check command passes; `.harness/architecture/app.md` lists the package.
