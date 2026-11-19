status: todo
kind: feature
commit: feat(model): stringly state and bool pairs become named types
scope: internal/*,.harness/architecture.md
depends_on: 127-conn-state

# Replace magic strings and bool pairs with types

Origin: the repo audit of 2026-11-19 (three read-only reviews against the harness rules). The user approved all phases and, with that, the deletions and the gate change named here. Behavior stays the same unless the task says otherwise; tests stay green and none is loosened. Update `architecture.md` in the same commit where it describes what changes.

- `model.Access.Kind`, `model.Auth.KeyIn`, `ProtoSource.Kind()` (key of a two-row table) become named types with constants; `editor.AuthFieldSet.Field` (keyed into `authFields`, panics on a miss) becomes a typed key.
- `stdioPrefix` defined twice (`access.go`, `mcp.go`): `model.StdioPrefix`; the multipart `@path` rule written three times (`access.go`, `rest.go`, `postman.go`): `model.FormFile(value) (path, ok)`.
- `IsErr bool` in `prefs.Toast`, `sentlog.Host.Toast`, `calltrace.Entry`, `widget/toast.go`: a `Severity` type (the toast `Note` already takes a tone). `prefs.MarkUnsaved` / `MarkSaved` become one message; `prefs.Change` and `prefs.Edit` have identical fields: one type. `FileDialog.Choose(ctx, title, save bool, name)`: a mode enum.
- `uitoolkit`: `view/overlay.go` `mode int` and `FloatAt` storing `y, x` in `top, right` become a typed `floatMode` and `x, y` fields; `bars.go` `bar int` becomes `barKind`; `ScrollState` carries eight HScroll-only fields: split or nest them.
- Persisted values (auth types, body modes, keyIn) keep their wire strings; existing fixtures must still load unchanged.
## Model
Named string/int types with constants at the model or port that owns the concept; parse the wire string once at the boundary and refuse unknown values with an error. Alternative rejected: leave as strings. Revisit never; a new value is a new constant.
