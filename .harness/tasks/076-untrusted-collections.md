# 076 Untrusted collections: commands, files and secrets

Decided by the user on 2026-11-16 (input audit): include the stdio trust prompt, restricted file reads and secrets on export. Not included: `.workshop.*` read from the working directory, shortcuts while the settings dialog is open.

**Goal:** opening a collection someone else wrote cannot run a command, read an arbitrary file or leak a secret without the user knowing.

**Objectives**
1. An MCP `stdio:<command>` server from an imported collection or client config does not start before the user agreed to that command.
2. A multipart `@path`, a `BodyFile` and a `.proto` import read only inside the collection folder, or a file the user chose.
3. Postman export and every other export leave out the values of `Secret` variables and credentials.

Done (answers in decision 21): the stdio command and outside-file prompt with remembered answers; `.proto` imports cannot leave their import paths; secret collection and folder variables kept out of `folder.json`; Postman export leaves out literal credentials and secret values; captured values are never expanded again.

Left alone on purpose: `.proto` files given by path are not behind the prompt (they are compiled, and only compile errors can show their content); `.workshop.*` in the working directory (decided against); a user who edits a captured variable's value in the Vars pane keeps it `Literal`, so `{{...}}` they type there stays text until the row is recreated (rare; revisit if it surprises someone).

Not checked by a screenshot: the prompt is the existing confirm dialog with new text, so no new look.

Scope: all items done. The user decides whether this file is deleted (rules: ask before deleting).

Acceptance: met; `make check` green.
