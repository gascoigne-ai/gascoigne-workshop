# Feature: settings

The settings the user can change: the schema, the settings file, the theme built from it, the dialog and its pages, and the system information it shows.

## Layout
- `internal/feature/settings`: the model (`Settings`, `Schema`, the codec, `Minimal`, `Validate`, `Complete`) in `model.go` and its neighbours.
- `prefs`: the messages a page sends; `theme`: how settings become a theme.
- `form`: a controller, the state of the settings form and what a user's action on it means: a `Model` (the refusal note of each setting), typed messages (`Typed`, `Stepped`, `Reset`, `Choose`, `Picked`) and a pure `Update(model, settings, msg)` that returns the next model and the effects the view carries out (`Apply`, `Restore`, `Fill`, `ForgetTexts`, `ApplyFonts`, `AskFile`). It also holds the pure facts of the form: where a row puts its control (`PlaceOf`), the font and size pairs (`Faces`), the search match (`Matches`, `Lists`). It imports the settings root only, so it runs without a window. The typed text stays a `TextField` in the view (toolkit state); the values stay in `Settings`.- `adapter/settingsfile`: `settings.json` on disk.
- `ui/settingsui`: the dialog shell (title bar, list of pages, search field, raw JSON view, Save) behind its `Host`. `ui/formpage`: the form view and the font drop-down, on top of `form`. `ui/colorpage`, `ui/shortcutpage`, `ui/systempage`: the three table pages. No `ui` package imports another, so the App draws the form through `Host.FormPage` and the tables through `Host.ExtraPage`, and composes the font list over the dialog (`menuView`).
- Installed fonts come from the toolkit's `desktop/fonts` (`Family`, `Find`, `Matching`, the scan); this package keeps only the family names (`UIFont`, `CodeFont`).
- Imports: the kernel (`kernel/jsonview`), `storage` and the base feature leaf `feature/request/severity` plus the toolkit for the views and `theme`.

## Known gaps
- `ui/formpage` is about 900 lines because the font drop-down (`fontlist.go`, `fonts.go`) sits in it: the form row draws the font control from the drop-down's state, and a `ui` package cannot import another. It splits when the toolkit has a drop-down widget for a long list of rows.

```mermaid
flowchart LR
  schema["settings.Schema rows (settings)"] --> formpage["formpage: rows and controls"]
  formpage -- "form.Update" --> form["form: Model, messages, effects"]
  formpage -- "prefs messages" --> app["App.Send (settingshost.go)"]
  app --> theme["theme.ThemeWith, Ctx.SetTheme"]
  app --> file["settingsfile: settings.json"]
  file -- "Poll every PollTicks frames" --> shell["settingsui: dialog, raw JSON, search"]
```

- **One row per setting.** `settings.Schema` (`schema.go`) is the form, the raw
  view's validation and completion, the search, reset and the order of keys in the file. A row is
  built from the accessor of its `Settings` field, so its kind and its value type cannot disagree.
  A new setting is one row, plus an entry of `settingControls` when the plain control does not fit.
- **Rows and controls.** A row is the title block (title, reset icon while changed, description) and
  one control. `form.PlaceOf` puts the control beside the block or under it from the setting's
  `Kind` (a bool, a stepped number and an enum of at most three values go right) with a table
  (`form.PlaceOf`, `customPlace`) for the custom controls; a bool is a `widget.Switch`. A font and its size are one row (`form.Faces`, `ui/formpage/face.go`): the title block is the font's (its description ends with the size's range, read from the size row), the control line holds the font drop-down and the stepper with its own Reset; `formpage.Page.View` skips the size row, and a search lists the line once whichever of the two settings it names. The reset icon, the
  category heading (search results only), the page title row and the search field in the page list
  follow `style.md` (Settings row). The Colors, Shortcuts and System pages stay tables and get only the
  title row.
- **Dialog and host** (`ui/settingsui`, `ui/formpage`): `settingsui.SettingsDialog` is `App.prefs`, `formpage.Page` is `App.form`, and
  the `App` is the `Host` of both. Host methods only read; everything a page asks for is a
  `prefs` message applied by `App.Send`. `SetFonts` stays a method because its
  error is the answer. The Colors, Shortcuts and System pages are separate `ui` packages that the
  App draws through `Host.ExtraPage`.
- **Closing forgets.** Closing the dialog drops page, search, raw view and typed text
  (`SettingsDialog.Fresh` and `formpage.Page.Fresh`); only its size (resized by any edge, `widget.Resizable`) stays, in
  `state.json`. A file chooser it opened is
  taken back when it closes (`App.pickFile` returns the cancel).
- **settings.json** holds only what differs from the defaults (`settings.Minimal`), grouped in
  sections (`sections.go`) after its `version`; the Go struct stays flat. The older flat form is
  read too and regrouped on the next write. The written bytes are pinned by
  `testdata/encode-*.golden`.
- **Font sizes** are three number settings; the interface and code sizes sit on the line of their font, Small text size is a row of its own: `fontSize` (interface,
  10 to 24 px, default 14), `smallFontSize` (captions, hints, table headers; 8 to 24, default 12)
  and `codeFontSize` (bodies and code views; 8 to 32, default 13). Each is an `Int` field shown in
  the form as a minus button, a field and a plus button (`formpage.Page.stepControl`; the rule is `form.Update`): a step
  starts from the setting, does nothing at an end of the range and refreshes the field; a typed
  number outside the range shows a note and keeps the last good value. `Clamp` brings a loaded
  value into the range its schema row states (0 is the face's default). The sizes are independent
  (`theming.FontSizes.Resolved` builds the theme's three sizes); control heights follow the
  interface size through `Ctx.Sizes`.
  The file versions 0 and 1 wrote the size as `textSize`: small, medium or large; their reader
  (`upgradeTextSize`) maps those to 13, 14 and 16 (anything else to 14, `fontSize` wins when both
  are present), and `Validate` accepts the old key only below version 2 (at 2: "replaced by
  fontSize"). Versions 0 to 2 had one size for the three faces: `deriveFaceSizes` sets whichever
  of `smallFontSize` and `codeFontSize` the file does not state from its `fontSize` (small =
  size - 2, never under 9; code = size - 1; `fontSize` clamped to 10 to 24 first), so an upgrade
  changes nothing on screen. A version 3 file is read as it is: missing keys are the defaults.
- **Raw view**: text with problems (`settings.Validate`) changes nothing and lists them; text
  without is applied and written as typed. `settings.Complete` offers keys and values at the caret.
- **Reload**: the dialog polls the file's time and size and applies an outside change; a file with
  problems is reported and ignored. The program's own writes are stamped and not reloaded, and
  nothing is reloaded while changes are unsaved.
- **Auto-save** (`autoSave`, on by default): a change applies at once either way; with auto-save off
  it is only marked unsaved until Save. Turning it off writes that choice once; turning it on
  writes what was pending. Closing with unsaved changes asks Cancel, Discard or Save.
- **Theme**: how settings become a theme (`theme.ThemeWith`) is in `style.md`, Theming pipeline.
- **Fonts** (`uitoolkit/desktop/fonts`): the scan runs in a goroutine at start and its result is
  applied on the UI goroutine. A saved family that is not installed stays saved, shows as
  `(not installed)` and draws in the built-in Go font. The font list builds only the rows in view,
  so only their files are read.
- **Shortcut editor** (`ui/shortcutpage`): a filterable keymap table (Action, Keystrokes as
  key chips, Source) over the text editing keys. Clicking a row, or Enter on the selected row
  (`Page.Navigate`, which the app calls while this page is what the dialog shows; Up and Down move the
  selection through the rows the filter leaves), selects it and records the next combination for the
  action. The selection and the dragged column widths are view state, not saved. Source is derived:
  User only when the settings hold an override for the action that parses (the test `Effective` uses),
  else Default; Unbind, Reset, the conflict with Replace and the refusal or plain-key note are in the
  detail under the selected row. What it refuses (`Refuse`) derives from the text widgets' own keys
  (`textKeys`) and Escape. A combination another action has is not saved until the user picks Replace.
  Only overrides are stored. The binding model itself is in Keyboard, focus and overlays (app.md).
