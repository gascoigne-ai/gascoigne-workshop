status: todo
kind: refactor
commit: refactor(widget): SVG parsing is table-driven
scope: uitoolkit/widget/*
depends_on: 116-parser-dedup

# Make the SVG parser table-driven

Origin: the repo audit of 2026-11-19 (three read-only reviews against the harness rules). The user approved all phases and, with that, the deletions and the gate change named here. Behavior stays the same unless the task says otherwise; tests stay green and none is loosened. Update `architecture.md` in the same commit where it describes what changes.

- `parsePathData` (150 lines, nesting 5): a `pathState` plus a table from command letter to argument count and handler; remove the leftovers (`op := up`, double `prev = up`, command letters written twice, the `angle` temp).
- `shapeCmds`, `applyStyle`, `parseSVGAs`: `shapes` map of element name to builder, with the "unknown element" list derived from it; `lineCaps` and `lineJoins` maps instead of switches; extract `parseRoot`; replace the `picture bool` mode with a small paint-parser value.
- A fuzz target exists for SVG? If not, add one (untrusted input rule) and list it in `FUZZ_TARGETS`.
Acceptance: SVG tests and rasterized goldens unchanged; `make check` green.
