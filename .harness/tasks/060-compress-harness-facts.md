# 060 Drop outdated facts from the harness, compress past decisions

Asked for by the user. The harness grew with the work: decisions.md has 61 entries, many superseded or describing removed code, and the migration files describe a finished job.

**Goal:** `.harness/` holds current facts only; a new reader gets the design in one pass.

**Objectives**
1. decisions.md is at most half its present size, with one entry per surviving fact, each keeping date and why.
2. No entry describes code that no longer exists (for example decision 5 "VML is a Go-native DSL", 14, 16, 17, 31).
3. architecture.md has no stale section.
4. The migration artifacts (`migration.md`, `loop.md`, `agent-brief.md`) are gone; every row is done.

Scope
1. Audit decisions.md, architecture.md, style.md, rules.md against the code; list each stale or superseded item in this file with its verdict (drop, merge, keep).
2. Merge superseded chains into one entry each: the task 033 package scheme (19 to 32), Model/Msg/Update (34 to 37), toolkit layering (40 to 49), views in markup (50 to 54). Keep dates and the reason.
3. Delete the three migration files.
4. Fix references: no dangling "decision N" in any harness file or code comment (grep).

Acceptance: objectives 1 to 4 hold, `make check` green.
