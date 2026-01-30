/**
 * Lobby-related payload validators
 *
 * lobby_created, lobby_joined, lobby_list_result, lobby_state message payloads
 */

import { createPayloadValidator } from "../factory";
import type { PayloadTypeMap } from "../types";

/**
 * Validate lobby_created payload
 * Expected: { lobby_id: string }
 */
export const validateLobbyCreatedPayload = createPayloadValidator<PayloadTypeMap["lobby_created"]>([
  { name: "lobby_id", required: true, type: "string" },
]);

/**
 * Validate lobby_joined payload
 * Expected: { lobby_id: string }
 */
export const validateLobbyJoinedPayload = createPayloadValidator<PayloadTypeMap["lobby_joined"]>([
  { name: "lobby_id", required: true, type: "string" },
]);

/**
 * Validate lobby_list_result payload
 * Expected: { lobbies: LobbySummary[] | null }
 */
export const validateLobbyListResultPayload = createPayloadValidator<PayloadTypeMap["lobby_list_result"]>([
  { name: "lobbies", required: true, type: "array" },
]);

/**
 * Validate lobby_state payload
 */
export const validateLobbyStatePayload = createPayloadValidator<PayloadTypeMap["lobby_state"]>([
  { name: "lobby_id", required: true, type: "string" },
  { name: "leader_nick", required: true, type: "string" },
  { name: "leader_id", required: true, type: "string" },
  { name: "player_nicks", required: true, type: "array", itemType: "string" },
  { name: "player_ids", required: true, type: "array", itemType: "string" },
  { name: "player_count", required: true, type: "number" },
  { name: "status", required: true, type: "string" },
  { name: "player_avatars", required: false, type: "array", itemType: "string" },
]);
