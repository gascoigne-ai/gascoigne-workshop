status: todo
kind: refactor
commit: refactor(markup): include, slot and icon are table rows
scope: uitoolkit/markup/*,uitoolkit/view/*
depends_on: 123-logic-out-of-ui

# Remove the special cases from the markup builder

Origin: the repo audit of 2026-11-19 (three read-only reviews against the harness rules). The user approved all phases and, with that, the deletions and the gate change named here. Behavior stays the same unless the task says otherwise; tests stay green and none is loosened. Update `architecture.md` in the same commit where it describes what changes.

- `buildOne` (113 lines) special-cases `include`, `slot`, `icon` (and `placeholder`, `standsIn`, `name` hacks): make `name` a normal parameter; `slot` becomes `scoped[view.Node]("name")`; `icon` becomes `textParam("name")` with the `HasIcon` check of `iconButton`; `include` gets its scope through `elementParts`. Delete `placeholder`, `standsIn` and the three loops.
- `Parse` (84 lines, nesting 4): `startElement`, `checkAttr`, `charData`; `spec.takes(name)` replaces the five-clause condition; one lookup instead of two; the error text lists the elements from the table.
- Twenty hand-written adapters over `[]any`: generic arity helpers (`args1[A]`, `args2[A,B]`, with option and item variants), no reflection.
- Atoi/bool parsing copied: `wholeNumber`, `truth`, `interpolated(name, parse)`.
- `spacer`, `hrule`, `vrule`, `vrule-inset`, `include`, `slot` hold text by mistake (`<spacer>abc</spacer>` parses and drops the text): they hold nothing (regression test first).
Acceptance: the template reference and all markup tests unchanged; `make check` green.
