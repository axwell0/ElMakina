# server/replay

This module records game events and serves replays over HTTP.
It is wired into `server/ws` via a `Recorder` interface.

## Responsibilities

- Persist match metadata, events, and snapshots.
- Serve replay payloads for client playback.
- Provide async recording to avoid blocking gameplay.

## Key Files

- `models.go`: database models and replay payload shapes.
- `store.go`: GORM-backed storage and query APIs.
- `async.go`: buffered, batched recorder implementation.
- `http.go`: `/replay/{match_id}` HTTP handler.

## Invariants and Sharp Edges

- Match IDs must be stable across a session.
- Async recorder may drop events if configured to do so under load.
- Replay visibility distinguishes public vs. private events.

## Extension Points

- Add new event types with explicit visibility and schema.
- Swap storage backends by implementing the `Recorder` interface.
