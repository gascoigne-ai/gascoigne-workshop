status: todo
kind: refactor
commit: refactor(proto): split mcp.go into session, transports and sampling
scope: internal/request/adapter/*,uitoolkit/view/*,uitoolkit/widget/*
depends_on: 119-split-markup-component

# Split mcp.go and the grab-bag toolkit files

Origin: the repo audit of 2026-11-19 (three read-only reviews against the harness rules). The user approved all phases and, with that, the deletions and the gate change named here. Behavior stays the same unless the task says otherwise; tests stay green and none is loosened. Update `architecture.md` in the same commit where it describes what changes.

- `mcp.go` (771 lines, section dividers) into session, HTTP transport, stdio transport, server requests and sampling; `Connect` (62 lines) split.
- `uitoolkit/view/el.go` options regrouped into `options_size.go`, `options_look.go`, `options_events.go`; `OnRightClick`, `DismissMargin`, `OnChange`, `KeepInView`, `OnSelect` and `When` move next to their group.
- `U/widget/textfield.go` split is in the small-dedup task: do not repeat it.
Acceptance: moves only; `make check` green.
