status: todo
kind: feature
commit: feat(settings): a size setting for each font next to its choice
scope: internal/settings/**, uitoolkit/theming/**, uitoolkit/core/theme.go, uitoolkit/core/sizes.go, uitoolkit/core/sizes_test.go, uitoolkit/core/sizes_ctx_test.go, uitoolkit/examples/**, cmd/workshop/**, internal/workspace/**, .harness/style.md, .harness/architecture/settings.md, .harness/architecture/persistence.md, CHANGELOG.md
not_before: 2026-01-01

# One size setting per font

Goal: the user sets the size of every font type: interface, small and code. Each size is a number
field with the stepper that "Text size" has today, placed right next to the setting that picks that
font. Today one `fontSize` (10..24, default 14) drives the interface face and the small (size-2, min 9)
and code (size-1) faces are derived. User decision: Interface, Code and Small each get their own
field; control heights keep following the interface size through `Ctx.Sizes`.

## Model
Settings gains `SmallFontSize` and `CodeFontSize` (px) beside `FontSize`, which keeps its key and
meaning (interface face; bold uses it). Defaults 14 / 12 / 13, so default screens do not change.
Ranges: interface 10..24 (as now, `MaxLine` 28 limit), small 8..24, code 8..32; one clamp per field
(`clampFontSize` style), a 0 from skipped validation becomes the default. `theming.TextSizesFor(font)`
no longer derives; theme building takes the three sizes (`theming.Prefs` carries all three) and
`Catalog.For` applies them. Small is clamped to the interface line only inside `SizesFor` (already
there). File version 2 -> 3: a version-2 reader sets the two new keys from the old rule when absent
(small = max(9, fontSize-2), code = fontSize-1), so nobody sees text change on upgrade; a v3 file keeps
what it says. Old `textSize` (v0/v1) reader keeps working through the existing chain. Schema rows, in
the Appearance page, in this order: Interface font, Interface font size (renamed from "Text size"),
Small text size, Code font, Code font size; "Interface scale" stays where it is. Layout
(`settingsui`): each size row uses the existing stepper control; the row text says which face it sizes.
Access pattern, structure and alternatives for the settings struct stay as is (flat struct, schema
table); rejected: a per-face offset from the interface size (hard to explain, two sources of truth);
revisit when a fourth face (heading, say) appears.

## Scope
1. Tests first: legacy readers (v0/v1 textSize, v2 fontSize-only file -> derived small and code), v3
   round trip, `Clamp` per field, schema order and `Validate`/`Expectation` per row, `Minimal` omits
   defaults, theme building from three sizes, form stepper per row incl. out-of-range typing, fuzz seeds.
   Fixtures are real writer output: version-2 fixtures already exist, add a v3 one.
2. Settings model, schema, file version 3 reader, fixtures, `encode-*.golden` regenerated.
3. `theming` and `appearance` take the three sizes; callers in `cmd/workshop` (`-set`, scenes) and the
   app (call sites only) follow; the `-set` keys are `fontSize`, `smallFontSize`, `codeFontSize`.
4. Settings form rows in the order above; goldens: regenerate only scenes that legitimately change
   (the Appearance form, `settings-menu`) with the container recipe in `.harness/release.md`; look at
   each image; add a scene with a changed code size so the mono face is covered.
5. Docs: `style.md` Type, `architecture/settings.md`, `architecture/persistence.md` (settings v3),
   CHANGELOG line (users: sizes per font; existing sizes stay as they were).

## Acceptance
- At default settings and after upgrading from v2 nothing changes on screen except the settings
  dialog (new rows).
- Each size row works from the form and from `-set`; out-of-range input shows the range note.
- `make check` passes.
