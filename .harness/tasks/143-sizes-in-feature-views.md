status: todo
kind: refactor
commit: refactor(ui): feature views read sizes from Ctx.Sizes
scope: internal/*/ui/**, !internal/workspace/ui/**, .harness/style.md
not_before: 2026-01-01
depends_on: 141-sizes-in-toolkit

# Feature views use Ctx.Sizes

Goal: destpicker, calltrace, mcpform, sentlog, colorpage, settingsui (dialog, fontlist, form),
shortcutpage and systempage read sizes from `Ctx.Sizes` and lose their raw literals (about 40 template
attributes). Each `Host` hands over the scope, or the view takes the `*Ctx`.

## Model
Model: unchanged.

## Acceptance
- Identical at default settings: no golden or `view.Dump` change.
- Guard test: no raw size literal in these packages.
- `make check` passes.
