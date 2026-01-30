# engine/input/websocket

This module implements a WebSocket-based `InputProvider` and a minimal server
adapter for the engine. It is transport-focused and mirrors the public contract
in `PROTOCOL.md`.

## Responsibilities

- Encode engine prompts into WebSocket messages with `request_id` correlation.
- Decode client replies into `models.PlayerAction` and step results.
- Enforce protocol rules (valid IDs, timely responses, and message shapes).

## Key Files

- `server.go`: HTTP upgrade handler and connection lifecycle.
- `provider_*.go`: request/response wiring and validation.
- `messages.go`: message and payload definitions.
- `PROTOCOL.md`: protocol spec used by clients and tests.

## Protocol Summary

- First client message must be `hello`.
- Every request includes `request_id`; clients echo it in the reply.
- Unknown or late `request_id` values are protocol errors.

## Invariants and Sharp Edges

- Each connection maps to one player index for a game session.
- Prompt ordering matters; do not send replies out of order.

## Tests

- `provider_test.go` and `server_test.go` validate protocol behavior.
