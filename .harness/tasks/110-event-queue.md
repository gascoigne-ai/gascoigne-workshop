status: todo
kind: refactor
commit: refactor(proto): one event queue for every session type
scope: internal/request/*
depends_on: 109-ci-dedupe

# Share the session event queue

Origin: the repo audit of 2026-11-19 (three read-only reviews against the harness rules). The user approved all phases and, with that, the deletions and the gate change named here. Behavior stays the same unless the task says otherwise; tests stay green and none is loosened. Update `architecture.md` in the same commit where it describes what changes.

`mcp.go`, `sse.go`, `grpcstream.go` and `websocket.go` each carry `events`, `done`, `stop sync.Once`, `mu`, `ended`, `closing`, `emit`, `finish`. They have drifted: the WebSocket `emit` has no `ended` guard, only two inject `now`.
- One `eventLog` type (`emit`, `finish`, `close`, `closing`) embedded by all four; pin the current close/emit-race behavior of each with a test first, then switch them over.
- Include the abnormal-close code `1006` literal in `mcp.go` and `sse.go`: use `ws.CloseAbnormal`.
Acceptance: `make check` green; the four files shrink.
