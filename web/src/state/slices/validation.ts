/**
 * Slice Validation Utilities (L3 + L4 Validation)
 *
 * Helper functions for validating message payloads in slice handlers
 * and dispatching actions only when validation passes.
 */

import type {
  InboundMessageType,
  PayloadTypeMap,
  ValidationResult,
} from "@/validation/types";
import { validationLogger } from "@/validation/logging";
import type { SlicedGameState } from "@/state/slices";
import type { WsEnvelope } from "@/network/socket";

// Import all payload validators
import {
  validateHelloAckPayload,
  validateHelloErrorPayload,
  validateLobbyCreatedPayload,
  validateLobbyJoinedPayload,
  validateLobbyListResultPayload,
  validateLobbyStatePayload,
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
  validateGamePausedPayload,
  validateGameResumedPayload,
  validateKickVoteUpdatePayload,
  validatePlayerKickedPayload,
  validateGameLogPayload,
  validateInvestigateResultPayload,
  validateChatMessagePayload,
} from "@/validation";

/**
 * Map of message types to their validators
 */
const payloadValidators: {
  [K in InboundMessageType]?: (payload: unknown) => ValidationResult<PayloadTypeMap[K]>;
} = {
  hello_ack: validateHelloAckPayload,
  hello_error: validateHelloErrorPayload,
  lobby_list_result: validateLobbyListResultPayload,
  lobby_created: validateLobbyCreatedPayload,
  lobby_joined: validateLobbyJoinedPayload,
  lobby_state: validateLobbyStatePayload,
  lobby_started: validateLobbyStartedPayload,
  game_config: validateGameConfigPayload,
  game_state: validateGameStatePayload,
  request_action: validateRequestActionPayload,
  challenge_window: validateChallengeWindowPayload,
  counter_window: validateCounterWindowPayload,
  request_step: validateRequestStepPayload,
  hand_state: validateHandStatePayload,
  prompt_closed: validatePromptClosedPayload,
  turn_timer: validateTurnTimerPayload,
  game_over: validateGameOverPayload,
  player_eliminated: validatePlayerEliminatedPayload,
  game_paused: validateGamePausedPayload,
  game_resumed: validateGameResumedPayload,
  kick_vote_update: validateKickVoteUpdatePayload,
  player_kicked: validatePlayerKickedPayload,
  game_log: validateGameLogPayload,
  investigate_result: validateInvestigateResultPayload,
  chat_message: validateChatMessagePayload,
};

/**
 * Validate and dispatch a message payload
 *
 * L3 validation: Validates the payload structure matches the expected schema
 * L4 validation: Validates business logic within the onValid callback
 *
 * @param messageType - Type of message being validated
 * @param envelope - WebSocket envelope
 * @param currentState - Current state before processing
 * @param onValid - Callback to process validated payload and return new state
 * @returns New state (or current state if validation fails)
 */
export function validateAndDispatch<T extends InboundMessageType>(
  messageType: T,
  envelope: WsEnvelope,
  currentState: SlicedGameState,
  onValid: (data: PayloadTypeMap[T]) => SlicedGameState
): SlicedGameState {
  const validator = payloadValidators[messageType];

  // If no validator exists for this type, skip L3 validation
  if (!validator) {
    // Log warning but still process (backward compatibility)
    validationLogger.logValidationWarning({
      layer: "L3_PAYLOAD",
      messageType,
      error: `No validator found for message type: ${messageType}`,
    });
    return onValid(envelope.payload as PayloadTypeMap[T]);
  }

  // L3: Payload validation
  const startTime = performance.now();
  const validation = validator(envelope.payload);
  const duration = performance.now() - startTime;

  validationLogger.logValidationMetric(messageType, duration);

  if (!validation.valid) {
    validationLogger.logValidationError({
      layer: "L3_PAYLOAD",
      messageType,
      requestId: envelope.request_id,
      error: validation.errors.map((e) => `${e.path}: ${e.message}`).join(", "),
      details: validation.errors,
      rawEnvelope: envelope,
    });
    // Return current state unchanged on validation failure
    return currentState;
  }

  // L4: Business logic validation happens in the onValid callback
  try {
    return onValid(validation.data!);
  } catch (error) {
    validationLogger.logValidationError({
      layer: "L4_BUSINESS_LOGIC",
      messageType,
      requestId: envelope.request_id,
      error: error instanceof Error ? error.message : String(error),
    });
    return currentState;
  }
}

/**
 * Type guard to check if a message type has a validator
 *
 * @param messageType - Message type to check
 * @returns True if a validator exists for this message type
 */
export function hasPayloadValidator(
  messageType: string
): messageType is InboundMessageType {
  return messageType in payloadValidators;
}

/**
 * Get validator for a specific message type
 *
 * @param messageType - Message type
 * @returns Validator function or undefined
 */
export function getPayloadValidator<T extends InboundMessageType>(
  messageType: T
): ((payload: unknown) => ValidationResult<PayloadTypeMap[T]>) | undefined {
  return payloadValidators[messageType];
}
