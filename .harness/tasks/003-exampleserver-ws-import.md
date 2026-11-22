status: todo
kind: refactor
commit: refactor(exampleserver): stop importing the websocket protocol adapter
scope: internal/exampleserver/**, internal/request/**
not_before: 2026-01-01
depends_on:

# Example server does not import a protocol adapter

Goal: `internal/exampleserver` imports `request/adapter/proto/ws` (no layer rule covers it). Either
the server brings its own small WebSocket echo, or the shared piece moves into a place both may
import. Write the chosen rule into `internal/CLAUDE.md`.

## Model
Model: unchanged

## Acceptance
- Example server tests stay green and none changes.
- `exampleserver` imports no `adapter` package.
- The repo check command passes.
