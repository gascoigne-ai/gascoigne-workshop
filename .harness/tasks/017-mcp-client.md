status: todo

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
