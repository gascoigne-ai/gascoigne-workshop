## Example server and client config

Decision 10. `make example-server` runs `cmd/example-server`
(`internal/request/adapter/exampleserver`), one port per service on `127.0.0.1`: REST 8081 (`/echo`,
`/status/{code}`, `/delay/{ms}`, `/auth`), WebSocket 8082 (echo), SSE 8083
(`/events?count=&interval=`), MCP 8084 (`/mcp`, tools `echo` and `add`), gRPC 8085 (health and
reflection) and gRPC without reflection 8086 (`grpc-no-reflection`: the same health service, reached
only through a `.proto` file, the server's own copy in
`exampleserver/protos/grpc/health/v1/health.proto`). Flags change host and ports; port 0 picks a
free one. The services are a table (`services`), so a new protocol or port is a new row.

The default build's `collection.Sample()` is only the `Example Server` collection (no GitHub calls;
the App itself still falls back to `GitHubSample` when no `Sample` option is given), with a request
for every endpoint of the example server on those ports: the REST routes are one table
(`restRoutes`, which also feeds the index page) and tests in `exampleserver` fail when a route or a
service has no sample request or a request uses another port (a gRPC request with proto files is for
the port without reflection); the sample's proto path is absolute, found from the debug build's
source tree (`exampleHealthProto`), since the sample lives in no folder and a relative path would
depend on where the program was started, and it reads no config file (`clientConfigs` in
`cmd/workshop/clients_*.go`, tested for both builds). It reopens what it remembered, like the
release build, from its own `gascoigne-workshop-dev` directory (`userdir.Name`, decision 20), so a
run from an IDE never touches the release build's data and the example server is what a first start
shows; `-workspace` still opens a folder. `userdir.Path(name)` is a file inside that directory ("" without a
config directory) and `userdir.WriteFileAtomic` the one temp-file-then-rename write every file adapter
uses (decision 26). Screenshot runs and the golden scenes keep settings and
state in memory, so they always start from the sample. `make release` builds `bin/workshop-release`
with `-tags release`, and `make install` installs that binary (no example servers; collections and
servers come from the client config in `~/.config/gascoigne-workshop/`): `Sample()` is empty and
`cmd/workshop` hands `clientfile.Files` to the App (`Options.Clients`). With no workspace (flag or
remembered), the App opens the client config's tree instead of the sample; an unreadable config is
named in the status line and the sample stays. `make check` also vets and tests with the tag.

`bus.Run` recovers a panic in a background command and dispatches `bus.Failed`, which the App shows
in the status line, so a protocol or parser bug does not end the app and its unsaved edits. A
whole-number setting outside its schema range is clamped when the file loads (`settings.Clamp`,
reported as a `*settings.ClampedError` that the App shows in the status line or a toast while using
the clamped settings; 0 stays "not set"). A settings file that does not parse is copied to
`settings.json.bak` before a save replaces it (`settingsfile.WriteText`). A client config is
`.workshop.yaml` (or `.yml`, `.json`, `.toml`) in the settings directory (global) and in the working
directory (local). An entry of the same name in the local file replaces the global one; two files in
one directory are an error. YAML is read by `miniyaml` (a subset), TOML by
`github.com/BurntSushi/toml`, JSON by `encoding/json`; all three become one generic tree that is
decoded strictly, so a misspelled key is an error. The client config opens as one collection, the
start one, so each configured collection becomes a folder (variables on the folder, environments on
the root) and servers become requests in a `Servers` folder.

```yaml
servers:
  - name: Local REST
    protocol: rest
    url: http://127.0.0.1:8081/echo
    auth: {type: bearer, token: secret}
collections:
  - name: Demo
    vars: {baseUrl: http://127.0.0.1:8081}
    envs:
      - name: Dev
        vars: {token: abc}
    requests:
      - name: Echo
        protocol: rest
        method: POST
        url: "{{baseUrl}}/echo"
        headers: {X-Trace: "1"}
        body_type: json
        body: '{"hello": "world"}'
    folders:
      - name: More
        requests: []
```

Request keys: `name`, `protocol`, `method`, `url`, `params` and `headers` (maps), `body`,
`body_type`, `timeout` (seconds), `auth` (`type`, `token`, `user`, `password`, `key_name`,
`key_value`, `key_in`). A server or request whose `protocol` is missing or unknown, or whose
`body_type` or auth `type` is unknown, is an error that names it.
