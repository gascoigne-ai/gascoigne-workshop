## Look

-  **Spacing rhythm**: one `Pad` (12 px) is the inset between panes and content; bars (sidebar
  header, tabs, toolbars, status) are separated with one token, `Border` (never `Rule`, which is for
  outlines and table headers). The request and response panes both start their first row `Pad` below
  the pane tabs, so rows line up across the divider
  (`TestRequestAndResponsePanesShareTheirFirstRowBand`).
-  **Click targets are outlined**: buttons (`Border`; primary buttons `AccentLine`), unfocused text
  fields and tables (all the same gentle `Border`) carry a 1 px outline drawn after the children so
  regions to click are visible at rest; a focused field swaps its outline to `AccentLine`.
-  **Spacing details**: table checkbox, delete and header cells center their icon; a caption row and
  its table are `Gap(6)` apart; the URL field has a `Pad-4` spacer after the method divider; the URL
  bar draws a divider only between its chips, never after the last one; the response toolbar's chips
  and icon buttons are one height (`core.CompactH`), centered.
-  **Theme files and control styles** (`uitoolkit/theming/themefile.go`,
  `uitoolkit/theming/themes/*.json`, `internal/settings/model/appearance/themes/*.json`): themes are
  embedded JSON in the style of editor themes: `variables` (referenced as `$name`), a flat `colors`
  table (`#rrggbb` or `#rrggbbaa`) and a `controls` table of per-control styles: `radius` (px, a
  large value makes a pill), `gradient` (a strength 0..0.5 that shades the fill toward white above
  and black below, or `[top, bottom]` colors) and `border` (a color token).
  `theming.ParseTheme(data, base)` merges a file over a base theme and rejects unknown fields,
  variables, tokens and bad gradients. The toolkit's palettes and control styles are
  `uitoolkit/theming/themes/{dark,light}.json` (`theming.Dark()/Light()` load them); the app lays
  its own colors (HTTP methods, JSON syntax) over them from
  `internal/settings/model/appearance/themes`. The look follows the CLI dashboard: 4 px buttons and
  fields, 6 px tables, 8 px menus, flat fills, and a burgundy hero banner with the one gradient
  behind the window title.
-  **Controls draw from their style**: `view.Control(name)` makes an element take its fill
  (gradient, rounded), outline and shadow from `Theme.Style(name)`: `button`, `primary`,
  `icon-button`, `field`, `table`, `menu`, `tooltip`, `tabstrip`, `statusbar`, `hero`.
  `Ctx.FillStyled` fills a (rounded) path with per-vertex gradient colors and `Ctx.OutlineStyled`
  strokes it; outlines are drawn after an element's children, and a field's focus ring over its
  outline. Two settings act on every style: `corners` (square / soft / round scale the radii 0 / 1 /
  1.5, `theming.CornerChoices()`) and `flat` (every gradient becomes a flat fill), both on the
  Appearance page. A control a theme does not list is plain, so a theme file can restyle or flatten
  any control without code.
-  **Surfaces and type** (after the CLI dashboard): the page (`Frame`: #140f12 / #f6f3f4) with
  surface panes and lists on it (`Bg` = `Sunken`: #1c1519 / white) and `Alt` for the faint tint;
  light inputs are lighter than their pane (the light `bg` is #f9f6f7 and inputs sit on `sunken`,
  white). Body text is 14 px (small 12, mono 13; large is 16) and the window title is bold. Tables
  have no zebra stripes: the header row sits on the `panel` tint with a `Border` rule under it and
  small, bold, muted, uppercase labels, and a `BorderSoft` hairline runs under every row but the
  last. Response status is a pill (`pill` control, radius 999) in the OK / warn / error color on its
  `*-soft` background, like the dashboard's badges. Tokens for these: `border-soft`, `ok-soft`,
  `warn-soft`, `error-soft`, `accent-soft`, `warn`, `track`, `faint`; `Fonts` has `Bold` and
  `SmallBold` (Go Bold) behind `view.Bold()`.
-  **Outlines and fills** (after the dashboard CSS): buttons and icon buttons are surface-filled
  with a `border-strong` outline and no gradient, as the dashboard's; fields, menus and tooltips use
  `border-strong`, tables `border-soft`; the tab strip and status bar are flat. Gradients stay a
  theme feature: only `hero` uses one by default, dark at the top like the dashboard's night
  gradient. Dark `panel`/`alt` are the dashboard's surface-2, `hover` its accent-tinted mix, and
  `info`/`info-soft` back `method.PUT`. What the dashboard has and the window does not is under
  Limits.
-  **Tabs** (`widget.Tabs`, `widget.ActiveTab`, the request strip `requesttabs.vml`): a request tab
  row is 42 px (`tabsH`) and a pane tab row 40 px (`core.TabBarH`), each with `core.TabGap` (6 px)
  of empty space above the tabs, so the open tab's outline never touches the banner or the bar above
  it. The strip draws one continuous bottom line behind its tabs; inactive tabs are transparent so
  it shows under them, and the open tab has the pane color and a 2 px accent outline (`AccentLine`)
  on its top, left and right (`view.EdgeT`, `EdgeL`, `EdgeR`) with the bottom open, so it runs into
  the pane below. In the pane strips the open tab's name is in the text color and the others are
  muted; no tab name is bold. The outline is square: the edge options take no radius.
-  **Size rules and rounding**: heights come in steps (`core.RowH` 28 for rows and standard
  controls, `CompactH` 24 for chips, icon buttons, inline fields and the slider, `LabelH` 22 and
  `NoteH` 18 for captions and notes) and icon buttons are `IconBtnW` 32 wide; gaps between
  neighbours come in two steps (`core.TightGap` 6 between controls side by side, `LooseGap` 8
  between the blocks of a pane); new components use these instead of literals, and a template reads
  them from its scope (`{{rowH}}`, `{{tightGap}}`, `{{looseGap}}`), since markup has no constants of
  its own. Corner radii come from the theme's control styles (4 px buttons and fields, 6 px tables,
  8 px menus), and every component takes its corners through `Ctx.Radius(name)` or `view.Control`,
  so the Corners setting reaches them all: besides the buttons, fields, tables and menus, also
  `checkbox`, `slider`, `scrollbar`, `row` (tree rows and menu items), `swatch` and `match`.
-  **Custom themes**: every `*.json` file in the themes folder (`-themes DIR`, default `themes` in
  the user directory, `<user config>/gascoigne-workshop` or `gascoigne-workshop-dev` by build
  (`userdir.Name`)) is a theme: the file name is its id, `name` its label, and `"extends": "dark"`
  (default) or `"light"` picks the built-in theme it is laid over (`theming.ThemeExtends`,
  `theming.ParseTheme`). It can set `variables`, `colors` (any token, including `method.*` and
  `json.*`) and `controls` (radius, gradient, border) and is listed in the Theme row of the settings
  next to the built-ins; the user's color overrides are kept per theme id. A file that does not
  parse, or extends an unknown theme, is skipped and reported in the status bar; a missing folder is
  fine. Example:
  `{"name": "Ocean", "extends": "light", "colors": {"accent": "#115577", "accent-line": "#115577"}, "controls": {"button": {"radius": 10}}}`.
-  **High contrast** (`theming.Catalog.Theme`, `highBars`): the Contrast setting `high` moves the
  customizable colors until their text reaches 7:1 and also the main text (17:1), muted text (12:1),
  error and ok colors (8:1), accent line (6:1), borders and rules (7:1), soft borders (3:1) and the
  fills that show state (selected row 2.2:1 and hovered row 1.6:1 against the list, chips 2:1
  against the pane), each in its own hue until it reaches the ratio on every surface it is drawn on;
  the surfaces themselves stay. The ratios sit above WCAG's minimums on purpose: both palettes
  already met those, so a switch that only enforced them changed nothing visible.
  `theming/highcontrast_test.go` holds the bars and checks that the change is visible in both
  themes.
-  **Interface scale**: `settings.UIScalePct` (default 100, bounded 50..300) multiplies the
  display's scale factor and the platform's base factor (`settings.BaseScale(goos)`: 1.1 on Linux, 1
  elsewhere, set by `main` as `Options.BaseScale`; `App.uiScale` is the product; screenshots skip
  it), so the slider reads 100% for the platform's normal size. `frameSize` turns the window size
  into the logical size the UI lays out in (window / scale), the total scale and the screen size;
  the Appearance page has a `widget.Slider` (50..200%, step 5, 100% at the middle of the track via
  `SliderMid`). The UI resizes live while the thumb moves (`onChange`), and the settings are saved
  on release (`OnSlideEnd`). A drag follows the pointer's movement in device pixels since the press,
  using `Ctx.PointerX/PointerY` (the pointer before any clipping container hides it), so the
  resizing track and the dialog's clip never feed back into the value. Remembered window sizes are
  stored in window pixels (logical size times the window scale) so they restore the same window.
  **Display scale on Linux**: Ebitengine reports 1 for an Xwayland window on a scaled monitor, so
  when it reports 1 `App.displayScale` asks the `settings/port.DisplayScale` port (adapter
  `settings/adapter/displayscale`; sources tried in order; `Factor` runs on the frame loop, so it
  never runs a command itself: it returns the cached scale, 0 until the first answer lands, and
  starts one background refresh when the answer is older than 2 s, so a changed desktop scale
  applies live and a hung compositor command cannot stall a frame): `hyprctl monitors -j`, `swaymsg -t get_outputs -r` (focused monitor's scale),
  then `GDK_SCALE` and `QT_SCALE_FACTOR`. `frameSize` then lays out in window pixels divided by the
  real scale and keeps the screen at window size (no supersampling); Windows and macOS use
  Ebitengine's own factor.
