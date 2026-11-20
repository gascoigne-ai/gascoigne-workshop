## Files, state and platforms

-  **Remembered state** (`internal/workspace/adapter/statefile`,
  `internal/workspace/ui/app/statesync.go`): `state.json` next to the settings holds the window
  geometry, sidebar width, pane split, expanded folders (by path: `expanded` for the collection that
  is not saved, `expandedIn` by the folder of a saved one), open tabs (the folder of the request's
  collection, request path, selected request and response pane; an empty folder is the unsaved
  collection, else the first, so a file written when one collection was open restores into it), the
  active tab and table column weights. It is separate from settings, written atomically, and a
  missing, broken or partly stale file only means a fresh start (missing requests are dropped,
  indexes clamped; a broken file shows a status note). `App.stateTick` snapshots each frame and
  writes after the state was unchanged for one second; `main` writes once more on exit and restores
  the window clamped to the monitor (`state.Window.Clamp`). `-state PATH` picks the file, empty
  keeps it in memory, and screenshot runs do not touch it unless `-state` is given.
-  **File dialogs and file bodies**: `internal/workspace/adapter/filedialog` runs the native panel
  through osascript on macOS, a PowerShell Windows Forms dialog on Windows, else the first installed
  of zenity, kdialog, yad, qarma and matedialog (`Open`, `Save`, `ErrCancelled`, `Available`).
  Elsewhere than macOS and Windows the order comes from the desktop (`detect.go`): the table
  `desktops` maps names, matched without case and an `X-` prefix, against each entry of
  `XDG_CURRENT_DESKTOP` (a colon-separated list, `ubuntu:GNOME`), then `XDG_SESSION_DESKTOP`, then
  the base name of `DESKTOP_SESSION`, to an order: KDE Plasma kdialog, qarma, zenity, yad,
  matedialog; MATE matedialog first (MATE's zenity fork, used when installed), then zenity, yad,
  kdialog, qarma; LXQt qarma (a Qt zenity) first, then zenity, kdialog, yad, matedialog; GNOME,
  Cinnamon, Budgie, Pantheon, Unity, XFCE and LXDE zenity, yad, kdialog, qarma, matedialog; any
  other desktop, a window manager with no desktop variable (Sway, Hyprland, i3), WSL or an empty
  environment zenity, kdialog, yad, qarma, matedialog. `preferred` and `installed` are pure
  (environment and path lookup passed in) and tested per desktop without a display. Each program is
  a row (`tool`) with its open and save command lines and the exit codes that mean a cancel: zenity,
  matedialog and qarma take
  `--file-selection [--save --confirm-overwrite] --title=TITLE --filename=START` (`zenityLike`),
  kdialog `--title TITLE --getopenfilename START` or `--getsavefilename START`, yad
  `--file [--save --confirm-overwrite] --title=TITLE --filename=START` and exits 252 when closed
  with Escape or the window's button, 1 on Cancel like the others. They start in the user's home
  folder (`os.UserHomeDir`, as the in-app browser; task 068, user's decision 2026-11-15): START for
  open is the folder in the form each program reads as a folder (`HOME/` for zenity, matedialog and
  yad, `HOME/.` for qarma, `HOME` for kdialog), for save the suggested name in it (`HOME/NAME`,
  `savePath`), which each opens on that folder with the name filled in. With no home folder they
  start without a folder (no `--filename` on open, kdialog `.`, the bare name on save, which the
  programs read against the working directory). The macOS and Windows panels choose their own start
  folder. `App.pickFile(title, mode, name, done)` takes a `port.FileMode` (`OpenFile` or `SaveFile`, which `port.FileDialog.Choose` and the in-app browser's mode table both read) and the first usable row of `fileChoosers`: the desktop's dialog
  (`askDesktop`, in a goroutine, the answer through the bus: a dispatched `func()` runs on the UI
  goroutine in the next frame), else the in-app file browser (decision 18). The body type `File`
  (`model.BodyFile`, `reqTab.file`) sends a file as it is; its editor says "found" or "not found"
  under the path, from `Folders.IsFile` asked once per path in the background (`reqTab.fileCheck`, a
  `bodyFileCheck` of the path asked about and its state: unknown while the answer is on its way,
  found or missing; an answer for an older path is dropped), so a file made after the path was typed
  shows as found once the path is typed again; a multipart form value `@path` sends a file part and
  `@@x` the literal `@x`.
-  **In-app file browser** (`internal/workspace/ui/app/filebrowser.go`, rules in
  `workspace/controller/filebrowser`, listing through `port.Folders` and `adapter/folders`, task
  068, decision 18): the file chooser when the desktop has no dialog program (Linux without zenity,
  kdialog, yad, qarma or matedialog), for Open and Save. A dialog (style.md File browser) with the
  caller's title and the count of items shown as subtitle, a toolbar with a parent-folder button,
  the path field and a Hidden files check, the list (folders first with a folder icon, then files,
  names ignoring case; names starting with a dot hidden unless checked or typed), and a footer with
  the problem of the last Enter, Cancel and Open or Save. While there is no problem the footer's
  left side is a muted hint that names the program to install for native dialogs
  (`port.FileDialog.Suggested`, the `suggest` of the desktop's row in `filedialog/detect.go`:
  kdialog on Plasma, zenity on the GTK desktops, MATE and LXQt, "zenity or kdialog" elsewhere; none
  on macOS and Windows), so the user learns the browser is a fallback (objective 2 of task 068). It
  starts in the user's home folder every time (`Folders.Home`, else the root) and does not remember
  the last folder (task 068, user's decision 2026-11-15). The path field holds the folder and a
  name: while opening the name filters the list (a part of a name, or a pattern such as `*.json`)
  and selects the exact name, else the first match; a path to another folder (absolute, relative or
  `~/`) is listed on Enter and then taken. While saving the name is the save name, kept across
  folders; picking a file takes its name; a name that exists asks `Replace file` first, as zenity's
  `--confirm-overwrite`. Keys: Up and Down move the selection, Enter takes the selected row or the
  typed path, Backspace with nothing after the folder and Alt+Up go to the parent, Escape cancels; a
  click selects, a double click (`view.OnDoubleClick`) opens. Only the rows in view are built. The
  `Update` of the controller asks for listings through its `Effect`, which the App reads on the UI
  goroutine (`followBrowser`). The screenshot flag `-browse DIR` opens it on a folder.
-  **File questions taken back** (`App.pickFile` returns a cancel function): a file chooser the
  settings dialog opened (`prefs.PickFile`) is closed when the dialog closes: the native dialog's
  process is killed through its context and its late answer is ignored, or the in-app file browser
  is closed (only the one it opened). A question asked by anything else (the request body's file) is
  left alone. `App.chooseFile` and `App.hasFileDialog` are the test seams for the native dialog.
-  **Platforms**: Linux is the tested target. `core.Input.Ctrl` means the platform's primary
  modifier (`core.PrimaryDown`: Control, plus Command on macOS), so every shortcut, text-editing key
  and menu works with Cmd there; settings and key strings keep the name `Ctrl`, and only displayed
  key text goes through `core.PlatformKeyText` (menu rows, the Shortcuts page, tooltips, toasts).
  Both read the OS from `Ctx.HostOS` (`runtime.GOOS` from `core.NewCtx`; `Sampler.HostOS` for input;
  `app.Options.HostOS` sets it for the app and `shortcutpage.Host.HostOS` hands it to the page);
  there is no package variable, so the `app` and `shortcutpage` tests pin it to `linux` per context
  or `Options`, and run in parallel with other values. The reconnect wait is
  `app.Options.RetryDelay` (two seconds when zero), kept in the `App`. The CI `build` job compiles
  and vets for Windows and macOS. Nothing has been run on them yet.
-  **IME and dead keys**: text widgets take `Input.Typed` (what Ebitengine's `AppendInputChars`
  returns), which is the text after the window system's dead-key and input-method composition, so
  accents, CJK and emoji arrive as whole runes (`uitoolkit/widget/composed_test.go`). Ebitengine
  does not report the composition in progress (preedit text and candidate window position), so none
  is drawn; the system's candidate window may sit away from the caret. This is a limit of the
  toolkit, not something the app can fix.
-  **Accessibility**: colors are held to contrast ratios by the palette tests
  (`theming/palette_test.go`); Tab moves the keyboard focus through the text widgets and code views
  in reading order (sidebar filter, URL field, response view) and cycles; every icon-only control
  has a tooltip (`focusorder_test.go` fails on one without). Buttons are reached by pointer or
  shortcut, not by Tab. There is no screen-reader support: Ebitengine has no accessibility tree to
  publish, so labels and hints exist for sighted users only.
-  **Icon and packaging**: `internal/workspace/ui/appicon.Image(size)` rasterizes `appicon.svg`
  (burgundy rounded square with a gradient, a light spanner on the diagonal) with
  `widget.RasterizeSVG`, which reads the icon subset plus `#rrggbb` fills and `linearGradient`s,
  fills only (non-zero rule, a hole is a subpath wound the other way), in software with
  `golang.org/x/image/vector`, so it works before any window exists; `main` sets it as the window
  icon, `workspace/adapter/dockicon` as the macOS Dock icon (the window icon does not reach the
  Dock; cgo and a small `.m` file), and `-icon-png FILE` writes it for packaging. `make install`
  installs the binary and, on Linux only, `packaging/gascoigne-workshop.desktop` and the icon (it
  uses `mkdir -p` and `install -m`, so BSD `install` on macOS runs it too);
  `.github/workflows/check.yml` runs `make check` under xvfb.
-  **Release packaging** (`.github/workflows/release.yml`, `packaging/`): a `v*` tag runs three
  build jobs, all with `-tags release`, Linux (tarball through `make install`), macOS (`go build`,
  `packaging/macos/make-app.sh` makes the `.app` with the drawn icon as `icon.icns`, zipped with
  `ditto`) and Windows (a `-H=windowsgui` build plus `icon.png`, zipped), and a final job publishes
  the three archives as one GitHub release. `packaging_test.go` checks that the property list is
  well formed and matches the script; the jobs themselves only run in CI. The Windows `.exe` carries
  the icon as a resource: the job writes `icon.png` with `-icon-png`, `go-winres simply` turns it
  into a `.syso` in `cmd/workshop`, and the `-H=windowsgui` build links it in.
-  **Screenshots with live settings** (`-set key=value`, `screenshot.set`): `make shot` can change
  settings after the window is up, through the same path the settings dialog uses (`App.Change`), so
  a capture shows what a live change draws, not a start-up one. Repeatable, e.g.
  `go run ./cmd/workshop -screenshot out.png -send -set flat=true -set contrast=high`. Used to check
  that Flat colors and High contrast change the picture (a flat header, about 5% of the pixels
  different under high contrast).
