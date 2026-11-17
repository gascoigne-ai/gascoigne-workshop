# 078 Protocol as a type with a capability table

From the harness audit of 2026-11-18: `Request.Protocol`, `BodyType` and the auth type are plain strings with about 65 literal comparisons outside the registry (`model.go`, `wsui.go`, `view.go`, `editor.go`, `access.go`, `postman.go`). Decision 2 says nothing switches on a protocol.

**Goal:** a protocol is a typed value; what differs per protocol is a table row, not a comparison.

Scope
1. `Protocol`, `BodyType` and auth type as named types with constants; parsers turn stored text into the type once, at the edge (unknown value is an error or an explicit `Unknown`).
2. A capability table per protocol (keeps a connection, has a composer, formats JSON, panes, label) in the `proto` registry or `panes.go`.
3. Replace `isSocket`, the URL bar flags, `composerHint` and the `editor.go` checks with table lookups.
4. Fix the `model.go` doc comment that lists 3 of 5 protocols.

Acceptance: no protocol literal outside the table rows and the format parsers (a test greps for it); decision 2 is true as written; `make check` green.
