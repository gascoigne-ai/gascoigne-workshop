status: todo
kind: chore
commit: test(app): split app_flow_test.go by what it tests
scope: internal/workspace/ui/app/*_test.go
depends_on: 121-app-substructs

# Split the big flow test file and the screenshot tests

Origin: the repo audit of 2026-11-19 (three read-only reviews against the harness rules). The user approved all phases and, with that, the deletions and the gate change named here. Behavior stays the same unless the task says otherwise; tests stay green and none is loosened. Update `architecture.md` in the same commit where it describes what changes.

- `app_flow_test.go` (3853 lines, 186 tests) into `tree_flow_test.go`, `table_flow_test.go`, `tabs_flow_test.go`, `send_flow_test.go`, `auth_flow_test.go`, `menu_dialog_toast_test.go`, `colors_shortcuts_test.go`, `find_view_test.go` additions, `regressions_test.go`; helpers are in `harness_test.go` (earlier task).
- `shot_test.go`: `runScene(t, opts, steps...)` plus two tables (scenes that succeed, scenes that fail); `TestTreeArrowKeysDoNothingWithoutRows` moves to the tree tests.
- The 47 literal `click(a, N, ...)` calls use `clickText` / `textCenter`; tests named after implementation (`TestEveryTableChangeGoesThroughOnePlace`, `TestTextFocusedCoversEveryEditorOfTheTab`, `TestCommittingACellOfARowThatNoLongerExistsAddsARow`) are renamed to the behavior they check; assertions are not loosened.
Acceptance: the test count is the same or higher; `make check` green.
