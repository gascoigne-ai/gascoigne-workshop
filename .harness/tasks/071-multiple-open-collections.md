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
- Item 1 is done (6d9c086): `collections.Set` with Entry, Open, Close, Move, Located, ChooseEnv; identity is the folder path; opening an already open folder changes nothing (the caller focuses it); two entries never share a folder; new ones are appended.
- Decided 2026-11-15: the header environment chip shows and sets the environment of the collection that owns the active tab (with no tab, the selected tree collection); closing the last collection leaves an empty tree (nothing created on disk); the first save of an unsaved collection goes to its own new folder under `collections/`, as New does.
- Item 2 is done (9045298, 61823bf, e667eb1): settings key `general.collections`, state tabs name their collection, `App` holds a `collections.Set`; migration reads the old `workspace`/`environment` keys when `collections` is absent. New collection and Ctrl+O still replace the set until item 3.
- Decided (user, 2026-11-15): root rows get a bold name, nothing else; `close-collection` defaults to Ctrl+Shift+W. Done (c736c52), goldens redrawn.
- Item 4 is verified (60d35fc): `testdata/one-collection` holds files the one-collection version wrote, opened as a set of one and rewritten in the new form. Found while capturing, not changed: a folder collapsed at exit is open again at the next start, in that version as in this one (Limits in architecture.md).
- Item 3's goldens (e6eb38c): `tree-two-collections` (`-add-collection`) and `tree-empty` (`-close-all`). Item 5: decision 17 and the docs.
- Open for the user: client configs still open as one collection with each configured collection as a folder; whether each should become its own root is not decided.
- Conventional defaults (2026-11-15), done at the start of item 3: history and the sent log are keyed by collection folder plus request path, with the bare path as a fallback for the first or an unsaved collection so old history stays readable; an environment pick updates only the matching entries of the remembered list when the set did not come from it (debug build, `-workspace X`), instead of replacing the list.
