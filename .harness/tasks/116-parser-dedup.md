status: todo
kind: refactor
commit: refactor(model): importers and checks lose their duplicated tables
scope: internal/collection/model/*,internal/request/model/*,internal/request/adapter/*,internal/response/*
depends_on: 115-file-adapters

# Remove duplication inside the parsers

Origin: the repo audit of 2026-11-19 (three read-only reviews against the harness rules). The user approved all phases and, with that, the deletions and the gate change named here. Behavior stays the same unless the task says otherwise; tests stay green and none is loosened. Update `architecture.md` in the same commit where it describes what changes.

- Postman import and export use two type families and two switches (`kv`/`outKV`, `auth`/`outAuth`, body modes and auth types as string literals twice): one body-mode table and one auth table used both ways; a named options type instead of the anonymous struct written three times; inherited auth computed once; a bad URL goes to `Skipped` instead of being dropped (`parseURL`).
- cURL flags are in the `ParseCurl` switch and again in `curlValueFlags`: one flag table; split `ShellSplit` (83 lines, 5 levels) per quote kind.
- `assert`: number comparison twice, header lookup twice, the missing-header branch twice: an operator table and one `header(response, name)`.
- `miniyaml`: the sequence-item test written five times, `sequence` has two near-identical blocks; add `miniyaml.ToJSON(data)` used by `openapi` and `clientconfig`; assign the error before returning the tree in `clientconfig.go` (`return tree, json.Unmarshal(data, &tree)` depends on evaluation order).
- `failureText` and `stopReason` near-duplicates; two different `looksJSON`; the ordered-JSON writers in `mcpschema.go`, `mcpform.go`, `sections.go`.
Acceptance: fixtures and tests unchanged; `make check` green.
