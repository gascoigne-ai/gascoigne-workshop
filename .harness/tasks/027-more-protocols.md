status: todo

# SSE, GraphQL and gRPC

## Goal
Protocols are a registry; REST, WebSocket and MCP exist.

## Scope
- **SSE:** a streaming protocol (`proto.Streamer`) with the event log pane of the WebSocket tab; `Last-Event-ID` and reconnect.
- **GraphQL:** query editor and variables pane, introspection to fill a schema outline, sends as REST under the hood; response viewer as usual.
- **gRPC:** decide first (new dependencies are needed for HTTP/2 framing and protobuf: ask before adding anything); reflection-based method list, unary calls with JSON input, streaming later.
- Each protocol is one registry row plus its pane builders; no switch on protocol name in the UI.

## Acceptance
- Tests against local stub servers for each; `make check` passes; decisions.md records any dependency decision.
