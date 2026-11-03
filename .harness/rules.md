# Rules

Adds to the shared `common/rules.md`, `common/commit-style.md` and `common/code-style.md`.

- Approved deps: `github.com/hajimehoshi/ebiten/v2` (GPU window, input, 2D drawing, `text/v2`) `golang.org/x/image` (embedded Go fonts, already an Ebitengine requirement) and Ebitengine's other transitive requirements. Everything else is stdlib; REST uses `net/http`.
- cgo is allowed here (Ebitengine needs it for windowing on Linux). Tests for non-UI packages must still run without a display.
- Verify UI changes with `make shot` and look at the PNG; layout and widget logic still get unit tests.
- Layers: `widget` imports nothing from this module and `vml` only `widget` (a test enforces it); `internal/ui` uses both. `internal/model` and `internal/proto` never import `internal/ui` or Ebitengine. `internal/ui` never does network I/O; it calls `proto` through the `Registry` from a goroutine and receives results over a channel.
- UI is written as VML views (`internal/ui/view.go`): no hand-rolled hit-testing or geometry in the app; a new reusable piece goes into `vml` as a composition of primitives, a new drawing primitive into `widget`.
- Protocols are a table (`proto` registry keyed by name), widgets are values with `Update`/`Draw`; no switch on protocol or widget kind.
- Drawing and layout stay separate: layout is pure (rects in, rects out) and unit-tested without a GPU. Draw code is thin.
- UI colors, spacing and font sizes come from the theme tokens in `internal/ui`, never literals in widgets.
- Fixtures under `testdata/` are real captured responses (trimmed), never invented.
