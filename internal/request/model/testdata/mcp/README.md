# MCP fixtures

Real output of the in-repo example server (`make example-server`, MCP endpoint
`/mcp`), captured with curl on 2026-11-20:

    curl -X POST http://127.0.0.1:<port>/mcp -H 'Content-Type: application/json' \
      -H 'Accept: application/json, text/event-stream' \
      -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'

- `tools-list-result.json`: the `result` member of the answer, unchanged
  (the JSON-RPC envelope `{"jsonrpc","id"}` is the only thing removed).

The example server answers `resources/list` and `prompts/list` with error
-32601 (method not found), so there is no real result to capture for them.
Captures from third-party MCP servers and from OpenAPI documents are still
missing.
