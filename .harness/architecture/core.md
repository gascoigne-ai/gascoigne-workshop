## core: the toolkit

Plain values; everything except the drawing calls is tested without a GPU.

-  **Palette source.** `theming.Dark()` and `theming.Light()` take their text, muted, accent (rose
  on dark, burgundy on light), OK, error and surface colors from the color tokens of the gascoigne
  CLI dashboard's stylesheet (in the gascoigne-cli repository); `TestPalettesFollowTheCLIDashboard`
  pins them to those values. Borders, rules, chips, hover and selected rows keep the CLI's hues but
  are stronger than its soft web borders, because a native window's edges have to be visible
  (`TestNonTextElementsAreVisible`).
-  **Theme and tokens.** A `Theme` maps color `Token`s (`Bg`, `Frame`, `Sunken`, `Alt`, `Selected`,
  `Chip`, `Rule`, `Text`, `Muted`, `Accent`, ...) to colors; `theming.Dark()` and `theming.Light()`
  are the dark and light palettes (Palette source above). Applications add tokens with `Theme.With`
  (the app adds `method.GET`, `json.key`, ...). Unknown tokens fall back to the text color. A test
  checks text/surface contrast for both themes, and that non-text elements are visible: dividers
  (`Border`) at 2:1 on panes and lists, rules at 2.5:1, `AccentLine` (the accent used as an outline,
  underline or checkbox mark) and icons at 3:1, selected and hovered rows distinct from the list.
-  **`Ctx`** is what drawing and hit-testing use: theme, fonts, this frame's `Input`, tick and
  scale. Everything works in logical pixels; `Ctx` primitives (`Fill`, `Text`, `Clip`, `Line`, ...)
  multiply by `Scale` when drawing, text measures divide by it, and `Input.Scaled` converts the
  pointer back. `SetScale` and `SetTheme` rebuild the fonts. This is what makes the window sharp on
  HiDPI.
-  **Input**: `ReadInput` samples mouse (including held, for drags), wheel, typed runes, edit keys
  with repeat (the word modifier turns Left, Right, Backspace and Delete into word-wise edits:
  Option on macOS, where Command is the shortcut key and Control belongs to the system, Ctrl
  elsewhere; the primary modifier turns Home and End into document start and end; on macOS Command
  turns Left and Right into Home and End and Backspace and Delete into deletion to the line start
  and end (`EditDeleteToLineStart`, `EditDeleteToLineEnd`), and wins over Option; `modifiedEdit` is
  the table, tested per platform), Enter, Escape, Tab, Shift/Ctrl and the Ctrl+A/C/X/V/Z/Y
  shortcuts. A `Sampler` adds the click count (double and triple clicks within 400 ms and 4 px).
-  **`TextField`** (in `widget`, decision 3): single-line text with a rune cursor, a selection
  (shift+arrows, drag, shift+click, double click selects a word, triple click everything, Ctrl+A),
  word movement and deletion (Ctrl+Left/Right/Backspace/Delete, Option on macOS; with Shift it
  selects by word; the same in `TextArea` and the single-line fields), line edits on macOS
  (Cmd+Left/Right go to the line start and end, with Shift they select; Cmd+Backspace/Delete delete
  to the line start and end, and at the line edge itself take the line break so the lines join),
  horizontal scroll that keeps the cursor visible (`Reveal`, `ScrollX`) and undo/redo with
  coalescing (typing and single-character deletes merge into one step; paste, cut, word and line
  deletion are their own steps; 100 steps kept). A word (`isWordRune`) is a run of letters of any
  script with their combining marks, digits and `_`; everything else (punctuation, spaces, line
  breaks) separates words and is skipped, so word left stops at a word's start and word right at its
  end (`user.name` is two words, `https://h/v1/users` four). `Apply(typed, edits, extend)`, `Insert`
  (paste: line breaks become spaces, control characters are dropped), `DeleteSelection`,
  `SelectedText`, `Undo`, `Redo`.
-  **`TextArea`** (in `widget`): the multi-line editor, a `TextField` in multiline mode plus view
  state (`ScrollY`, optional per-line `Styler` for syntax colors). It keeps line breaks (CRLF and CR become
  LF), moves by line with a remembered goal column (Up/Down/PageUp/PageDown; past the first or last
  line goes to the start or end), Home/End by line and Ctrl+Home/End by document, Enter inserts a
  line break plus the current indentation, Tab inserts spaces to the next stop (every selected line
  is indented when the selection spans lines), Shift+Tab dedents, triple click selects a line. Line
  numbers, scrollbars (the drag and page logic is shared with `CodeView` through `barDrag`), a
  current-line highlight and cursor-follows-scroll. Tab belongs to the editor, so Escape leaves it.
  A tab character in pasted text counts as one column.
-  **Scrollbars**: `Thumb`, `PosFromOffset` and `ClampScroll` are the pure math (thumb length has a
  24 px minimum); `Ctx.Scrollbar` draws a thumb for a vertical or horizontal track.
-  **Clipboard**: `core.Clipboard` (read/write text). Ebitengine has no clipboard API, so
  `SystemClipboard()` runs the first complete pair of platform tools (`wl-paste`/`wl-copy`, `xclip`,
  `xsel`, `pbpaste`/`pbcopy`) with a 2 s timeout; with none installed every call fails with a hint
  to install wl-clipboard or xclip. No dependency is added.
-  **`CodeView`** (in `widget`): lines of colored spans (`Span{Text, Style Token}`), a fold set, a
  pixel scroll and a character-level selection (drag, also across lines; double click a word, triple
  click a line; a click without dragging selects nothing; Ctrl+A selects all). `Find` / `SetMatches`
  / `RevealLine` give it search highlighting and scroll-to-line. Ctrl+C on the focused view copies
  the selected lines, or the whole body when nothing is selected. Long lines scroll sideways
  (Shift+wheel, a bottom scrollbar; the gutter stays put), a vertical scrollbar thumb can be
  dragged, and PageUp/PageDown/Home/End scroll while it has focus. Shift with Left, Right, Up, Down,
  Home or End moves the end of the selection (the caret is where the last click or key left it, the
  first visible line when there is none): characters cross line ends, the word keys (Ctrl or Option
  with Left/Right) select by word with the text editor's word rules, skipping lines without a word,
  Cmd+Left/Right on macOS arrive as Home and End and select to the line edges, Up/Down keep the
  column, folded lines are skipped, and the view scrolls to the caret (`extendSelection`). Selection
  and copy follow the shown text: a folded line is its preview (`shownSpans`), lines inside a fold
  are skipped, and a selection of everything copies the whole text. `Visible()` hides everything
  from a folded line through its closing line, found by indentation. `Update` handles wheel and fold
  clicks; `Draw` paints with line numbers and fold arrows.
- **`Rect`** and its split helpers.
