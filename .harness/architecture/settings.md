## Settings

-  **Settings dialog** (`internal/settings/ui/settingsui`; `settingsui.Dialog` is `App.prefs` and
  the App is its `settingsui.Host`: the host's methods only read, and everything a page asks for is
  a `prefs` message (`internal/settings/controller/prefs`: `Change`, `Edit`, `ScaleLive`, `Save`,
  `Replace`, `Persist`, `FileMarked{Saved}`, `Toast`, `OpenMenu`, `PickFile`, `Close`, ...) sent with `Host.Send` and
  applied by `App.Send` in `settingshost.go`; `Change` from the form also rebuilds the Colors page,
  `Edit` from the Colors and Shortcuts pages does not (both carry a `prefs.EditFunc`); `SetFonts` stays a method because its error
  is the answer): a 760x560 modal dialog over a dimmed window that the user can resize by dragging
  any edge or corner (`widget.Resizable`: the size changes by twice the pointer's movement so the
  edge follows it; 520x360 at least, the window less a 40 px margin at most; a double click on an
  edge resets it) and whose size is remembered in `state.json` (`settingsW`, `settingsH`). Closing
  it always forgets what was done in it (`Dialog.Fresh`: page, search, raw view, typed text, open
  lists), so it opens fresh; only the size stays. The dialog is the template `dialog.vml`, which Go
  sizes to the window and hands the title bar (`titlebar.vml`, with Save from `save-pending.vml` or
  `save-idle.vml`, or none while auto-save is on and nothing is pending), the built list items and
  the page, through slots; the font list floats beside it. It has: a title bar with a search field
  and the close button, a list of pages on the left (`pages`: Appearance, Colors, Shortcuts,
  Requests, General, Experimental, System; `Dialog.Cat`; each page is `navitem-open.vml` or
  `navitem-closed.vml`, picked in Go; Colors, Shortcuts and System are drawn by the App through
  `Host.ExtraPage`) and the page on the right. Appearance, Requests, General and Experimental are
  generated from the settings schema (`settings.Schema`): each setting shows its title, a wrapped
  description and its control (chips for an enum, Off/On for a boolean, a field for text and
  numbers; `settingControls` holds the few special ones: theme, scale slider, font pickers), a dot
  and a `Reset` button while it differs from the default (reset removes the key from the file). The
  form is markup (`form.go`): each setting is `settingrow.vml` over a `settingView`, with the
  control built by its kind's template and slotted in (`chips.vml`, a `<segment-chip>` per value, the
  current one lit; `scale.vml`; `textcontrol.vml`; `file-empty.vml` or `file-chosen.vml`;
  `openpage.vml`; `fontcontrol.vml`), and the page is `formpage.vml` over the built rows, since a
  search puts `formcaption.vml` captions between them (`formempty.vml` when nothing matches). The
  search field filters the settings of every category by title, description, key and allowed values
  and lists them grouped by category; the Colors and Shortcuts settings are tables and show an
  `Open` button. A number outside its range is explained under its field and not applied. The text
  fields and refusal notes of the form, the Colors page and the Shortcuts page are one
  `widget.NotedFields[K]` each (a field is created once per key, a note of `""` is no note); the
  form sends its edits through `setKey`, which toasts a refusal of `settings.Set`.
  `settings.json` holds only what differs from the defaults (`settings.Minimal`), grouped by section
  (below). Esc, the close button or a click outside close it. New settings are one row of
  `settings.Schema` (and one entry of `settingControls` if the plain control does not fit).
-  **Raw settings and reload** (`internal/settings/ui/settingsui/raw.go`,
  `internal/settings/model/settings/complete.go`): the `JSON` button in the settings title bar swaps
  the dialog for `settings.json` as text in a `TextArea` (`Form` swaps back); that page is the
  template `raw.vml`, which Go hands the area's state, the status line and its color, and the
  problems as lines. Every change runs `settings.Validate`: text with problems changes nothing and
  lists them under the editor (syntax errors with line and column, unknown keys, wrong types, values
  out of range), the text is colored as JSON (`jsonview.Tokenize`, the response colors) and keys
  with a problem are drawn in the error color, both through the area's `Styler`; text without
  problems is applied (`Host.Replace`, the App's `applySettings`: theme, fonts, response view,
  fields) and written to the file as typed. Typing the opening quote of a key or a value opens, by
  itself, a menu of what `settings.Complete` offers there (`Dialog.RawChanged`; a paste or a closing
  quote does not, and nothing offered means nothing shown), and Ctrl+Space opens it on demand; the
  menu is what `settings.Complete` offers at the caret (unused keys, enum and boolean values,
  shortcut action ids inside `shortcuts`). `Dialog.Poll` checks the file's time and size every 30
  frames and applies an outside change to the form and the open raw view; a file with problems is
  reported in a toast and ignored. The program's own writes are remembered (`Dialog.Stamp`) and not
  reloaded.
-  **settings.json sections** (`internal/settings/model/settings/sections.go`): the file starts with
  its `version` (Persistence) and is grouped into `appearance`, `colors`, `shortcuts`, `requests`,
  `general` and `experimental`, always in that order and with the keys of a section in schema order,
  for example
  `{"version": 1, "appearance": {"theme": "light"}, "requests": {"timeoutSec": 30}, "general": {"autoSave": false}}`.
  `appearance`, `requests`, `general` and `experimental` are objects of settings (`experimental`
  holds the opt-in `grpc` and `sse` switches, off by default) (`SectionOf` says where a setting
  goes; the hidden `workspace` and `environment` are in `general`); `colors` and `shortcuts` are the
  tables themselves. Reading accepts the older flat form too, alone or mixed with sections, and the
  next write regroups it. `Validate` checks a section is an object and each key is in its own
  section (`belongs in requests`), with the offsets of the key inside the whole text; `Complete`
  offers the sections at the top, the keys of the section around the caret, and values for a key of
  that section. The Go `Settings` struct stays flat; only the text is sectioned.
-  **Auto-save and the Save button** (`settings.AutoSave`, `App.settingsUnsaved`,
  `settingshost.go`): the setting `autoSave` (page General, on by default) writes every change at
  once. With it off a change still applies at once but is only marked unsaved; the dialog's title
  bar has a `Save` button (lit while something is unsaved; absent while auto-save is on and nothing
  is pending) that writes the settings in effect, or the raw text as typed when that is what was
  held back. Turning the option off writes that choice once, turning it on writes whatever was
  pending. Closing the dialog with unsaved changes asks (`Cancel`, `Discard` which reverts to the
  file, `Save`), and a change made to the file outside the program is not loaded while changes are
  unsaved.
-  **File settings** (`settings.File`, `Dialog.fileControl`): the certificate settings (`caCert`,
  `clientCert`, `clientKey`) are of kind `File`: with no file one button (a folder icon and
  `Choose file`), with a file a text field showing the path with a folder button inside it at the
  right (`Dialog.fileControl`), which opens the file chooser through `Host.PickFile` (the native
  dialog or, without one, the in-app file browser) and puts the chosen path in the setting. In JSON
  they are text, and `Expectation` says `the path of a file`.
-  **Font families** (`internal/settings/adapter/fonts`, `core.Ctx.SetFonts`,
  `internal/settings/ui/settingsui/fonts.go`, `fontlist.go`): `settings.UIFont` and `CodeFont` name
  installed families. `fonts.Scan` walks the platform font folders (`SystemDirs`) for .ttf/.otf
  files and groups them by the family name inside each file; a file without a glyph for every letter
  A-Z and a-z (a symbol or icon font) is left out (regular and bold; a family with no bold file uses
  its regular one); at start the scan runs in a goroutine and `Dialog.FontsScanned` applies the
  saved names on the UI goroutine. `Ctx.SetFonts(ui, code FontData)` rebuilds the faces
  (`Ctx.Chosen` is kept across theme and scale changes); a failing font changes nothing. A saved
  name that is not installed uses the built-in Go font, stays saved and shows as `(not installed)`.
  The setting's control is a dropdown (`Dialog.fontControl`): a field-like row showing the current
  family drawn in its own font (`fontcontrol.vml`, from the same `fontChoice` the list's rows use);
  clicking opens `FontList` below the pointer, a modal list with a search field and `Built-in` plus
  every installed family, each name drawn in that family's font (`view.FontSource`). The list is the
  `fontlist.vml` template with the visible rows slotted in, each built from `fontrow.vml`, a
  `<list-row>` selected for the family in use, from a `fontChoice` view model. Only the rows in view
  are built, so only their font files are read (once, cached by file); the list opens scrolled so
  the family in use is centered in view (`selectedRow`), a pick applies and saves at once and closes
  the list, Escape or a click outside closes it.
-  **Shortcut editor** (`internal/settings/ui/shortcutpage`; its page is the host's
  `ExtraPage("shortcuts")`): the Shortcuts page has a search field (matches the action name or its
  key text) over a table of actions; it is the template `shortcutpage.vml`, fed one `actionRow` view
  model per matching action and the text editing keys. A key button records the next combination
  (`startRecording`, `recordKey`); Backspace while recording, or the unlink icon button, leaves the
  action unbound; a replay icon button appears for overrides (a row at its default key keeps only
  the room). A combination a text widget answers (Ctrl+C...) or Escape is refused with a note and
  recording goes on. A combination another action has is not saved: the row says `Used by X (key)`
  with a `Replace` button, which unbinds X and gives the key to this action (`replaceBinding`);
  recording again drops the pending choice. A key that types text (a plain letter) is saved with a
  muted warning. Only overrides are stored, in `shortcuts` of settings.json. The text editing keys
  are one unexported table (`textKeys`: action and its combinations); what `Refuse` keeps from other
  actions is derived from it, and a row shows its combinations joined with `or`.
