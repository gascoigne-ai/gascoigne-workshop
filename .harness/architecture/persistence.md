## Persistence

Every file the workshop writes carries `"version": <int>` as the first key of its object (on every
line of `sent.jsonl`); a file without one is version 0, the layout written before versions existed
(decision 22). The version belongs to the file, not to the domain: each adapter has a wire type that
embeds or mirrors the model type (`statefile.stateFile`, `store.folderMeta`, `history.sentLine`...),
so `state.State`, `port.Entry`, `port.Sent`, `settings.Settings` and `model.Request` have no version
field. `fileversion.Read` (`internal/settings/model/fileversion`) probes the version and calls the
reader of that version from a table, a slice indexed by version (`readers[v]`); the newest version
is `len(readers)-1` and is what the writer writes. Versions 0 and 1 share their layout, so both rows
are the same decoder (`fileversion.Plain`), except `.secrets.json`; the old-format fallbacks of
decision 17 stay in that decoder.

| File | Owner | Version | Version 0 | Newer version |
|---|---|---|---|---|
| `state.json` | `workspace/adapter/statefile` | 1 | the same layout without the key | an error toast at start; the session starts with the empty state and never writes the file (`App.stateAccess`) |
| `settings.json` | `settings/model/settings` (`Decode`, `Minimal`, `Validate`), written by `settingsfile` | 1, always written, so the defaults are `{"version": 1}` | the same layout (sectioned, the older flat form, `workspace`/`environment`) | an error toast at start; defaults for the session and no write of the file, from the form, the raw editor or a remembered collection (`App.settingsAccess`, `readOnlySettings`). In the raw editor a newer version is a problem on the `version` key; `version` is not in the schema, so the form, reset and completion never show it, and inside a section it "belongs at the top level" |
| `folder.json`, `*.req.json`, `environments/*.env.json` | `collection/adapter/store` | 1 | the same layout without the key | the collection is not opened: the error toast names the file and both versions, also at start for a remembered one, which stays remembered (`App.toastNewer`); nothing is written |
| `.secrets.json` | `collection/adapter/store` | 1: `{"version": 1, "files": {path: secrets}}` | the bare map of path to secrets | as the other collection files |
| `history/<hash>.json` | `response/adapter/history` | 1; the keys `Key`, `Entries` and those of an entry (`entryFile`) are spelled out | the same layout without the key | an error toast when the request opens, no responses shown; the next response overwrites the file. The version is checked before the key |
| `history/sent.jsonl` | `response/adapter/history` | 1 on every line, since a header line would be lost when the log is trimmed | a line without the key | the line is skipped; trimming does not parse lines, so it stays until it is among the oldest |

Structure: a slice of readers indexed by version, because a version is a small dense integer read
once per file and every version up to the newest has a reader; a `map[int]reader` adds nothing over
the slice and a `switch` per format is the sprawl the tables replace. Each row is a full decoder of
its version rather than a chain of upgrades from version 0, because no layout has changed yet.
Revisit when a layout changes (version 2): then the old row decodes its own wire type and converts
it, and a chain of upgrade steps is worth it once there are three or more layouts. The version is
per file, not one for a whole collection in its root `folder.json`, so a file read alone (a request
file copied between collections, a history file) still says what it is. Not versioned:
`.workshop.{yaml,json,toml}` client configs (written by people), Postman export (Postman's format),
and the in-memory keys `stateKey` and `tree.Key`.
