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
- Save the settings in the user config directory (`os.UserConfigDir`), per platform.
- Save the user's collection state (open tabs, expanded folders, sizes) in the user config directory.
- Logging view: a place where the user can trace the logs of calls (requests sent, answers, errors, timings).
- Icons on the right-click menu rows.
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
