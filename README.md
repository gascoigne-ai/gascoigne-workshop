# gascoigne-workshop

GPU-accelerated desktop client for REST, WebSocket, MCP and other API calls, written in Go.

Three-pane window: collections tree, request editor, response viewer. Drawn with
[Ebitengine](https://ebitengine.org) and a small declarative widget toolkit (`uitoolkit`), so it
starts fast and stays sharp on HiDPI screens.

Status: early. REST works; WebSocket and MCP are stubs. See
[`.harness/architecture.md`](.harness/architecture.md) for what exists today.

## Features

- Collections as a tree of folders and requests: rename, duplicate, move, delete.
- REST requests over `net/http`: params table kept in sync with the URL, headers, JSON, text, form
  and multipart bodies, `{{variables}}` from collection vars.
- Response viewer with pretty-printed, foldable and searchable JSON.
- Import of Postman and OpenAPI files.
- Dark and light themes, extendable with JSON theme files.

## Install

Download the Linux tarball from the
[latest release](https://github.com/gascoigne-ai/gascoigne-workshop/releases/latest), or build
from source.

## Build

Needs Go (see `go.mod`) and the Ebitengine Linux dependencies
(`libgl1-mesa-dev libxcursor-dev libxi-dev libxinerama-dev libxrandr-dev libxxf86vm-dev libasound2-dev`).

```sh
make build                    # bin/workshop
make run                      # run without installing
make install PREFIX=$HOME/.local   # binary, desktop entry, icon
```

## Develop

```sh
make check                    # gofmt, go vet, go test -race
make shot OUT=shot.png        # render the window once to a PNG (needs a display)
```

Rules and design notes live in [`.harness/`](.harness): `rules.md`, `architecture.md`,
`decisions.md`.

## Release

Pushing a tag `vX.Y.Z` runs `.github/workflows/release.yml`, which builds the Linux binary and
publishes it as a GitHub release.

```sh
git tag v0.1.0 && git push origin v0.1.0
```
