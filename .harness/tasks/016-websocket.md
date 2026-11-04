status: in progress (left: a JSON-formatting composer)

# WebSocket client

## Goal
WebSocket is a stub that answers "not implemented yet".

## Scope
- Connect / disconnect, send text (and JSON-formatted) messages, live message log with direction, time and size, ping/pong, close codes.
- Headers and subprotocols; reconnect option.
- UI: a message-log pane in place of the response, and a composer in place of the body.
- `proto.Protocol` grows a streaming shape (event channel); REST keeps its one-shot result. Record the interface change in `decisions.md`.

## Acceptance
- Tests against an `httptest` WebSocket server (echo, server close, slow consumer).
- `make check` passes.

## Progress
- Done: `internal/proto/ws` (hand-written RFC 6455 client and server handshake), `proto.WebSocket` (`Connect` session with an event channel, one-shot `Send`), `Registry.Connect`, tests against an `httptest` server (echo, server close, slow consumer, ping/pong, fragments, size limit).
- Done: `internal/ui/wsui.go` - Send button becomes Connect / Disconnect, the response side of a socket tab is a `Messages` pane (log with time, direction arrow and text, Ping and Clear, follow-the-end scrolling) with a composer under it; tests against an echo server.
- Done: per-tab `Auto-reconnect` chip (not saved with the request): a drop the user did not ask for is retried after `retryDelay` (2s) until Disconnect.
- Left: subprotocols and a JSON-formatted composer, binary send.
- Done later: subprotocols work through a `Sec-WebSocket-Protocol` header in the Headers pane (the handshake already offered them); the composer sends binary with `hex: de ad be ef` (`proto.BinarySender`).
