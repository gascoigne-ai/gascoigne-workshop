## Persistence

Every file the workshop writes has `"version": <int>` as the first key of its object
(of every line in `sent.jsonl`); a file without one is version 0. The version belongs to the file,
not the domain: each adapter has a wire type that embeds or mirrors the model type, so model types
have no version field. `fileversion.Read` picks the reader of the version from a table (its doc says
why a slice); the newest is what the writer writes. Versions 0 and 1 share their layout and decoder,
except `.secrets.json`; the fallbacks for files from before several collections could be open stay in that decoder. The version is
per file, not per collection, so a file read alone (a request file copied between collections, a
history file) still says what it is. A whole-file write goes through `userdir.WriteFileAtomic`
(temp file, then rename; its doc says why it is in `settings/userdir`); the sent log appends.
Accepted (the user's answer): the first save after upgrading rewrites every collection file once, a
one-time diff under version control; `CHANGELOG.md` says so.

| File | Owner | Version | Version 0 | Newer version |
|---|---|---|---|---|
| `state.json` | `workspace/adapter/statefile` | 1 | the same layout without the key | an error toast at start; the session starts with the empty state and never writes the file (`App.stateAccess`) |
| `settings.json` | `settings` (`Decode`, `Minimal`, `Validate`), written by `settingsfile` | 3, always written, so the defaults are `{"version": 3}` | versions 0 and 1 share a layout (sectioned, the older flat form, `workspace`/`environment`) and write the text size as `textSize` (small, medium, large), read as `fontSize` 13, 14, 16 (`upgradeTextSize`); versions 0 to 2 have one `fontSize` for every face, so the reader derives the missing `smallFontSize` (size - 2, at least 9) and `codeFontSize` (size - 1) from it (`deriveFaceSizes`); version 3 has the three sizes | an error toast at start; defaults for the session and no write of the file, from the form, the raw editor or a remembered collection (`App.settingsAccess`, `readOnlySettings`). In the raw editor a newer version is a problem on the `version` key; `version` is not in the schema, so the form, reset and completion never show it, and inside a section it "belongs at the top level" |
| `folder.json`, `*.req.json`, `environments/*.env.json` | `collection/adapter/store` | 1 | the same layout without the key | the collection is not opened: the error toast names the file and both versions, also at start for a remembered one, which stays remembered (`App.toastNewer`); nothing is written |
| `.secrets.json` | `collection/adapter/store` | 1: `{"version": 1, "files": {path: secrets}}` | the bare map of path to secrets | as the other collection files |
| `history/<hash>.json` | `response/adapter/history` | 1; the keys `Key`, `Entries` and those of an entry (`entryFile`) are spelled out | the same layout without the key | an error toast when the request opens, no responses shown; the next response overwrites the file. The version is checked before the key |
| `history/sent.jsonl` | `response/adapter/history` | 1 on every line, since a header line would be lost when the log is trimmed | a line without the key | the line is skipped; trimming does not parse lines, so it stays until it is among the oldest |

Not versioned: `.workshop.{yaml,json,toml}` client configs (written by people), the Postman export
(Postman's format), and the in-memory keys `stateKey` and `tree.Key`.
