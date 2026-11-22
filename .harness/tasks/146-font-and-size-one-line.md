status: todo
kind: feature
commit: feat(settingsui): a font and its size share one line
scope: internal/settings/**, cmd/workshop/**, internal/workspace/**, .harness/style.md, .harness/architecture/settings.md
not_before: 2026-01-01

# Font and size on one line

Goal: the Appearance page uses less vertical space. The interface font and its size, and the code
font and its size, each sit on one line: the font drop-down on the left, the size stepper (the
`-` field `+` control) on the right. User request: "font selection and size ... in single line".
The small text size has no font of its own and keeps its own row, directly under the interface line.

## Model
Model: unchanged. The schema rows (`uiFont`, `fontSize`, `smallFontSize`, `codeFont`, `codeFontSize`),
their order, ranges, keys, file format, `-set`, search and the raw JSON view stay as they are. This is a
form layout change only: `settingsui` renders the size row of a face inside the line of its font row
and no longer lists it as a separate row. Each size keeps its own Reset, its note for out-of-range
typing and its description text (shown under the line, one description per face covering both
settings; wording decided by the implementer, kept short).

## Scope
1. Tests first: `view.Dump`/layout tests of the Appearance page at the wide and narrow widths: the
   font control and the stepper share one row at the wide width and wrap under each other when the
   dialog is narrow (no clipping, no overlap); the size row of each face is not listed separately;
   searching "size" still finds the face's line; Reset of the size and of the font work independently;
   a typed out-of-range size still shows the range note; row order and the small text row.
2. Form layout in `settingsui`: a paired control built from view primitives with sizes from
   `ctx.Sizes` (no raw numbers; the size guard test applies); the stepper keeps working as today.
3. Goldens: regenerate only the scenes that legitimately change (the Appearance form, `settings-menu`)
   with the container recipe in `.harness/release.md`; look at each image.
4. Docs: `style.md` (settings form rules: a setting that belongs to another may share its line),
   `architecture/settings.md`.

## Acceptance
- The Appearance page is shorter than before (fewer rows) with identical behavior of every setting.
- `make check` passes.
