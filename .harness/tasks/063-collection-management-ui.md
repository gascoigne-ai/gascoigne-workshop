# 063 Create, rename, save and delete collections from the UI

Asked for by the user. Today a collection is a directory of JSON files (decision 7) edited outside the app.

**Goal:** manage collections without leaving the app.

**Objectives**
1. New collection, rename, save and delete are each reachable from the sidebar (menu entry; shortcut where one fits).
2. Delete asks for confirmation; only files the store owns are removed (decision 7).
3. Failures show as a toast, never a silent loss.
4. The rules sit in a pure controller, tested without a window.

Scope
1. (decided, user's answers) A collection is a folder on disk, one open at a time (`openWorkspaceAt`). **New**: prompt for a name, create a folder in the default workspace directory, open it. **Rename**: change the name (`folder.json`) and the slug directory. **Save**: write the open collection now (flush; autosave stays). **Delete**: confirm, remove the folder's owned files (decision 7), then open another collection or a fresh one. The actions are entries of the sidebar's Collections chip menu, with shortcuts for New and Save; prompts and confirms reuse `prompt` and `confirm`.
2. `collection/controller/tree`: create, rename, delete rules, tested.
3. `collection/port.Store` gains the operations; `store.Files` implements them; tested on a temp dir.
4. UI: sidebar menu entries, prompts and confirm through the existing `prompt` and `confirm`.
5. Docs: architecture.md, a decision, a golden scene of the menu.

Acceptance: an App test runs create, rename, save, reload, delete; `make check` green; `make shot` of the menu looked at.
