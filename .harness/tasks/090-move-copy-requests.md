status: blocked
kind: feature
commit: feat(collection): move and copy requests between collections
scope: internal/*,uitoolkit/*,.harness/*
depends_on: 089-welcome-golden-scene

# 090 Move and copy requests between collections

Decision 17 calls it "a later task". Unfamiliar feature: per `common/rules.md`, search open-source clients first and propose 3 options (open source, creative, best); wait for the user's pick.

Scope
1. Research and 3 options; user picks (recorded here).
2. Design note (`data-model`), then test-first implementation per the pick.
3. Docs.

Acceptance: set after the pick; `make check` green.

## Model
To be set after the user picks an option.

Blocked: needs the user's pick among the 3 options.
