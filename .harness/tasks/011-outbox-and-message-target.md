status: todo
kind: refactor
commit: refactor(send): outbox model and tab-id message targets
scope: internal/**, cmd/**, e2e/**, .harness/architecture.md, .harness/architecture/**, .harness/overview.md, uitoolkit/CLAUDE.md, Makefile
not_before: 2026-01-01
depends_on:

# Outbox model and message targets

Goal: the websocket view cannot leave `internal/app` while its remaining state sits in `reqTab` and
its messages carry a `*reqTab`. The connection state machine already exists (`tabs.ConnState`:
idle, connecting, open, closing, retrying, with a transition table and tests; the first text of this
task was wrong about it). What remains in `socketBlock` (`internal/app/request_tab.go`): the log
widget `wsView`, the composer `msg`, the send queue (`outbox []queuedSend` plus `outboxBusy`) and
`wsCancel`. `wsMessage` and `wsTarget` carry a `*reqTab` and are type-switched in `handleWS` and
`applyWS`. Two changes, behavior unchanged:

1. A pure outbox model under `internal/feature/send` (a controller package, as `viewer` is):
   `Enqueue` says whether to send now, `Done` returns the next send; `outbox` and `outboxBusy`
   become one value of it (they are two facts about one queue today).
2. `wsTarget` and `wsMessage` carry a tab id plus the connection generation (`Gen`) instead of a
   `*reqTab`, so `App.handle` only routes and a stale answer is dropped by the generation.

User's answer: rewrite task 011 as this (outbox and message target); the websocket view move stays
task 012.

## Model
The outbox as one type: the queue and the in-flight flag with the transitions Enqueue and Done;
one owner (the model), the tab holds a value; the target is a tab id and a generation, never a
pointer.

## Acceptance
- Test first: a table-driven test of the outbox (empty queue, one in flight, several queued, ping
  ordering as today), written against a stub and seen failing for the expected reason.
- Existing app tests stay green and none changes in what it asserts.
- The repo check command passes.
