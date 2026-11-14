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
   - A floating or modal view used as a test's layout root fills the window: wrap it in `Stack(...)` in the test. A bool field printed with `{{...}}` works as a conditional flag; a conditional color is a `core.Token` passed through the scope (`color="{{color}}"`); an optional sub-view is a nil `Node` field plus `<slot if="x.Field" name="x.Field"/>`. To capture dumps, write them from the pin test once behind an environment variable and remove that code before committing.
   - A `for` cannot pick between two roots per item: Go builds each item from a one-root template and the parent walks them with `<slot for="x in xs" name="x"/>`. A big view may be swapped one control kind at a time (one pin commit, then a swap commit per kind). Slider arguments do not show in `Dump`: pin them by pressing at a fraction (at most 0.99) of the slider's width.
   - Widget compositions with a Go constructor have an element (task 011): `<choice selected>`, `<form-row label label-width>`, `<check-row on>` (label and hint as children, `on-click` on the row), `<lines color>` (text from `{{path}}`, literal text loses its newlines) and `<note tone>`; use them, never the primitive tree. Another gap copied into three views becomes a task (like 007).
   - `-pane LABEL` scene flag exists. Dump does not show ticked checkboxes: pin with `%+v` of the `widget.checkB` behavior. XML comments work in `.vml`.
   - Row 13 notes: view-mode scene = `-set responseView=<mode>` with `-send`; `newTestApp` starts in Pretty (app default Compact), set the mode explicitly (row 20). `toolbar` is written out twice (a third copy makes it a task). Computed tooltip via scope. Nil Node in a scope map also leaves out `<slot if>`.
   - Before step 0 check the row's function is reachable (grep its kind/builder); row 12 `codePane` was dead.
   - "Stable over 3 runs" means the whole TestGoldenScenes set or `make check`, not one scene alone. A shot flag that changes state without input must call `markStale()` (fixed for `-type-url`; check `-filter`, `-request`, `-open-all`). Two-root view = two templates from one `//go:embed a.vml b.vml`.
   - One-root item: `<row for="x in xs">` straight over a typed slice, no slot walk. Per-item flag: `active-tab="{{x.Active}}"`. Shared strings/handlers go in the scope, not each item.
   - Pin what Dump misses (min-w, placeholder, segments, on-enter, state pointer) with `findWidget` + `fmt.Sprintf("%+v", node.Behavior())` (urlbar_view_test.go). Mutually exclusive variants differing in sizing options: `if` siblings driven by bools from Go. `-sending` scene flag exists. `vrule-inset` has one written-out copy (urlbar.vml).
   - A conditional option is switched off by its value (decision 54): a color attribute given an empty `core.Token` from the scope applies nothing (`border="{{row.DropBorder}}"`, the sidebar drop outline), and `separator="{{x}}"` with 0 applies none (a tab strip's first tab). No Go-side patching of a built node.
   - A two-root sub-view inside an item becomes `if`/`if="!x"` siblings in the item template. A fragment shared by two `MustLoad` templates is copied with a comment pointing at the other.
   - `-filter TEXT` scene flag exists. Task 010: filter match color is near invisible.
   - A scene that needs state after a response: act once `sendsDone()` is true and make `ready` wait for it (`-find`). A field filled with `Insert` leaves the caret always drawn, so no mask needed.
   - `Dump` does not print placeholders; carry them over by hand.
   - A test that ends while a background save runs breaks TempDir cleanup: use `t.Cleanup(a.waitWrites)`.
   - Known gap with two copies so far (rows 5, 10): no `tab-bar` element (written out as a `<row border-b ...>`). A third copy makes it a task.
   - If markup lacks something, stop and report it; do not special-case.
7. Progress count after each iteration: `grep -rEc '\b(Row|Col|Stack)\(' internal/*/ui` outside `Custom` behaviors, reported in one line.
8. Next wakeup: 60 to 270 s while an agent runs or work remains; 1200 s or more as a fallback.
