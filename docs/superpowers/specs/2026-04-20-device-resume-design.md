# Device Resume Discovery Design

## Goal

Allow a returning browser to discover unfinished games it previously joined and prompt the user with "Resume your game?" before reconnecting. This should work without accounts or cross-device sync and should fit the existing in-memory room/session model.

## Non-Goals

- Cross-device identity or login
- Using browser identity as authorization to control a session
- Persistent storage beyond the current process lifetime
- Automatic silent resume without user confirmation

## Current Context

The codebase already has:

- A durable per-room `ClientSession` model in `app/session`
- A per-session `resumeToken` used to authorize websocket reconnects
- In-memory room storage in `server/ws`
- A browser playground that stores the session ID and resume token locally

Today, the server knows how to reconnect a specific session when the client already has `sessionId + resumeToken`, but it does not know how to answer "does this browser have an unfinished game?" on a fresh site visit.

## Proposed Model

Introduce a second identity layer:

- `deviceId`: a stable browser-local identifier stored in local storage
- `ClientSession`: the existing per-room player identity
- `resumeToken`: the existing secret used to authorize control of a specific session

The key distinction is:

- `deviceId` is for discovery
- `resumeToken` is for authorization

This lets the server answer "which resumable sessions are associated with this browser?" without weakening session security.

## Data Model Changes

Add a `DeviceID string` field to `ClientSession`.

Behavior:

- Set when a player creates or joins a room
- Remains attached to that session for the lifetime of the session
- Is not treated as secret
- Is not sufficient to authorize websocket resume or room mutation

No new global index is required in the first version. The server can scan active rooms and sessions in memory when serving resume-discovery requests.

## API Design

### Create Room

Extend `POST /api/rooms` to accept `deviceId`.

Server behavior:

- Validate that `deviceId` is present and non-empty
- Create the leader `ClientSession`
- Store `deviceId` on that session
- Continue issuing `sessionId` and `resumeToken` as today

### Join Room

Extend `POST /api/rooms/{roomID}/join` to accept `deviceId`.

Server behavior:

- Validate that `deviceId` is present and non-empty
- Create the joining `ClientSession`
- Store `deviceId` on that session
- Continue issuing `sessionId` and `resumeToken` as today

### Resume Discovery

Add `GET /api/me/resumable-games?deviceId=...`

Response shape should be a small list of resumable session summaries, for example:

- `roomId`
- `roomCode`
- `playerName`
- `playerIndex`
- `phase`
- `canReconnect`
- `lastSeenAt`

Optional but useful:

- `joinUrl`
- `connected`
- `turnNumber`

Server behavior:

- Require `deviceId`
- Scan all active rooms
- For each room, inspect its sessions
- Return sessions whose `DeviceID` matches and whose room/session state is still resumable

Resumable in v1 should mean:

- Room still exists
- Session still exists
- Room is not deleted

The server may return both connected and disconnected sessions. The UI can decide how to label them. If needed, the server can later narrow this to disconnected sessions only.

## Client Flow

### Browser Identity

On app startup:

- Read `deviceId` from local storage
- If missing, generate a random ID and store it

This `deviceId` should be long and unguessable enough for uniqueness, but it is not a security token.

### Room Create / Join

When creating or joining a room:

- Include `deviceId` in the request body
- Store returned `sessionId` and `resumeToken` as today

### Fresh Site Visit

When the app loads:

- Read `deviceId` from local storage
- Call `GET /api/me/resumable-games?deviceId=...`
- If one or more resumable games are returned, show a "Resume your game?" prompt

The prompt should be explicit, not automatic. It should show enough context for the user to recognize the game they left, such as room code and player name.

### Resume Action

When the user chooses to resume:

- Use the existing stored `sessionId` and `resumeToken` for websocket reconnect if still present
- If the UI has lost `sessionId` or `resumeToken`, the discovery response alone is not enough to reclaim the session

For v1, the resume prompt is primarily a discovery and convenience feature for the same browser that still has its local session state.

## Security and Trust Boundaries

This design intentionally keeps discovery separate from authority.

- `deviceId` says "this browser likely participated in these sessions"
- `resumeToken` says "this client is allowed to control this session"

Rules:

- Never accept `deviceId` alone as proof of session ownership
- Never allow room mutation or websocket resume using only `deviceId`
- Continue verifying `resumeToken` during websocket handshake and protected room actions

This preserves the current security model while improving UX.

## Server Implementation Notes

Recommended implementation steps:

1. Extend `ClientSession` with `DeviceID`
2. Extend create/join HTTP request types to include `deviceId`
3. Store `deviceId` during session creation
4. Add a resumable-games response type in `server/httpapi`
5. Add a `HandleListResumableGames` endpoint in `server/ws` or the HTTP server package
6. Scan `s.rooms` and matching sessions to build the response
7. Wire the route in `server/http`

No server-side persistence or indexing is needed initially. If lookup cost becomes important later, add an in-memory index such as `map[string][]sessionRef` maintained alongside room lifecycle changes.

## Error Handling

Expected server responses:

- `400` when `deviceId` is missing or malformed
- `200` with an empty list when no resumable games exist
- `200` with matching sessions when one or more are found

Expected client behavior:

- If the lookup fails, continue loading the normal home screen
- If the lookup returns no matches, do nothing special
- If the lookup returns matches, show a resume prompt

This feature should fail soft. Discovery problems must not block the base app.

## Testing

Add tests for:

- `ClientSession` stores and exposes `DeviceID`
- Create room stores `deviceId` on the leader session
- Join room stores `deviceId` on the joining session
- Resume-discovery endpoint returns sessions only for the requested device
- Resume-discovery endpoint ignores sessions from other devices
- Resume-discovery endpoint returns empty when nothing matches
- Websocket reconnect still requires `resumeToken` even when `deviceId` matches

Frontend tests should cover:

- Generating and persisting `deviceId`
- Calling the resumable-games endpoint on load
- Rendering the resume prompt when matches exist
- Not auto-resuming without user action

## Trade-Offs

Why this design:

- Minimal change to the current session model
- Preserves existing reconnect authorization
- Uses the existing in-memory architecture cleanly
- Easy to evolve later into a stronger identity system

Known limitation:

- The feature only works for the same browser profile because identity is local-storage based
- Discovery state is lost on server restart because rooms and sessions are in memory
- If local storage loses `sessionId` or `resumeToken`, the browser can discover the game but cannot securely reclaim it in v1

## Future Extensions

Possible follow-ups after v1:

- Add a small signed browser identity token instead of raw `deviceId`
- Add an in-memory device index for faster lookup
- Add account-based resume across devices
- Add explicit session reclaim flows when the browser knows the game but lost the stored `resumeToken`

## Implementation Decision

Proceed with:

- Local-storage-backed `deviceId`
- Server-side `DeviceID` field on `ClientSession`
- Resume-discovery endpoint that scans active rooms
- Explicit "Resume your game?" prompt on app load
- Existing `resumeToken` retained as the only authorization mechanism for actual resume
