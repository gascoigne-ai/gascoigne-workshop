status: todo
kind: feature
commit: feat(mcp): server messages are bounded and the parsers are fuzzed
scope: internal/*,Makefile,.harness/architecture.md
depends_on: 131-toolkit-api-rules

# Bound MCP server work; fuzz the remaining parsers

Origin: the repo audit of 2026-11-19 (three read-only reviews against the harness rules). The user approved all phases and, with that, the deletions and the gate change named here. Behavior stays the same unless the task says otherwise; tests stay green and none is loosened. Update `architecture.md` in the same commit where it describes what changes.

- `mcp.go`: one goroutine per server message (`go s.answerServer`, list refresh) and every `sampling/createMessage` opens a user prompt: a bounded worker or semaphore; a flood test first.
- Fuzz targets (untrusted input rule) for `deliver`, `deliverJSON`, `answerServer`, `sample`, `model.ToolFields` / `Arguments`, `jsonview.Lines`, `parseMonitors` (`displayscale`), the key-repeat parsers; list each in `FUZZ_TARGETS` and in `architecture.md`.
- A test that fails when a `Fuzz*` function exists but is missing from `FUZZ_TARGETS` (today 31 equal 31).
- MCP stdio: the server's stderr is discarded, so a crash reads "the server exited": keep the last lines and show them. Silent errors without a debug log (`ws.go` close and pong writes, `mcp.go` `listen` and `stdin.Close`, `openapi.go` dropped path items) get a debug log with the reason.
- Desktop query helper copied in `displayscale.go` and `keyrepeat.go` (1 s timeout, `log.Printf`): check whether `Factor()` is called from the frame loop; if it is, it blocks the UI up to 1 s every 2 s when `hyprctl` hangs: cache it off the frame loop (test first).
## Model
A bounded worker pool owned by the MCP session (size a named constant). No persisted data. Alternative rejected: drop messages silently. Revisit if legitimate servers hit the bound.
