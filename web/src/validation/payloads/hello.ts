/**
 * Hello-related payload validators
 *
 * hello_ack, hello_error message payloads
 */

import { createPayloadValidator } from "../factory";
import type { PayloadTypeMap } from "../types";

/**
 * Validate hello_ack payload
 * Expected: { player_id: string, token: string }
 */
export const validateHelloAckPayload = createPayloadValidator<PayloadTypeMap["hello_ack"]>([
  { name: "player_id", required: true, type: "string" },
  { name: "token", required: true, type: "string" },
]);

/**
 * Validate hello_error payload
 * Expected: { error: string }
 */
export const validateHelloErrorPayload = createPayloadValidator<PayloadTypeMap["hello_error"]>([
  { name: "error", required: true, type: "string" },
]);
