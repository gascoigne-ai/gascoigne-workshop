status: todo

# One message bus and a command runtime in place of three channels

## Goal
Everything that arrives from a goroutine reaches the UI through one bus as a typed message, and effects (network, files, timers) run as commands off the UI goroutine. First step of the MVU pattern (decision to record in step 1).

## Scope
- [done] Record the decision in `decisions.md`: per-domain Model, Msg, Update, Cmd; `Update` is pure and may type-switch on `Msg` (one switch per `Update`); views dispatch messages instead of mutating `App`; tests drive `Update`, not pixels. Describe the target in `architecture.md`.
- [open] `internal/workspace/controller/bus`: `Dispatch(msg)` safe from any goroutine and never blocking the caller, drained once per frame; any message marks the frame dirty (replaces `drained` / `pruned` flags by hand). Unit tests: ordering, dispatch from the UI goroutine with a full queue, dirty mark.
- [open] Cmd runtime: runs a `Cmd` with a `context.Context` on a goroutine and dispatches its result message. It owns the protocol registry and a settings snapshot, so no goroutine reads `App` fields.
- [open] Move the `results` channel (`dispatch`, `apply`) onto the bus.
- [open] Move `wsEvents` (`connect`, `applyWS`, retries via `time.AfterFunc`) onto the bus.
- [open] Move the `later` channel and its users (file dialog, gRPC method list, MCP sampling, `settingsui.Host.Later`) onto the bus; delete the three channels.

## Acceptance
- No behavior change and no assertion changed in existing tests (names and imports only).
- No goroutine started by `App` touches an `App` field; `go test -race` green.
- `make check` green; `architecture.md` describes the bus.

## Notes
- Do task 034 first. Plan: `~/.claude/plans/analyze-internal-codebase-with-keen-newt.md`.
