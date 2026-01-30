/**
 * Game-related payload validators
 *
 * Game state, actions, prompts, and related message payloads
 */

import { createPayloadValidator } from "../factory";
import type { PayloadTypeMap } from "../types";

/**
 * Validate lobby_started payload
 */
export const validateLobbyStartedPayload = createPayloadValidator<PayloadTypeMap["lobby_started"]>([
  { name: "lobby_id", required: true, type: "string" },
  { name: "match_id", required: true, type: "string" },
  { name: "player_index", required: true, type: "number" },
  { name: "player_count", required: true, type: "number" },
  { name: "player_names", required: true, type: "array", itemType: "string" },
  { name: "index_mapping", required: true, type: "object" },
  { name: "player_avatars", required: false, type: "array", itemType: "string" },
]);

/**
 * Validate game_config payload
 */
export const validateGameConfigPayload = createPayloadValidator<PayloadTypeMap["game_config"]>([
  { name: "roles", required: true, type: "array", itemType: "string" },
]);

/**
 * Validate game_state payload
 */
export const validateGameStatePayload = createPayloadValidator<PayloadTypeMap["game_state"]>([
  { name: "turn_number", required: true, type: "number" },
  { name: "active_player_index", required: true, type: "number" },
  { name: "players", required: true, type: "array" },
]);

/**
 * Validate request_action payload
 */
export const validateRequestActionPayload = createPayloadValidator<PayloadTypeMap["request_action"]>([
  { name: "actor_index", required: true, type: "number" },
  { name: "allowed_actions", required: false, type: "array", itemType: "string" },
  { name: "prompt", required: false, type: "string" },
]);

/**
 * Validate challenge_window payload
 */
export const validateChallengeWindowPayload = createPayloadValidator<PayloadTypeMap["challenge_window"]>([
  { name: "actor_index", required: true, type: "number" },
  { name: "action_id", required: true, type: "string" },
  { name: "claimed_role", required: true, type: "string" },
  { name: "kind", required: true, type: "string" },
  { name: "eligible", required: true, type: "boolean" },
  { name: "target_index", required: false, type: "number" },
  { name: "timeout_ms", required: false, type: "number" },
  { name: "prompt", required: false, type: "string" },
]);

/**
 * Validate counter_window payload
 */
export const validateCounterWindowPayload = createPayloadValidator<PayloadTypeMap["counter_window"]>([
  { name: "actor_index", required: true, type: "number" },
  { name: "action_id", required: true, type: "string" },
  { name: "allowed_actions", required: false, type: "array", itemType: "string" },
  { name: "target_index", required: false, type: "number" },
  { name: "eligible", required: true, type: "boolean" },
  { name: "timeout_ms", required: false, type: "number" },
  { name: "prompt", required: false, type: "string" },
]);

/**
 * Validate request_step payload
 */
export const validateRequestStepPayload = createPayloadValidator<PayloadTypeMap["request_step"]>([
  { name: "step", required: true, type: "object" },
  { name: "prompt", required: false, type: "string" },
]);

/**
 * Validate hand_state payload
 */
export const validateHandStatePayload = createPayloadValidator<PayloadTypeMap["hand_state"]>([
  { name: "player_index", required: true, type: "number" },
  { name: "hand", required: true, type: "array", itemType: "string" },
]);

/**
 * Validate prompt_closed payload
 */
export const validatePromptClosedPayload = createPayloadValidator<PayloadTypeMap["prompt_closed"]>([
  { name: "reason", required: true, type: "string" },
]);

/**
 * Validate turn_timer payload
 */
export const validateTurnTimerPayload = createPayloadValidator<PayloadTypeMap["turn_timer"]>([
  { name: "active_player_index", required: true, type: "number" },
  { name: "turn_number", required: true, type: "number" },
  { name: "duration_ms", required: true, type: "number" },
  { name: "state", required: true, type: "string" },
]);

/**
 * Validate game_over payload
 */
export const validateGameOverPayload = createPayloadValidator<PayloadTypeMap["game_over"]>([
  { name: "winner_index", required: true, type: "number" },
  { name: "winner_name", required: true, type: "string" },
]);

/**
 * Validate player_eliminated payload
 */
export const validatePlayerEliminatedPayload = createPayloadValidator<PayloadTypeMap["player_eliminated"]>([
  { name: "player_index", required: true, type: "number" },
  { name: "reason", required: true, type: "string" },
  { name: "turn", required: true, type: "number" },
]);
