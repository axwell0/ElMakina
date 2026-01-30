# WebSocket Contract (ElMakina)

This document describes the public WebSocket contract used by the frontend. The
canonical JSON schema lives at `docs/ws_schema/envelope.json`.

## Transport

- Endpoint: `GET /ws` (WebSocket upgrade)
- Encoding: JSON
- Envelope: every message is a JSON object with `type`, optional `request_id`, and
  optional `payload`.

```json
{
  "type": "<message_type>",
  "request_id": "req-123",
  "payload": {"...": "..."}
}
```

### request_id

- Required for messages that expect a response (prompts and their replies).
- Optional for broadcast messages and state updates.

## Message Types

### Handshake

- `hello` (client -> server): payload `HelloPayload`
- `hello_ack` (server -> client): payload `HelloAckPayload`
- `hello_error` (server -> client): payload `HelloErrorPayload`

### Lobby

Client requests:
- `lobby_create`
- `lobby_list`
- `lobby_join` (payload `LobbyJoinPayload`)
- `lobby_start` (payload `LobbyStartPayload`)

Server responses/events:
- `lobby_created` (payload `LobbyCreatedPayload`)
- `lobby_joined` (payload `LobbyJoinPayload`)
- `lobby_list_result` (payload `LobbyListPayload`)
- `lobby_started` (payload `LobbyStartedPayload`)
- `lobby_state` (payload `LobbyStatePayload`)
- `lobby_error` (payload `{ "error": "..." }`)

`LobbyStartedPayload` now includes `match_id` for replay and analytics.

### Game Prompts

- `request_action` (payload `RequestActionPayload`)
- `challenge_window` (payload `ChallengeWindowPayload`)
- `counter_window` (payload `CounterWindowPayload`)
- `request_step` (payload `RequestStepPayload`)

### Game Responses

- `action` (payload `ActionPayload`)
- `challenge` (payload `ChallengePayload`)
- `counter` (payload `ActionPayload`)
- `step_result` (payload: arbitrary JSON, often `int[]`)

### Game Events

- `game_log` (payload `GameLogPayload`)
- `chat_message` (payload `ChatMessagePayload`)
- `game_state` (payload `GameStatePayload`)
- `game_config` (payload `GameConfigPayload`)
- `game_over` (payload `GameOverPayload`)
- `prompt_closed` (payload `PromptClosedPayload`)
- `investigate_result` (payload `InvestigateResultPayload`)
- `hand_state` (payload `HandStatePayload`)
- `player_eliminated` (payload `PlayerEliminatedPayload`)
- `turn_timer` (payload `TurnTimerPayload`)

## Schema

Use the JSON schema in `docs/ws_schema/envelope.json` for validation. It defines:

- The envelope shape.
- All message types and payloads.
- Field optionality and array structures.

## Examples

```json
{"type":"hello","payload":{"nickname":"Ada"}}
```

```json
{"type":"lobby_create","request_id":"req-1"}
```

```json
{"type":"request_action","request_id":"req-42","payload":{"actor_index":1,"allowed_actions":["coup","income"],"prompt":"choose your action"}}
```
