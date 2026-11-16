# 075 Fuzz targets and input gap tests

From the input audit of 2026-11-16: the repo has no `func Fuzz` and no test of hostile input for most parsers.

**Goal:** a short fuzz run on every parser or editor that takes user or imported text finds no panic, hang or runaway allocation, and the cases it finds stay as normal tests.

**Objectives**
1. Native Go fuzz targets (stdlib `testing.F`, no new dependency) for each entry point below, seeded from real files in `testdata/`.
2. `make fuzz` runs each target briefly (30 to 45 s); it is not part of `make check`.
3. The audit's gap tests exist as normal table-driven tests.

Done: `make fuzz` with 17 targets (listed in `architecture.md` Limits); each ran 12 to 20 s with no crasher. Gap tests pinned: hostile `state.json`, zero-size editors, split and code view, hostile pasted text (NUL, invalid UTF-8, bidi, C1, combining marks, ZWJ).

Dropped by the bug rating rule (frequency x severity below 5): AltGr typing is lost only where AltGr reports as Ctrl+Alt (Windows; this app targets Linux); `CodeView` selection after `Lines` is mutated directly reads clamped values and is cosmetic; duplicate shortcut bindings in raw settings fire both actions (the user wrote them).

Not done, no ask needed: `FuzzTarget` for `Input.Scaled` (covered by a table test), startup with a missing collection and a corrupt state file together (each is covered alone).

Scope: all items done or dropped. The user decides whether this file is deleted (rules: ask before deleting).

Acceptance: met; `make fuzz` passes; `make check` green.
