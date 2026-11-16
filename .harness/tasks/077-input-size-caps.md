# 077 Input size caps and editor speed

Decided by the user on 2026-11-16 (input audit): cap the size of text a user can put into an editor.

**Goal:** pasting or loading a huge text into a field or editor does not freeze or exhaust memory.

**Objectives**
1. `TextField.Insert` and the clipboard read refuse text past a limit and say so (a toast, not a silent cut). Starting value 1 MiB; confirm it with the user in the first item.
2. A single-line `Field` with a long text stays responsive: `Ctx.CursorAt` (`core/ctx.go:310-319`) is O(n^2) on every click and drag frame.
3. `TextArea.geometry` and `CodeView` layout are not rebuilt several times per frame for unchanged text.
4. The undo history is bounded by bytes as well as by count (100 snapshots of the full text).

Scope (one commit per item, test first; speed items get a test that counts work or bounds time, not a wall-clock guess)
1. Limit constant and the refusal path in `textfield.go` `Insert` and `core/clipboard.go` `Read`; the toast text; tests with a 2 MiB paste.
2. `CursorAt`: binary search over prefix widths. Test: a 1 MB line answers within a fixed number of `Width` calls.
3. `TextArea.geometry`: cache by text version; `CodeView` `layoutAt` and `Visible()` cache by lines version. Test: calls per frame.
4. Undo: cap the sum of snapshot bytes; test that old snapshots drop first.
5. Docs: `architecture.md` Limits section.

Acceptance: tests fail before and pass after; `make check` green; `make shot` of a field with long text looked at; the file is deleted in the last commit.
