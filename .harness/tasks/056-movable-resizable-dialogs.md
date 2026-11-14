# 056 Dialogs move like windows and resize without vanishing

Ebitengine has one OS window per process (`cmd/workshop/main.go`: one `ebiten.RunGame`), so a dialog cannot be an OS child window. It becomes an in-app floating window: it stays in the window Stack, but the user drags it by its title bar and resizes it from any edge. Decided with the user: all dialogs (message, prompt, settings, MCP form, call log), in-app approach.

Bug: when the user resizes a dialog it disappears. Suspected cause: `edges()` in `uitoolkit/widget/resizable.go` accepts a press up to 2 px outside the rect, while a click outside a modal element's rect calls `onDismiss` (`uitoolkit/view/runtime.go`), so a press on the outer pixels closes the dialog. Verify with a test before fixing; if the cause is another, record it here.

Scope
1. Regression test first, then fix: a press on the resize edge (including the 2 px outside the rect) starts a resize and does not dismiss; the dialog stays open through a whole drag.
2. `FloatFree` placement with a `PlaceState` (X, Y offset from the centre, 0 is centred) in `view` (`overlay.go`, `floatRect` in `layout.go`): the rect is clamped so the title bar stays inside the stack. Layout is pure and unit-tested.
3. Drag by the title bar in `widget` (`DialogTitle` takes the place state; cursor, press-move-release, a double click puts the dialog back to the centre). Resize keeps the opposite edge fixed instead of resizing about the centre.
4. `Dialog`, `ResizableDialog`, `MessageDialog` take the place and size state; the app owns it (`internal/workspace/ui/app/overlay.go`, `settingsui/dialog.go`). A window resize clamps the place again.
5. Template attributes for new options (toolkit rule 9), doc comments, package doc index lines and template table rows (rule 10).
6. Docs: `architecture.md` Overlays section and a decision in `decisions.md` (in-app window, why not a second OS window or process).

Acceptance: every dialog drags by its title bar and resizes from any edge or corner without closing; it stays inside the window after a window resize; `make check` green; `make shot` PNGs of the settings and a message dialog moved off centre looked at.
