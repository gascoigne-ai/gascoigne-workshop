status: todo

# Remember window and UI state

## Goal
Only the theme and the text size persist. Everything else resets on every start.

## Scope
- A state file next to settings (`state.json` in the user config dir, written atomically, never fatal when missing or broken): window size and position, sidebar width and split ratios (task 011), expanded folders, open tabs by request path, the active tab, each tab's selected request and response pane.
- Restore it at startup: window geometry through Ebitengine, tabs whose requests no longer exist are dropped, an empty result opens the first request.
- Save on a debounce after changes and on exit; unsaved edits are not part of the state (task 014 owns saving requests).
- Settings and state stay separate files: settings are preferences users edit, state is what the app remembers.
- A `-state PATH` flag (empty keeps it in memory) so tests and screenshots do not touch the real file.

## Acceptance
- Round-trip tests in `internal/settings` style with a temp dir; restore with missing requests, a corrupt file, a window off-screen (clamp to the monitor), and unknown fields ignored.
- `make check` passes.
