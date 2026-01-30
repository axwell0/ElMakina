# server/store

This module provides lobby persistence implementations.
It currently ships with a file-backed store for local use.

## Responsibilities

- Implement `server.LobbyStore` for snapshot load/save.
- Keep snapshot writes atomic and durable.

## Key Files

- `file.go`: JSON snapshot storage with atomic rename.

## Invariants and Sharp Edges

- Snapshots are full-state dumps; partial updates are not supported.
- `Load` returns `(nil, nil)` when the snapshot file does not exist.

## Extension Points

- Add Redis or SQL-backed implementations for production deployments.
