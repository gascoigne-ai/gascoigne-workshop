status: todo

# Harden the toolkit: focus, events, drag, purity, rules and docs

## Goal
The toolkit follows its own rules and has one answer for focus and event handling. Independent of tasks 035 to 038.

## Scope
- [done] Focus owner in `core.Ctx` (`FocusOwner`, kept by `view.Update`): one owner, hidden elements lose focus, `focusURL` stops clearing by hand. The `Focused` bool stays as the way to ask for focus and to read it (about 120 sites); it is no longer the source of truth. The settings dialog has no `Ctx`, so a `ctx.Focus()` call there would need a message; asking through the field kept one mechanism.
- [done] One event rule: left click reaches every ancestor, right click the innermost, wheel the first user; documented in `architecture.md` and tested. The position comes with the new `OnClickAt` (changing `OnClick` itself would touch every call site and the templates); `env.go`, `panes.go` and `graphql.go` use it. `wsbar.go` and `grpcui.go` still read `a.ctx.In` because their click goes through a template or a Cmd.
- [done] One drag helper (`grab` in `drag.go`: begin, held) replaces the hand-written press/hold/release code of `split`, `components` columns, `scroll` and `scrollbar`, and `scrollB.press` and `barDrag.press` merge into `pressThumb`. `slider` (a ratio of a device-pixel track, with an end callback) and `resizable` (two axes from a start rectangle) keep their own state: they do not follow the grab-offset shape, and forcing them in would add options to the helper.
- [done] Layout is pure: `scroll.go` and `hscroll.go` place children for the state they find (clamped locally) and `Update` settles it (scroll clamp, the KeepInView reveal, `Overflowing`). `layout.go` and `split.go` only wrote their own derived rects, which is what layout is for. A state change now shows after the next `Update`, not after `Layout` alone; the draw pass lays out again, so nothing is a frame late on screen.
- [open] Rule breaks: `FloatTopRight` (rule 1), `Edges(t, top, left, right bool)` (rule 4), template names that do not mirror Go (`tip`, `justify`, `onclick`, `hr`, `vr`, `pad="12 0"`) and options with no attribute or comment (rule 9), missing doc index lines (rule 10).
- [open] No switch on widget kind in focus code (`runtime.go`, `layout.go`); `Checkbox` gets `OnChange`; `Tabs` and `ScrollTabs` share `pick`.
- [open] Doc drift: package `widget` (does not exist) in `architecture.md`, `decisions.md`, `rules.md`; "no template parser" line in `architecture.md`; names that no longer match (`Center`, `Float`, `Reveal`, `Separator(show, inset)`); mislabeled doc comments (`BorderB`, `FloatTopRight`, `TextCenter`, `KeepInView`, `Color`, `Ctx`).
- [open] Small: `Sampler.Read` uses the clock port, not `time.Now`; `El.hover` unused; `Icon` lookup and `_ = ic`; `viewCache` grows without bound.

## Acceptance
- Each item has a test that failed first; toolkit API rule exceptions go into `decisions.md`.
- `make check` and `make shot` green and looked at; `layers_test.go` still passes.

## Notes
- Task 034 `textFocused` item may be closed by the focus owner here.
