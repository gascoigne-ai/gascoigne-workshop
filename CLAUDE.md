# gascoigne-workshop

Desktop GUI client in Go for running REST, WebSocket, MCP and other protocol calls. GPU-rendered with Ebitengine and custom widgets; layout is a three-pane window: collections tree, request editor, response viewer.

- `make check` must pass (fmt-check + vet + `go test -race`).
- Shared harness is the submodule `.harness/common`; read its `rules.md`, `commit-style.md` and `code-style.md`, then `.harness/rules.md` and `.harness/architecture.md`. Update architecture.md in the same commit as behavior changes.
- Design decisions: `.harness/decisions.md`.

@.harness/common/rules.md
@.harness/common/commit-style.md
@.harness/common/code-style.md
@.harness/rules.md
