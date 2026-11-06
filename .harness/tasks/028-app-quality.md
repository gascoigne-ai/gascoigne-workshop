status: todo

# App quality

## Goal
Make the app dependable to ship.

## Scope
- Error states for every failure a user can see (network, parse, disk, clipboard): one wording rule, a toast or an in-pane message, never a silent failure; a review pass over `internal/ui`.
- Window: minimum size, app icon, title with a dirty marker for unsaved edits.
- Text input: IME and dead keys.
- CI workflow (`make check` plus the screenshot compare), release packaging and a desktop entry.
- Accessibility pass: contrast is tested; add focus order review and screen reader hints where the platform allows.
- Font family setting for the interface, the code views and other text roles.
- Support for Windows and macOS (paths, shortcuts with Cmd, packaging, CI builds).

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
