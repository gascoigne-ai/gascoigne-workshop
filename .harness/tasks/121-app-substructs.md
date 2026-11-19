status: todo
kind: refactor
commit: refactor(app): App and reqTab state grouped by concern
scope: internal/workspace/ui/app/*,.harness/architecture.md
depends_on: 120-split-mcp-and-others

# Group the App and reqTab fields

Origin: the repo audit of 2026-11-19 (three read-only reviews against the harness rules). The user approved all phases and, with that, the deletions and the gate change named here. Behavior stays the same unless the task says otherwise; tests stay green and none is loosened. Update `architecture.md` in the same commit where it describes what changes.

- `App` has about 90 flat fields. Group into `treeState`, `tableState`, `ports`, `frameState`, `persistState`, `settingsState` (names may change while writing; the groups follow the audit: tree selection/filter/drag/expanded/scroll; table edit/drag/bulk/selection/widths).
- `reqTab` has 40 fields for every protocol: `socketTab`, `grpcTab`, `mcpTab`, `responseTab`; the `outbox`/`outboxBusy` pair uses `controller/serial` (already a queue) instead of two fields.
- `NewApp` (108 lines): `opts.withDefaults()` beside the null ports, one struct literal, `loadSettings` and `loadState` helpers. `step` (68 lines): `beginFrame`, `routeKeys`, `endFrame`; compute `busy` once. `cmd/workshop/main.go`: `defineFlags`, `adapters`, `isolateScreenshot`, `openWindow`; the tests' `withAdapters` reuses the adapter list.
Acceptance: tests unchanged apart from field paths; `make check` green.
