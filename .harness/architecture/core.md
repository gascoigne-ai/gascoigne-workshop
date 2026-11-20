## core: the toolkit

`core` holds the primitives (`rules.md`, Toolkit API rule 11). Everything except the drawing calls
is tested without a GPU.

- **Logical pixels.** Widgets, layout and hit tests work in logical pixels. `Ctx` drawing calls
  multiply by `Scale`, text measures divide by it, and `Input.Scaled` maps the pointer back.
  `SetScale` and `SetTheme` rebuild the fonts and drop cached masks. This keeps HiDPI sharp.
- **Theme and tokens**: the look, the tokens and the theme pipeline are in `style.md`.
- **Input.** `Sampler.Read` samples the frame's pointer, wheel, runes and edit keys and counts
  clicks. `modifiedEdit` is the per-platform table that turns modifiers into word and line edits.
- **Clipboard.** `SystemClipboard` runs the platform's clipboard tools: Ebitengine has no clipboard
  API, and no dependency is added for it.
- **Text editors live in `widget`** (`uitoolkit/doc.go`): `TextField` (one line, undo), `TextArea` (a
  `TextField` in multiline mode plus view state) and `CodeView` (read-only spans, folds, selection).
  They share the word rule (`isWordRune`), the scrollbar math and drag (`widget/scrollbar.go`,
  `barDrag`) and the monospace geometry (`monoGrid`), so the three agree on words, caret and clicks.
