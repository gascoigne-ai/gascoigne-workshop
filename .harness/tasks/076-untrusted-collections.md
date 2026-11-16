# 076 Untrusted collections: commands, files and secrets

Decided by the user on 2026-11-16 (input audit): include the stdio trust prompt, restricted file reads and secrets on export. Not included: `.workshop.*` read from the working directory, shortcuts while the settings dialog is open.

**Goal:** opening a collection someone else wrote cannot run a command, read an arbitrary file or leak a secret without the user knowing.

**Objectives**
1. An MCP `stdio:<command>` server from an imported collection or client config does not start before the user agreed to that command.
2. A multipart `@path`, a `BodyFile` and a `.proto` import read only inside the collection folder, or a file the user chose.
3. Postman export and every other export leave out the values of `Secret` variables and credentials.

Scope (ask first on every point marked ask; one commit per item, test first)
1. Trust prompt: where it shows (dialog like the confirm dialog), what is remembered (per command, in settings or the state file) and how it is forgotten. Ask the user. Tests with a fake host.
2. File reads: `rest.go:136-196`, `postman.go:182` (`"@"+src`), `protosource.go` (`LoadedPath`, `StoredPath`), `grpcfiles.go:119-135`. A path outside the collection folder asks, or is refused with a clear error; ask which. Absolute paths the user picked in the file browser stay allowed.
3. Secrets: `postman.go:281-283, 359-367` export, and `store.go:151,187` collection and folder variables flagged `Secret` still write `Value` to JSON (only environments divert to `.secrets.json`): divert them too, test with a round trip.
4. Captured values re-expanded: a value captured with `set x = json x` can hold `{{secret}}`, which a later variable substitutes into the next request (`vars.go`, `assert.go:75`). Decide with the user: expand once, or never expand captured values. Ask.
5. `decisions.md` records the answers; `architecture.md` describes the prompt and the rules.

Acceptance: each objective has a test that failed before; `make check` green; docs updated; the file is deleted in the last commit.
