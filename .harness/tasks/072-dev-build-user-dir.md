# 072 Dev build uses its own user directory

Asked for by the user while closing task 071: the release build opens from the user config; the dev build opens from the user config too, but with a `-dev` suffix.

**Goal:** the default (dev) build behaves like a real install, remembers its collections, and never touches the data of the release build.

**Objectives**
1. The release build reads and writes settings, remembered collections and state in the user config directory as today.
2. The default (dev) build uses the same layout in a directory with a `-dev` suffix, so the two never share data.
3. The dev build no longer ignores the remembered collections (today `ignoreRememberedWorkspace` makes a run from an IDE always show the Example Server): it opens what it remembered, and the Example Server is the first-start sample.
4. Client configs (`.workshop.toml`) still open as one collection with each configured collection a folder inside it (unchanged, decided by the user on 2026-11-15).

Scope
1. Find where the user directory name is derived (settings, state, history, `collections/`, logs, `DefaultWorkspace`) and give it one source of truth with the suffix per build; tests for both builds without a display.
2. Dev build remembers its collections; the example-server behavior of the golden scenes and `-workspace` stays (scenes use empty settings: goldens must not change).
3. Docs: architecture.md (the paragraph about the default build and `ignoreRememberedWorkspace`), README, decisions.md, `.harness/release.md` if it mentions the dirs.

Acceptance: both builds tested for their directory; no golden changes; `make check` green; docs updated.

**Decided (user, 2026-11-15)**
- Separate user directory with a `-dev` suffix for the dev build; client-config collections stay one collection.

**Open (ask the user first)**
- Is an existing dev data directory (the unsuffixed one a dev build wrote earlier) migrated, left alone, or ignored?
