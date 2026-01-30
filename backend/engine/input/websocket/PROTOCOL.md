# WebSocket Input Protocol

This document defines the client/server message contract for the WebSocket
InputProvider. The protocol is request/response with correlation via
`request_id`.

## Envelope
All messages are JSON objects:

```json
{
  "type": "request_action",
  "request_id": "req-1",
  "payload": { }
}
```

Rules:
- Every server request includes a `request_id`.
- Clients must echo the same `request_id` in their response.
- Unknown or late `request_id` values are treated as protocol errors.

## Handshake
The first client message must be:

```json
{"type":"hello","payload":{"player_index":1}}
```

## Server → Client requests

### request_action
```json
{"type":"request_action","request_id":"req-1","payload":{"actor_index":0}}
```

### request_step
```json
{"type":"request_step","request_id":"req-2","payload":{"step":{...}}}
```

### challenge_window
```json
{"type":"challenge_window","request_id":"req-3","payload":{
  "actor_index":0,
  "action_id":"investigate",
  "claimed_role":"Policewoman",
  "kind":"main",
  "target_index":1
}}
```

### counter_window
```json
{"type":"counter_window","request_id":"req-4","payload":{
  "actor_index":0,
  "action_id":"steal",
  "allowed_actions":["block_steal","tax"],
  "target_index":1
}}
```

## Client → Server responses

### action
```json
{"type":"action","request_id":"req-1","payload":{
  "id":"steal",
  "source_index":0,
  "target_index":1
}}
```

### challenge
```json
{"type":"challenge","request_id":"req-3","payload":{
  "challenger_index":1,
  "actor_discard_index":0,
  "pass":false
}}
```

Passing the challenge:
```json
{"type":"challenge","request_id":"req-3","payload":{
  "challenger_index":1,
  "pass":true
}}
```

### counter
```json
{"type":"counter","request_id":"req-4","payload":{
  "id":"block_steal",
  "source_index":1,
  "pass":false
}}
```

Passing the counter window:
```json
{"type":"counter","request_id":"req-4","payload":{
  "source_index":1,
  "pass":true
}}
```

### step_result
For card selection:
```json
{"type":"step_result","request_id":"req-2","payload":[0,1]}
```

For other step kinds, payload may be any JSON value.

## Validation rules
- `id` and `source_index` are required on action/counter payloads.
- `guess` requires `target_index` (used for Accuse).
- `challenger_index` is required on challenge payloads.
- `pass` is allowed on counter/challenge payloads; when true, other fields are ignored.
- `request_id` must match a pending request; unknown ids are errors.

## Error handling
Protocol violations or disconnects terminate the current turn and are surfaced
as provider errors to the orchestrator.

## Quick start (server)
Minimal HTTP server wiring:

```go
package main

import (
	"ElMakina/backend/engine/input/websocket"
)

func main() {
	server := websocket.NewServer(":8080", nil)
	_ = server.ListenAndServe()
}
```
