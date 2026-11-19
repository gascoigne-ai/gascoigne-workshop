status: todo
kind: chore
commit: chore: delete code with no caller
scope: internal/*,uitoolkit/*,cmd/*
depends_on: 102-limits

# Delete code nothing calls

Origin: the repo audit of 2026-11-19 (three read-only reviews against the harness rules). The user approved all phases and, with that, the deletions and the gate change named here. Behavior stays the same unless the task says otherwise; tests stay green and none is loosened. Update `architecture.md` in the same commit where it describes what changes.

The user approved these deletions (audit of 2026-11-19). Confirm each with a compiler check (`go vet`, and a grep over `internal`, `uitoolkit`, `cmd`, including tests and the `release` tag) before deleting; keep anything that has a caller.
- `sendComposed` (`internal/workspace/ui/app/wsui.go`), the empty `const ()` in `overlay.go`.
- `withOpts` (`uitoolkit/markup/component.go`); the dead assignment at `uitoolkit/widget/textfield.go` ~585.
- `ws.CloseProtocol`, `CloseUnsupported`, `CloseTooBig`; `calltrace.titleH`; `settingsui.DialogTitleH` (the unused one); `SettingsDialog.RawStyle`; `shortcutpage.Page.PendingCount`; the always-true `wroteSection` flag in `settings/sections.go`.
- The duplicate `opts.GRPCSchema` assignment in `cmd/workshop/main.go`.
- Unused parameters: `El.color` ctx, `settle` ctx, `split.handle` and `firstSize` element, `dropOptional` flow, `calltrace.titleBar` and `toolbar` host.
- Redundant overrides: `fragment.draw`, `colHandlesB.Measure`; the unreachable error in `sourceKindOf`.
Acceptance: `make check` green; each deleted name is gone from the tree.
