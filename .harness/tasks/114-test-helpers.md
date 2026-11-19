status: todo
kind: chore
commit: test: one shared harness for the app tests
scope: internal/workspace/ui/app/*_test.go,internal/*_test.go
depends_on: 113-app-lookups-and-importers

# Share the duplicated test helpers

Origin: the repo audit of 2026-11-19 (three read-only reviews against the harness rules). The user approved all phases and, with that, the deletions and the gate change named here. Behavior stays the same unless the task says otherwise; tests stay green and none is loosened. Update `architecture.md` in the same commit where it describes what changes.

About 300 lines of helpers are copied across the app tests. Move to `harness_test.go` and delete the copies; tests keep their assertions.
- Pollers: `until`, `stepUntil`, `statusSoon`, `waitForResponse`. Dump pinners: `pinAssertDump`, `assertDumpPinned`. Tree walkers by type name: `fieldStates`, `areaStates`, `checkStates`, `findWidget`, `findWidgets` become one `findWidgets(root, kind)`. Toast helpers: `toastWith`, `toastSays`, `toastTexts`, `errorToasts`. Menu lookup: `itemByLabel`, `menuItem`. Key input: `key`, `altKey`, `combo`, `pressCtrl` become `combo`. File reads: `fileText`, `readFile`. Rect parsing: `textCenter`, `clickDumped`.
- About 25 app builders (`newTestApp`, `newSettingsTestApp`, ...) become one `testApp(t, options...)` where the difference is only an option.
- Four session fakes (`recordingSession`, `slowSession`, `requesterSession`, `closeTrackingSession`) become one configurable fake.
- Fix the stale comment "request tabs 40" (`tabsH = 42`).
Acceptance: no assertion changes; `make check` green.
