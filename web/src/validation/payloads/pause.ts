/**
 * Pause/Vote-related payload validators
 *
 * game_paused, game_resumed, kick_vote_update, player_kicked message payloads
 */

import { createPayloadValidator } from "../factory";
import type { PayloadTypeMap } from "../types";

/**
 * Validate game_paused payload
 */
export const validateGamePausedPayload = createPayloadValidator<PayloadTypeMap["game_paused"]>([
  { name: "paused_by_player_id", required: true, type: "string" },
  { name: "paused_by_index", required: true, type: "number" },
  { name: "paused_by_name", required: true, type: "string" },
  { name: "deadline_ms", required: true, type: "number" },
  { name: "duration_ms", required: true, type: "number" },
  { name: "pause_reason", required: true, type: "string" },
  { name: "eligible_voters", required: true, type: "array", itemType: "number" },
  { name: "kick_votes", required: true, type: "array", itemType: "number" },
]);

/**
 * Validate game_resumed payload
 */
export const validateGameResumedPayload = createPayloadValidator<PayloadTypeMap["game_resumed"]>([
  { name: "resumed_by_player_id", required: true, type: "string" },
  { name: "resumed_by_index", required: true, type: "number" },
  { name: "resumed_by_name", required: true, type: "string" },
  { name: "resume_reason", required: true, type: "string" },
]);

/**
 * Validate kick_vote_update payload
 */
export const validateKickVoteUpdatePayload = createPayloadValidator<PayloadTypeMap["kick_vote_update"]>([
  { name: "eligible_voters", required: true, type: "array", itemType: "number" },
  { name: "kick_votes", required: true, type: "array", itemType: "number" },
]);

/**
 * Validate player_kicked payload
 */
export const validatePlayerKickedPayload = createPayloadValidator<PayloadTypeMap["player_kicked"]>([
  { name: "player_index", required: true, type: "number" },
  { name: "reason", required: true, type: "string" },
]);
