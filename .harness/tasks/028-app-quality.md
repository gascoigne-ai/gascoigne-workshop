status: todo

# App quality

## Goal
Make the app dependable to ship.

## Scope
- Window: minimum size, app icon, title with a dirty marker for unsaved edits.
- Release packaging for Windows and macOS (see below); the screenshot compare in CI is dropped by decision (no golden screenshots).
- Accessibility pass: contrast is tested; add focus order review and screen reader hints where the platform allows.
- Windows and macOS: installers and packaging (a `.app` bundle with the icon, a Windows icon resource and zip), and a run on real machines (window, clipboard, file dialogs, browser preview, IME).

## Acceptance
- A checklist of failure paths each with a test; screenshots compared in CI; `make check` passes.

## Progress
- Done: the window title shows the collection and a dot while something is unsaved (`App.Title`); a minimum window size of 720x480.
- Done later: a drawn program icon (`internal/appicon`, set as the window icon, `-icon-png FILE` writes a 256 px PNG), `packaging/gascoigne-workshop.desktop` and `make install PREFIX=...` (binary, desktop entry, icon); a GitHub Actions workflow running `make check` under xvfb.
- Reviewed: ignored errors in `internal/ui` are the clipboard (it reports through `statusClipboard`), cookie jar creation (cannot fail without options) and closes on shutdown; no silent failure path was found.
- Left: IME and dead keys (depends on Ebitengine's text input), an accessibility pass.
- Done later: the settings (hamburger) icon turned the dark text color on hover over the banner; `vml.HoverColor` / `hover-color` now sets it to the banner's text token, which the palette tests hold to the banner's contrast.
- Done later: selection on folded lines follows what is shown (the block's preview): columns, words, triple click and copy use the preview text, lines hidden inside a fold are left out of a selection across it, and select-all still copies the whole text.
- Done later: the move cursor shows while a table row, a tab or a tree row is dragged (dividers and column handles already had resize cursors).
- Already so: settings (`settings.json`), the collection state (`state.json`: tabs, expanded folders, sizes), themes, the default workspace and the response and sent histories all live under `os.UserConfigDir()/gascoigne-workshop` (`settings.DefaultPath`, `state.DefaultPath`); the path is per platform through the standard library. Windows and macOS runs are the open part (see the platform item).
- Done later: right-click menu rows have icons: `menuItem.icon`, else the table `menuIcons` by the first word of the label (`Copy`, `Cut`, `Paste`, `Delete`, `Rename`, `Close`, `Find`, `New`, `Save`, `Open`, `Undo`...); rows without one keep an empty icon column so labels line up. Seven new icons (pencil, scissors, clipboard, save, external, undo, redo). Not looked at in a live window.
- Done later: call trace: a `Log` button in the status bar opens a dialog with the last 500 entries (request sent, answer with status, time and size, failures, connections opened, lost or refused), a filter and Clear; in memory only. Messages of an open socket stay in the Messages pane.
- Done later: Windows and macOS groundwork: the code builds for both (CI job `build` on windows and macos runners runs `go build` and `go vet`; a Windows cross build and a macOS vet pass locally); the shortcut modifier is Command on macOS (`core.PrimaryDown`) and every shown key text says `Cmd` there (`core.PlatformKeyText`; stored bindings keep `Ctrl`); paths use `os.UserConfigDir` and `filepath`, the browser preview has its per-OS opener. Not run on either system.
- Done later: font families: Appearance has `Interface font` and `Code font` fields; typing lists matching installed families as buttons, Enter or Apply picks one (`settings.uiFont`, `codeFont`). Installed fonts are found in the background (`internal/fonts`: platform font folders, .ttf and .otf, grouped by the family name inside the file, regular and bold). Left: the same fields as rows of the settings schema when task 030 lands, a monospace-only filter for the code font, `.ttc` collections, italic styles and per-role sizes.
- Done later: failure paths: `failurepaths_test.go` is the checklist (network, clipboard unreadable, not cURL / OpenAPI / Postman, collection and settings and state and body saves, no response to save or open), each row asserting an error toast or the status line. Wording rule: `<what> failed: <cause>` or `Not a <thing>: <cause>`. The review found one silent failure: a history write error went to the status line and was replaced at once by the send result; history and sent-log errors are now error toasts, and an identical error toast is shown once.
- Done later: IME and dead keys (decision: test and document the limit): `composed_test.go` pins that composed text (accents, CJK, emoji, decomposed accents) arrives as whole characters in fields and editors and that Backspace removes one. Ebitengine delivers only the finished text, so an in-progress IME composition cannot be shown; this is recorded in architecture.md.
