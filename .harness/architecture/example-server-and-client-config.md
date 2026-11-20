## Example server and client config

The default build carries example endpoints and a release build reads client configs instead
(the user's choice). `make example-server` runs `cmd/example-server`, one port per service on `127.0.0.1`
(flags change host and ports; port 0 picks a free one):

| Port | Service |
|---|---|
| 8081 | REST: `/echo`, `/status/{code}`, `/delay/{ms}`, `/auth` |
| 8082 | WebSocket echo |
| 8083 | SSE: `/events?count=&interval=` |
| 8084 | MCP: `/mcp`, tools `echo` and `add` |
| 8085 | gRPC health and reflection |
| 8086 | gRPC health without reflection, reached only through its `.proto` file |

Builds:

- Default build: `collection.Sample()` is the `Example Server` collection, one request per endpoint
  (tests in `exampleserver` fail when one is missing). It reads no client config and keeps its data
  in its own directory (`userdir`; why in its package doc), so a run from an IDE never touches the release
  build's data. Screenshot runs and golden scenes keep settings and state in memory.
- Release build (`make release`, `-tags release`, installed by `make install`): no example server,
  `Sample()` is empty, and with no workspace the App opens the client config's tree. An unreadable
  config is named in the status line and the sample stays.

Client config:

- `.workshop.yaml` (or `.yml`, `.json`, `.toml`) in the settings directory (global) and in the
  working directory (local). A local entry replaces a global one of the same name; two files in one
  directory are an error.
- TOML is read with `github.com/BurntSushi/toml` (approved by the user), YAML with the `miniyaml`
  subset reader. All three formats become one generic tree, decoded strictly: a misspelled key is an error.
- It opens as one collection: each configured collection is a folder (variables on the folder,
  environments on the root), servers are requests in a `Servers` folder.

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
`key_value`, `key_in`). A missing or unknown `protocol`, or an unknown `body_type` or auth `type`, is
an error that names the server or request.
