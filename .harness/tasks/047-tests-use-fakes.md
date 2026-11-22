status: todo
kind: refactor
commit: refactor(tests): view tests use fakes, test-only layer rows go
scope: internal/**, e2e/**, .harness/architecture.md, .harness/overview.md, internal/CLAUDE.md
not_before: 2026-01-01
depends_on: 046-docs-final

# Tests use fakes, test-only layer rows go

Reorg note: this task runs last, after the packages sit in their final place (`internal/feature`,
`internal/ui`, `internal/storage`). The paths below are the old ones and `platform/fonts` is gone
(fonts live in `uitoolkit/desktop`): the designer first re-reads the real test imports and the
layer table, and lists the rows that exist then. Fixtures go into the test, not into
`feature/request`. Components already take fakes through `Deps`.

Goal: the layer test (`e2e/layers_rules_test.go`) carries six test-only rows, one per test file
that imports something its zone may not: `exampleserver` tests import `feature/collections` and
`feature/send/proto`; the `sentlog` tests import `history/adapter`; the `formpage` tests import
`platform/fonts` and `settings/adapter/settingsfile`; the `settingsui` tests import
`settings/adapter/settingsfile`. Replace each real adapter with a small fake of the port the view
uses (fakes over mocks, `common/code-style.md`) and for the `exampleserver` tests move the fixtures
they need into the test itself or a kernel type; then delete the six rows and the known-gap lines
about them.

## Model
Model: unchanged. Tests only; a fake implements the existing port, no new production type.

## Acceptance
- Each changed test keeps what it asserts; a test whose subject was the real adapter (a round trip
  through the file) moves to the adapter's own package unchanged.
- The six test-only rows and their known-gap lines are gone and `TestRealTreeObeysLayers` is green
  without them (`TestEveryRuleMatchesSomething` still passes).
- The repo check command passes.
