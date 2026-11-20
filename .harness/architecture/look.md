## Look

Dialog, pane and sidebar layouts are in `style.md`; this section is the theme system and the
app-wide rules. Sizes and tokens are in `uitoolkit/core/theme.go`.

- **Spacing.** Insets are `core.Gutter`; heights come in steps (`RowH`, `CompactH`, `LabelH`,
  `NoteH`) and gaps in two (`TightGap`, `LooseGap`). Markup has no constants, so a template reads
  them from its scope (`{{rowH}}`). Bars are separated by `Border`, never `Rule` (outlines and table
  headers). Both panes start their first row one `Gutter` below their tabs, so rows line up across
  the divider (`TestRequestAndResponsePanesShareTheirFirstRowBand`).
- **Click targets are outlined** at rest (buttons, unfocused fields and tables in `Border`,
  primary buttons in `AccentLine`), drawn after the children; a focused field turns `AccentLine`.
- **Theme files** (`uitoolkit/theming/themes/*.json`, format in `themefile.go`): `variables`, a
  flat `colors` table and per-control styles (`radius`, `gradient`, `border`). `ParseTheme` merges
  a file over a base and rejects unknown fields, variables and tokens. The app lays its HTTP-method
  and JSON colors over them from `internal/settings/model/appearance/themes`.
- **Controls draw from their style.** `view.Control(name)` takes fill, gradient, radius, outline
  and shadow from `Theme.Style(name)`; a control a theme does not list is plain, so a theme restyles
  any control without code. Every component takes its corners through `Ctx.Radius(name)` or
  `view.Control`, so the Corners setting (square, soft, round) reaches all of them; Flat turns
  every gradient into a flat fill.
- **After the CLI dashboard.** Surfaces, type, outlines and radii follow the dashboard: 4 px
  buttons and fields, 6 px tables, 8 px menus, flat fills, no zebra stripes, the response status a
  `pill`, and one gradient, `hero`, behind the window title. What the dashboard has and the window
  does not is under Limits.
- **Tabs.** A request tab row is `tabsH` (42), a pane tab row `core.TabBarH`, each with
  `core.TabGap` above. The strip draws one bottom line under transparent inactive tabs; the open tab
  has the pane color and a square 2 px `AccentLine` outline on top, left and right (`EdgeT`,
  `EdgeL`, `EdgeR`), open at the bottom so it runs into the pane. No tab name is bold.
- **Custom themes.** Every `*.json` in the themes folder (`-themes DIR`, default
  `settingsfile.ThemesDir()`) is a theme: the file name is its id, `name` its label, `extends`
  `dark` (default) or `light`. It may set any token, `method.*` and `json.*` included, and controls.
  It is listed beside the built-ins; color overrides are kept per theme id. A file that does not
  parse or extends an unknown theme is skipped and reported in the status bar; no folder is fine.
- **High contrast** (`theming.Catalog.Theme`, `highBars`) moves text, borders and state fills in
  their own hue until each reaches its ratio on every surface it is drawn on; surfaces stay. The
  ratios sit above WCAG's on purpose: both palettes already met those, so the switch would change
  nothing visible.
- **Interface scale.** `settings.UIScalePct` multiplies the display factor and the platform base
  (`settings.BaseScale`: 1.1 on Linux, 1 elsewhere), so 100% is the platform's normal size;
  screenshots skip it. `frameSize` turns the window into the logical size the UI lays out in. The
  Appearance slider resizes live and saves on release; it follows the pointer in device pixels since
  the press, so the resizing UI never feeds back into the value. Remembered window sizes are in
  window pixels.
- **Display scale on Linux.** Ebitengine reports 1 for an Xwayland window on a scaled monitor, so
  then `App.displayScale` asks the `settings/port.DisplayScale` port (adapter
  `settings/adapter/displayscale`). `Factor` runs on the frame loop and never runs a command: it
  returns the cached scale and refreshes it in the background, so a hung compositor command cannot
  stall a frame. The UI then lays out in window pixels over the real scale, with no supersampling;
  Windows and macOS use Ebitengine's factor.
