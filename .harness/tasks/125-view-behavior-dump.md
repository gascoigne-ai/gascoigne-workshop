status: todo
kind: refactor
commit: refactor(view): one Behavior contract; Dump shows every look attribute
scope: uitoolkit/view/*,uitoolkit/widget/*,uitoolkit/markup/*
depends_on: 124-markup-build-table

# Remove the private twin of Behavior; complete Dump

Origin: the repo audit of 2026-11-19 (three read-only reviews against the harness rules). The user approved all phases and, with that, the deletions and the gate change named here. Behavior stays the same unless the task says otherwise; tests stay green and none is loosened. Update `architecture.md` in the same commit where it describes what changes.

- `view` has a private twin of its own extension point: `behavior` and `plain` against `Behavior` and `Base`; `preUpdater`, `overDrawer`, `customB`; `Linear` wraps `flow` again. Built-in kinds implement `Behavior` directly; delete the lowercase interfaces. `Dump` kind names are pinned by tests: keep them. Fold `Describer` into `Behavior` if it still has one implementation.
- `Dump` omits `minW`, `maxW`, `minH`, `maxH`, `alignItems`, `alignSelf`, `justify`, `clip`, `dismissMargin`, `dragHandle`, `roundCtl`, `roundCorners`, `reveal`, `moved`, while its doc and decision 4 say equal dumps mean equal looks: one table of name and getter covering every `El` attribute (test first: two elements differing only in one omitted attribute dump differently). Pinned dumps in tests may change only by gaining these attributes.
- `OnClick`/`OnClickAt` and `OnRightClick`/`OnRightClickIn`: four fields and four runtime branches for two events; fold to two.
Acceptance: `make check` green.
