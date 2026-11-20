## Limits

-  Key repeat output (`keyrepeat`): a count above `maxSetting` = 1,000,000 (milliseconds, ticks or presses a second) is unknown, as is an `xset` repeat rate of 0, so a desktop's output cannot overflow `time.Duration` into a negative timing.
-  MCP server work: answers to server requests and list refreshes run on at most `maxServerWork` =
  16 workers (`mcpSession.spawn`, a counting semaphore). A request that finds none free is answered
  with the JSON-RPC error -32000 `client busy`, written by one goroutine behind a queue of
  `maxPending` (1024) so the reader of the stream never waits for a slow send; a full queue drops
  the reply. A refresh that finds no worker is dropped. Either logs one `client busy: refused N ...`
  note per second, not one per message. Sampling prompts open at once: `maxSamplePrompts` = 1.
-  WebSocket dials directly (`ws.Dial` has no proxy support), so the proxy setting does not apply to
  `ws` and `wss`; the CA file, client certificate and Insecure TLS do.
-  Windows and macOS: the CI `build` job compiles and vets for them, but nothing has been run there
  (decision 12; Platforms).
-  IME: text arrives composed, but the composition in progress (preedit text, candidate window
  position) is not drawn, because Ebitengine does not report it (IME and dead keys).
- No screen reader: Ebitengine has no accessibility tree to publish (Accessibility).
-  Look: the CLI dashboard's outlined primary button and segmented pill groups, blurred shadows, 2
  px focus rings, a medium font weight and letter-spacing, 36 px table rows, a selected-row accent
  bar and floating cards need a renderer or layout change or an approval; changing `border`,
  `selected` or the scrollbar thumb color would break `TestNonTextElementsAreVisible`.
- Fractional scales round logical text metrics to whole pixels.
-  Saved file versions (decision 22): a version is a whole number from 0 to the newest the build
  reads; anything else (negative, fractional, text, `null`, a document that is not an object) is an
  error naming the file, and a higher one is a `fileversion.NewerError`, never a panic (`FuzzRead`).
-  Fuzzing: `make fuzz` runs every native fuzz target for 30 s each (`FUZZTIME`; the list is
  `FUZZ_TARGETS` in the Makefile: the YAML, OpenAPI, Postman and client config readers, cURL import,
  URL parameters, assertions, the compact JSON view, the WebSocket frame reader, templates, text
  editing, find and wrapping, GraphQL schemas, JSON Schemas, MCP lists, MCP messages and server info, protocol, body and auth
  type names, collection folders, the saved state, settings, request history and the sent log,
  images, theme files, the event-stream reader of SSE and MCP, settings text, bulk-edited table rows,
  key bindings, batches and requests an MCP server sends, tool forms, the JSON line view, display scale and key repeat output, and the SVG icon parser `FuzzParseSVG`, which has no size or depth limit of its own:
  an arc gives at most 4 segments and a number that overflows is an error). The list has 46 rows and `TestEveryFuzzTargetIsInTheMakefile` (`internal/fuzztargets_test.go`) keeps it in step with the code: a `Fuzz` function without a row, a row without a function and a duplicate row each fail `make check`. The fuzzing itself is not part of `make check`. A crasher is saved under the package's
  `testdata/fuzz`; fix it and keep the file as a regression test.
-  Protocol, body type and auth type names (`model.ParseProtocol`, `ParseBodyType`, `ParseAuthType`, `ParseKeyIn`)
  are matched exactly against a fixed list: no case folding, no trimming, no length to grow;
  anything else is an error. No authentication is stored as an absent auth `type`; the older
  `"none"` still reads as none, and saving rewrites it.
-  Input limits that keep a hostile or huge input from freezing or crashing the app: a response body
  over `respview.ViewLimit` (2 MiB) is shown as its first part in plain lines with a note (no
  formatting, folding or filtered search; Save body keeps all of it); variable expansion stops at 16
  MiB (`model.maxExpanded`); JSON deeper than 256 levels shows as plain text (`jsonview.maxDepth`);
  YAML nests at most 100 levels (`miniyaml.maxDepth`); a
  request timeout is at most 24 hours; gRPC reflection fetches at most 512 files per symbol; a gRPC
  message schema expands at most 500 messages (`maxSchemaMessages`); one line or one event of an
  event stream (SSE and MCP over HTTP) is at most 16 MiB (`proto.maxEventBytes`): a larger one ends
  the stream with the reason (`Connection lost: ...`; an MCP session ends), and a broken stream
  reports its read error instead of looking closed by the server; an image over 64 million pixels is
  not decoded (`respview.Decode`); a paste is at most 1 MiB (`core.MaxPasteBytes`, refused with a
  toast); a text field keeps at most 100 undo steps and 8 MiB of snapshots. More bounds: a WebSocket
  message is at most 16 MiB (`ws.DefaultMaxMessage`, `Options.MaxMessage` overrides); an MCP message
  is at most 16 MiB (`proto.mcpMaxMessage`); the WebSocket handshake response (status line and
  headers) is at most 64 KiB (`ws.maxHandshakeBytes`), a larger one fails the connect with
  `response too large`; an image is decoded only up to 64 Mi pixels width times height
  (`respview.maxImagePixels`); an OpenAPI schema follows `$ref` at most 6 levels deep
  (`openapi.maxRefDepth`); a decoded JSON Schema nests at most 32 levels
  (`model.maxSchemaDepth`), keeps at most 1000 properties per object (`maxSchemaProperties`) and
  1000 enum entries (`maxSchemaEnum`), and what is beyond is left out; the schema sits inside an MCP
  message of at most 16 MiB; the kept stderr of an MCP stdio server is at most 20 lines of 512
  bytes (`mcpStderrLines`, `mcpStderrLine`; the rest of a longer line is dropped).
- **Debug logs**: a failure nobody can act on is logged at debug level with its reason through
  `slog.Debug`, never dropped: the WebSocket close frame and pong writes (`ws.Options.OnError`
  replaces the log), the MCP HTTP `listen` errors and its normal "no server stream" answer (405 and
  the like, never a note), the session `DELETE`, and the stdio input close. Adapters take the sink
  as a `debugFunc` field set to `slog.Debug`, so a test records it without a global logger.
-  Postman, cURL and client config parsers (`postman.Parse`, `model.ParseCurl`, the `.workshop.*`
  reader) take bytes already in memory and set no size limit of their own: the caller bounds the
  input. cURL text comes from the clipboard; Postman and client config files are read whole with no
  size cap, so a huge file costs memory but is parsed without panic or hang (fuzz targets).
-  A folder collapsed at exit is open again at the next start: opening a collection opens every
  folder and the remembered open folders only add to that (`migration_test.go` notes it).
