status: in progress (save, load, autosave done; close prompt and import pending)

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

## Progress
- Done: `internal/store` (format in decisions 9, round-trip tests with unicode, duplicate names, empty folders, secrets kept out of request files), `-workspace` flag, remembered workspace, `Ctrl+S`, debounced autosave.
- Left: unsaved-changes prompt on close (autosave makes it rare), cURL import (paste), an open/new workspace dialog (only the flag and the default folder exist), OpenAPI import.
