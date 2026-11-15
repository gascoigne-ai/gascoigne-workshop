# 071 Multiple open collections

Known limitation: one collection is open at a time.

**Goal:** several collections can be open together.

**Objectives**
1. The collections tree shows every open collection as a root.
2. A collection can be opened, closed and reordered without closing the others.
3. Open request tabs from different collections coexist; save, dirty state and rename act on the right collection.
4. The set of open collections is restored at the next start.
5. Variables and environments stay per collection, with no leakage across.

Scope
1. Model and controller: a set of open collections instead of one (`internal/collection`), with tests.
2. Workspace composition and persistence of the open set.
3. Tree UI and menu or shortcut changes; `make shot` looked at.
4. Migration of the saved state with one collection.
5. Docs: architecture.md, decisions.md, user docs.

Acceptance: two collections open, edited and saved independently; restart restores both; `make check` green.

**Decided (user, 2026-11-15)**
- One active environment per collection; variables never leak across collections.
- No limit on open collections.
- Copying or dragging requests across collections is out of scope (later task).
