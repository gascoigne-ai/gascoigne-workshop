status: todo
kind: refactor
commit: refactor(widget): CodeView and TextArea share one monospace grid
scope: uitoolkit/widget/*,.harness/*
depends_on: 111-http-builder

# Share the monospace grid of CodeView and TextArea

Origin: the repo audit of 2026-11-19 (three read-only reviews against the harness rules). The user approved all phases and, with that, the deletions and the gate change named here. Behavior stays the same unless the task says otherwise; tests stay green and none is loosened. Update `architecture.md` in the same commit where it describes what changes.

The two widgets duplicate: geometry (`lineH`, `charW`, `textX`, `adv`, `colX`, `colAt`), wrap width, `keepTopLine`, mouse wheel scrolling (magic `*6`, `*3`), span row drawing, selection x0/x1.
- Introduce an unexported `monoGrid` embedded in both; name the wheel constants; keep behavior, goldens and tests unchanged.
Acceptance: `make check` green; about 90 lines fewer.
