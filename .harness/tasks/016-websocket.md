status: in progress (protocol layer done; UI pending)

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
- Left: the message-log pane in place of the response, the composer in place of the body, connect/disconnect/ping buttons, reconnect option.
