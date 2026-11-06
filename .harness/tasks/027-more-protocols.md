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

## Progress
- Done: SSE (`internal/proto/sse.go`, protocol `sse`, chip `SSE`): a receive-only streaming protocol on the Messages pane without a composer; event id shows as `[id] data`, the event name is the kind; one-shot send returns the first event; the Auto-reconnect chip applies. Tested against a streaming `httptest` server.
- Left: GraphQL (query editor, variables, introspection), gRPC (ask before adding dependencies), `Last-Event-ID` on reconnect.
- Done later: GraphQL as a body type of REST requests (`model.BodyGraphQL`, `Request.GraphQLVars`): the Body tab has a query editor, a JSON variables editor with validation, and `Load schema`, which sends the introspection query (`proto.IntrospectionQuery`) to the tab's URL with its headers and credentials and shows the answer as the response, without touching the tab's own query. The body type chips scroll sideways when the pane is narrow.
- Done later: `Last-Event-ID`: an SSE tab that reconnects offers the last event id it saw (`Event.ID`, `reqTab.lastEventID`), and a deliberate disconnect forgets it.
- Done later: gRPC unary calls with JSON messages through reflection (`internal/proto/grpc.go`, decision 13); the `gRPC` chip, a method chip that lists the server's methods, the Body tab for the message.
- Done later: GraphQL schema outline: `Schema (n)` menu of root operations that fill the query and variables with a starter (`proto.ParseSchema`).
- Left: gRPC streaming.
