status: todo
kind: refactor
commit: refactor(ui): feature ui packages join the screen tree
scope: internal/**, e2e/**, Makefile, .harness/**, internal/CLAUDE.md
not_before: 2026-01-01
depends_on: 027-theme-to-ui

# Every feature/*/ui package moves under internal/ui

Goal: moves: `collections/ui/destpicker` to `ui/overlays/destpicker`; `history/ui/sentlog` to
`ui/sidebar/sentlog`; `settings/ui/{settingsui,formpage,colorpage,shortcutpage,systempage}` to
`ui/overlays/settingsdialog{,/formpage,/colorpage,/shortcutpage,/systempage}` (`settingsui` becomes
`settingsdialog`); `send/ui/{respview,calltrace}` to `ui/response/{respview,calltrace}`; `send/ui/mcpform` to
`ui/editor/mcp/mcpform`; `workspace/ui/appicon` to `ui/appicon`. Delete the feature `ui` zones; the layer table
switches to the final rows of `.harness/reorg.md`. `FEATURE.md` files drop their UI parts and point to `ui/`.
Rewrite `internal/CLAUDE.md` (layout, imports, component anatomy, tests), `architecture.md`, `overview.md`,
`architecture/app.md` and the layer line of `rules.md`. Follow the move recipe.

## Model
Model: unchanged.

## Acceptance
- No `feature/*/ui` directory remains; the layer test enforces the new rows (`ui` to adapter is absent).
- Common acceptance of `.harness/reorg.md`.
