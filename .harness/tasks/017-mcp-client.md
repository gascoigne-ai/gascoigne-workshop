status: in progress (left: the notification stream, sampling)

# MCP client

## Goal
MCP is a stub that answers "not implemented yet".

## Scope
- Connect over streamable HTTP and stdio; `initialize`, then list tools, resources and prompts.
- Tool call form generated from the tool's JSON schema (strings, numbers, booleans, enums, objects as JSON), result viewer, raw JSON-RPC log.
- Needs a dependency decision (official SDK vs a small JSON-RPC client): ask the user before adding anything.

## Acceptance
- Tests against a stub MCP server (`cmd/mcpmock`-style fake): initialize, list, call, error, cancellation.
- `make check` passes.

## Progress
- Decision: hand-written client, no SDK (decisions 10).
- Done: streamable HTTP and stdio transports, initialize / lists / calls / ping, raw JSON-RPC log in the Messages pane, shorthand composer; tests against an HTTP stub (JSON and event-stream answers, errors, cancellation) and a stdio stub (the test binary re-run).
- Left: tool-call form generated from the tool's JSON schema, resource and prompt browsers, a result viewer, server-initiated requests (sampling, roots), the long-lived GET stream for server notifications.
- Done later: after the server announces its tools, resources and prompts the Messages toolbar gets `Tools (n)`, `Resources (n)` and `Prompts (n)` buttons. Picking one fills the composer with a ready-to-edit message: `call NAME {...}` with the arguments the tool's JSON schema asks for (`proto.Skeleton`: required properties first, defaults and enums used), `read URI`, or `prompt NAME {...}` with the required arguments. The reply is in the log, where responses are JSON.
- Left on purpose (together with the notification stream and sampling): a typed form with one input per schema property (the generated skeleton covers strings, numbers, booleans, enums, arrays and nested objects as JSON the user edits), server-initiated requests and the long-lived notification stream.
- Done later: requests the server sends are answered (`ping` and `roots/list` with an empty list, anything else refused with `method not found`), so a server that waits is not left hanging; both directions show in the log.
- Done later: a typed form per tool property (`proto.ToolFields`, `proto.Arguments`, `App.openToolForm`): picking a tool opens a dialog with one input per schema property and fills the composer with the typed call.
- Left: the long-lived GET stream for server notifications and sampling requests.
