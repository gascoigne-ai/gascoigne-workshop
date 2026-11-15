# 069 Code view wrapping and word-wise cursor movement

Known limitation: no text wrapping in the code view and no word-wise cursor movement. IME is out of scope by decision of the user.

**Goal:** long lines and keyboard editing behave as in a normal editor.

**Objectives**
1. Long lines can wrap at the view width; the horizontal scroll still works when wrap is off.
2. Cursor, selection, click hit-testing, scrolling and line numbers stay correct with wrapped lines.
3. Ctrl+Left/Right (Option on macOS) move by word; with Shift they select by word; Ctrl+Backspace/Delete (Option on macOS) delete a word.
4. Word boundaries are defined and tested (letters, digits, `_`, punctuation, unicode).

Scope
1. Word motion in the text state (pure, tested), then the key bindings per platform.
2. macOS line edits (decided 2026-11-15): Cmd+Left/Right go to line start/end, Cmd+Backspace deletes to line start, Shift selects; in the code view and every text field; platform table tested like the word keys.
3. Wrap layout as a pure function (rects and visual lines in, rects out), tested without a GPU.
4. Wrap in `Code` and the editing views through the toolkit, with an option and a template attribute (toolkit API rules 9 and 10).
5. Golden scenes, `make shot` looked at; docs and package doc index.

Acceptance: items tested, `make check` green, docs updated.

**Decided (user, 2026-11-15)**
- Wrap is off by default, with a toggle in the UI.
- Wrap applies to the response viewer and the request body editor.
- Word-wise movement and deletion also work in the single-line fields.
- IME stays out of scope.
- Item 1 (word motion) is done in 87dba94. On macOS Cmd+Left/Right and Cmd+Backspace become line edits (item 2).
- The word keys are not listed on the Settings Shortcuts page; not part of this task unless the user asks.
