# Brief for the agent that migrates one row

You work in a git worktree of gascoigne-workshop (Go, Ebitengine UI toolkit). First `git merge --ff-only main` if behind.

Read, in order: CLAUDE.md, .harness/common/rules.md, .harness/common/commit-style.md, .harness/common/code-style.md, .harness/rules.md, .harness/decisions.md entries 11, 50, 51, 52, 53, .harness/tasks/loop.md (all of it, including the conventions learned), .harness/tasks/migration.md, the markup package doc (uitoolkit/markup/component.go), the finished examples (internal/settings/ui/colorpage and internal/response/ui/sentlog: .go, .vml and the pin tests), and cmd/workshop/golden_scenes_test.go.

Your row is named in the prompt. Move its view construction into embedded `.vml` templates (go:embed + markup.MustLoad, parsed once at package level); Go keeps the scope, typed view models and handlers, and Custom behaviors reached through slot. Pixels must not change.

Order, mandatory:
0. If the row says a scene is missing: add a golden scene first (flag test first as in internal/workspace/ui/app/shot_test.go; generate with `go test ./cmd/workshop -run TestGoldenScenes/<name> -update`; LOOK at the PNG; mask anything that changes between runs; stable over 3 runs; update the screenshot-mode line in architecture.md). Commit `test(shot): golden scene for <view>`.
1. Pin the view with tests that pass on the current Go code: `view.Dump` after `view.Layout` at two sizes and the key states, plus simulated input through the package's fakes. Dump misses field text and ellipsis: pin those through input. Commit `test(<pkg>): pin the <view> view`.
2. Swap: template(s) replace the Go construction. Test files and testdata have a ZERO-line diff in this commit. Commit `refactor(<pkg>): <view> is a markup template` with the trailer `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`. Update architecture.md where it describes the view.
If markup lacks something the view needs, STOP and report exactly what; do not add toolkit features or special-case. If the manifest names the wrong package, use the real one.

Rules: no new dependencies, no one-letter parameter names, comments say why, never skip/delete/loosen a test, layer rules. The sandbox may refuse chained commands or heredocs: use the Write tool and one git command at a time. Do not push.

Verify before reporting: `make check` green in the worktree (needs xvfb-run, installed, ~3 min; golden scenes must run, not skip); `git diff --stat main -- '*.png'` lists only PNGs of scenes you added. Final report (short): commits with hashes, what the pins cover and that they passed before and after the swap, make check and golden results, construction calls removed, blockers, recipe changes for later rows.
