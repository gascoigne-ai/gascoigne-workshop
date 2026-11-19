status: todo
kind: docs
commit: docs(harness): architecture.md matches the code, links the package map
scope: .harness/*,CLAUDE.md

# Bring the harness docs in line with the code

Origin: the repo audit of 2026-11-19 (three read-only reviews against the harness rules). The user approved all phases and, with that, the deletions and the gate change named here. Behavior stays the same unless the task says otherwise; tests stay green and none is loosened. Update `architecture.md` in the same commit where it describes what changes.

Scope:
- `architecture.md`: remove the hand-written package lists of the diagram and the package table; link `packagemap.md` instead. Fix `controller/transfer` "not wired yet", the `Move` message that does not exist in `controller/tree`, `ui` named for package `app`, the pane kinds list, the Host-package list (add `destpicker`, `systempage`), the unbalanced parenthesis near the app section.
- Wrap or split the 2600-character lines of `architecture.md` so diffs read (split per domain only if wrapping is not enough).
- Say the import-layer rule and the toolkit layer rule once (`rules.md` and the generated `packagemap.md`); `overview.md`, `architecture.md` and `decisions.md` link to them.
- `CLAUDE.md` line 5: `make check` also runs the `release`-tag vet and tests.
- Record in `rules.md` that layers are enforced by `internal/layers_test.go` instead of `layers.txt` and `scripts/check-layers.sh`, and why.
Acceptance: `make check` green; no fact in the docs contradicts the code.
