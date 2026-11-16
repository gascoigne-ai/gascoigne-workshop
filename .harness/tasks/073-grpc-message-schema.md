# 073 gRPC message schema in the editor

Follow-up of task 070, decided by the user on 2026-11-15: 070 closed with services and methods shown, and the message schema left for this task.

**Goal:** the user can see the request message of the chosen gRPC method and start from a valid body.

**Objectives**
1. The request message schema of the chosen method (fields, types, nesting, repeated, enums, oneofs) is shown, from reflection and from proto files alike.
2. The Body can be filled with a JSON skeleton of the request message (default values), on the user's action.
3. A tab remembered on a pane after Body reopens the right pane now that Proto sits after Body (known 070 gap).

Decided by the user on 2026-11-16: the schema shows in a new read-only **Schema** pane after Proto in the gRPC request panes (a tree of fields: type, repeated, map, enum values, oneof, nesting), with a **Fill body** button that writes the JSON skeleton of the request message into Body.

Scope
1. `model` schema type and JSON skeleton (default values; nested messages expanded once, a recursive message cut off); adapter turns a method's input descriptor into it, for reflection and `.proto` files; port `GRPCSchema`. Tests with the real `health.proto`.
2. Schema pane and Fill button (`editor` message for the body), golden scene.
3. Remembered pane by label, so a state file written before Proto existed still opens the right pane (with a test).
4. Docs: `architecture.md`, `packagemap.md`, template tables.

Acceptance: every Objective met; `make check` green; the pane looked at in a `make shot` PNG.
