status: todo
kind: refactor
commit: refactor(ui): statusbar, sizes and notices become components
scope: internal/**, cmd/**, e2e/**, Makefile, .harness/**
not_before: 2026-01-01
depends_on: 028-existing-ui-to-tree

# ui/sizes, ui/statusbar and the notice store

Goal: the smallest real component, to prove the component recipe of `.harness/reorg.md`. Three parts:
`ui/sizes` (from `window_sizes.go`: the size scope every component needs), `ui/statusbar`
(`window_statusbar.go`; `responseStatus` stays in `app` until 037) and `feature/workspace/notice` (status line
and toasts; `App.SetStatus` and `Toast` delegate to it while other code still calls them). The designer reads
the real files first and reports fields that another area also touches.

## Model
New store `notice.Notices`: owner of the status line (text, severity) and the toast list (text, severity,
expiry); invariants: a toast expires by tick count, the status line holds one entry. Access pattern: read
every frame, written by handlers and by the bus; a slice of toasts pruned in order (small, ordered by age), no
map. Rejected: keeping them as `App` fields (the problem this task removes). Severity comes from the place 021
chose.

## Acceptance
- Test first: `notice` unit tests (set, prune on tick, expiry order); statusbar dump test against a fake `Deps`.
- `App.statusBar()` calls the component; the app cap is lowered.
- Common acceptance of `.harness/reorg.md`.
