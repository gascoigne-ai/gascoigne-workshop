status: todo

# Collections on disk

## Goal
The sample collection is built in and nothing is saved.

## Scope
- File format decision recorded in `decisions.md` (one file per request in a folder tree, git-friendly, stable key order, no secrets inline).
- Open / create a workspace directory, autosave on change (debounced) or Ctrl+S, atomic writes, reload on start, remember the last workspace in settings.
- Dirty state and "unsaved changes" prompt on close.
- Import: cURL command (paste), and an OpenAPI or other collection format as a follow-up task.

## Acceptance
- Round-trip tests (save, load, deep-equal) including unicode names, duplicate names and empty folders; real-world cURL fixtures.
- `make check` passes.
