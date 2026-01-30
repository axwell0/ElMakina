# engine/input

This module provides `InputProvider` implementations for the engine.
It translates transport-specific inputs into engine decisions.

## Responsibilities

- Implement the `engine.InputProvider` interface.
- Convert transport payloads into `models.PlayerAction` and step results.
- Enforce prompt/response ordering at the transport boundary.

## Implementations

- `stdio.go`: line-based JSON provider for manual or CLI testing.
- `websocket/`: WebSocket provider and server wiring for remote clients.

## Invariants and Sharp Edges

- Providers are called sequentially from a single goroutine.
- Challenge and counter responses stream until context cancel or an explicit end.

## Extension Points

- Add new providers for HTTP, bots, or scripted simulations.
