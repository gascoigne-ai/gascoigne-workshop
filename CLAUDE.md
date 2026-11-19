# gascoigne-workshop

Desktop GUI client in Go for running REST, WebSocket, MCP and other protocol calls. GPU-rendered with Ebitengine and custom widgets; layout is a three-pane window: collections tree, request editor, response viewer.

- `make check` must pass (fmt-check + vet + `go test -race`, then the same vet and tests with `-tags release`).
- Shared harness is the submodule `.harness/common`; read its `rules.md`, `commit-style.md` and `code-style.md`, then `.harness/rules.md` and `.harness/architecture.md`. Update architecture.md in the same commit as behavior changes.
- Start with `.harness/overview.md` (the 3000 ft view); the package tree is `.harness/packagemap.md`.
- Design decisions: `.harness/decisions.md`. Visual style: `.harness/style.md`.

@.harness/common/rules.md
@.harness/common/commit-style.md
@.harness/common/code-style.md
@.harness/common/architecture-style.md
@.harness/rules.md

Task queue: `.harness/tasks/`, run with `/task-loop` (verify: `make check`).
