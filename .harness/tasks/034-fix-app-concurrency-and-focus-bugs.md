status: in progress

# Fix the concurrency, settings and focus bugs found in the UI analysis

## Goal
The bugs the analysis of `App` turned up are fixed, each with a regression test that failed first. They do not wait for the MVU work (tasks 035 to 041); the fixes are small and the later tasks keep them.

## Scope
- [done] Send, connect and the gRPC method listing read `a.reg` and `a.settings` from goroutines while `clearCookies` replaces `a.reg` (commit `fd8b4f0`).
- [done] The connect timeout does not stop the dial: `connect` (`wsui.go`) dials with `context.Background()` and checks the timeout only after the dial returns. Dial with `ctx`.
- [done] Network settings (proxy, certificates) never reach the protocol registry: `newRegistry` runs only at start and in `clearCookies`. Changing them in the settings dialog rebuilds the registry, on the UI goroutine.
- [done] `textFocused` (`treeops.go`) lists the focused fields by hand and misses `tests`, `gqlVars`, `file` and `msg`, so plain-key shortcuts and table undo can fire while those have the keyboard. One list, or the focus owner of task 039, covers every field.
- [open] `settingsui.Host.Change` and `pageHost.Edit` reach two different functions (`changeSettings`, `saveSettings`); the second skips the response-view refresh (`menu.go`). One path.

## Acceptance
- Each item has a regression test, with a comment above it saying what went wrong, that was seen to fail for the reported reason before the fix. The race test fails under `go test -race`.
- `make check` green.

## Notes
- Found by the analysis of 2026-11-10 (see task 035 for the pattern that prevents this class of bug).
