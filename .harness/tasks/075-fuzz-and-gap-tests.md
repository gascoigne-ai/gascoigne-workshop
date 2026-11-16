# 075 Fuzz targets and input gap tests

From the input audit of 2026-11-16: the repo has no `func Fuzz` and no test of hostile input for most parsers.

**Goal:** a short fuzz run on every parser or editor that takes user or imported text finds no panic, hang or runaway allocation, and the cases it finds stay as normal tests.

**Objectives**
1. Native Go fuzz targets (stdlib `testing.F`, no new dependency) for each entry point below, seeded from real files in `testdata/`.
2. `make fuzz` runs each target briefly (30 to 45 s); it is not part of `make check`.
3. The audit's gap tests exist as normal table-driven tests.

Scope (one commit per item)
1. Add `make fuzz` and the first target, `miniyaml.Parse`; run it, fix what it finds under task 074 rules (regression test first).
2. Targets: `model.ShellSplit` and `ParseCurl`; `ParamsFromURL` and `ApplyPathParams`; `assert.Run`; `jsonview.Compact` and `Lines`; `openapi.Parse`; `postman.Parse`; `clientconfig` decode; `ws` frame reader; `markup.Parse` and `interpolate`.
3. Targets for the editor: `TextField.Apply` with random typed runes and edits plus `Insert`, `CodeView.Find`, `wrapRunes`, `Input.Scaled`.
4. Gap tests: invalid UTF-8, CRLF, combining marks and C1/bidi characters in `Insert`; zero-size and negative `Rect` for `TextArea`, `CodeView`, `splitB`; AltGr (Ctrl+Alt) typing; hostile `state.json` (huge `sidebarW`, zero or negative column weights, `Active` out of range, duplicate tabs); startup with a missing collection and a corrupt state file together; duplicate and reserved shortcut bindings in raw settings; a send result arriving after its tab closed; `CodeView` selection after `Lines` is mutated directly.
5. Docs: a line in `rules.md` or `architecture.md` on `make fuzz` and where seeds live.

Acceptance: `make fuzz` passes on a clean tree; `make check` green; every crasher found is a committed regression test; the file is deleted in the last commit.
