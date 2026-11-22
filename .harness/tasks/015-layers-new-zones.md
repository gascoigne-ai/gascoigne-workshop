status: todo
kind: chore
commit: chore(layers): new zones and rows join the layer table
scope: e2e/**, internal/CLAUDE.md, .harness/**
not_before: 2026-01-01
depends_on:

# Layer table learns the target zones

Goal: the reorganization in `.harness/reorg.md` needs the layer test to accept the new layout before anything
moves. Add the zones `base`, `adapter`, `uilogic` and `ui` and the rows of the layer table section of
`reorg.md`, beside the old ones so both layouts pass. Add the `Below` flag (target path has the importer path as
prefix) for a `ui` parent that imports its children. No package moves in this task.

## Model
Model: unchanged. Test tables only.

## Acceptance
- Test first: one table test per new zone and row, failing before the code.
- `TestEveryRuleMatchesSomething` and `TestRealTreeObeysLayers` stay green.
- The repo check command passes.
