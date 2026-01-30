# cmd/server

This is the HTTP/WebSocket server entrypoint for ElMakina.
It wires the lobby manager, game engine, and replay recorder.

## Responsibilities

- Load configuration from `.env` and environment variables.
- Build the lobby server and attach replay recording.
- Host the WebSocket endpoint and optional replay HTTP handlers.
- Handle graceful shutdown on SIGINT/SIGTERM.

## Environment Variables

- `ELMAKINA_HTTP_ADDR` (default `:8080`)
- `ELMAKINA_WS_PATH` (default `/ws`)
- `ELMAKINA_GRACE` (default `60s`)
- `ELMAKINA_CHALLENGE_TIMEOUT` (default `15s`)
- `ELMAKINA_COUNTER_TIMEOUT` (default `15s`)
- `ELMAKINA_TURN_TIMEOUT` (default `20s`)
- `ELMAKINA_POSTGRES_DSN` (required for replay storage)
- `ELMAKINA_REPLAY_AUTOMIGRATE` (default `false`)
- `ELMAKINA_LOBBY_STORE_PATH` (optional file-based lobby persistence)

## CORS

Allowed origins are currently restricted to `http://localhost:3000` and
`http://127.0.0.1:3000`. Adjust `withCORS` in `cmd/server/main.go` if you need
additional origins in development.

## Replay Endpoint

When replay recording is enabled, the server mounts handlers under `/replay/`.
