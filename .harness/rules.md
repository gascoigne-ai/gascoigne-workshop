# Rules

Adds to the shared `common/rules.md`, `common/commit-style.md` and `common/code-style.md`.

- Approved deps: `github.com/hajimehoshi/ebiten/v2` (GPU window, input, 2D drawing, `text/v2`) and its transitive requirements. Everything else is stdlib; REST uses `net/http`.
- cgo is allowed here (Ebitengine needs it for windowing on Linux). Tests for non-UI packages must still run without a display.
- Layers: `internal/model` and `internal/proto` never import `internal/ui` or Ebitengine. `internal/ui` never does network I/O; it calls `proto` through a `Protocol` value and receives results over a channel.
- Protocols are a table (`proto` registry keyed by name), widgets are values with `Update`/`Draw`; no switch on protocol or widget kind.
- Drawing and layout stay separate: layout is pure (rects in, rects out) and unit-tested without a GPU. Draw code is thin.
- UI colors, spacing and font sizes come from the theme tokens in `internal/ui`, never literals in widgets.
- Fixtures under `testdata/` are real captured responses (trimmed), never invented.
