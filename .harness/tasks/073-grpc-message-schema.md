# 073 gRPC message schema in the editor

Follow-up of task 070, decided by the user on 2026-11-15: 070 closed with services and methods shown, and the message schema left for this task.

**Goal:** the user can see the request message of the chosen gRPC method and start from a valid body.

**Objectives**
1. The request message schema of the chosen method (fields, types, nesting, repeated, enums, oneofs) is shown, from reflection and from proto files alike.
2. The Body can be filled with a JSON skeleton of the request message (default values), on the user's action.
3. A tab remembered on a pane after Body reopens the right pane now that Proto sits after Body (known 070 gap).

Scope (to be written with the user before work starts; ask first)
1. Decide where the schema shows (Body pane side panel, a tooltip, a tab) and what the skeleton action looks like.
2. Implement on the descriptors both sources already give; tests with real `.proto` fixtures.
3. Fix the remembered pane index (state file) with a test.
4. Docs, golden scene.

Acceptance: to be written with the scope.
