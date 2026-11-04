# Rules

Adds to the shared `common/rules.md`, `common/commit-style.md` and `common/code-style.md`.

- Approved deps: `github.com/hajimehoshi/ebiten/v2` (GPU window, input, 2D drawing, `text/v2`) `golang.org/x/image` (embedded Go fonts, already an Ebitengine requirement) and Ebitengine's other transitive requirements. Everything else is stdlib; REST uses `net/http`.
- cgo is allowed here (Ebitengine needs it for windowing on Linux). Tests for non-UI packages must still run without a display.
- Verify UI changes with `make shot` and look at the PNG; layout and widget logic still get unit tests.
- Layers: `uitoolkit/core` imports nothing from this module, `vml` only `core`, `component` and `theming` only `core` and `vml`, and nothing under `uitoolkit` imports `internal/*` or `cmd/*` (`uitoolkit/core/layers_test.go` enforces it); `internal/ui` uses them. `internal/model` and `internal/proto` stay free of UI.
- VML views put attributes before children, keep handlers as named closures or methods when they are longer than a call, and dot-import `vml` (`core` is aliased `w`). UI is written as VML views (`internal/ui/view.go`): no hand-rolled hit-testing or geometry in the app; a new reusable piece goes into `vml` as a composition of primitives, a new drawing primitive into `widget`.
- Protocols are a table (`proto` registry keyed by name), widgets are values with `Update`/`Draw`; no switch on protocol or widget kind.
- Drawing and layout stay separate: layout is pure (rects in, rects out) and unit-tested without a GPU. Draw code is thin.
- UI colors, spacing and font sizes come from the theme tokens in `internal/ui`, never literals in widgets.
- Fixtures under `testdata/` are real captured responses (trimmed), never invented.

## Toolkit API rules (`uitoolkit/*`)

Apply when adding or changing anything exported. They exist so that a guess is usually right; a change that needs an exception records it in `decisions.md`.

1. **Sides and axes.** One setting that can differ per side comes as the full family: all sides (`Pad`), an axis (`PadX`, `PadY`) and one side by letter suffix L, T, R, B (`PadL`...). This holds for padding, margin and border (`Border`, `BorderX`, `BorderY`, `BorderL`...). Never `Top`, `Bottom`, `Left` or `Right` words in option names, and never a half family.
2. **Elements and options.** A constructor makes an element (`Row`, `Field`, `Button`); an option sets an attribute of it (`Gap`, `Bg`, `OnClick`). Constructors take their required data positionally and everything else as `opts ...Option`; containers take `items ...Item`. Options may stand anywhere and a later one wins; style puts them first.
3. **Callbacks are options, named `OnX`.** Never a positional `func` parameter (`OnSelect`, `OnSlide`, `OnDismiss`, `OnChange`, `OnEnter`, `OnClick`). A mode and its handler are two options (`Modal()` and `OnDismiss(f)`), not one option with an argument.
4. **Flags take no argument.** `Small()`, `Lighten()`, `Optional()`. Conditions are the caller's `If` for elements and `When(cond, opt)` for options, never a bool parameter of the option (`When(i > 0, Separator(10))`).
5. **One meaning per name.** A name does not mean two things in the toolkit, and a constant does not share a name with an option or a function (`core.Gutter`, not `Pad`; `Ctx.Color`, not `Col`, which is a layout). Same-named tokens and options (`Bg`, `Muted`) are the accepted exception: tokens live in `core` and are always package-qualified.
6. **Suffix by kind.** Icons end in `Icon` (`TrashIcon`). State types that hold what a widget edits are named after the widget (`TextField`, `CodeView`) or end in `State` when the widget has none of its own (`ScrollState`); an element has the same stem as its state (`Field` for `TextField`, `Area` for `TextArea`, `Code` for `CodeView`).
7. **Constructors.** `NewX` returns what the caller stores; one family, one kind of return (all pointers or all values). A `Must` form exists only where the input is source code (`component.Must`).
8. **One data shape, one type.** Two types with the same fields are one type (`vml.Span` is used by `CodeView` and by `Field`'s segments).
9. **Templates mirror Go.** Every option that can sensibly be written in markup has a template attribute that is its Go name in kebab-case (`BorderT` is `border-t`, `PadX` is `pad-x`); an option without one says why in its doc comment.
10. **A new option or element comes with** a doc comment that starts with its name, a test of what it does, a line in the `vml` package doc index, and its row in the template tables where rule 9 applies.
11. **Packages.** `core` has no layout and no widgets with state; `vml` is everything that is a view or an input; `engine`, `component` and `theming` depend downward only.

Known debt against these rules is listed in `.harness/backlog.md` (Toolkit).
