## Toolkit: API and templates

The package map is `uitoolkit/doc.go`; import and naming rules are `rules.md` (Layers, Toolkit API
rules). `uitoolkit/core/layers_test.go` enforces the layers, also for `uitoolkit/examples`, the
tested examples, which use the exported API only.

- **Families stay whole.** An option family (`MinW`, `AlignStart`, `JustifyBetween`, `PadL`...) is
  exported even where the app uses part of it, since a half family breaks rule 1; helpers only the
  toolkit uses are unexported. Borders are one-pixel hairlines in a token; `EdgeT/L/R/B` are the
  separate two-pixel open-tab outline. The `Round` family is only the corner masks a panel needed
  (`RoundT`, `RoundB`, `RoundBL`; see Exceptions).
- **Mistakes fail loudly.** A component option on the wrong kind of element panics, naming the
  option and the element's control (`behaviorOf`). A template error names the file and line,
  suggests the nearest name and lists the valid ones.
- **Theme pipeline.** `core` has the `Theme` and `Token` types and constants, `ControlStyle`, the
  color math (`core/color.go`) and `Ctx`. `theming` has the JSON parser (`ParseTheme`,
  `ThemeExtends`), the palettes and their files, and the preferences (`Catalog`, `Prefs`,
  `Catalog.Theme(Prefs)`, the choice tables). `theming` imports `core`, never the reverse, so
  `core`'s tests build a small `testTheme`. `internal/settings/model/appearance` is only the app's
  part: its overlay themes (HTTP method and JSON colors), `MethodToken`, `LayoutChoices`, and the
  mapping from settings to `theming.Prefs`.
- **Templates** (`markup`; format and why in its package doc). The app writes each view as a
  template in a `const` at the end of its Go file, parsed once with `markup.Must`. A template has
  one root, so a view that shows one of two things is two templates and Go picks. Parse does all
  the work that does not depend on the scope (element specs, `for`, params, `{{path}}` splits, the
  options of static attributes, shared by every build); a build reads only handlers, `if`, `for`,
  params and `{{...}}` from the scope. A path reads map keys and exported fields only: no methods,
  comparisons or filters, so anything computed is a field of a view model. A wrong scope type, a
  nil state pointer or a widget-only option on another element is a parse or build error, never a
  panic at layout. `Load`/`MustLoad` (templates in an `fs.FS`) are for other programs.
- **Slots.** `<slot name="path"/>` puts a scope `view.Node` into a template: another template's
  tree or a `view.Custom` component, whose measure, layout and input run as in a Go tree
  (`custom_test.go`).
- **Dumps pin views.** `view.Dump` prints a laid-out tree with every non-zero attribute (one
  name-and-getter table, `dumpedAttributes`). Pinned dumps keep a view from changing, and tests
  compare a template with its Go form this way.

### Exceptions to the toolkit API rules

Each is also said in the doc comment of what breaks the rule; a new exception adds a row here.

- Rule 1: `Round` is not a per-side family but the corner masks a panel needed (`RoundT`, `RoundB`,
  `RoundBL`); a mask is added with the panel that needs it, not ahead.
- Rule 7: state types are values (`NewTextField`, `NewTextArea`, `NewMenu`); `OpenMenu` stores a
  pointer to its own copy.
- Rule 11: `core.Ctx.Menu` is `any`, because `widget` owns `MenuState` and `core` cannot name it;
  that is the price of any element opening a menu without a host. `core.Ctx.FocusOwner` is typed,
  since its interface (`core.Focuser`, aliased by `view.Focuser`) uses no widget type.
- Exported mutable tables are copies (`theming.SizeChoices` and the other choice lists);
  `engine.keyNames` is unexported, since nothing outside `engine` reads it.
