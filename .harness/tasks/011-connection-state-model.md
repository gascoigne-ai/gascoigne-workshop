status: todo
kind: refactor
commit: refactor(app): connection state gets its own model outside reqTab
scope: internal/**, cmd/**, e2e/**, .harness/architecture.md, .harness/architecture/**, .harness/overview.md, uitoolkit/CLAUDE.md, Makefile
not_before: 2026-01-01
depends_on:

# Connection state model

Goal: the websocket view cannot leave `internal/app` because its state lives in `reqTab`
(`socketBlock`: `wsView`, `msg`, `outbox`, `outboxBusy`, `wsCancel`; also `tab.Tools`, `Resources`,
`Prompts`, `grpcStream`) and is read by `driver.go`, `game.go`, `request_tab.go`,
`request_send.go`, `statesync.go` and `window.go`. `wsMessage` and `wsTarget` carry a `*reqTab` and
are type-switched in `App.handle`. Give a connection its own type with typed messages and a pure
`Update` (a controller package under `internal/feature/send`, as `tabs` and `viewer` are), so
`reqTab` holds a value of it and `App` only routes messages. Behavior unchanged.

User's answer: the websocket connection only. States: idle, connecting, open, retrying, closing, one
enum with a transition table. MCP (`Tools`, `Resources`, `Prompts`) and gRPC stream state stay in
`reqTab` for later tasks.

## Model
The connection status as one enum with a transition table instead of the flags that change together;
one owner (the controller `Model`); tab holds a value, not shared fields.

## Acceptance
- Test first: a table-driven test of the transition table and `Update` (each message from each
  state), written before the code and seen failing.
- Existing app tests stay green and none changes in what it asserts.
- The repo check command passes.
