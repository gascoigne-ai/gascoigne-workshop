status: todo
kind: feature
commit: feat(proto): SSE, MCP and WebSocket honor the TLS and proxy settings
scope: internal/request/*,.harness/architecture.md
depends_on: 110-event-queue

# One HTTP request builder for every HTTP-based protocol

Origin: the repo audit of 2026-11-19 (three read-only reviews against the harness rules). The user approved all phases and, with that, the deletions and the gate change named here. Behavior stays the same unless the task says otherwise; tests stay green and none is loosened. Update `architecture.md` in the same commit where it describes what changes.

Behavior change the user noticed in the audit: only REST uses the configured HTTP client, so a self-signed `wss`, SSE or MCP server cannot be reached, and header and auth handling differ (`Add` vs `Set`; an unknown auth type is an error in REST and WebSocket but silently skipped in SSE and MCP).
- One `newHTTPRequest(ctx, method, request)` and one client builder taken from `REST.client`; `Configured` (`proto.go`) passes the config to all protocols; WebSocket sets `ws.Options.TLS`.
- Decide and apply one rule for an unknown auth type: an error, as REST does.
- Tests first: an httptest TLS server with a client CA / InsecureTLS for each of SSE, MCP, WebSocket; a header-set test.
## Model
No new types; `proto` gets one request builder and one client factory. Access pattern: called once per connect. Alternative rejected: leave four copies. Revisit if a protocol needs a different transport.
