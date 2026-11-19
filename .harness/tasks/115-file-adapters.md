status: todo
kind: refactor
commit: refactor(adapters): one atomic write and one config path lookup
scope: internal/*,.harness/*
depends_on: 114-test-helpers

# Share the file helpers of the adapters

Origin: the repo audit of 2026-11-19 (three read-only reviews against the harness rules). The user approved all phases and, with that, the deletions and the gate change named here. Behavior stays the same unless the task says otherwise; tests stay green and none is loosened. Update `architecture.md` in the same commit where it describes what changes.

- Atomic write (temp file then rename) copied in `store.go`, `history.go`, `sent.go`, `statefile.go`, `settingsfile.go`; four use a fixed `.tmp` name, one `CreateTemp`. The layer rule forbids a shared adapter across domains: put the helper where every domain may import it (check `internal/layers_test.go`; if none fits, ask the user and record the answer in `decisions.md`).
- `os.UserConfigDir()` joined with `userdir.Name` in three adapters: `userdir.Path(name)` in the model.
- Store adapters export each operation twice (package function and a method that forwards): keep the methods, drop the function forms; update the tests.
- `proto/aliases.go` re-exports 12 port types only for old names: delete and update callers.
- `trimSent` re-reads the whole log after every append: trim from the size already known.
Acceptance: `make check` green.
