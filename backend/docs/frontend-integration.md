# Frontend Integration Guide

This guide explains how a frontend client connects to the ElMakina server,
joins a lobby, plays a game, and reacts to server events.

## Audience

Frontend engineers building a browser or desktop client that speaks the
ElMakina WebSocket protocol.

## Connection and Handshake

1) Open a WebSocket connection to `ws://<host>:<port><WS_PATH>`.
   - Default address: `ws://localhost:8080/ws`.
   - `WS_PATH` defaults to `/ws` and can be changed via `ELMAKINA_WS_PATH`.

2) Send `hello` as the first message:

```json
{"type":"hello","payload":{"nickname":"Ada"}}
```

3) The server responds with `hello_ack` and a reconnect token:

```json
{"type":"hello_ack","payload":{"player_id":"...","token":"..."}}
```

4) Store the token and use it to reconnect:

```json
{"type":"hello","payload":{"reconnect_token":"..."}}
```

Notes:
- You may include `avatar` in the hello payload. The server persists it for the lobby.
- If the payload is invalid, the server sends `hello_error` and closes the socket.

## Lobby Lifecycle

Client requests:
- `lobby_create`
- `lobby_list`
- `lobby_join` with `LobbyJoinPayload`
- `lobby_start` with `LobbyStartPayload`

Server responses/events:
- `lobby_created`
- `lobby_joined`
- `lobby_list_result`
- `lobby_started` (includes `match_id` for replay)
- `lobby_state`
- `lobby_error`

Key behaviors:
- Only the lobby leader can start a lobby.
- Lobby state broadcasts on join/leave and on start.

## Game Prompts and Responses

The game engine asks for input using prompt messages. Every prompt includes a
`request_id` that the client must echo back.

Prompts:
- `request_action`
- `challenge_window`
- `counter_window`
- `request_step`

Responses:
- `action`
- `challenge`
- `counter`
- `step_result`

Rules:
- Always respond with the same `request_id` you received.
- Late or unknown `request_id` values are protocol errors.
- `step_result` payloads vary by step type (often an array of card indices).

## Game Events

The server emits game events that do not require a response:
- `game_state`: authoritative snapshot for rendering.
- `game_log`: public log lines for the UI.
- `hand_state`: private hand view for the local player.
- `turn_timer`: countdowns for the active prompt.
- `prompt_closed`: indicates a prompt expired or was canceled.
- `investigate_result`: private card reveal result.
- `player_eliminated`: player is out of the game.
- `game_over`: final result and winner data.

## Timeouts and Disconnects

The server enforces time windows for turn choice, challenge, and counter prompts.
If a player disconnects:
- Their session is detached from the lobby or game.
- The server schedules a forfeit after a grace period.
- Reconnecting with the token before the grace period cancels the forfeit.

Timeouts and grace period are controlled by server config:
- `ELMAKINA_TURN_TIMEOUT`
- `ELMAKINA_CHALLENGE_TIMEOUT`
- `ELMAKINA_COUNTER_TIMEOUT`
- `ELMAKINA_GRACE`

## Error Handling

The server reports protocol errors in `*_error` messages where applicable.
For prompt flows, errors can also be surfaced as `prompt_closed` or by
terminating the socket. Treat disconnects as fatal to the in-flight prompt and
reconnect with the saved token.

## Schema and Source of Truth

- Human-readable contract: `docs/ws_contract.md`
- JSON schema: `docs/ws_schema/envelope.json`
- OpenAPI stub: `docs/openapi.yaml`

Use the JSON schema to validate payload shapes and to generate client types.

## Local Development

Defaults:
- WebSocket address: `ws://localhost:8080/ws`
- Allowed origins for CORS: `http://localhost:3000` and `http://127.0.0.1:3000`

If your dev server runs on a different origin, update the CORS list in
`cmd/server/main.go` or proxy through a compatible origin.

## Replay Endpoint (Read-Only)

If replay recording is enabled, the server exposes HTTP handlers at `/replay/`.
Use these endpoints for post-game playback or analytics ingestion.

## Build and Deploy Notes

- The server uses WebSocket only; there is no REST API beyond the WS upgrade.
- Ensure your deployment allows WebSocket upgrades on the WS path.
- Persist the reconnect token in local storage to survive refreshes.
