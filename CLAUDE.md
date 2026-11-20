# gascoigne-workshop

Desktop GUI client in Go for REST, WebSocket, MCP and other protocol calls: Ebitengine, own widget toolkit, three panes (collections tree, request editor, response viewer).

- `make check` must pass before work is called done.
- Start with `.harness/overview.md`. Read only the part of `.harness/architecture.md` (index at its top; one file per area in `.harness/architecture/`) for the area you touch. Reasons are in `decisions.md`, the look in `style.md`, the package tree in `packagemap.md`.
- Docs: behavior in `architecture`, the look in `style.md`, choices in `decisions.md`; each changes in the same commit as the behavior.
- Shared harness is the submodule `.harness/common`.

@.harness/common/rules.md
@.harness/common/commit-style.md
@.harness/common/code-style.md
@.harness/common/architecture-style.md
@.harness/rules.md

Task queue: `.harness/tasks/`, run with `/task-loop`.
