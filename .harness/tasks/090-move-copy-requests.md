status: todo
kind: feature
commit: feat(collection): move and copy requests between collections
scope: internal/*,uitoolkit/*,.harness/*,CHANGELOG.md,cmd/workshop/scene.go,cmd/workshop/golden_scenes_test.go,cmd/workshop/testdata/golden/*

# 090 Move and copy requests between collections

Decision 17 calls it "a later task". Unfamiliar feature: per `common/rules.md`, search open-source clients first and propose 3 options (open source, creative, best); wait for the user's pick.

Scope
1. Research and 3 options; user picks (recorded here).
2. Design note (`data-model`), then test-first implementation per the pick.
3. Docs.

Acceptance: set after the pick; `make check` green.

## Model
Set by the design note (data-model skill).

## Decision (user, 2026-11-18)
Option "Best": row menu `Move to...` / `Copy to...` opens a destination picker (open collections and folders); dragging onto another collection's folder moves, Alt-drag copies; one core operation behind both. Name clashes get an automatic rename. Variables are not rewritten (a request expands with its target collection's variables); after the operation a toast lists variables the request uses that the target collection lacks. Also decided (user, 2026-11-18): Move and Copy into an unsaved collection are both allowed (no `ErrTargetUnsaved`, no "save first" refusal); a name clash renames in the same collection too (the old `ErrNameTaken` refusal on a drag goes); Alt-drag copies, the row menu is the reliable path where a window manager takes Alt-drag; scope widened to the screenshot flag and golden scenes under cmd/workshop and to CHANGELOG.md. Steps: 1 transfer controller and model helpers, 2 drag across collections, 3 picker dialog, 4 menu entries, screenshot flag and goldens, docs, delete this file.
Considered and rejected: Bruno-style drag only (no menu), cut/copy/paste in the tree.

