status: todo
kind: chore
commit: build: the release-tag test run covers only the packages that differ
scope: Makefile,.github/*,.harness/*,CLAUDE.md,README.md
depends_on: 107-small-dedup-helpers

# Limit the release-tag test run

Origin: the repo audit of 2026-11-19 (three read-only reviews against the harness rules). The user approved all phases and, with that, the deletions and the gate change named here. Behavior stays the same unless the task says otherwise; tests stay green and none is loosened. Update `architecture.md` in the same commit where it describes what changes.

`make check` runs the whole race suite twice (default tags and `release`). Only `cmd/workshop`, `internal/collection/model/collection` and `internal/settings/model/userdir` have non-test files behind the `release` tag. The user approved narrowing the second run to those packages and the packages that import them (compute the list with `go list -deps` / `go list -f` so it cannot drift); `go vet -tags release ./...` stays on everything.
- Prove it: a test or make target fails when a new file with a `release` build tag appears outside the computed set.
- Update `Makefile`, the CI workflow, `overview.md`, `README.md`, `CLAUDE.md`.
Acceptance: `make check` green and faster; say the before and after times.
