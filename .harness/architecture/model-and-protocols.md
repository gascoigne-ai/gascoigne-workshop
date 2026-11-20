## Model and protocols

`model.Request` is protocol-neutral: `Protocol` picks the implementation, `Method` only matters for
REST (an HTTP method) and gRPC (a service method). `Protocol`, `BodyType` and `Auth.Type` are named
string types (`model.Protocol`, `model.BodyType`, `model.AuthType`) with constants; stored text
becomes one of them once, at the edge, with `ParseProtocol`, `ParseBodyType` and `ParseAuthType` (an
unknown or empty protocol, an unknown body or auth type or API key location (`model.KeyIn`,
`ParseKeyIn`; `""` stays the zero value, read as a header) is an error: a collection request file fails
the load naming the file, a client config names the server or request, a sent-log line is skipped).
What differs per protocol is a row of `model.Protocols` (`ProtocolInfo`: name, chip label,
`Connection` (`OneShot`, `Always`, `OnStream` for gRPC while its Stream chip is on), `Method`
(`MethodNone`, `MethodHTTP`, `MethodRPC`), `RunsCommand` (MCP `stdio:`), `ResumeHeader` (SSE
`Last-Event-ID`)), in the order the protocol chip cycles; `Protocol.Info()` finds it, and the zero
`ProtocolInfo` is the row of no protocol. `internal/protocol_literals_test.go` fails on a protocol
name literal, or a comparison, case or map key against a `model.Protocol*` constant, outside the
rows, the registry and a short list of files that use the same words for something else (settings
keys, the HTTP Upgrade token, the example server's service names and flags). `{{name}}` in a URL is
expanded from the collection vars (`model.Expand`) and the params table is folded into the URL with
`model.WithParams`; `model.ParamsFromURL` goes the other way (`model.MergeParams` rebuilds the table
after a URL edit, keeping descriptions and disabled rows), so the URL bar and the Params table stay
in sync. The caller does both before sending.

`proto.Registry` is a `model.Protocol`-to-`Protocol` table with one implementation per row of
`model.Protocols` (`rest`, `websocket`, `mcp`, `sse`, `grpc`; a test checks both directions, and
that a row that keeps a connection is a `Streamer`). `Send` returns an error only for misuse
(unknown protocol, malformed method or URL). Transport failures arrive in `Response.Err` so the UI
always has a value to render. `rest` uses `net/http`, keeps at most 32 MiB of body, and skips
disabled header rows. The request body is encoded by a table keyed by `Request.BodyType` (`none`,
`json`, `text`, `form`, `multipart`, `graphql`, `file`; `BodyRaw`, stored as "", sends `Body` as is
with no Content-Type) which also gives the automatic Content-Type; an enabled `Content-Type` header
of the user replaces it. Form and multipart bodies use the enabled, named rows of `Request.Form`; a
multipart value `@path` (one rule, `model.FormFile`, shared by the encoder, the access check and the Postman export) sends that file as a file part (named by its base name) and `@@x` sends the
text `@x`. `websocket`, `sse`, `mcp` and `grpc` are streaming or call clients of their own (decision
2).

`rest`, `sse`, the HTTP transport of `mcp` and the `websocket` handshake share one request builder
and one client factory (`proto/httpclient.go`). `newHTTPRequest` adds every enabled, named header
with `Add` (same names are all sent) and applies the Auth pane last, so it wins over a header typed
by hand; `newHTTPClient` applies the cookie jar, the `Network` proxy and certificates and the
request's `NoRedirects` and `InsecureTLS`; `applyTLS` is the TLS part, also used by `websocket`
(`ws.Options.TLS`). An unknown auth type is an error in all four (`unknown auth type "x"`; SSE and
MCP fail at connect, before any request). `Configured` builds one `Network` from the protocol config
and gives the jar, proxy, CA file and client certificate to `rest`, `sse` and `mcp`, and only the
TLS setting to `websocket`. SSE sets `Accept` and `Cache-Control` only when the user did not type
them. An API key sent in the query reaches the WebSocket server in the dialed URL.
