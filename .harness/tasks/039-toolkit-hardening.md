status: todo

# Harden the toolkit: focus, events, drag, purity, rules and docs

## Goal
The toolkit follows its own rules and has one answer for focus and event handling. Independent of tasks 035 to 038.

## Scope
- [done] Focus owner in `core.Ctx` (`FocusOwner`, kept by `view.Update`): one owner, hidden elements lose focus, `focusURL` stops clearing by hand. The `Focused` bool stays as the way to ask for focus and to read it (about 120 sites); it is no longer the source of truth. The settings dialog has no `Ctx`, so a `ctx.Focus()` call there would need a message; asking through the field kept one mechanism.
- [open] One event rule: say which events propagate (left click reaches every ancestor, right click the innermost, wheel the first user) in the `view` doc and test it; `OnClick` carries the pointer position so the app stops reading `a.ctx.In.X/Y` (`env.go`, `grpcui.go`, `panes.go`, `graphql.go`, `wsbar.go`).
- [open] One drag helper replaces the six hand-written drag machines (`split`, `resizable`, `components` columns, `scroll`, `slider`, `scrollbar`); `scrollB.press` and `barDrag.press` merge.
- [open] Layout is pure: `hscroll.go`, `scroll.go` and `layout.go` stop writing state during layout; the double build per frame stops repeating the writes.
- [open] Rule breaks: `FloatTopRight` (rule 1), `Edges(t, top, left, right bool)` (rule 4), template names that do not mirror Go (`tip`, `justify`, `onclick`, `hr`, `vr`, `pad="12 0"`) and options with no attribute or comment (rule 9), missing doc index lines (rule 10).
- [open] No switch on widget kind in focus code (`runtime.go`, `layout.go`); `Checkbox` gets `OnChange`; `Tabs` and `ScrollTabs` share `pick`.
- [open] Doc drift: package `widget` (does not exist) in `architecture.md`, `decisions.md`, `rules.md`; "no template parser" line in `architecture.md`; names that no longer match (`Center`, `Float`, `Reveal`, `Separator(show, inset)`); mislabeled doc comments (`BorderB`, `FloatTopRight`, `TextCenter`, `KeepInView`, `Color`, `Ctx`).
- [open] Small: `Sampler.Read` uses the clock port, not `time.Now`; `El.hover` unused; `Icon` lookup and `_ = ic`; `viewCache` grows without bound.

## Acceptance
- Each item has a test that failed first; toolkit API rule exceptions go into `decisions.md`.
- `make check` and `make shot` green and looked at; `layers_test.go` still passes.

## Notes
- Task 034 `textFocused` item may be closed by the focus owner here.
