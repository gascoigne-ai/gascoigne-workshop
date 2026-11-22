status: todo
kind: feature
commit: test(layout): make check fails on a layer violation
scope: Makefile, e2e/**, internal/CLAUDE.md, uitoolkit/CLAUDE.md
not_before: 2026-01-01
depends_on: 001-collection-root-no-sibling-imports, 002-settings-workspace-cycle, 003-exampleserver-ws-import

# Layer check in make check

Goal: the import rules in `internal/CLAUDE.md` and `uitoolkit/CLAUDE.md` are only prose.
Enforce them from `make check`.

OPEN QUESTION (ask the user before building): commit `7d856abc` deliberately deleted
source-scanning tests, including the import-layer test. How should the check run?
Options: (1) a Go test over `go list -deps` output with a rule table, recommended, because it
reads the compiler's imports and not source text; (2) a small Go program under `e2e/` called
from the Makefile; (3) leave the rules as prose. Record the answer here.

## Model
Model: unchanged (a rule table: zone pattern to allowed imports, including the written exceptions).

## Acceptance
- Test first: a table-driven test with one failing import per rule (ui to adapter, ui to ui,
  domain to another domain past its root, toolkit to internal, root to sibling package).
- The check passes on the current tree after tasks 001 to 003.
- The repo check command passes; `overview.md` no longer says no check exists.
