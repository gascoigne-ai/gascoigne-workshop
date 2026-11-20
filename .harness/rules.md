# Rules

Adds to the shared `common/rules.md`, `common/commit-style.md` and `common/code-style.md`.

- Approved deps: `github.com/hajimehoshi/ebiten/v2` (GPU window, input, 2D drawing, `text/v2`) `golang.org/x/image` (embedded Go fonts, already an Ebitengine requirement) and Ebitengine's other transitive requirements, and `google.golang.org/grpc` with `google.golang.org/protobuf` and their transitive requirements, `github.com/BurntSushi/toml`, and `github.com/bufbuild/protocompile`, and the build tool `github.com/tc-hib/go-winres` (run by `release.yml` at `v0.3.3`; not a module requirement). Everything else is stdlib; REST uses `net/http`.
- cgo is allowed here (Ebitengine needs it for windowing on Linux). Tests for non-UI packages must still run without a display.
- The look of the UI is written in `.harness/style.md`: a view follows it.
- Verify UI changes with `make shot` and look at the PNG; layout and widget logic still get unit tests. Golden files change only where the task changes the look.
- Layers: the toolkit knows nothing of the app (`uitoolkit/CLAUDE.md`), `internal/` is layered by domain and role (`internal/CLAUDE.md`); both are enforced by `layers_test.go`. `internal/request/model` and `internal/request/adapter/proto` stay free of UI.
- VML views put attributes before children, keep handlers as named closures or methods when they are longer than a call, and dot-import `view` (`core` is aliased `w`). UI is written as VML templates, inline in the Go file of the view (a `const` at its end, parsed once with `markup.Must`): no hand-rolled hit-testing or geometry in the app; a new component goes into `widget`, as a composition of `view` primitives or, when it needs its own layout, input or drawing, through `view.Custom`; a new primitive (color, geometry, drawing call) goes into `core`.
- Protocols are a table (`proto` registry keyed by name), widgets are values with `Update`/`Draw`; no switch on protocol or widget kind.
- Drawing and layout stay separate: layout is pure (rects in, rects out) and unit-tested without a GPU. Draw code is thin.
- UI colors, spacing and font sizes come from the theme tokens and size constants in `uitoolkit/core/theme.go` (the app adds its own tokens through `internal/settings/model/appearance`), never literals in widgets.

## Tasks
- Finish one task completely before starting another: every item of its Scope done and tested, its Acceptance met, `make check` green, docs updated. Do not switch tasks while items are open; an item that cannot be done (blocked, needs a decision, not possible on this platform) is written into the task file with the reason and what would unblock it, and the user decides whether the task closes.
- Work through a task in the order of its Scope. Commits follow the `task-loop` skill (one per task).

## Ambiguity
- When a request, a task item or a design can be read in more than one way, or a choice changes behavior the user would notice, ask before building: put the options, with a recommendation first, to the user and wait for the answer. Do not pick silently and do not guess to keep moving. This applies to tasks too: an unclear item is asked about, not interpreted.
- A question is for a real fork only. A choice with a conventional default, or a fact that can be read from the code, is decided and mentioned in the report, not asked.
- Record the answer in the task file or `decisions.md` when it will matter later.

## Layout of `internal/`
- `internal/<domain>/<role>/<package>`, enforced by `internal/layers_test.go`; the rules are in `internal/CLAUDE.md`, the tree in `packagemap.md`.

## Untrusted input
- Anything that comes from a file, a clipboard, an import or a server is untrusted: parsers return an error, never panic, hang or grow without a bound; a new parser or importer gets a fuzz target (`FUZZ_TARGETS` in the Makefile) and a limit written into `architecture/limits.md`.
- A request that runs a command or reads a file outside its collection goes through `App.allowed` (decision 21); a new protocol that does either extends `model.Request.Accesses`.
- Values from a server are never expanded as variables (`model.KV.Literal`).
