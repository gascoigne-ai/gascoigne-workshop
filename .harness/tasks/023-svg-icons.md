status: in progress (left: a mask cache if profiling asks, an open icon set)

# Icons as vector files (SVG), drawn and themed by the toolkit

## Goal
The line icons (`CrossIcon`, `TrashIcon`, `SearchIcon`, `PlusIcon`, chevrons, dots, copy, prompt, checkbox, triangle) are hand-written Go drawing functions in the `glyphs` table (`uitoolkit/vml/leaf.go`) with sizes and hover colors in two more tables. Adding an icon means writing path code; designers cannot supply one; colors and stroke weight are fixed in Go.

## Scope
- **Format:** icons are SVG files, one per icon, embedded in the toolkit (`uitoolkit/icons/*.svg`) and loadable from an application folder. Subset only: `path` (commands `M L H V C S Q T A Z`, relative and absolute), `line`, `polyline`, `polygon`, `rect`, `circle`, `ellipse`; `viewBox`; `fill` and `stroke` as `none`, `currentColor` or a theme token name (`stroke="token(muted)"`); `stroke-width`, `stroke-linecap`, `stroke-linejoin`. Anything else is a parse error naming file and line. Parsed with `encoding/xml`, no new dependency.
- **Rendering:** a parsed icon becomes `vector.Path`s in unit space; drawing scales them to the element's logical size at the device scale (crisp at fractional scales, using the same pixel snapping as lines), antialiased. `currentColor` is the element's text/icon token, so an icon follows the theme and hover state without per-icon tables.
- **Theming:** stroke width comes from the theme (a `icon` control style or a token), so "thin", "regular" and "bold" icon sets are a theme choice; hover and disabled colors come from tokens, not a Go map.
- **API:** `vml.Icon(name string, opts ...Option)` for any registered icon, plus `vml.RegisterIcons(fs.FS)`; the existing `XxxIcon()` functions stay as one-line wrappers or go away once the app uses names. Templates get `<icon name="trash"/>`.
- **Cache:** parse once per file; optionally cache the rendered mask per (icon, size, scale) if drawing paths every frame shows in profiles.
- **Icon set:** start by converting the existing icons by hand to match their current look; later adopt an open set (check the license: Lucide is ISC, Tabler is MIT) and vendor only what is used, with the license file.

## Decisions to take first
1. Subset above, or a full SVG library? Recommendation: the subset, hand-written parser, stdlib only (same reasoning as decision 7 and 10).
2. Where the files live: `uitoolkit/icons` (embedded, shared) with an app overlay folder. Recommendation: yes.
3. Stroke-only icons, or fills too? Recommendation: both, since checkbox and triangle are filled.

## Acceptance
- Every current icon is an SVG file and the Go drawing functions and the three icon tables are deleted.
- A screenshot comparison at 100%, 150% and 200% shows no visible regression; a test renders each icon without a window (path extents inside the viewBox).
- Parse errors name file and line; `component.Reference()` lists `<icon>`.
- Architecture, rules (new options get docs, tests, template attribute) and the `vml` index updated in the same commit.

## Progress
- Done: parser and renderer (`svg.go`, `icon.go`), eleven icons converted (screenshot at 200% checked), the Go drawing functions and the three icon tables deleted, `Icon`, `RegisterIcons`, `IconNames`, `HasIcon`, `<icon>` in templates, tests (parse errors with file and line, path commands, arcs, extents inside the viewBox, drawing at 100/150/200%).
- Decisions taken: subset parser, no dependency; files embedded in `vml/icons` with `RegisterIcons` for an application folder; both stroke and fill supported.
- Done later: stroke weight comes from the theme (`iconStroke` in a theme file, `Theme.IconStroke`, a multiplier; absent keeps icons as drawn).
- Left: an optional rendered-mask cache if profiling asks for it, and adopting an open icon set (license check) when more icons are needed.
