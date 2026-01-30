/**
 * L1 (Envelope Structure) and L2 (Type Whitelist) Validation
 *
 * L1: Validates the basic structure of WebSocket message envelopes
 * L2: Validates that the message type is in the whitelist of known types
 */

import type {
  ValidationResult,
  EnvelopeStructure,
  InboundMessageType,
  ValidationError,
} from "./types";

import { INBOUND_MESSAGE_TYPES } from "./types";

export { INBOUND_MESSAGE_TYPES } from "./types";

/**
 * Validates the envelope structure (L1 validation)
 *
 * Rules:
 * - Must be a non-null object
 * - Must have a 'type' property that is a non-empty string
 * - 'request_id' is optional but must be a string if present
 * - 'payload' can be any value (including undefined/null)
 *
 * @param data - Unknown data to validate
 * @returns ValidationResult with validated envelope or errors
 */
export function validateEnvelopeStructure(
  data: unknown
): ValidationResult<EnvelopeStructure> {
  const errors: ValidationError[] = [];

  // Must be a non-null object (not an array)
  if (typeof data !== "object" || data === null || Array.isArray(data)) {
    return {
      valid: false,
      errors: [
        {
          path: "",
          message: "Message must be an object",
          code: "NOT_OBJECT",
        },
      ],
    };
  }

  const obj = data as Record<string, unknown>;

  // Must have type property
  if (!("type" in obj)) {
    errors.push({
      path: "type",
      message: "Missing required field 'type'",
      code: "MISSING_TYPE",
    });
  } else if (typeof obj.type !== "string" || obj.type.length === 0) {
    errors.push({
      path: "type",
      message: "'type' must be a non-empty string",
      code: "INVALID_TYPE",
    });
  }

  // request_id must be string if present
  if ("request_id" in obj && obj.request_id !== undefined) {
    if (typeof obj.request_id !== "string") {
      errors.push({
        path: "request_id",
        message: "'request_id' must be a string",
        code: "INVALID_REQUEST_ID",
      });
    }
  }

  // payload can be any type, including undefined - no validation needed

  if (errors.length > 0) {
    return { valid: false, errors };
  }

  // At this point, we've validated that type exists and is a string
  const validated: EnvelopeStructure = {
    type: obj.type as string,
    payload: obj.payload,
  };

  if (obj.request_id !== undefined) {
    validated.request_id = obj.request_id as string;
  }

  return {
    valid: true,
    data: validated,
    errors: [],
  };
}

/**
 * Type guard for valid inbound message types (L2 validation)
 *
 * @param type - String to check
 * @returns True if type is a known inbound message type
 */
export function isValidInboundMessageType(
  type: string
): type is InboundMessageType {
  return INBOUND_MESSAGE_TYPES.includes(type as InboundMessageType);
}
