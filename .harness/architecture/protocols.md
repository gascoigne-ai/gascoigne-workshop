## Protocols

-  **Streaming and WebSocket** (`Streamer` and `Session` in `internal/request/port`;
  `proto/websocket.go`, `proto/ws/`): `Registry.Connect` opens a `Session` with
  a protocol that implements `Streamer`. `WebSocket.Send` is the one-shot form (connect, send the
  body, return the first reply as a `Response` with status 101). Headers, Auth pane credentials and
  a `Sec-WebSocket-Protocol` header (split into subprotocols) go into the handshake. A session logs
  a text or binary send as a `Sent` event before writing it, so an echo is never logged ahead of the
  message it answers (a failed write is logged after it, as the returned error). Every `Session`
  method that writes (`Send`, `Ping`, `Close`, `BinarySender.SendBinary`) takes the context of that
  write first: one that has already ended sends and logs nothing and returns its error, and `Close`
  with an ended context skips the goodbye (the MCP `DELETE`). The context ends the write, never the
  session; the App's outbox runs each send with the context of its background command, and closes
  after a timed-out connect use `context.WithoutCancel` so the goodbye still goes out. The tab that
  shows a session is WebSocket tabs below.
-  **WebSocket tabs** (`internal/workspace/ui/app/wsui.go`): a tab whose protocol keeps a connection
  (`isSocket`: `Connection` is `Always`, or `OnStream` while gRPC streams) keeps a `port.Session`.
  The URL bar button reads Connect / Disconnect (`toggleConnection`); dialing runs in a goroutine
  and its answer, every session `Event` and the end of the stream come back as five message types,
  `wsOpened`, `wsFailed`, `wsReceived`, `wsClosed` and `wsRetry`, each carrying one `wsTarget` (the
  tab and its `Connection.Gen`). `App.handle` drops a message whose generation is no longer the
  tab's (an abandoned or replaced connect) and closes the session of a stale `wsOpened`; the rest
  go to `applyWS`, a type switch with one arm per message. A `wsOpened` the connection refuses
  (the tab is no longer connecting) closes its session the same way. The response side is the row's `streamPanes` (`responsePanesFor`),
  for WebSocket the single `Messages` pane: a `CodeView` log with `Follow` set (pinned to the end
  while new lines arrive, paused when the user scrolls up, resumed at the end), Ping / Clear buttons
  and a composer field. The pane is the template `messagespane.vml`: Go hands it the toolbar
  (`messagesToolbar`, in a slot), the log's `CodeView`, the composer's `TextField` with its hint and
  the send and format handlers; flags from the protocol's `protocolViews` row leave out the composer
  of an event stream (`composer`) and the `{ }` button outside WebSocket (`formatsJSON`); the hint
  while connected is the row's `composerHint`, "Message" when it has none. Changing the protocol
  disconnects.
-  **WebSocket auto-reconnect**: the connection of a tab is the machine `tabs.ConnState`
  (`controller/tabs/connstate.go`): idle, connecting, open, closing, retrying. `Connection` keeps
  the state and the session private (the session exists exactly in open and closing); the events
  are `Begin`, `Opened`, `Failed`, `Closed` and `Disconnect`, and a transition table
  `[state][event]` decides them: 12 cells are legal, the 13 others are refused and change nothing
  (the app ignores a refused event, or closes the delivered session of a refused `Opened`). The
  `Auto-reconnect` chip sets `Connection.Reconnect` (per tab, not persisted), read when a connect
  fails or an open session ends: with it a move into retrying happens, without it the connection
  goes idle at once, so the button returns to Connect. `Active` is any state but idle, so while
  retrying the button reads Disconnect, which stops it. A session the user closed ends idle, never
  retrying. In retrying `scheduleRetry` logs it and a timer sends a `wsRetry` after `retryDelay`,
  which reconnects only if the tab is still retrying; the timer is not an event of the machine.
  Structure: the access pattern is one lookup per event on a handful of states, so an array table
  of cells (zero cell = refused); a map or a switch was rejected because the array makes the
  refused cells visible and a test walks all 25 of them; revisit when states or events double.

  ```mermaid
  stateDiagram-v2
    [*] --> idle
    idle --> connecting: Begin
    connecting --> connecting: Begin (newer connect)
    connecting --> open: Opened
    connecting --> retrying: Failed (Reconnect on)
    connecting --> idle: Failed (Reconnect off)
    connecting --> idle: Disconnect (abandoned)
    open --> retrying: Closed (Reconnect on)
    open --> idle: Closed (Reconnect off)
    open --> closing: Disconnect
    closing --> closing: Disconnect
    closing --> idle: Closed
    retrying --> connecting: Begin
    retrying --> idle: Disconnect
  ```
-  **Sending state** (`controller/tabs/sendstate.go`): whether a request is in flight is a second,
  separate machine, `SendState` (idle, sending) with its own table and events `BeginSend`,
  `Accept` and `Cancel`; `Seq` numbers the sends inside it so the late result of a cancelled or
  replaced send is dropped. It is not merged into `ConnState` because the two are independent: a
  gRPC tab can have a unary call in flight and then switch Stream on, which one merged machine
  would forbid (`TestASendInFlightDoesNotForbidAConnect` pins it).
-  **JSON composer for WebSocket** (`App.formatMessage`, `textfmt.CompactJSON`): the `{ }` button
  next to Send in a WebSocket tab turns the composer text into one line of JSON (key order and
  number text kept) or, when it is not JSON, leaves it and toasts where it breaks (a decoder reports
  the position).
-  **REST client options** (`proto.REST`): `Jar` (cookies kept for the app's lifetime, `App.jar`,
  replaced by `clearCookies`), `Proxy` (`settings.proxy`; changing the proxy or a certificate in the
  settings rebuilds the protocol registry, `saveSettings`), `MaxBody` (cut and flagged
  `Response.Truncated`), per request `InsecureTLS` and `NoRedirects`; `REST.client(req)` builds the
  `http.Client` for a request from those. Every send is traced with `net/http/httptrace` into
  `Response.Timing` (`internal/request/adapter/proto/trace.go`), shown by `timelineText`.
-  **SSE** (`internal/request/adapter/proto/sse.go`): `proto.SSE` is a `Streamer` over a long GET
  with `Accept: text/event-stream`; `readEventStream` (`eventstream.go`, shared with the MCP HTTP
  transport's streams) parses the event-stream format (event, data lines joined with newlines, id,
  comments and retry ignored) and stops with an error at a read failure or an event over
  `maxEventBytes` (Limits). Events arrive as `Received` `Event`s (kind `message`, the server's event name in `Event.Name`), the end of the stream as an info line; the session refuses `Send` and `Ping`. The UI
  treats it as a socket tab (`isSocket`) without the composer row.
-  **Session events** (`internal/request/port/port.go`): every line of a socket log is a
  `port.Event` whose `Kind` is a `port.EventKind`, one of `KindOpen` (connection up), `KindClose`
  (ended), `KindEnd` (this side of a stream closed), `KindInitialize` (an MCP server's description
  of itself, Info), `KindInfo` (a note of the UI), `KindError` (a problem when Info, a JSON-RPC
  error response when Received), `KindText` and `KindBinary` (WebSocket data frames), `KindPing`
  and `KindPong`, `KindMessage` (a gRPC message or a server-sent event), `KindRequest`,
  `KindResponse`, `KindNotification` and `KindInvalid` (JSON-RPC). `Event.Name` is what the peer
  called the event (the SSE `event:` field), empty otherwise. Consumers compare the constants
  (`Connection.Received`, `App.logEvent`) and never `Name`; an SSE name never reaches `Kind`, so a
  server cannot pose as an error or as the initialize note. `Event.Label` (the name of a
  `KindMessage` that has one, else the kind) is shown only when an event has no text. The name is
  untrusted text: a label, never expanded as a variable. Events are not persisted or serialized.
-  **GraphQL** (`internal/request/adapter/proto/graphql.go`,
  `internal/workspace/ui/app/graphql.go`): body type `graphql` encodes `{"query", "variables"}` as
  JSON (`encodeGraphQL`; bad variables and an empty query are errors); choosing it on a GET request
  makes it POST. `App.dispatch(tab, request)` is the one place a request is sent for a tab, so
  helper requests such as `loadSchema` reuse the tab's URL, headers and credentials. The editor
  (query, Load schema, variables with whether they parse) is the template `graphql.vml`.
-  **GraphQL schema outline** (`model.ParseSchema`, `App.rememberSchema` / `schemaMenu`): every
  successful response of a tab with a GraphQL body is tried as an introspection answer; one that is
  keeps the query and mutation root fields (`reqTab.schema`) and any other answer leaves the schema
  alone. The Body tab then shows `Schema (n)`, a menu with one row per operation
  (`query user(id: ID!): User`). Picking one replaces the tab's query and variables with a starter:
  the call with its arguments as variables, a selection of the result type's first eight scalar and
  enum fields (`__typename` when there are none), and zero values for required scalar arguments
  (`null` for the rest). The schema is not saved with the tab.
-  **MCP** (`internal/request/adapter/proto/mcp.go`, with the session in `mcpsession.go`, the transports in `mcphttp.go` and `mcpstdio.go`, server requests in `mcpserver.go` and sampling in `mcpsampling.go`): `proto.MCP` is a `Streamer` over two
  transports behind `mcpTransport`: streamable HTTP (POST per message, JSON or event-stream answers,
  `Mcp-Session-Id` and `MCP-Protocol-Version` headers, user headers and auth applied, DELETE on
  close) and stdio (`stdio:` + command line, newline-delimited JSON; the process runs under
  `exec.CommandContext` with a context the session owns, not Connect's, which ends with the
  handshake: closing ends its input and cancels that context a second later, so a server that
  ignores the end of its input is killed; its stderr is kept in a `tailLines` ring, the last 20
  lines of at most 512 bytes each, and shown only when the process ends, as `the server exited:
  <err>` then `stderr:` and the lines). `Connect` runs `initialize`, sends
  `notifications/initialized` and lists tools, resources and prompts the server announces; every
  message in both directions is an `Event` (kind request / response / error / notification), so the
  MCP tab reuses the WebSocket message-log pane (`isSocket`). The composer takes `call NAME {json}`,
  `read URI`, `prompt NAME {json}`, `method {json}` or a raw JSON-RPC object. After `initialize` it
  emits an Info event of kind `initialize` carrying the result, which `Connection.Received` parses
  with `model.ParseServerInfo` into `Connection.Server` (name, version, protocol, capabilities,
  instructions); the log does not show that event; a result that is not an object is noted as an
  Info `error` event and the session goes on without server info. A reply to a request the server
  sent (ping, `roots/list`, sampling) that cannot be written is an Info `error` event naming the
  method, since the server waits for it. The messages the client writes are structs
  (`mcprpc.go`: `rpcRequest`, `rpcReply`, `rpcError`, `initializeParams`) with named error codes
  (`rpcMethodNotFound`, `rpcInvalidParams`, `rpcRejected`), and their JSON keys are in
  alphabetical order, as the maps they replaced marshaled; a test pins the bytes. `port.Requester` (an MCP session implements it) sends a
  command like the composer does and returns the answer: the result JSON, or the server's error
  message.
-  **JSON Schema model** (`internal/request/model/jsonschema.go`): `model.JSONSchema` (`Type` a
  `SchemaType`, `Description`, `Properties` as `[]SchemaProperty` in document order with `Required`
  derived once from `required`, `Items`, `Enum`, `Default`, `Const` as `JSONValue` (the exact bytes;
  `Text()` is a string unquoted, anything else as compact JSON) and `Ref`) is decoded once by
  `DecodeJSONSchema` (also its `UnmarshalJSON`), once at the server answer (`MCPTool.Schema`) and
  once at the OpenAPI import; `Skeleton`, `ToolFields`, `Arguments` and the OpenAPI importer read
  the typed schema, with no map and no type assertion. Only these keywords are read; `anyOf`, `$defs`, `pattern`, `format` and the rest are
  ignored. A schema that is not an object (`true`, `false`, a string, `null`, an array) is the zero
  schema and no error; invalid JSON is the only error. A list of types takes the first that is not
  `null`, an unknown or non-string type is none, properties without a type make it an object, a
  `required` name without a property is dropped and a duplicate property key keeps the later value.
  Parts beyond the bounds under Limits are left out and logged at debug. Fuzz targets:
  `FuzzDecodeJSONSchema`, and `FuzzMCPDeliver` for what an MCP server sends. Real captures of the
  example server are under `internal/request/model/testdata/mcp`.
-  **MCP pickers** (`internal/request/model/mcpschema.go`, `internal/workspace/ui/app/wsui.go`):
  `model.DecodeTools`, `DecodeResources` and `DecodePrompts` read the result of a list answer once,
  in the MCP session, which sends it as `Event.Payload` (`port.ToolsList`, `ResourcesList`,
  `PromptsList`; nil for every other event). The session keeps `pending`, the method of each request
  it sent by id (also a raw JSON message typed in the composer; at most 1024, cleared when the
  session ends), and a response whose id maps to a list method carries the payload; the same shape
  as the result of `tools/call`, in a notification or from another protocol carries none. A result
  that does not decode is an Info line of kind `error` and no payload; one bad entry is skipped and a
  tool whose `inputSchema` is not an object keeps the zero schema (an empty form), so neither drops
  the list.
  `Connection.Received` fills `Tools`, `Resources` and `Prompts` from the payload only: a WebSocket
  frame never fills `Tools`, and an empty `ToolsList` (a list that changed to nothing) clears them.
  `model.Skeleton(schema)` builds a JSON value that fits a tool's input schema (required properties
  and those with a default, by name); a tool with the zero schema composes `{}`. A tab keeps what the
  server announced (`reqTab.tools`, `resources`, `prompts`, cleared when the session ends) and the
  toolbar buttons open menus whose entries compose a `call`, `read` or `prompt` line in the
  composer.
-  **MCP explorer** (`internal/workspace/ui/app/mcpexplorer.go`,
  `internal/request/controller/mcpexplorer`): the response side of an MCP tab is `mcpPanes` (the
  `streamPanes` of its `protocolViews` row): `Explorer` (`paneExplorer`) and `JSON-RPC` (the message
  log with the composer, every message in both directions as compact JSON with time and direction).
  The explorer shows the server info (`Connection.Server`), a list of tools, resources or prompts on
  the left (`mcpexplorer.Entries`) and the open entry on the right: the argument form
  (`mcpform.Form`, built from the tool's schema, or from a prompt's required arguments via
  `MCPPrompt.Schema`), a Call / Read button and the answer. The explorer is markup: `explorer.vml`
  lays out the server info header (an `explorerHeader` view model: the connection state until the
  server answered `initialize`, then its name, protocol, offers and instructions) above the list and
  the open entry, slotted in. `explorerlist.vml` walks `explorerEntry` view models in `<list-row>`
  rows under a `<tab-bar>` of `<tabs>`, the open entry is `explorerdetail.vml` over an `openEntry`
  (the form and the answer slotted in, the answer from `explorerresult.vml`) or `explorernone.vml`
  when nothing is open. `mcpexplorer.State` is the UI-free state (kind, selection, result and a
  `CallState`: `Idle`, `Calling`, `Answered` or `Failed`, read through `IsCalling()` and
  `IsFailed()`). `Begin` makes it `Calling`; `Finish` makes it `Answered`, or `Failed` for a server
  error or a result that says `isError`; `Select` drops the answer and goes `Idle`, except that a
  call in flight stays `Calling` (its answer still lands); `Choose` resets to `Idle`, also during a
  call. The open entry shows "Connect first" without a session and "Waiting for the server..." while
  `IsCalling()`; both can show together. A call goes through `port.Requester` on the bus
  (`App.exploreCall`), so the answer is matched to its request; `isError` in a tool result and a
  server error both show as Error. Enter in a field of the form calls. The toolbar pickers on the
  JSON-RPC pane still compose into the composer, tool forms there still open the dialog. Choosing
  another protocol resets both pane indexes.
-  **MCP tool form** (`model.ToolFields`, `model.Arguments`, `internal/workspace/ui/app/mcpform.go`,
  the fields built by `internal/request/ui/mcpform` from the markup template `form.vml` over one
  typed row per field): picking a tool from the `Tools (n)` menu opens a dialog with one input per
  property of its input schema, required ones first and starred: a text field for strings, numbers
  and integers, a checkbox for booleans (an optional one starts unset and is not sent until
  clicked), a row of buttons for enums (optional ones have `none`), a JSON text field for objects,
  arrays and untyped properties; defaults fill the fields. `Fill composer` builds the call with
  `model.Arguments`, which leaves out empty optional values, keeps a required empty string, parses
  numbers and JSON and refuses a value that does not fit with a message naming the field (shown in
  the dialog, which stays open); the composer then holds `call NAME {json}` and Send sends it as
  typed. A tool with no properties is composed at once.
-  **MCP notifications and sampling** (`mcphttp.go`, `mcpserver.go`, `mcpsampling.go`, `mcpform.go`): after `initialized` a streamable
  HTTP session opens the server's optional GET event stream (`mcpHTTP.listen`; a 405 or any
  non-stream answer is not an error) and everything on it goes through `deliver` like POST answers.
  Both kinds of stream are read by `mcpHTTP.readStream`. A POST answer stream that breaks or sends
  an event over `maxEventBytes` ends the session with
  `Connection lost: read the server's stream: ...`, since the call cannot complete. On the optional
  GET stream only an event over `maxEventBytes` ends the session; any other read error (such as an
  unexpected EOF) is an Info line of kind `error` and the session goes on without notifications. A
  stream cut by `Close` reports nothing. `notifications/{tools,resources,prompts}/list_changed` make
  the session ask for that list again (`listChanged`), so the toolbar counts and menus follow the
  server; one refresh of a list runs at a time and a notification that comes meanwhile makes it run
  once more (`refreshList`). `proto.MCP.Sampler` (nil: sampling is neither offered in `initialize` nor answered) puts a
  server's `sampling/createMessage` to a function; the app's `sampleViaDialog` shows the system
  prompt and messages in a dialog where the person types the reply (`Send reply`, model name `user`)
  or declines (the standard "User rejected sampling request" error). It runs on a session
  worker, only one prompt at a time (`maxSamplePrompts`): a second `sampling/createMessage` while
  one is open gets `User rejected`-code (-1) error "another sampling request is open" at once. It opens the dialog on the UI goroutine through `App.later`, waits while another dialog is
  open and takes the question off the screen when the session ends.
-  **gRPC** (`internal/request/adapter/proto/grpc.go`, `grpcreflect.go`, decision 2): `proto.GRPC`
  is a registry row; `Send` makes one unary call. It finds the method in the request's source of
  methods (`grpcsource.go`): `methodSources` is a table keyed by `model.SourceKind` (`ProtoSource.Kind`), with two
  rows behind one `methodSource` interface (service names, a service's descriptor). `reflection`,
  for a request without proto files, asks the server's reflection (v1, then v1alpha on
  `Unimplemented`) for the service's file and its imports (well-known types come from the program's
  own registry) and leaves the reflection service out of the list; `proto files` compiles the
  request's files (below) and lists the services those files declare, not their imports'. A file
  that is gone or does not parse is the send's error, with its path (and line); it never falls back
  to reflection. From the descriptor on, both are one path: it builds `dynamicpb` messages, turns
  the JSON body into the input with `protojson`, calls `conn.Invoke` with the enabled headers and
  the Auth pane's Authorization as lowercase metadata, and shows the answer as indented JSON
  (indented by `encoding/json`, because `protojson` randomizes its spacing). Header and trailer
  metadata are the response headers. `ListGRPCMethods` lists `package.Service/Method` for a picker.
  `Send` refuses streaming methods with a message. In the UI (`internal/workspace/ui/app/grpcui.go`)
  the protocol chip cycles REST, WS, MCP, SSE, gRPC; a gRPC tab shows a method chip instead of the
  HTTP one (`Pick method...` until set): clicking it lists the server's methods in the background
  (`pickGRPCMethod`, a `Cmd` whose result is a `func()` for the UI goroutine) in a context menu. The
  Body tab holds the JSON message, Send goes through the normal one-shot path, and switching back to
  REST replaces a `Service/Method` name by GET. `GRPCInputSchema` resolves the request's method the
  same way and turns the input descriptor into a `model.MessageSchema` (field names and types,
  repeated, maps, enum values, oneofs, nested messages; a message that contains itself is cut off at
  its second visit, `Cut`); `Skeleton` writes it as JSON with every field at its default and the
  first member of each oneof. `ListGRPCMethods` dials only for a source that asks the server (the
  table row's `asksServer`), so methods are listed from `.proto` files with no server entered, and
  the chip does not ask for one then.
-  **Proto pane** (`internal/workspace/ui/app/protoui.go`, template `protopane.vml`, decision 19): a
  gRPC request's panes are `grpcRequestPanes`, the HTTP ones with Proto after Body. Two chips choose
  the source: Reflection (a muted note, "Methods come from the server's reflection") or Proto files:
  the FILES and IMPORT PATHS lists, a path field and a Remove icon button per row, Add file...
  (`pickFile`, so the desktop dialog or the in-app browser) and Add path (an empty row to type
  into). The source is derived from the request (`ProtoSource.Kind`): choosing Proto files with no
  file yet only changes the tab (`protoState.filesChosen` of the tab), so the request still saves as
  reflection; choosing Reflection drops the files and import paths (`editor.ReflectionChosen`),
  which marks the tab edited. Edits are `editor` messages (`ProtoPathAdded`, `ProtoPathSet`,
  `ProtoPathRemoved`), so the dirty mark and the autosave see them. A row shows its path as the
  collection saves it (relative to the folder when inside, `model.StoredPath`) and a typed relative
  path is read from the folder (`LoadedPath`); a field is made again when its path changed by
  something other than typing. While the lists show and the files changed since the last read, the
  pane reads them in the background (`checkProtoFiles`, through the `GRPCMethods` port) and shows
  each problem under the lists as small `Error` text at `file:line:col`, the file named as the
  collection names it; an answer for an older source is replaced by a new read.
-  **Schema pane** (`internal/workspace/ui/app/schemaui.go`, template `schemapane.vml`): a gRPC
  request's panes are `grpcRequestPanes`, with Schema after Proto. With no method chosen
  (`methodChosen`: the name holds a `/`) it says to pick one. Otherwise it asks for the method's
  request message in the background (`loadMessageSchema`, through the `GRPCSchema` port, the same
  way the Proto pane reads files; an answer for an older URL, method or source is dropped by the
  `schemaKey` comparison) and lists one row per field (`schemaRows`:
  `name: repeated type (enum values), oneof x`, a nested message's fields one level deeper, a
  recursive one marked). Reload asks again; an error shows as small `Error` text. Fill body replaces
  the body with `MessageSchema.Skeleton` as typing it would (`editor.BodyTyped`, so it can be undone
  and marks the tab edited) and shows the Body pane. `-pane Schema` captures after the answer
  (`schemaLoading`); the `grpc-schema` golden scene shows it.
-  **gRPC from `.proto` files** (`internal/request/adapter/proto/grpcfiles.go`, decision 19):
  `compileProtoFiles` reads `.proto` files with protocompile into linked descriptors. Relative files
  and import paths are taken from a base directory given by the caller; a request holds absolute
  paths (the store resolves them against the collection's folder on load), so the `proto files` row
  passes none. A file is named as protoc would name it, by its path below the first import path that
  holds it, or by its base name with its folder searched after the import paths; imports are looked
  up on the import paths in order, then the standard `google/protobuf/*` files. Every problem in the
  sources comes back as a `model.ProtoError` (path, line, column, message), joined; a missing import
  is one at its `import` line.
-  **gRPC streaming** (`internal/request/adapter/proto/grpcstream.go`, `App.toggleStream`): `GRPC`
  is also a `Streamer`. `Connect` resolves the method from the request's source, as `Send` does,
  refuses a unary one, opens `conn.NewStream` with the method's own direction flags and sends the
  Body as the first message (blank: `{}` for a server stream, nothing for client and bidirectional
  streams, which stay open on this side). Received messages are `Received` events with compact JSON;
  every sent message is a `Sent` event; the end of the stream is an `Info` `close` event with the
  gRPC code (none when the user closed it). `Session.Send` takes a JSON message of the input type,
  or the text `end` to close this side (not valid JSON, so never a message). In the UI the `Stream`
  chip beside the method chip sets `reqTab.grpcStream`; while it is on `isSocket` is true, so the
  tab has Connect / Disconnect and the Messages pane with its composer. Turning it off disconnects.
  The setting is not saved with the request.
