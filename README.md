# gascoigne-workshop

GPU-accelerated desktop client for REST, WebSocket, SSE, MCP and gRPC calls, written in Go.

Three-pane window: collections tree, request editor, response viewer. Drawn with
[Ebitengine](https://ebitengine.org) and a small declarative widget toolkit (`uitoolkit`), so it
starts fast and stays sharp on HiDPI screens.

Start with [`.harness/overview.md`](.harness/overview.md), the 3000 ft view of the code. Rules and
design notes live in [`.harness/`](.harness): `rules.md`, `architecture.md`, `decisions.md`.

## Features

- **Protocols:** REST over `net/http` (params table kept in sync with the URL, headers, JSON, text,
  form, multipart and file bodies, auth, assertions), WebSocket, Server-Sent Events, MCP (tools
  explorer, streamable HTTP and stdio) and gRPC (server reflection, unary calls and streams).
- **Collections** are folders of JSON files, one open at a time: create, rename, save and delete a
  collection from the sidebar (right-click the *Collections* chip), and a tree of folders and
  requests with rename, duplicate, move and delete. Secrets are kept out of the request files.
- **Variables and environments:** `{{name}}` in URLs, headers and bodies, resolved from the
  collection, its folders and the active environment.
- **Response viewer:** pretty-printed, foldable and searchable JSON, raw and compact views,
  history, timeline, a call log.
- **Import and export:** cURL, OpenAPI and Postman (from the clipboard), export as Postman.
- **Settings** in one `settings.json` with a form and a raw editor, a shortcut editor, dark and
  light themes extendable with JSON theme files, and a System page showing the GPU rendering.

## Install

Download a build from the
[latest release](https://github.com/gascoigne-ai/gascoigne-workshop/releases/latest): a Linux
tarball, a macOS arm64 app bundle or a Windows zip. Only Linux is tested so far (see
[Known limitations](#known-limitations)). Or build from source.

## Build

Needs Go (see `go.mod`) and, on Linux, the Ebitengine dependencies
(`libgl1-mesa-dev libxcursor-dev libxi-dev libxinerama-dev libxrandr-dev libxxf86vm-dev libasound2-dev`).

```sh
make build                         # bin/workshop, with example endpoints
make release                       # bin/workshop-release, reads client configs instead
make run                           # run without installing
make install PREFIX=$HOME/.local   # binary, desktop entry, icon
make example-server                # one local server per protocol, for the example requests
```

## Use

### Collections

A collection is a folder on disk; the program opens the last one. Right-click the *Collections*
chip (or the collection's name in the tree) for *New collection*, *Open collection folder*,
*Rename collection*, *Save collection* and *Delete collection*. New collections are made in
`collections/` next to the default workspace, in the settings directory. Edits are saved a moment
after you stop typing; `Ctrl+S` saves at once.

### Client configs

A release build (`make release`) starts without example requests and reads a client config instead:
`.workshop.yaml` (or `.yml`, `.json`, `.toml`) in the settings directory and in the working
directory; an entry in the local file replaces one of the same name in the global file.

```yaml
servers:
  - name: Local REST
    protocol: rest
    url: http://127.0.0.1:8081/echo
    auth: {type: bearer, token: secret}
collections:
  - name: Demo
    vars: {baseUrl: http://127.0.0.1:8081}
```

### Shortcuts

| Keys | Action |
|---|---|
| Ctrl+Enter | Send the request (Connect on a socket) |
| Ctrl+L | Focus the URL |
| Ctrl+T | New request |
| Ctrl+W | Close the tab |
| Ctrl+Tab, Ctrl+Shift+Tab | Next and previous tab |
| Ctrl+S | Save the collection now |
| Ctrl+O | Open a collection folder |
| Ctrl+F | Find in the response |
| Ctrl+Space | Complete a `{{variable}}` |
| Ctrl+D, Alt+Up, Alt+Down, Delete | Duplicate, move and delete a table row |
| Ctrl+, | Settings |

All of them can be changed in Settings, Shortcuts.

## Known limitations

- Windows and macOS builds have never been run on a real machine: window and HiDPI behavior,
  clipboard, file dialogs, the browser opener and font folders are untested there. The Windows
  program has no embedded icon.
- File dialogs need `zenity` or `kdialog` on Linux; without them the program asks for a path.
- No text wrapping in the code view, no IME and no word-wise cursor movement.
- gRPC has no `.proto` file import: methods come from server reflection.
- One collection is open at a time.

## Develop

```sh
make check                    # gofmt, go vet, go test -race, also with -tags release
make shot OUT=shot.png        # render the window once to a PNG (needs a display)
make packagemap               # rewrite .harness/packagemap.md after adding a package
```

Golden screenshots (`cmd/workshop/testdata/golden`) are compared under software GL: on Linux run
`xvfb-run -a make check`; update them with
`xvfb-run -a go test ./cmd/workshop -run TestGoldenScenes -update`.

## Release

[`.harness/release.md`](.harness/release.md) is the checklist. Pushing a tag `vX.Y.Z` runs
`.github/workflows/release.yml`, which builds the Linux, macOS and Windows files and publishes
them as a GitHub release; running the workflow by hand (a dry run) builds them and keeps them as
workflow artifacts without a release.

```sh
git tag v0.1.0 && git push origin v0.1.0
```
