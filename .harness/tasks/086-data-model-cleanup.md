status: todo
kind: feature
commit: refactor(app): typed state instead of any and bool pairs
scope: internal/*,.harness/*
depends_on: 085-toolkit-api-gaps

# 086 Data model cleanup

From the audit: repeated and untyped state in the App.

Scope (design note with the `data-model` skill first)
1. Screenshot scenes as a list of steps instead of ~25 repeated `App.Options`/`screenshot` fields (`app.go:277-302`, `shot.go:32-66`).
2. Typed tree node instead of `any` for `selected`, `dragNode`, `dropTarget` (`app.go:232-242`).
3. One choice type for the Vars target instead of `varsFolder` + `varsCollection` (`app.go:259`).
4. Enum for the MCP explorer state instead of `Disconnected`/`Calling` bools (`mcpexplorer.go:189`).
5. Typed set for `trusted` instead of `map[string]bool` (`app.go:264`).

Acceptance: tests unchanged in behavior; `make check` green.

## Model
Scene step list, typed tree node, one choice type for the Vars target, MCP explorer state enum, typed trusted set. Detail in the design note.
