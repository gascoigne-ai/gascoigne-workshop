# Feature: environments

Named sets of variables a collection offers, and which one is active. The variables and environments are part of the request model (`request.Environment`, `Collection.VarsFor` in `feature/request`); this feature owns the selection and the header chip's menu.

## Layout
- `internal/feature/environments`: package `envs` (the short name callers use): `Selection` (the active environment of a collection, by index, `None()` for none) in `envs.go` and the chip's menu rows and `Action`s (`Select`, `New`, `Rename`, `Delete`) in `menu.go`. There is no `ui/` or `adapter/`: the chip is drawn by the App and environments are saved by the collection store.
- Imports: `feature/request` only.

## Behavior
- Resolution order (`Collection.VarsFor`): collection < active environment < folders, outermost
  first, nearest wins. A request expands only with its own collection's variables. Every send,
  cURL copy, connect, URL tooltip and highlight uses `App.varsFor`.
- Each open collection has its own active environment, remembered by name in its
  `settings.collections` entry. The header chip and the Vars pane act on the focused collection.
- The Vars pane edits the active environment, else the collection variables, unless a `varsTarget`
  (`env_varstarget.go`) points it at a folder or at the collection's own variables.

## Files
`environments/*.env.json` in a collection folder (written by `feature/collections/adapter/collectionstore`); format and version in `.harness/architecture/persistence.md`.

## Known gaps
- The active environment is remembered by name inside the collections feature's `settings.collections` entry (`collections.Set`), and the store writes the environment files, so the two features meet only through `feature/request`; the App wires them.
- The package name is `envs`, the directory `environments`.
