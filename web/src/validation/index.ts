/**
 * Validation Module
 *
 * 4-Layer WebSocket Message Validation System
 *
 * L1: Envelope Structure - JSON structure validation in SocketManager
 * L2: Type Whitelist - Known message type checking
 * L3: Payload Validation - Type-safe payload validation
 * L4: Business Logic - State-aware validation in reducers
 *
 * @example
 * ```typescript
 * // In SocketManager - L1 + L2 validation
 * const l1Result = validateEnvelopeStructure(rawData);
 * if (!l1Result.valid) {
 *   logValidationError({ layer: "L1_STRUCTURE", error: "Invalid envelope" });
 *   return;
 * }
 *
 * if (!isValidInboundMessageType(l1Result.data.type)) {
 *   logValidationWarning({ layer: "L2_TYPE_WHITELIST", error: "Unknown type" });
 *   return;
 * }
 *
 * // In slice handlers - L3 validation
 * const l3Result = validateGameStatePayload(envelope.payload);
 * if (!l3Result.valid) {
 *   logValidationError({ layer: "L3_PAYLOAD", error: "Invalid payload" });
 *   return state;
 * }
 * ```
 */

// Core types
export type {
  ValidationResult,
  ValidationError,
  ValidationErrorCode,
  ValidationLayer,
  InboundMessageType,
  EnvelopeStructure,
  PayloadTypeMap,
  PayloadValidator,
  FieldValidator,
} from "./types";

// L1 + L2 validation
export {
  validateEnvelopeStructure,
  isValidInboundMessageType,
  INBOUND_MESSAGE_TYPES,
} from "./envelope";

// L3 validator factory
export { createPayloadValidator, checkType } from "./factory";

// Logging
export {
  validationLogger,
  productionValidationLogger,
  developmentValidationLogger,
  createErrorContext,
} from "./logging";

// Payload validators (L3)
export {
  validateHelloAckPayload,
  validateHelloErrorPayload,
} from "./payloads/hello";

export {
  validateLobbyCreatedPayload,
  validateLobbyJoinedPayload,
  validateLobbyListResultPayload,
  validateLobbyStatePayload,
} from "./payloads/lobby";

export {
  validateLobbyStartedPayload,
  validateGameConfigPayload,
  validateGameStatePayload,
  validateRequestActionPayload,
  validateChallengeWindowPayload,
  validateCounterWindowPayload,
  validateRequestStepPayload,
  validateHandStatePayload,
  validatePromptClosedPayload,
  validateTurnTimerPayload,
  validateGameOverPayload,
  validatePlayerEliminatedPayload,
} from "./payloads/game";

export {
  validateGamePausedPayload,
  validateGameResumedPayload,
  validateKickVoteUpdatePayload,
  validatePlayerKickedPayload,
} from "./payloads/pause";

export {
  validateGameLogPayload,
  validateInvestigateResultPayload,
  validateChatMessagePayload,
} from "./payloads/ui";
