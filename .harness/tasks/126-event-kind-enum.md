status: todo
kind: feature
commit: feat(port): event kinds are a type; server event name is its own field
scope: internal/*,.harness/architecture.md
depends_on: 125-view-behavior-dump

# Give session events a typed kind

Origin: the repo audit of 2026-11-19 (three read-only reviews against the harness rules). The user approved all phases and, with that, the deletions and the gate change named here. Behavior stays the same unless the task says otherwise; tests stay green and none is loosened. Update `architecture.md` in the same commit where it describes what changes.

`port.Event.Kind` is a string: the comment lists 6 kinds, adapters emit about 14, and SSE puts the server's event name in the same field, so a server-chosen kind can collide with an internal one. Consumers compare strings (`controller/tabs/connection.go`, `app/wsui.go`).
- `port.EventKind` with constants; SSE event name in its own field; consumers switch on the type.
- Test first: an SSE event named `error` does not render as an error event.
## Model
`type EventKind` (string-backed enum, constants for each emitted kind, a `String()` for display), owned by `request/port`. Access pattern: set once per event, compared by consumers. Alternative rejected: keep strings and document them. Revisit if kinds differ per protocol so much that each needs its own type. Persisted formats: kinds are not persisted; if the call trace log saves them, version and fixture it.
