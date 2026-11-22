status: todo
kind: feature
commit: feat(settingsui): settings rows follow a Zed-like layout
scope: internal/settings/**, cmd/workshop/**, internal/workspace/**, .harness/style.md, .harness/architecture/settings.md
not_before: 2026-01-01

# Settings in a Zed-like layout

Goal: the settings dialog presents its settings the way Zed's settings page does (reference image:
/tmp/claude-1000/-home-lukaszb-Development-gascoigne-gascoigne-workshop-main/9915e3f2-18df-4415-b424-d658430aedc4/images/1.png; read it). What the user likes in it, to take over in our
look and tokens (not a copy of its colors or fonts):
- A row is: the title (stronger weight) on the left with a small reset icon next to it when the value
  differs from the default, the description below it in the muted face, and the control on the right of
  the same row (drop-down, toggle, number stepper), vertically centered on the title block.
- A category is a small muted heading (e.g. "Base Keymap", "Modal Editing") with a thin rule under it;
  rows are separated by a light rule, with generous vertical air.
- Booleans are toggle switches, not text chips; enumerations with few values stay as chips or a
  drop-down by width.
- Left navigation with a search box on top; the page title row shows where the settings live and an
  "Edit in settings.json" button (our existing JSON/raw view).
Also the user asked, in this order: (1) a font and its size on one line: the font drop-down and the size
stepper side by side (interface font + interface size; code font + code size), small text size under the
interface line; (2) the keybindings (shortcuts) page gets a better presentation in the same spirit:
actions listed with their name and description on the left and the key chips on the right, grouped by
category headings, with a search box, readable at a glance (design it from the same row pattern).

## Model
Model: unchanged. The schema rows, keys, ranges, file format, `-set`, search and the raw JSON view stay.
Layout and widgets only: a settings row template, a category heading, a toggle switch (check whether
`widget` has one before adding; a new component goes into `widget` with its tests and doc index line),
a reset icon, and the keybinding row. Where two settings share a line, a table in `settingsui` (not a
switch) says which size row belongs to which font row.

## Scope
1. Design first (task-designer): look at the current pages (settingsui form, shortcutpage, colorpage,
   systempage), the reference image and `.harness/style.md`; split the work into steps that each leave
   `make check` green; put the visual rules into `.harness/style.md` (Rows, headings, toggle, spacing
   from the size rules and tokens, never literals).
2. Tests first for every step (`view.Dump` and layout tests at wide and narrow widths: no clipping or
   overlap; the control wraps under the title when the dialog is narrow; search, Reset, typed
   out-of-range note and descriptions keep working).
3. Implement in steps: row pattern and heading, toggle, font+size line, shortcuts page.
4. Goldens: regenerate only scenes that legitimately change (settings pages, `settings-menu`,
   `settings-shortcuts*`) with the container recipe in `.harness/release.md`; look at each image.
5. Docs: `style.md` (settings form rules), `architecture/settings.md`.

## Acceptance
- Every setting behaves as before; the Appearance page is shorter; the shortcuts page reads as a
  grouped list with chips on the right.
- `make check` passes.
