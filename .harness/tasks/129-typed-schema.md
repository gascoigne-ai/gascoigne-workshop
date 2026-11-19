status: todo
kind: feature
commit: feat(mcp): tool schemas and MCP payloads are typed at the boundary
scope: internal/*,.harness/architecture.md
depends_on: 128-named-types

# Type the JSON Schema and MCP payloads

Origin: the repo audit of 2026-11-19 (three read-only reviews against the harness rules). The user approved all phases and, with that, the deletions and the gate change named here. Behavior stays the same unless the task says otherwise; tests stay green and none is loosened. Update `architecture.md` in the same commit where it describes what changes.

- JSON Schema is `map[string]any` throughout the core (`request/model/mcpschema.go`, `mcpform.go`, `collection/model/openapi/openapi.go`); the `required` set is parsed twice; about 15 unchecked type assertions. A typed `Schema` (Type, Properties, Required, Enum, Default, Const) decoded once at the server answer (`ParseTools`) and at the OpenAPI import.
- `controller/tabs/connection.go` `Received` runs `ParseTools`, `ParseResources`, `ParsePrompts` (three `json.Unmarshal`) on every received event of every protocol, on the UI goroutine; a WebSocket echo shaped like a tools list fills `Tools`. The MCP adapter parses once and emits typed payloads on `Event`.
- JSON-RPC messages in `mcp.go` are `map[string]any` (31 uses): `rpcRequest` / `rpcReply` structs and named error codes (`-32601` built twice).
- Test first with real captured fixtures under `testdata/` (trimmed real server output), a fuzz target for the decoder.
## Model
`model.Schema` (recursive, ordered properties kept as a slice to keep form order), `port.Event.Payload` as a small sum type (tools list, resources list, prompts list, none). Owner: request model and port. Alternative rejected: keep maps and add helper getters. Revisit if a schema keyword outside the listed set is needed.
