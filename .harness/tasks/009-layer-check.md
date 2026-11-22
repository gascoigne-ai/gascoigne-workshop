status: todo
kind: feature
commit: test(layout): make check fails on a layer violation
scope: Makefile, e2e/**, internal/CLAUDE.md, uitoolkit/CLAUDE.md, .harness/architecture.md, .harness/overview.md
not_before: 2026-01-01
depends_on:

# Layer check in make check

Goal: the rules in `internal/CLAUDE.md` are prose. After the layout move (kernel, features, platform,
app) they are few and checkable, so `make check` enforces them:
- `internal/kernel/**` imports no other `internal` package;
- a feature imports kernel and its own packages only, never another feature (tests included);
- a platform adapter imports kernel and the feature that owns its port (the written exception);
- inside a feature `ui` imports no `adapter` and no other `ui`; a controller imports no toolkit;
- the toolkit (`uitoolkit/**`) imports nothing of `internal`;
- `app` imports anything; `cmd` and `e2e` are not checked;
- written exceptions: `internal/exampleserver` may import `feature/send/proto/ws`;
  `feature/settings/theme` imports `uitoolkit/core` and `uitoolkit/theming`.

User's answer: a Go test over `go list -deps -test` output with a rule table (it reads the
compiler's imports, not source text), run by `make check`. The rule table lives in the test, one
row per rule, exceptions as extra rows with the reason in a field. Not `common/scripts/check-layers.sh`.

User's answer on the real-tree result: the checker found 6 violations, all in `_test.go` files
(`exampleserver` tests import `feature/collections` and `feature/send/proto`; the `sentlog`,
`formpage` and `settingsui` tests import their own feature's adapter; the `formpage` tests import
`platform/fonts`). Each gets one explicit test-only rule row with a reason ("test fixture: real
adapter"), valid for test imports only; the rule stays default-deny for new code. A follow-up task
replaces those tests' real adapters with fakes and deletes the rows.

## Model
Model: unchanged. The rule table is data: zone pattern, allowed imports, reason; no logic per rule.

## Acceptance
- Test first: a table-driven test with one failing import per rule.
- The check passes on the current tree; every written exception is a table row with its reason.
- The repo check command passes; `overview.md` no longer says no check exists.
