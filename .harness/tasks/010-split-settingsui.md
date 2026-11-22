status: todo
kind: refactor
commit: refactor(settings): split settingsui into form and dialog packages
scope: internal/**, cmd/**, e2e/**, .harness/architecture.md, .harness/architecture/**, .harness/overview.md, uitoolkit/CLAUDE.md, Makefile
not_before: 2026-01-01
depends_on:

# Split settingsui

Goal: `internal/feature/settings/ui/settingsui` is 1626 non-test lines in one package. Task 004
moved the feature and left this split open: `form.go` (537) is not generic, every form function is
a method on `*SettingsDialog` reading its state (`fields`, `scale`, `list`), the 13 tests are
`package settingsui`, and the rule that no `ui` package imports another `ui` package forbids the
shell importing the form. A cut needs a form-state type owned outside `ui` (for example in the
feature root or a `form` controller package with a typed `Model` and pure `Update`), the form view
on top of it, and the dialog shell on top of both. Tests are redistributed by the behavior they
state, none loosened.

Two test files import another feature's package (`theme/theme_test.go` and
`ui/settingsui/rawcolor_test.go` import `respview`); give them a fake or a kernel type so the
feature has no import of another feature, tests included.

User's answer: the form state lives in a new controller package `internal/feature/settings/form`
with a `Model`, typed messages and a pure `Update`; the form view and the dialog shell sit on top of it.

## Model
Per the open question: the form-state type, its owner, the transitions; one source of truth for
field values.

## Acceptance
- Test first for the new seam: a test of the form `Model` that states the behavior the dialog needs.
- Existing tests stay green and none changes in what it asserts; each resulting package is under
  about 800 lines and no `ui` package imports another `ui` package.
- The repo check command passes.
