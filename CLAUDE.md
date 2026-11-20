# gascoigne-workshop

Desktop GUI client in Go for REST, WebSocket, MCP and other protocol calls: Ebitengine, own widget toolkit, three panes (collections tree, request editor, response viewer).

- The check command is `make check`.
- Start with `.harness/overview.md`. Read only the part of `.harness/architecture.md` (index at its top; one file per area in `.harness/architecture/`) for the area you touch. Reasons for a choice are in the doc comment of the code it shapes (else in the matching architecture file), the look in `style.md`, the package tree in `packagemap.md`.
- Docs: behavior in `architecture`, the look in `style.md`, the why of a choice (with the rejected alternative and when to revisit) in the doc comment of the owning package, type or function.
- Shared harness is the submodule `.harness/common`.

@.harness/common/rules.md
@.harness/common/commit-style.md
@.harness/common/code-style.md
@.harness/common/architecture-style.md
@.harness/rules.md

Task queue: `.harness/tasks/`, run with `/task-loop`.
