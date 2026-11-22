# uitoolkit

Loaded when you work under `uitoolkit/`. Adds to `.harness/rules.md`.

- Layers: `core` imports nothing from this module, `view` only `core`, `widget` only `core` and `view`, `theming` only `core`, `markup` only those and `widget`; nothing here imports `internal/*` or `cmd/*`.

## Toolkit API rules

Apply when adding or changing anything exported. They exist so that a guess is usually right; a change that needs an exception says so in the doc comment of what breaks the rule and adds it to the exceptions in `.harness/architecture/toolkit.md`.

1. **Sides and axes.** One setting that can differ per side comes as the full family: all sides (`Pad`), an axis (`PadX`, `PadY`) and one side by letter suffix L, T, R, B (`PadL`...). This holds for padding, margin and border (`Border`, `BorderX`, `BorderY`, `BorderL`...). Never `Top`, `Bottom`, `Left` or `Right` words in option names, and never a half family.
2. **Elements and options.** A constructor makes an element (`Row`, `Field`, `Button`); an option sets an attribute of it (`Gap`, `Bg`, `OnClick`). Constructors take their required data positionally and everything else as `opts ...Option`; containers take `items ...Item`. Options may stand anywhere and a later one wins; style puts them first.
3. **Callbacks are options, named `OnX`.** Never a positional `func` parameter (`OnSelect`, `OnSlide`, `OnDismiss`, `OnChange`, `OnEnter`, `OnClick`). A mode and its handler are two options (`Modal()` and `OnDismiss(f)`), not one option with an argument.
4. **Flags take no argument.** `Small()`, `Lighten()`, `Optional()`. Conditions are the caller's `If` for elements and `When(cond, opt)` for options, never a bool parameter of the option (`When(i > 0, Separator(10))`).
5. **One meaning per name.** A name does not mean two things in the toolkit, and a constant does not share a name with an option or a function (`core.Gutter`, not `Pad`; `Ctx.Color`, not `Col`, which is a layout). Same-named tokens and options (`Bg`, `Muted`) are the accepted exception: tokens live in `core` and are always package-qualified.
6. **Suffix by kind.** Icons end in `Icon` (`TrashIcon`). State types that hold what a widget edits are named after the widget (`TextField`, `CodeView`) or end in `State` when the widget has none of its own (`ScrollState`); an element has the same stem as its state (`Field` for `TextField`, `Area` for `TextArea`, `Code` for `CodeView`).
7. **Constructors.** `NewX` returns what the caller stores; one family, one kind of return (all pointers or all values). A `Must` form exists only where the input is source code (`markup.Must`).
8. **One data shape, one type.** Two types with the same fields are one type (`widget.Span` is used by `CodeView` and by `Field`'s segments).
9. **Templates mirror Go.** Every option that can sensibly be written in markup has a template attribute that is its Go name in kebab-case (`BorderT` is `border-t`, `PadX` is `pad-x`); an option without one says why in its doc comment.
10. **A new option or element comes with** a doc comment that starts with its name, a test of what it does, a line in the package doc index (`view` or `widget`), and its row in the template tables where rule 9 applies.
11. **Packages.** `core` holds primitives: tokens, colors and color math, geometry, input, clipboard, fonts and the `Ctx` drawing calls, with no layout and no widget state; `view` is the framework: the element tree, layout, runtime, the generic options and the extension point (`Behavior`, `Custom`); `widget` is every component that has its own drawing, input or state (why: `doc.go`); `markup` builds both from templates; `engine`, `widget`, `markup` and `theming` depend downward only.

