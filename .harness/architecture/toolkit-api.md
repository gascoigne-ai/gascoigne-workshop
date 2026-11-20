## Toolkit: API and templates

The package map is `uitoolkit/doc.go`; import and naming rules are `rules.md` (Layers, Toolkit API
rules). `uitoolkit/core/layers_test.go` enforces the layers, also for `uitoolkit/examples`, the
tested examples, which use the exported API only.

- **Families stay whole.** An option family (`MinW`, `AlignStart`, `JustifyBetween`, `PadL`...) is
  exported even where the app uses part of it, since a half family breaks rule 1; helpers only the
  toolkit uses are unexported. Borders are one-pixel hairlines in a token; `EdgeT/L/R/B` are the
  separate two-pixel open-tab outline. The `Round` family is only the corner masks a panel needed
  (`RoundT`, `RoundB`, `RoundBL`; decision 27).
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
- **Templates** (`markup`; format in its package doc and decision 4). The app writes each view as a
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
