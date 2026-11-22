## Limits

Bounds that keep a hostile or huge input from freezing or crashing the app. Past a bound the input
is cut, refused or shown plainly, never a panic.

| Input | Bound | Past it | Constant |
|---|---|---|---|
| Response body in the viewer | 2 MiB | first part as plain lines, with a note; Save body keeps all | `respview.ViewLimit` |
| REST response body kept | 32 MiB | cut, `Response.Truncated` | `proto.maxBody` |
| Image response | 64 Mi pixels | not decoded | `respview.maxImagePixels` |
| JSON nesting in the viewer | 256 levels | plain text | `jsonview.maxDepth` |
| YAML nesting | 100 levels | error | `miniyaml.maxDepth` |
| Variable expansion | 16 MiB | stops | `model.maxExpanded` |
| Request timeout | 24 hours | capped | `settings.MaxTimeout` |
| Paste | 1 MiB | refused with a toast | `core.MaxPasteBytes` |
| Text field undo | 100 steps, 8 MiB of snapshots | oldest dropped | `widget.historyLimit`, `historyBytes` |
| Event-stream line or event (SSE, MCP over HTTP) | 16 MiB | ends the stream with the reason; an MCP session ends | `proto.maxEventBytes` |
| WebSocket message | 16 MiB | refused; `Options.MaxMessage` overrides | `ws.DefaultMaxMessage` |
| WebSocket handshake response | 64 KiB | connect fails, `response too large` | `ws.maxHandshakeBytes` |
| MCP message (stdio line, HTTP answer) | 16 MiB | refused or cut | `proto.mcpMaxMessage` |
| MCP stdio stderr kept | 20 lines of 512 bytes | older lines and line tails dropped | `mcpStderrLines`, `mcpStderrLine` |
| MCP request ids tracked for list payloads | 1024 | not tracked, no payload | `proto.maxPending` |
| MCP server requests and list refreshes at once | 16 workers | request answered with -32000 `client busy`, refresh dropped; one note per second | `proto.maxServerWork` |
| MCP busy replies queued | 1024 | reply dropped; the stream reader never waits | `proto.maxPending` |
| MCP sampling prompts open | 1 | user-rejected error at once | `proto.maxSamplePrompts` |
| JSON Schema | 32 levels, 1000 properties, 1000 enum entries | the rest left out, logged at debug | `model.maxSchemaDepth`, `maxSchemaProperties`, `maxSchemaEnum` |
| OpenAPI `$ref` chain | 6 levels | deeper refs not resolved (ends cycles) | `openapi.maxRefDepth` |
| gRPC reflection files per symbol | 512 | error | `proto.maxReflectedFiles` |
| gRPC message schema | 500 messages | the rest not expanded | `proto.maxSchemaMessages` |
| Key repeat output | 1,000,000 (ms, ticks or presses a second); an `xset` rate of 0 | unknown, so a timing never overflows `time.Duration` | `keyrepeat.maxSetting` |
| Saved file version | 0 to the newest the build reads | error naming the file; higher is `fileversion.NewerError` (Persistence) | `fileversion.Read` |
| Protocol, body type, auth type, key location names | exact match of a fixed list | error; the older auth `"none"` reads as none and is rewritten as absent | `model.Parse*` |

- The Postman, cURL, client config and SVG icon parsers set no size limit of their own: a huge
  input costs memory but parses without panic or hang. An SVG arc gives at most 4 segments.

### Fuzzing

`make fuzz` runs every target in `FUZZ_TARGETS` in the Makefile for `FUZZTIME` (30 s) each; it is not
part of `make check`. `TestEveryFuzzTargetIsInTheMakefile` (`e2e/fuzztargets_test.go`) fails on
a `Fuzz` function without a row, a row without a function, or a duplicate row. A crasher is saved
under the package's `testdata/fuzz`: fix it and keep the file as a regression test.

### Known gaps

- WebSocket dials directly (`ws.Dial` has no proxy support): the proxy setting does not apply; the
  CA file, client certificate and Insecure TLS do.
- Windows and macOS are compiled and vetted in CI but never run (`release.md`; Platforms).
- IME: the composition in progress is not drawn; Ebitengine does not report it (IME and dead keys).
- No screen reader: Ebitengine has no accessibility tree (Accessibility).
- Look: what the CLI dashboard has and the window does not is in `style.md`, Decisions (Not done).
- Fractional scales round logical text metrics to whole pixels.
- A folder collapsed at exit is open again at the next start: opening a collection opens every
  folder (`migration_test.go` notes it).
- Failures nobody can act on (WebSocket close and pong writes, MCP `listen` errors and `DELETE`,
  stdio input close) are logged at debug only. Adapters take the sink as a `debugFunc` field, so a
  test records it without a global logger.
