## Model and protocols

- `model.Request` is protocol-neutral. What differs per protocol is a row of `model.Protocols`;
  `internal/protocol_literals_test.go` fails on a protocol name literal or a comparison with a
  `model.Protocol*` constant outside the rows, the registry and a short list of files that use the
  same words for something else.
- Protocol, body type, auth type and API key location are parsed once, at the edge. An unknown one
  fails the collection file (named), names the server or request of a client config, and skips a
  sent-log line.
- The URL bar and the Params table stay in sync through `model.WithParams`, `ParamsFromURL` and
  `MergeParams`; the caller expands `{{variables}}` and folds the params in before sending.
- `proto.Registry` maps each row of `model.Protocols` to a `port.Protocol` (why the table is split: the `model.Protocols` doc); a test
  checks both directions and that a connected row is a `Streamer`. `Send` errors only on misuse;
  transport failures arrive in `Response.Err`, so the UI always has a value to render.
- `rest`, `sse`, the HTTP transport of `mcp` and the `websocket` handshake share one request
  builder and client factory (`proto/httpclient.go`). The Auth pane is applied last, so it wins over
  a header typed by hand; a user `Content-Type` replaces the automatic one.
- `Configured` gives the cookie jar, proxy, CA file and client certificate to `rest`, `sse` and
  `mcp`, and only the TLS settings to `websocket` (Limits).
- A multipart value `@path` sends a file; one rule, `model.FormFile`, serves the encoder, the access
  check (`App.allowed`) and the Postman export.
