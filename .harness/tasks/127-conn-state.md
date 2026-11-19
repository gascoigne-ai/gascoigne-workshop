status: todo
kind: feature
commit: feat(tabs): connection state is one enum with a transition table
scope: internal/*,.harness/architecture.md
depends_on: 126-event-kind-enum

# Replace the connection flags with a state machine

Origin: the repo audit of 2026-11-19 (three read-only reviews against the harness rules). The user approved all phases and, with that, the deletions and the gate change named here. Behavior stays the same unless the task says otherwise; tests stay green and none is loosened. Update `architecture.md` in the same commit where it describes what changes.

- `tabs/connection.go`: `Session != nil`, `Connecting`, `WantOpen`, `Reconnect` change together, with `Active`, `CanRetry`, `RetryWanted` as predicates; `Tab.Sending` with `Seq` has the same shape. Replace by a `ConnState` enum with a transition table; `Connecting` with a session becomes unrepresentable.
- `app/wsui.go` `wsEvent` has `sess`, `err`, `ev`, `hasEv`, `closed`, `retry`: five message types (`wsOpened`, `wsFailed`, `wsReceived`, `wsClosed`, `wsRetry`) dispatched in `App.handle`.
- Test first: every legal transition, and every illegal one is refused.
## Model
`ConnState` (idle, connecting, open, closing, retrying; final names written by the designer) with `transition(state, event) (state, ok)` table. Owner: `controller/tabs`. Not persisted. Alternative rejected: keep flags with asserts. Revisit if a protocol needs a state the others do not have.
