# Gascoigne Workshop

**One fast desktop app for every API call you make.** REST, WebSocket, Server-Sent Events, MCP and
gRPC in a single window, with your requests kept as plain files you own.

> **Alpha.** 0.1.0 alpha is the first public release: ready for daily use and still growing, so
> details will keep improving. Your feedback shapes what comes next:
> [tell us what you find](https://github.com/gascoigne-ai/gascoigne-workshop/issues).

## Highlights

- **Five protocols, one place.** Stop switching tools: REST, WebSocket, SSE, MCP and gRPC share
  the same tabs, variables, history and response viewer.
- **Your requests are files.** A collection is a folder of small JSON files you can read, diff and
  commit to git. Secrets go to a separate file that stays out of version control.
- **Fast and sharp.** Drawn on the GPU, so it starts quickly and stays crisp on HiDPI screens.
- **Made for the keyboard.** Send with `Ctrl+Enter`, jump to the URL with `Ctrl+L`, rebind any
  shortcut.
- **Works with what you have.** Paste a cURL command, an OpenAPI spec or a Postman collection and
  start sending; export back to Postman when you need to share.
- **No account, no cloud.** Everything stays on your machine, and the program talks only to the
  servers you point it at.

## What you can do

- **Call anything:** REST with params, headers, JSON, form, multipart and file bodies, auth and
  assertions; WebSockets and SSE streams; MCP servers (tools explorer, streamable HTTP and stdio);
  gRPC from server reflection or `.proto` files, unary calls and streams.
- **Organize:** folders and requests in a tree with rename, duplicate, move and delete. Open several
  collections side by side; create, rename, save, reorder, close and delete them from the sidebar
  (right-click the *Collections* chip or a collection's name).
- **Reuse:** `{{name}}` variables in URLs, headers and bodies, resolved from the collection, its
  folders and the active environment.
- **Read responses well:** pretty-printed, foldable and searchable JSON, raw and compact views,
  history, a timeline and a call log.
- **Make it yours:** dark and light themes (extend them with JSON theme files), an editable
  shortcut map, and settings in one `settings.json` with a form and a raw editor.

Start with [`.harness/overview.md`](.harness/overview.md) if you want the 3000 ft view of the
code. Rules and design notes live in [`.harness/`](.harness).

## Install

Download an alpha build from the
[latest release](https://github.com/gascoigne-ai/gascoigne-workshop/releases/latest): a Linux
tarball, a macOS arm64 app bundle or a Windows zip. Or build from source.

On Linux, native open and save dialogs need one of `zenity`, `kdialog`, `yad`, `qarma` or
`matedialog` (most desktops ship one; `zenity` on GNOME and others, `kdialog` on KDE Plasma).
Without any of them the program opens its own file browser, whose footer names the one to install.

## Build

Needs Go (see `go.mod`) and, on Linux, the Ebitengine dependencies
(`libgl1-mesa-dev libxcursor-dev libxi-dev libxinerama-dev libxrandr-dev libxxf86vm-dev libasound2-dev`).

```sh
make build                         # bin/workshop, with example endpoints
make release                       # bin/workshop-release, reads client configs instead
make run                           # run without installing
make install                       # binary under ~/.local, plus desktop entry and icon on Linux (PREFIX=/usr/local for all users)
make example-server                # one local server per protocol, for the example requests
```

The settings directory holds `settings.json`, the remembered window and tabs (`state.json`),
history, themes and new collections. A release build uses `gascoigne-workshop` in the user config
directory (`~/.config` on Linux, `~/Library/Application Support` on macOS, `%AppData%` on Windows);
the default build (`make build`, `make run`) uses `gascoigne-workshop-dev` beside it, so the two
never share data. Both reopen the collections open last; the default build's first start shows the
example server.

## Use

### Collections

A collection is a folder on disk. Several can be open at once, each a root of the tree with its own
variables and environment; the program reopens the ones open last. Right-click the *Collections*
chip (or a collection's name in the tree) for *New collection*, *Open collection folder*,
*Rename collection*, *Save collection*, *Close collection* and *Delete collection*; New and Open add
the collection beside the open ones (a folder already open is shown, not opened twice). A
collection's name in the tree also has *Move up* and *Move down* to reorder them. Closing one keeps
its files; closing the last leaves an empty tree that offers to open or create one. New collections are made in
`collections/` next to the default workspace, in the settings directory. Edits are saved a moment
after you stop typing; `Ctrl+S` saves at once.

### gRPC with .proto files

A gRPC request learns the server's methods from its reflection. For a server without reflection,
open the request's *Proto* tab and choose *Proto files*: *Add file...* picks a `.proto` file and
*Add path* adds an import path (a folder searched for the files it imports, in order, as protoc
does; `google/protobuf/*` is built in). Paths inside the collection's folder are saved relative to
it, so the folder can move. The method chip then lists the services of those files, without a
server, and a problem in a file shows under the lists at its file, line and column. Choosing
*Reflection* again removes the files. The debug build's sample has *gRPC Health from .proto*, which
calls the example server's port without reflection (8086).

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
| Ctrl+Shift+W | Close the collection of the open request |
| Ctrl+Tab, Ctrl+Shift+Tab | Next and previous tab |
| Ctrl+S | Save the collection now |
| Ctrl+O | Open a collection folder beside the open ones |
| Ctrl+F | Find in the response |
| Ctrl+Space | Complete a `{{variable}}` |
| Ctrl+D, Alt+Up, Alt+Down, Delete | Duplicate, move and delete a table row |
| Ctrl+, | Settings |

On macOS Ctrl is Cmd. All of them can be changed in Settings, Shortcuts.

## Known limitations

- File dialogs are native on macOS and Windows. Linux uses the first of `zenity`, `kdialog`, `yad`, `qarma` and `matedialog` that is installed, in the order the desktop prefers (`kdialog` first on KDE Plasma, `matedialog` on MATE and `qarma` on LXQt when installed, `zenity` first elsewhere); without any of them the program opens its own file browser.
- No IME.

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
git tag v0.1.0-alpha && git push origin v0.1.0-alpha
```
