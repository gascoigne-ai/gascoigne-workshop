status: todo
kind: feature
commit: feat(settings): schema rows get typed getters and setters
scope: internal/settings/*,.harness/architecture.md
depends_on: 129-typed-schema

# Give settings schema rows typed getters and setters

Origin: the repo audit of 2026-11-19 (three read-only reviews against the harness rules). The user approved all phases and, with that, the deletions and the gate change named here. Behavior stays the same unless the task says otherwise; tests stay green and none is loosened. Update `architecture.md` in the same commit where it describes what changes.

`settings/schema.go` `asMap`, `Value`, `Changed`, `Reset`, `With`, `Minimal` marshal the whole `Settings` to a `map[string]any` on every call; the form calls `Value` per field per frame; errors are dropped at five sites. Each schema row gets `Get`/`Set` functions on the typed struct; delete the map round trip.
- Also in this task: `settingsui` pages repeat the "field plus refusal note" state (`inputs`/`notes` maps in `dialog.go`, `form.go`, `colorpage.go`, `shortcutpage.go`): a small generic `noted[K]` helper; `ReservedForText` and `TextKeys` state the same keys twice and are exported mutable slices: one key table, unexported; the raw-settings write, toast and stamp duplicated in `raw.go` and `dialog.go`.
- Also: `model.Node` migration. `request/model/tree.go` operations take `any` although `model.Node` exists (38 call sites, `Node.Any()`); `Parent`, `Rename`, `Duplicate`, `Delete` take and return `Node`; delete `Any()`.
- Test first: a table of every schema row round-tripping a value; the settings file format and fixtures unchanged.
## Model
Schema row = name, type, default, `get(Settings) Value`, `set(*Settings, Value) error`. Access pattern: per field per frame (reads), rare writes. Alternative rejected: cache the map. Revisit if rows exceed a few hundred.
