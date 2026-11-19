status: todo
kind: ci
commit: ci: workflows reuse make targets instead of repeating commands
scope: .github/*,Makefile,.harness/release.md,packaging/*
depends_on: 108-check-release-scope

# Remove repeated steps from the CI workflows

Origin: the repo audit of 2026-11-19 (three read-only reviews against the harness rules). The user approved all phases and, with that, the deletions and the gate change named here. Behavior stays the same unless the task says otherwise; tests stay green and none is loosened. Update `architecture.md` in the same commit where it describes what changes.

- `release.yml` Windows job builds `bin/workshop.exe` twice and writes the icon twice: build once.
- The apt package list is copied in `check.yml`, `release.yml` and `release.md`: one source (a script or composite action) and the doc links to it.
- `check.yml` build job retypes the two vet commands: call `make vet vet-release`.
- macOS release uses raw `go build` while Linux uses `make install`: use the same target if it works for both, otherwise say why in a comment.
Acceptance: workflow files lint (`actionlint` if present, else read twice); `make check` green. Nothing is pushed.
