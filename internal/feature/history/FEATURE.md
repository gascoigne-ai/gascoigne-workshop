# Feature: history

What the workshop remembers about sent requests: the responses of each request and the log of sent requests.

## Layout
- `internal/feature/history`: the model (`Entry`, `Sent`, `SentLimit`) and the ports (`History`, `SentStore`).
- `adapter`: the disk store (`Dir`, `SentLog`), package `history`. Reached from the app as `historyadapter`.
- `ui/sentlog`: the sidebar's `Sent` list, behind its `Host` seam (`Host.SentStore`); the `App` implements the host.
- Imports: the kernel and the base feature (`feature/request`; kernel: `kernel/fileversion`, `kernel/userdir`, `kernel/severity`) plus the toolkit for the view.

## Behavior
- **History in a tab**: the last 10 successful responses a new one replaced, in the `History` menu.
- **History across restarts**: the last 10 successful responses of a request in `history/<hash of its key>.json` next to the state file, mode 0600 because responses can hold tokens. The key is the collection folder and the request path; the file holds the key, so a file of another request is never shown. A request of the first open collection with no file under its key reads the bare path's file, written when one collection was open at a time.
- **Sent log** (`history/sent.jsonl`): one line per finished send through `dispatch` of a request in the tree (not WebSocket or MCP sessions). The sidebar's `Sent` list opens a row's request in its collection.
- With no state path (tests, `-state ""`) no history or sent log is written.

## Limits
| What | Limit | On overflow |
|---|---|---|
| Responses per request | 10 (`Limit`) | oldest dropped |
| Body kept on disk | 256 KiB (`MaxBody`) | shown once, not written |
| Sent log | 500 (`SentLimit`); the file is cut back when it holds twice that | oldest dropped |

## Files
Formats and versions are in `.harness/architecture/persistence.md` (`history/<hash>.json`, `history/sent.jsonl`); fixtures in `adapter/testdata`. Both are version 1 and unchanged.

## Known gaps
- The model is the one root file `model.go`; it moved here from the old `internal/response` because only this feature used it.
- The tab's in-memory history list lives in `internal/app` (`statesync.go`), not in this feature.
- `ui/sentlog` imports the toolkit (`uitoolkit/...`), as every `ui` does.
