# Rules

Adds to the shared `common/rules.md`, `common/commit-style.md` and `common/code-style.md`.

- Approved deps: `github.com/hajimehoshi/ebiten/v2` (GPU window, input, 2D drawing, `text/v2`) `golang.org/x/image` (embedded Go fonts, already an Ebitengine requirement) and Ebitengine's other transitive requirements, and `google.golang.org/grpc` with `google.golang.org/protobuf` and their transitive requirements, `github.com/BurntSushi/toml`, and `github.com/bufbuild/protocompile`, and the build tool `github.com/tc-hib/go-winres` (run by `release.yml` at `v0.3.3`; not a module requirement). REST uses `net/http`.
- cgo is allowed here (Ebitengine needs it for windowing on Linux). Tests for non-UI packages must still run without a display.
- The look follows `.harness/style.md`; its Rules section is part of these rules.
- Layers: the toolkit knows nothing of the app (`uitoolkit/CLAUDE.md`), `internal/` is laid out as base, features, ui and app (`internal/CLAUDE.md`). `internal/ui/format` and `internal/feature/send/proto` stay free of UI.
- VML views put attributes before children, keep handlers as named closures or methods when they are longer than a call, and dot-import `view` (`core` is aliased `w`). UI is written as VML templates, inline in the Go file of the view (a `const` at its end, parsed once with `markup.Must`): no hand-rolled hit-testing or geometry in the app; a new component goes into `widget`, as a composition of `view` primitives or, when it needs its own layout, input or drawing, through `view.Custom`; a new primitive (color, geometry, drawing call) goes into `core`.
- The name Postman is allowed where it names the import and export format (the user's answer): the format is a file the workshop reads and writes, and a generic name would hide which format a user can import. Elsewhere `common/rules.md` Hygiene applies.
- Protocols are a table (`proto` registry keyed by name), widgets are values with `Update`/`Draw`; no switch on protocol or widget kind.
- Drawing and layout stay separate: layout is pure (rects in, rects out) and unit-tested without a GPU. Draw code is thin.

## Design
- Every design or plan (a task design note, a layout change, a new package) is made by an Opus agent (`model: opus`), read-only, before any code is written. Do not design on a smaller model, and do not skip the Opus pass because the change looks small.

## Tasks
- Finish a task completely before starting another. An item that cannot be done (blocked, needs a decision, not possible on this platform) is written into the task file with the reason and what would unblock it; the user decides whether the task closes.

## Ambiguity
- When a request, a task item or a design can be read in more than one way, or a choice changes behavior the user would notice, ask before building: put the options, with a recommendation first, to the user and wait for the answer. Do not pick silently and do not guess to keep moving. This applies to tasks too: an unclear item is asked about, not interpreted.
- A question is for a real fork only. A choice with a conventional default, or a fact that can be read from the code, is decided and mentioned in the report, not asked.
- Record the answer in the task file; when it will matter after the task, also as the why in the doc comment of the code it shapes, or in the matching `architecture/*.md` when no one package owns it.

## Layout of `internal/`
- `internal/{feature,storage,ui,app}`; the rules are in `internal/CLAUDE.md`.

## Untrusted input
- Anything that comes from a file, a clipboard, an import or a server is untrusted: parsers return an error, never panic, hang or grow without a bound; a new parser or importer gets a fuzz target (`FUZZ_TARGETS` in the Makefile) and a limit written into `architecture/limits.md`.
- A request that runs a command or reads a file outside its collection goes through `App.allowed` (its doc says why it asks rather than refuses); a new protocol that does either extends `request.Request.Accesses`.
- Values from a server are never expanded as variables (`request.KV.Literal`).
