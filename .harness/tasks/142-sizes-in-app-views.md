status: todo
kind: refactor
commit: refactor(app): app views read sizes from Ctx.Sizes
scope: internal/workspace/ui/**, cmd/workshop/**, .harness/style.md
not_before: 2026-01-01
depends_on: 141-sizes-in-toolkit

# App views use Ctx.Sizes

Goal: the views under `internal/workspace/ui/app` (and `cmd/workshop` if any) take sizes from
`Ctx.Sizes` and `Sizes.Scope()`, and lose their raw size literals (about 45 template attributes in 17
files: urlbar, wsbar, responsepane, requesttabs, sidebar, env, mcpexplorer, bodypane, ...). Package-level
consts built from size constants (`browserFootH`) become functions. Templates built with
`MustBuild(markup.Scope{...})` without a `*Ctx` in reach get the scope from the caller.

## Model
Model: unchanged.

## Acceptance
- Identical at default settings: no golden or `view.Dump` change.
- Guard test (extend `sizeliterals_test.go`): no raw size literal in these files.
- `make check` passes.
