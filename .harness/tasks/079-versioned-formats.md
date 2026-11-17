status: todo
kind: feature
commit: feat(store): version every saved format
scope: internal/*,.harness/*,testdata/*
depends_on: 078-protocol-type

# 079 Versions on saved formats

From the audit: `state.json`, `folder.json`, `*.req.json`, history files, `sent.jsonl` and `settings.json` carry no version; old formats are recognised by guessing (`store/store.go`, `state/state.go`, decision 17). Rule: persisted formats are versioned and have a fixture of real output.

**Goal:** every written format has a `version`; a missing version reads as v0.

Scope
1. Design note with the `data-model` skill (access pattern, structure, alternative, signal to revisit).
2. Add `version` to each format; reader dispatches by version through a table.
3. A fixture of real output per format under `testdata/`.
4. Reading tests table-driven per version, including v0 and an unknown future version (error, no panic).
5. Update `architecture.md` persistence section.

Acceptance: every format writes a version and reads v0; `make check` green.

## Model
Each persisted format gets an integer version field; missing reads as v0; reader dispatch is a table keyed by version. Detail in the design note.
