status: todo
kind: feature
commit: test(layout): make check fails on a layer violation
scope: Makefile, e2e/**, internal/CLAUDE.md, uitoolkit/CLAUDE.md
not_before: 2026-01-01
depends_on: 008-app-features-table

# Layer check in make check

Goal: the rules in `internal/CLAUDE.md` are prose. After layout C they are few and checkable:
kernel imports no other `internal` package; a feature imports kernel and platform, never another
feature; platform imports kernel only; inside a feature `ui` imports no `adapter` and no other
`ui`; the toolkit imports nothing of `internal`; `app` imports anything; `exampleserver` may import
`feature/send/proto/ws`.

OPEN QUESTION (ask the user before building): commit `7d856abc` deliberately deleted
source-scanning tests, including the import-layer test. How should the check run? Options:
(1) a Go test over `go list -deps` output with a rule table, recommended, because it reads the
compiler's imports and not source text; (2) a small Go program under `e2e/` called from the
Makefile; (3) leave the rules as prose. Record the answer here.

## Model
Model: unchanged (a rule table: zone pattern to allowed zones, the one written exception).

## Acceptance
- Test first: a table-driven test with one failing import per rule.
- The check passes on the tree after task 008.
- The repo check command passes; `overview.md` no longer says no check exists.
