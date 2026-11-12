# Migration loop

Run with `/loop` in dynamic mode, prompt: "Follow .harness/tasks/loop.md".

Each iteration moves one view; the main session does the checks itself, it does not trust an agent's report.

1. Read `migration.md`. Take the first `todo` row whose `Needs` tasks are deleted (done). None left: stop the loop (`ScheduleWakeup` with `stop: true`) and report. Set the row `doing`.
2. Toolkit tasks first: while a needed task file `00N` still exists, work it (its Scope order, one commit per item, then delete the file in the last commit) before any view.
3. Spawn an Opus agent (`Agent`, `model: opus`, `isolation: worktree`) with the row, `common/rules.md`, `common/code-style.md`, `.harness/rules.md` and this recipe:
   a. Pin the view: a passing `view.Dump` plus rect test at two window sizes and the key states (empty, long text, selected, narrow) and, for input, simulated clicks through `newTestApp`. Commit it.
   b. Move the view to a `.vml` file; the pinned tests stay unchanged and green; delete the Go construction; update `architecture.md` if it described the view.
   c. `refactor(<scope>): <view> is a markup template`, trailer included.
4. Verify in the worktree, yourself: `make check` (also `-tags release`, run by it); the pinned tests are unchanged in the diff (only additions before the swap); `go test ./cmd/workshop` (golden scenes) green with no golden file changed (`git diff --stat -- '*.png'` empty); `make shot` PNG of an affected scene looked at; Go lines removed.
5. Pass: merge, set the row `done` with the commit hash. Fail: one retry with a fresh agent given the failure output. Second failure: set `blocked` with the reason; if a later row needs it, stop and ask the user, otherwise continue.
6. Conventions learned from row 1 (give them to every agent):
   - Sizes and colors that a theme token or constant provides (`w.RowH`, `w.CompactH`) are passed through the scope (`h="{{rowH}}"`), never written as literals (rules.md); a literal that is not a theme value stays a literal.
   - Typed view models: Go builds a small struct per row (fields only, no methods; decision 53), the template walks it; state objects in it are pointer fields.
   - `view.Dump` does not show a field's text or ellipsis truncation: pin those through input tests (a retype needs `EditEnd` first).
   - A row's package name in the manifest may be wrong; check the directory.
   - Net Go lines usually grow (view models); the measure is construction calls removed.
   - A test that calls a view helper keeps that helper's signature through the swap; if that blocks the typed view model, walk Go-built nodes with `<slot for=... name=.../>` and note it for a later cleanup.
   - Markup has one root per template: a view that returns one of two roots is two templates and Go picks. `<include>` only reaches templates registered with `markup.Define`, not `MustLoad` ones (a known gap, report it again only if it blocks a row).
   - App unit tests use the GitHub sample collection (there "Echo" is a WebSocket request); shot tests that send need `withAdapters`; `App.step` does not call `shotUpdate`.
   - Measure golden masks from a zoomed crop of the PNG; the pixel slack can hide a changed digit.
   - If markup lacks something, stop and report it; do not special-case.
7. Progress count after each iteration: `grep -rEc '\b(Row|Col|Stack)\(' internal/*/ui` outside `Custom` behaviors, reported in one line.
8. Next wakeup: 60 to 270 s while an agent runs or work remains; 1200 s or more as a fallback.
