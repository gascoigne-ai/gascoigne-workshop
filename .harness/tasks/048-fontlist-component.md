status: todo
kind: feature
commit: feat(widget): drop-down widget for the settings font list
scope: uitoolkit/**, internal/**, cmd/**, e2e/**, .harness/architecture.md, .harness/architecture/**, .harness/overview.md, .harness/style.md, Makefile
not_before: 2026-01-01
depends_on: 046-docs-final

# Font drop-down (was: drop-down widget)

Reorg note: this task runs after the reorganization (tasks 015-046). The reason for a toolkit
widget, "`ui` packages must not import each other", is gone: in the new tree a parent may import its
child. Recommended answer to the open question below: the font drop-down becomes the child component
`internal/ui/overlays/settingsdialog/formpage/fontlist` (own `logic/` and view, `Deps` for the font
store), no toolkit change, which matches the rule that the reorganization leaves `uitoolkit/`
alone. `formpage` then drops under about 800 lines. The loop asks the user before building:
child component (recommended) or the toolkit widget below. If the user picks the widget, the
scope keeps `uitoolkit/**` and the paths below are read through the new tree.

Original goal: `internal/feature/settings/ui/formpage` is 912 non-test lines because the font drop-down
(`fontlist.go`, `fonts.go`, about 304 lines) sits in it: the form row draws the font control from
the drop-down's state, and `ui` packages must not import each other. A drop-down widget in
`uitoolkit/widget` (a composition of `view` primitives, per `.harness/rules.md`) lets `formpage` use
it and drop under about 800 lines. The widget API follows `uitoolkit/CLAUDE.md` (the Toolkit API
rules) and the look follows `.harness/style.md`; it is a new component, so check first whether an
existing widget (menu, select, popover) already does the job and extend it instead.

OPEN QUESTION (ask the user before building): the widget's API and look (single choice only, a
filter field inside, grouped entries), and whether another view could use it. Recommend: single
choice with an optional filter field, as the font list works today.

## Model
The widget's state (open, filter text, highlighted index, selected value): one owner (the widget
value), a typed change message to the caller; no shared flags.

## Acceptance
- Test first: layout and hit tests without a GPU (rects in, rects out), then the interaction.
- Golden scene or screenshot of the open drop-down, in light and dark.
- `formpage` is under about 800 lines and its tests keep their assertions.
- The repo check command passes.
