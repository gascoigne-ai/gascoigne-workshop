status: todo
kind: refactor
commit: refactor: parameters are named by role, not by type
scope: internal/*,uitoolkit/*,cmd/*
depends_on: 104-delete-test-only-api

# Name parameters by role

Origin: the repo audit of 2026-11-19 (three read-only reviews against the harness rules). The user approved all phases and, with that, the deletions and the gate change named here. Behavior stays the same unless the task says otherwise; tests stay green and none is loosened. Update `architecture.md` in the same commit where it describes what changes.

Naming rule in `common/rules.md`. Pure renames.
- Parameters named after a builtin, type or package: `Text(string string)`, `Ctx` text calls, `mix(value, uint8 uint8)`, `btoi(bool bool)`, `SelectLines(value, int int)`, `applyNumber(..., float64 float64)`, `wsEvent wsEvent`, `sendResult sendResult`, `dialog *dialog`, `reqTab *reqTab` (several files), `state *state.State`, `flag *flag.Flag`, `string string` in `view.go` and `clipboard_test.go`.
- Meaningless names: every `count int` pixel option in `view` becomes `pixels`; `slider value` becomes `fraction`; `svg.go other` becomes `attrs`; the others listed by `grep` for `\b(int|string|bool|float64|uint8) (int|string|bool|float64|uint8)\b` in parameter lists.
- Remaining one-letter parameters: `view/el.go`, `custom.go`, `runtime.go`, `tabs.go`, `textfield.go`, `svgraster.go`, `markup/component.go`, `app/panes.go`, `shortcuts.go`, `filebrowser.go`, `statesync.go`, `shot.go`, `state.go` (`mw, mh`), `mcp.go` `setVersion`, `rest.go`, `postman.go` `exportable`, `settingsui/form.go`, `port/state.go` `Save`, and the test files named in the audit (`transfer_test.go`, `fileaccess_test.go`, `newerfiles_test.go`, `window_view_test.go`).
Acceptance: no test changes except renamed identifiers; `make check` green.
