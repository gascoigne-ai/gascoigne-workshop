# 083 Dropped errors and missing contexts

From the audit: errors dropped without a debug log (`openapi.go:111`, `mcp.go:86,712`, `main.go:85`, `fonts.go:42`); a failed MCP reply write is ignored so the server waits forever; `exec.Command` in `mcp.go:629` and `preview.go:100`; `Session.Send/Ping/Close` take no context.

Scope
1. Regression test: failed MCP reply write surfaces an error.
2. Log or return each dropped error with its reason.
3. `exec.CommandContext` for MCP stdio and the browser opener.
4. Context first on `Session` methods and their fakes.

Acceptance: no silent error drop in the listed places; `make check` green.
