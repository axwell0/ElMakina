# Replay System Plan (Player‑Specific)

## Context
ElMakina currently emits live WebSocket events (game_log, game_state, etc.) and keeps match state in memory during play. There is no persisted event stream, no deterministic replay inputs, and no replay UI. The goal is to enable chess.com‑style **player‑specific replay**: a user can replay a finished match and see their private information, while others only see public information.

## Goals
- Provide deterministic, on‑demand replay for a completed match.
- Support **player‑specific** visibility (public vs private events).
- Keep the design lightweight: Postgres only; avoid heavy microservices.
- Preserve authoritative game loop in memory during live play.

## Non‑Goals
- Multi‑region replay at launch.
- Full event sourcing rewrite of the engine.
- Real‑time spectator view (can be added later).

## High‑Level Approach
1. Persist match metadata and **all authoritative inputs** (actions, challenges, counters, timeouts, kicks, pauses) to Postgres as an append‑only event stream.
2. Store periodic snapshots of game state to enable fast seeking.
3. Expose replay endpoints that filter events by visibility for the viewing player.
4. Build a replay UI that can step through events and render state frames.

## Data Model (Postgres)
### matches
- `id` (pk)
- `created_at`
- `ended_at`
- `ruleset_version`
- `protocol_version`
- `rng_seed` (required for deterministic re-sim)

### match_participants
- `match_id` (fk)
- `player_id`
- `display_name`
- `avatar`

### match_events (append‑only)
- `match_id` (fk)
- `seq` (monotonic)
- `type` (action, challenge, counter, timeout, kick, pause, resume, etc.)
- `payload` (JSONB)
- `visibility` (`public` | `private`)
- `player_id` (nullable; for private events)
- `created_at`

### match_snapshots
- `match_id` (fk)
- `seq` (event index at which snapshot is valid)
- `state` (JSONB full engine state)
- `created_at`

## Event Visibility Rules
- Public events are visible to all replay viewers.
- Private events are visible only to the `player_id` who is replaying.
- For player‑specific replay, return `public + private where player_id = viewer`.

## Replay APIs (MVP)
- `GET /replay/:id?viewer_id=...`
  - returns match metadata + filtered events + snapshots (masked per viewer).
- Optional: `GET /replay/:id/frame?seq=...&viewer_id=...`
  - server computes and returns a “frame” (state at seq), avoiding client re‑simulation.

## UI Plan (MVP)
- Add a Replay view with:
  - timeline list of events
  - step/play controls
  - board rendering from snapshots + event re‑application
  - "Your hand" panel (only for viewer)

## Integration Points (Go)
- `sessionRunner` emits events and persists them after every authoritative input/outcome.
- Store RNG seed at match start for deterministic re-sim instead of relying on snapshots.
- Persist snapshots every N events or once per turn.
- Extend match start/finish logic to create/update `matches` and `match_participants`.

## MVP Scope
1. Persist match metadata and event stream to Postgres.
2. Provide replay endpoint returning filtered events.
3. Add a basic replay UI that can step through public + viewer‑private events.

## Out of Scope (for MVP)
- Real‑time spectating.
- Cross‑match analytics dashboards.
- Automated highlights or advanced search.

## Risks and Mitigations
- **Data leakage:** Ensure visibility filtering is strict and validated server‑side.
- **Schema drift:** Use protocol_version and ruleset_version to decode events.
- **Storage growth:** Apply retention policies (e.g., store logs for 30–90 days).

## Success Signals
- Replays load reliably for completed matches.
- Players can see their own hidden info but not others’.
- Support/debugging can reproduce disputes using replay.

## First Implementation Step (Recommended)
- Add `matches`, `match_events`, and `match_participants` tables.
- Add a small server‑side event logger in `sessionRunner`.
