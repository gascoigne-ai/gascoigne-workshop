status: todo
kind: feature
commit: feat(settingsui): a font and its size share one line
scope: internal/settings/**, cmd/workshop/**, .harness/style.md, .harness/architecture/settings.md
not_before: 2026-01-01
depends_on: 146-settings-row-layout

# Font and size on one line

Goal: the interface font and its size, and the code font and its size, each sit on one line: the font
drop-down on the left (grows, elides with an ellipsis) and the size stepper (`-` field `+`, with its own
Reset when changed) on the right. Small text size has no font of its own: it stays a plain row under the
interface line. Uses the row pattern from task 146.

## Model
Model: unchanged (no schema, file, key, range or `-set` change). One unexported UI table in
`settingsui`: `fontFaces = []fontFace{{Font:"uiFont", Size:"fontSize"}, {Font:"codeFont",
Size:"codeFontSize"}}` (keys of `settings.Schema`) with helpers `sizeOf(fontKey)` and `pairedSize(key)`.
Access pattern: lookup per schema row per frame over 2 entries; a slice of structs. Rejected: a map plus
inverse (two sources of truth), a `Pair` field on `settings.Field` (model change for a UI concern).
Revisit at more than ~6 faces or a non-UI consumer. Invariant pinned by a test: both keys exist, same
category, the size is Int, the font is not hidden.

## Scope
1. Tests first (settingsui, with the existing dump/Layout/testCtx helpers): no standalone "Interface
   font size"/"Code font size" row but "Small text size" stays directly under the interface line; font
   control and stepper share one row at the dialog width (same y range, disjoint, inside the row); no
   overlap or clipping at the smallest dialog width (MinWidth) with the size Reset showing and the
   longest family name (NO wrapping: the font control shrinks with an ellipsis and keeps a named
   minimum width; decided, the toolkit has no wrap primitive); search "size" finds the face line once;
   Reset of the size and of the font are independent (font Reset still applies fonts); typed
   out-of-range size shows the range note under the line; one description per face containing the range
   built from the size field's Min/Max (no new copy of the range).
2. Compose the paired control from view primitives (a template, widths as named constants or
   `ctx.Sizes`; the size-literals guard applies); `FormPage` skips paired size fields; search matches the
   face line through either field.
3. Dumps and goldens for the changed Appearance page and `settings-menu` (container recipe in
   `.harness/release.md`); look at each image.
4. Docs: `style.md` (a setting that belongs to another may share its line), `architecture/settings.md`.

## Acceptance
- Appearance page has two fewer rows; every setting behaves as before.
- `make check` passes.
