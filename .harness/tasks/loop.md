# Task loop

Run with `/loop` in dynamic mode, prompt: "Follow .harness/tasks/loop.md".

Order of tasks: 078, 079, 080, 081, 082, 083, 084, 085, 086, 087, 088, 089, 090. One task is finished before the next starts (`rules.md`). The main session checks the results itself; it does not trust an agent's report. Nothing is pushed.

1. Take the first task file in the order above that still exists. None left: stop the loop (`ScheduleWakeup` with `stop: true`) and report.
2. Take its first Scope item not yet done (`git log` shows the finished ones, one commit per item).
3. Spawn an Opus agent (`Agent`, `model: opus`, `isolation: worktree`) with the task file, the item, `CLAUDE.md`, `common/rules.md`, `common/commit-style.md`, `common/code-style.md`, `.harness/rules.md`, `architecture.md` and the commit trailer from the session. The agent works test first, one commit for the item, runs `make check`, updates the docs the item touches (`architecture.md` in the same commit, `style.md` if the look changes, the package doc index and template tables for toolkit API), and reports.
4. Verify in the worktree, yourself: `make check` green; the diff matches the item and nothing else; a UI change has a `make shot` PNG that was looked at; golden files changed only where the item changes the look.
5. Pass: merge into main. The last item of a task also deletes the task file in its commit, after every Acceptance line is checked. Fail: one retry with a fresh agent given the failure output; a second failure is written into the task file as blocked with the reason and the loop stops to ask the user.
6. Ambiguity: when an item can be read in more than one way or a choice changes behavior the user would notice, stop the loop, ask the user with a recommendation first (`AskUserQuestion`), record the answer in the task file or `decisions.md`, then continue. Never decide silently. The decisions already made are in each task file.
7. Task 070 adds `github.com/bufbuild/protocompile`: the dependency approval is already recorded in the task; `rules.md` gets the line in its first item.
8. Report one line per iteration: item, commit hash, `make check` result.
9. Next wakeup: 60 to 270 s while an agent runs or work remains; 1200 s or more as a fallback.
