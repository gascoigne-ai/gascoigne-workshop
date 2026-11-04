status: done (OpenAPI import is a follow-up)

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
- Done: cURL import (`proto.ParseCurl`, folder menu `Import cURL from clipboard`).
- Left: unsaved-changes prompt on close (autosave makes it rare), an open/new workspace dialog (only the flag and the default folder exist), OpenAPI import.
- Done later: `App.FlushWorkspace` saves pending edits when the window closes (`main`), so the unsaved-changes prompt is not needed.
- Done later: `Ctrl+O` opens another collection folder (pending edits saved first, tabs closed, remembered); an empty or new folder starts a collection.
- Left: OpenAPI import (its own follow-up task).
