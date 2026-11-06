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
- Hamburger (settings) button has the wrong color on hover: derive the hover color from the contrast rules instead of a fixed token.
- Font family setting for the interface, the code views and other text roles.
- Save the settings in the user config directory (`os.UserConfigDir`), per platform.
- Save the user's collection state (open tabs, expanded folders, sizes) in the user config directory.
- Selection in the compact response view does not work correctly on folded items.
- Logging view: a place where the user can trace the logs of calls (requests sent, answers, errors, timings).
- Icons on the right-click menu rows.
- Support for Windows and macOS (paths, shortcuts with Cmd, packaging, CI builds).
- Change the mouse cursor while grabbing (drags of rows, tabs, dividers, column handles).

## Acceptance
- A checklist of failure paths each with a test; screenshots compared in CI; `make check` passes.

## Progress
- Done: the window title shows the collection and a dot while something is unsaved (`App.Title`); a minimum window size of 720x480.
- Done later: a drawn program icon (`internal/appicon`, set as the window icon, `-icon-png FILE` writes a 256 px PNG), `packaging/gascoigne-workshop.desktop` and `make install PREFIX=...` (binary, desktop entry, icon); a GitHub Actions workflow running `make check` under xvfb.
- Reviewed: ignored errors in `internal/ui` are the clipboard (it reports through `statusClipboard`), cookie jar creation (cannot fail without options) and closes on shutdown; no silent failure path was found.
- Left: IME and dead keys (depends on Ebitengine's text input), an accessibility pass.
