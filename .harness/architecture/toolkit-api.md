## Toolkit: API and templates

-  **Toolkit layers** (`uitoolkit/`, decision 3; `uitoolkit/doc.go` describes them): `core` holds
  the primitives (the theme tokens in `core/theme.go`, colors and color math, the drawing context
  `Ctx`, pointer and keyboard `Input` with the `Edit` vocabulary of text keys, fonts, the
  clipboard); `view` the element tree, layout, runtime and the extension point; `widget` every
  component, the text inputs `TextField`, `TextArea` and `CodeView` with their elements `Field`,
  `Area` and `Code` included (a state type and its element share one stem); `markup` the templates;
  `theming` the palettes and theme files; `engine` the app-independent frame loop (`Pacer`,
  `MustDraw`, `InputActive`, and the `Binding` key model with parse/format, `Matches`, `BindingFrom`
  and `TypesText`/`IsFunctionKey`). App views dot-import `view` and reach `core` only for tokens,
  sizes and the context. Nothing under `uitoolkit` imports `internal/*` or `cmd/*`:
  `TestToolkitImportsStayInTheirLayer` (`uitoolkit/core/layers_test.go`) fails if it does and checks
  each toolkit package's imports against its layer.
-  **Toolkit API conventions** (developer experience): one naming rule (a side is a letter suffix,
  an axis is X or Y, sizes are `W H MinW MaxW MinH MaxH`); options may stand anywhere among the
  children and a later option overrides an earlier one (components rely on it), with options-first
  as the style; every package has a package doc, the tested examples of `view`, `widget`, `markup`
  and `theming` are one package, `uitoolkit/examples` (they use the exported API only, and
  `core/layers_test.go` checks its imports), and the `view` doc has a family index of the option
  names. Template errors name the file and line, suggest the nearest name and list the valid ones.
  `Ctx.PointerX/PointerY` are the pointer position that containers do not hide (drags); the slider
  measures drags in device pixels because its value can rescale the UI mid-drag.
-  **Every side the same way** (`view`): padding, margin and border follow one rule. All sides:
  `Pad`, `Margin`, `Border` (an outline); an axis: `PadX`/`PadY`, `MarginX`/`MarginY`,
  `BorderX`/`BorderY`; one side: a letter suffix L, T, R or B (`PadL`, `MarginT`, `BorderR`). Border
  lines are one-pixel hairlines in a theme token, `BorderInset` keeps the left and right lines off
  the corners, and `EdgeT`, `EdgeL`, `EdgeR` and `EdgeB` draw the separate two-pixel open-tab outline. The `Round` family is only the corner masks a panel needed (`RoundT`, `RoundB`, `RoundBL`; decision 27).
  Templates use the same names in kebab-case (`border-t`, `margin-x`, `pad-b`).
-  **Toolkit names**: the rules are in `rules.md` (Toolkit API rules). Icons end in `Icon`; the
  drawing color is `Ctx.Color` (`Col` is a layout); the inset constant is `core.Gutter` (`Pad` is an
  option); `widget.Span` is the one colored-run type; callbacks are options (`Modal()` +
  `OnDismiss(f)`, `OnSlide(f)` on `Slider`, `OnSelect(f)` on `Tabs` and `ScrollTabs`, `OnDragOver`);
  conditions are `If` and `When(cond, opt)`; `NewTextArea` returns a value; `Separator(inset)`,
  `TextCenter`, `KeepInView`, `FloatTR`, `CodeView.ScrollBy`. A component runs a handler through
  `El.Select(i)`, `El.Dismiss()` and `El.Changed()` (nothing happens without one). A component
  option put on the wrong kind of element panics with the option name and the element's control
  (`behaviorOf`), also on `Slider`'s options. `LoadUser` logs at debug why it skipped a theme
  folder whose name is no valid glob pattern.
-  **Toolkit surface trimmed**: helpers only the toolkit itself uses are unexported (`clampScroll`,
  `posFromOffset`, `tabWidth`, `tipDelayTicks`, `resizePair`, `newMultiline`, `blinkTicks`,
  `keyLabel`, `gradientEnds`, `newFonts`, `readInput`, `parseColor`, `themeExtends`). Option
  families (`MinW`, `AlignStart`, `JustifyBetween`...) stay exported even when the app does not use
  them yet: a half family breaks the naming rule.
-  **Theme pipeline lives in `theming`** (task 024, option A): `core` keeps the `Theme` and `Token`
  types, the token constants, `ControlStyle`, the color math (`RGB`, `Shade`, `Lighten`,
  `GradientFor`, `Mix`, `ParseHex` (also `ParseHexStrict`, `ParseHexAlpha`), `FormatHex`, `ContrastRatio`, `EnsureContrast`, all in `core/color.go`)
  and the drawing `Ctx`. `uitoolkit/theming` has the JSON theme parser (`ParseTheme`,
  `ThemeExtends`, `ParseColor`), the palettes and their files (`Dark`, `Light`, `themes/*.json`) and
  the preference mechanics (`Catalog`, `Prefs`). `theming` imports `core`, never the reverse, so
  `core`'s own tests build a tiny theme (`testTheme`) instead of calling `Dark()`.
-  **Theming mechanics are toolkit code** (`uitoolkit/theming`): `Catalog` (the built-in choices,
  the user's theme folder via `LoadUser`/`SetUser`, `For`, `ThemeID`), `Prefs` (theme, text size,
  corners, contrast, flat, color overrides) and `Catalog.Theme(Prefs)`, plus the choice tables
  (`SizeChoices()`, `CornerChoices()`, `ContrastChoices()`, `CustomColors()`, each a copy of an unexported table) and contrast rules.
  `internal/settings/model/appearance` is only the app's part: its embedded overlay themes (HTTP
  method and JSON colors), `MethodToken`, `LayoutChoices`, and thin wrappers that map
  `settings.Settings` to `theming.Prefs`.
-  **Templates** (`uitoolkit/markup`): `markup.View(src, scope)` builds an HTML-style template
  written inline in a Go method (parsed once per source string, panics on a mistake in the program);
  `markup.Parse`/`Must` and `Template.Build`/`MustBuild` are the explicit forms and
  `markup.Load`/`MustLoad` read a template from an `fs.FS` (kept for programs that keep templates in
  files; the app does not use them). The app writes each view as a template string in a `const` at
  the end of its Go file, parsed once at package level with `markup.Must("name.vml", nameSource)`
  (decision 4; the name is only what an error calls it; a template has one root, so a view that
  shows a note or a list is two templates and Go picks); a syntax error names the file and line.
  Parse does the work that does not depend on the scope, so a build does only the rest: each node
  keeps its element spec, its `for` and its params, every `{{path}}` text is split into literals and
  dotted paths once, and the option of an attribute with no `{{` is made once and reused by every
  build (the attribute readers capture only values and an option only sets fields of the element it
  is applied to, so it is safe to share). A static attribute whose value is bad is not an error at
  Parse: the build reports it with its line, and not at all while an earlier `if` hides the element,
  as before. Handlers, `if`, `for`, params and `{{...}}` attributes are read from the scope on every
  build. Building the app's view allocates about 4400 times instead of 6000.
  Event attributes are the `handlers` table: `on-click`, `on-click-at`, `on-right-click`,
  `on-right-click-in`, `on-middle-click`, `on-drag-over`, `on-change`, `on-select`, `on-dismiss`,
  each a scope function typed like the Go option's argument (`moved` is the same table's state
  entry: a `*view.MoveState`, nil rejected; `drag-handle` is a flag) (the error names the type); the
  same table holds `font-source`, which names a `*text.GoTextFaceSource` (nil keeps the interface
  font). Container elements mirror their widget constructors, whose arguments are attributes the
  element's table row lists (`params`), all required, so a missing one is a parse error; state and
  specs name a scope entry of their Go type, and a wrong type is an error naming it, as is a nil
  state pointer, which would otherwise panic at layout, and a nil `text-menu` field (decision 4).
  Only `text`, `dialog-subtitle`, `caption`, `button`, `primary-button`, `link-button`, `lines`, `note` and `choice` hold text (the
  table's `holds`): text inside `<spacer>`, a rule, `<slot>` or `<icon>` is a parse error, not dropped.
  The containers are `<scroll state>` and `<hscroll state>` (a child marked `keep-in-view` is
  scrolled to), `<split state default>` with the options only it takes, `vertical`, `as-ratio`,
  `min-first` and `min-second`, `<tabs items active>` and `<scroll-tabs state items active>` (no
  children; `on-select` gets the index), `<toolbar>` (`widget.Toolbar`, the bar's own row), `<tab-bar>` (`widget.TabBar`, the strip that holds them and
  anything beside them, its height, room above the tabs and line under them the widget's), with
  `separator` and `active-tab` for a hand-made strip, `<table spec>` (a `widget.TableSpec`,
  callbacks and edit state included), `<dialog>`, `<dialog-title title>` (its `<dialog-subtitle>` is the muted note, text only), `<dialog-toolbar>` and
  `<message-dialog state>` (a `*widget.DialogState`) and
  `<resizable-dialog state min-width min-height max-width max-height>` (a `*widget.ResizeState` and
  the size bounds, in the Go order), with `on-dismiss` for an outside click or the close button,
  `focus-scope` for a hand-made modal panel and `dismiss-margin` (`view.DismissMargin`: how many
  pixels outside the panel a click still counts as on it), `<menu ctx state>` (the `*core.Ctx` it
  measures its rows with and a `*widget.MenuState`; `on-select` gets the row) and
  `<toasts stack width height bottom-inset>` (a `*widget.ToastStack`; it takes no options, and an
  empty stack builds nothing). `<hrule>`, `<vrule>` and `<vrule-inset inset>` are the rules of
  `view`, the last one's inset a required param. Widget elements take their edit state the same way:
  `<field state>` edits a `*widget.TextField`, with `placeholder`, `field-pad`, `styled` (a
  `widget.Styler`, the whole text), `text-menu` (the `*widget.TextField` its right-click menu edits),
  `on-enter` and `on-tab` (a `func(back bool)`), and `<area state>` edits a `*widget.TextArea`, with
  `text-menu` naming its `&area.TextField`, and `<checkbox on>` (`true` or `false`, open to
  `{{path}}`; `on-change` runs on a click), and `<list-row selected>` is `widget.ListRow`
  (`selected` the same `true` or `false`; the row look, hover tint and selection background are the
  widget's, so a template gives only size, content and `on-click`), and `<icon-button icon>` is
  `widget.IconButton` around the named icon, grown to fill the chip (no children; size, `tooltip`
  and `on-click` are the generic attributes; an unknown icon is a build error naming the line), and
  `<segment-chip selected>` is `widget.SegmentChip`, one chip of a segmented choice (its text the label,
  `selected` `true` or `false` like `list-row`'s; a primary button when selected, a plain one
  otherwise, with the same size and `on-click`), and `<form-row label label-width>` is
  `widget.FormRow`, a settings-style row whose children are the controls after the label column
  (rows with the same `label-width` line their controls up; the row height and the label gap are the
  widget's), and `<check-row on>` is `widget.CheckRow`, a checkbox followed by its children (the
  label and a muted hint, since the Go constructor takes them as items too; `on` as `checkbox` reads
  it), given `on-click` so the whole row toggles, and `<lines color>` is `view.Lines`, one `LabelH`
  text per line of its text in that color (with `small`, as `view.Small()` among the Go options,
  every line is a small text `NoteH` high; `color` is the Go argument and required here, since the
  generic option would color the column and not its texts; literal text has its whitespace
  collapsed, so the lines come from `{{path}}`), and `<note tone>` is `widget.Note`, the one-line
  small message under a control (`error` for a problem, `muted` for a hint), and `<code state>`
  shows a `*widget.CodeView`, and `<image img>` a `*ebiten.Image` (nil draws nothing, as in Go, so
  it is not rejected), and `<slider state min max step value>` drags a `*widget.SliderState`, with
  `slider-mid`, `on-slide` and `on-slide-end` (each a `func(int)`); these are the element's own
  options and handlers (`options`, `handlers` in its row), so another element rejects them at parse
  instead of panicking at build. Scope values may be the app's own Go types (decision 4): each step
  of a `{{path}}` reads a map key or an exported struct field through pointers (a missing or
  unexported field is an unknown name, a nil pointer on the way an error naming the path), `for`
  walks a slice or array of any element type and nests (`for="child in row.Children"`, `$index` the
  item's index), and `if` hides on a value's zero by kind (false, nil pointer, func or map, empty
  string or slice, zero number); there are no methods, comparisons or filters, so anything computed
  is a field of a view model. A color attribute (`bg`, `border*`, `edge-*`, `hover-*`, `color`)
  given an empty token applies no option, so a view model switches it off per item
  (`border="{{row.DropBorder}}"`); a non-empty token is not checked at build, and an unknown one
  draws in the text color; `separator="0"` likewise applies none, so a tab strip gives its first tab
  0 (Go's `Separator(0)` is still a full-height line). Rules of the format are in decision 4 and the
  package doc. `view.Dump` prints a laid-out tree with every attribute that is not its zero value (one table
  of name and getter, `dumpedAttributes`, covers every look attribute of an element); pinned dumps
  are how a view is kept from changing. An element has one click and one right-click handler:
  `OnClick`/`OnClickAt` and `OnRightClick`/`OnRightClickIn` set the same one, the later option
  wins, and the dump names them `onClick` and `onRightClick`. The extension point of `view` is the
  public `Behavior`; the toolkit's own kinds (`flow`, `stack`, `textB`) implement it as well.
-  **Slots**: `<slot name="x"/>` in a template inserts the `view.Node` the scope holds under the
  path `x`, which is how a template contains another template's tree (the header's environment
  chip), a hand-built element or a `view.Custom` component, whose measure, layout and input run as
  in a Go tree (`custom_test.go`). The name is a path like `{{path}}`, so inside a `for` each item
  brings its own node (`<slot name="row.Knob"/>`). `<slot>` takes only `name`, `if`
  and `for`; any other attribute is a parse error, because there is no element of theirs to apply it
  to. `App.header` and `App.statusBar` are the embedded `header.vml` and `statusbar.vml`, their
  sizes from the scope; both are tested against the Go views they replaced.
-  **Layout attributes**: templates may set `align-items`,
  `align-self` (default, start, center, end, stretch, baseline) and `justify-content` (start,
  center, end, between, around) and use `<primary-button>` and `<link-button>`.
  `App.messagesToolbar` (`internal/workspace/ui/app/wsbar.go`, the embedded `messagestoolbar.vml`
  with its sizes from the theme through the scope and the auto-reconnect chip a `<segment-chip>`) is the
  first multi-handler template and is compared with its Go original in `wsbar_test.go`.
