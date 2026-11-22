status: todo
kind: refactor
commit: refactor(ui): the url bar owns its field, keys and sends
scope: internal/**, cmd/**, e2e/**, Makefile, .harness/**
not_before: 2026-01-01
depends_on: 032-tabstrip-component

# ui/editor/urlbar and the sends store

Goal: `request_urlbar`, `protocol_wsbar` and `request_send` move to `ui/editor/urlbar` and a new store
`feature/send/sends` over `outbox` and `bus`: send numbering, in-flight sends per tab, cancel, and `apply` of a
result to the tab. The URL field claims plain keys and Undo/Redo while focused (the rest of `textFocused`);
`commands.go` offers `send` and `focus-url`. `App.send`/`dispatch` call the store. Follow the component recipe of
`.harness/reorg.md`.

## Model
New `sends.Sends`: owner of in-flight sends keyed by tab; invariant: a late result of a cancelled send is
dropped by its number (as today); a map from tab to send state (looked up on every bus message, tens of
entries). Rejected: send state in `reqTab` (it is domain, not widget state). Goroutines never touch it: results
arrive on the bus and are applied on the UI goroutine.

## Acceptance
- Test first: `sends` tests (numbering, late result dropped, cancel); urlbar dump test; the existing
  send-flow tests move and keep their assertions.
- Common acceptance of `.harness/reorg.md`.
