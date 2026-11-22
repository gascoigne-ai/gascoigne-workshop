status: todo
kind: refactor
commit: refactor(app): the Driver moves to app/drive
scope: internal/app/**, e2e/**, .harness/architecture/app.md
not_before: 2026-01-01
depends_on: 017-apptest-seam

# Driver leaves the app package

Goal: `driver.go` (338 lines) pokes unexported state only because it lives in `app`. Move it to
`internal/app/drive` over the exported seam of 017; `e2e/cmd/shot` and `apptest` both use it. If an action needs
a private field, add one small exported method on `App` and say so in the report.

## Model
Model: unchanged.

## Acceptance
- `driver_test.go` moves with it, assertions unchanged.
- `e2e/cmd/shot` imports `app/drive`; `TestGoldenScenes` and the soft-render shot test are unchanged.
- Common acceptance of `.harness/reorg.md`.
