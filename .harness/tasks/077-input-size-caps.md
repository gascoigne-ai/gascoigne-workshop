# 077 Input size caps and editor speed

Decided by the user on 2026-11-16 (input audit): cap the size of text a user can put into an editor.

**Goal:** pasting or loading a huge text into a field or editor does not freeze or exhaust memory.

**Objectives**
1. `TextField.Insert` and the clipboard read refuse text past a limit and say so (a toast, not a silent cut). Starting value 1 MiB; confirm it with the user in the first item.
2. A single-line `Field` with a long text stays responsive: `Ctx.CursorAt` (`core/ctx.go:310-319`) is O(n^2) on every click and drag frame.
3. `TextArea.geometry` and `CodeView` layout are not rebuilt several times per frame for unchanged text.
4. The undo history is bounded by bytes as well as by count (100 snapshots of the full text).

Done: paste capped at 1 MiB with a toast (default chosen, no fork: a conventional limit); `CursorAt` by binary search (a 200k-rune line did not finish in 30 s); undo history bounded to 8 MiB of snapshots. Limits are in `architecture.md`.

Measured and dropped (rating below 5): per-frame layout of a 1 MiB `TextArea`, wrapped or not, costs about 11 ms a frame, so no layout cache. Not done: a cap on text loaded from files or typed key by key (only the paste path was a way to put megabytes in at once).

Scope: all items done or dropped. The user decides whether this file is deleted (rules: ask before deleting).

Acceptance: met; `make check` green.
