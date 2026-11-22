status: todo
kind: feature
commit: feat(settingsui): settings rows follow a Zed-like layout
scope: internal/settings/**, uitoolkit/widget/**, cmd/workshop/**, internal/workspace/**, .harness/style.md, .harness/architecture/settings.md
not_before: 2026-01-01

# Settings rows in a Zed-like layout

Goal: the settings dialog presents its settings the way Zed's settings page does (reference image
/tmp/claude-1000/-home-lukaszb-Development-gascoigne-gascoigne-workshop-main/9915e3f2-18df-4415-b424-d658430aedc4/images/1.png; read it). Take over the pattern in our look and tokens, not its colors or fonts:
- A row: the title (stronger weight) on the left with a small reset icon next to it when the value
  differs from the default; the description under it in the muted face; the control on the right of
  the same row (drop-down, toggle, number stepper), vertically centered on the title block.
- A category is a small muted heading with a thin rule under it; rows separated by a light rule, with
  generous vertical air.
- Booleans are toggle switches, not text chips. Enumerations with few values stay chips.
- Left navigation with a search box on top; the page title row shows where the settings live and an
  "Edit in settings.json" button (our existing JSON/raw view).

## Model
Model: unchanged. Schema rows, keys, ranges, file format, `-set`, search and the raw JSON view stay.
Layout and widgets only: a settings row template, a category heading, a reset icon, and a toggle
switch (grep `uitoolkit/widget` for an existing switch or checkbox to build on; a new component goes
into `widget` with its tests, doc comment and doc index line, sizes from the size rules, colors from
tokens).

## Scope
1. Look at the current pages (settingsui form, colorpage, systempage, general) and `.harness/style.md`;
   write the row/heading/toggle rules into `style.md` (sizes from Ctx.Sizes, tokens, never literals).
2. Tests first: `view.Dump` and layout tests at wide and narrow widths (no clipping or overlap; the
   control keeps its space; a long description wraps under the title); toggle widget tests (click,
   keyboard, focus, disabled look, state); search, Reset, notes and descriptions keep working.
3. Implement the row pattern, the heading and the toggle; every setting page uses them.
4. Goldens: regenerate only scenes that legitimately change (settings pages, `settings-menu`) with the
   container recipe in `.harness/release.md`; look at each image.
5. Docs: `style.md`, `architecture/settings.md`.

## Acceptance
- Every setting behaves as before; booleans are switches; pages read as rows with controls on the right.
- `make check` passes.
