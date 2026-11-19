status: todo
kind: chore
commit: chore: fix stale and orphaned doc comments
scope: internal/*,uitoolkit/*,cmd/*
depends_on: 100-docs-sync

# Fix stale and orphaned doc comments

Origin: the repo audit of 2026-11-19 (three read-only reviews against the harness rules). The user approved all phases and, with that, the deletions and the gate change named here. Behavior stays the same unless the task says otherwise; tests stay green and none is loosened. Update `architecture.md` in the same commit where it describes what changes.

Comments only, no code change. Fix:
- `internal/workspace/adapter/statefile/statefile.go` first line says "Package state"; `internal/collection/model/openapi/openapi.go` names `internal/miniyaml`; `internal/settings/ui/colorpage/colorpage.go` documents `colorEdited` above `edited`; `internal/request/adapter/proto/curl.go` has the `shellQuote` comment above `quoteUnlessToken`; `internal/workspace/ui/app/app.go` `NewApp` doc says "around the sample collection".
- Comments that still name old one-letter parameters (`app.go`, `panes.go`, `ctxmenus.go`, `wsui.go`, `statesync.go`, `shot.go`, `view.go`).
- Orphaned or wrong doc comments: `uitoolkit/view/el.go` (BorderB text on `BorderL`), `uitoolkit/core/ctx.go` ("Col resolves" on `Color`, "three faces" lists five, `Ctx.Menu`), `uitoolkit/widget/hscroll.go` (KeepInView text on `overflow`; names `Reveal`), `scroll.go`, `table.go`, `svg.go`, `scrollbar.go`, `codeview.go`, `checkbox.go`, `uitoolkit/markup/component.go` (`finiteFloat` between a comment and `realParam`).
- Missing doc comments: `HasFocus`, `SetFocus`, `CanRedo`, `Linear.Measure`, `Linear.Layout`, `MaxW`, `MinH`, `MaxH`, `MarginT/R/B`, `ChevronDownIcon`.
- Package doc index gaps: `FitW` in `view`; `RasterizeSVG`, `NewTextField`, `NewTextArea` in `widget/doc.go`; `doc.go` claims every widget is a plain composition, which is not true.
Acceptance: `make check` green, `git diff` shows comment lines only.
