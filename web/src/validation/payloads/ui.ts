/**
 * UI-related payload validators
 *
 * game_log, investigate_result, chat_message message payloads
 */

import { createPayloadValidator } from "../factory";
import type { PayloadTypeMap } from "../types";

/**
 * Validate game_log payload
 */
export const validateGameLogPayload = createPayloadValidator<PayloadTypeMap["game_log"]>([
  { name: "turn", required: true, type: "number" },
  { name: "scope", required: true, type: "string" },
  { name: "message", required: true, type: "string" },
  { name: "player_index", required: false, type: "number" },
]);

/**
 * Validate investigate_result payload
 */
export const validateInvestigateResultPayload = createPayloadValidator<PayloadTypeMap["investigate_result"]>([
  { name: "target_name", required: true, type: "string" },
  { name: "role", required: true, type: "string" },
]);

/**
 * Validate chat_message payload
 */
export const validateChatMessagePayload = createPayloadValidator<PayloadTypeMap["chat_message"]>([
  { name: "id", required: true, type: "string" },
  { name: "senderIndex", required: true, type: "number" },
  { name: "senderName", required: true, type: "string" },
  { name: "text", required: true, type: "string" },
  { name: "timestamp", required: true, type: "number" },
]);
