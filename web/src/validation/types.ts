/**
 * Core validation types for the 4-layer WebSocket validation system
 *
 * L1: Envelope Structure - JSON structure validation
 * L2: Type Whitelist - Known message type checking
 * L3: Payload Validation - Type-safe payload validation
 * L4: Business Logic - State-aware validation
 */

/**
 * Validation error with context for debugging
 */
export interface ValidationError {
  /** Path to the field that failed validation */
  path: string;
  /** Human-readable error message */
  message: string;
  /** Error code for programmatic handling */
  code: ValidationErrorCode;
}

/**
 * Error codes for validation failures
 */
export type ValidationErrorCode =
  // L1 Structure errors
  | "NOT_OBJECT"
  | "MISSING_TYPE"
  | "INVALID_TYPE"
  | "INVALID_REQUEST_ID"
  | "JSON_PARSE_ERROR"
  // L2 Type whitelist errors
  | "UNKNOWN_MESSAGE_TYPE"
  // L3 Payload errors
  | "MISSING_FIELD"
  | "TYPE_MISMATCH"
  | "INVALID_ARRAY_ITEM"
  | "CUSTOM_VALIDATION_FAILED"
  // L4 Business logic errors
  | "INVALID_STATE"
  | "STATE_MISMATCH";

/**
 * Result of validation operation
 */
export interface ValidationResult<T> {
  /** Whether validation passed */
  valid: boolean;
  /** Validated data (only present if valid) */
  data?: T;
  /** List of validation errors (empty if valid) */
  errors: ValidationError[];
}

/**
 * Validation layer identifier for logging and metrics
 */
export type ValidationLayer =
  | "L1_STRUCTURE"
  | "L2_TYPE_WHITELIST"
  | "L3_PAYLOAD"
  | "L4_BUSINESS_LOGIC";

/**
 * Inbound message types we can receive from the server
 */
export const INBOUND_MESSAGE_TYPES = [
  // Connection
  "hello_ack",
  "hello_error",

  // Lobby
  "lobby_list_result",
  "lobby_created",
  "lobby_joined",
  "lobby_state",

  // Game
  "lobby_started",
  "game_config",
  "game_state",
  "request_action",
  "challenge_window",
  "counter_window",
  "request_step",
  "hand_state",
  "prompt_closed",
  "turn_timer",
  "game_over",
  "player_eliminated",

  // Pause/Vote
  "game_paused",
  "game_resumed",
  "kick_vote_update",
  "player_kicked",

  // UI
  "game_log",
  "investigate_result",
  "chat_message",
] as const;

/**
 * Type for valid inbound message types
 */
export type InboundMessageType = typeof INBOUND_MESSAGE_TYPES[number];

/**
 * Basic envelope structure (L1 validation)
 */
export interface EnvelopeStructure {
  type: string;
  payload?: unknown;
  request_id?: string;
}

/**
 * Map of message types to their payload types (for L3 validation)
 * This will be extended in payload validators
 */
export interface PayloadTypeMap {
  hello_ack: { player_id: string; token: string };
  hello_error: { error: string };
  lobby_list_result: { lobbies: unknown[] | null };
  lobby_created: { lobby_id: string };
  lobby_joined: { lobby_id: string };
  lobby_state: {
    lobby_id: string;
    leader_nick: string;
    leader_id: string;
    player_nicks: string[] | null;
    player_ids: string[] | null;
    player_avatars?: string[] | null;
    player_count: number;
    status: string;
  };
  lobby_started: {
    lobby_id: string;
    match_id: string;
    player_index: number;
    player_count: number;
    player_names: string[] | null;
    player_avatars?: string[] | null;
    index_mapping: Record<string, number> | null;
  };
  game_config: { roles: string[] | null };
  game_state: {
    turn_number: number;
    active_player_index: number;
    players: unknown[] | null;
  };
  request_action: {
    actor_index: number;
    allowed_actions?: string[] | null;
    prompt?: string;
  };
  challenge_window: {
    actor_index: number;
    action_id: string;
    claimed_role: string;
    kind: string;
    target_index?: number | null;
    eligible: boolean;
    timeout_ms?: number;
    prompt?: string;
  };
  counter_window: {
    actor_index: number;
    action_id: string;
    allowed_actions?: string[] | null;
    target_index?: number | null;
    eligible: boolean;
    timeout_ms?: number;
    prompt?: string;
  };
  request_step: {
    step: unknown;
    prompt?: string;
  };
  hand_state: {
    player_index: number;
    hand: string[] | null;
  };
  prompt_closed: { reason: string };
  turn_timer: {
    active_player_index: number;
    turn_number: number;
    duration_ms: number;
    state: string;
  };
  game_over: {
    winner_index: number;
    winner_name: string;
  };
  player_eliminated: {
    player_index: number;
    reason: string;
    turn: number;
  };
  game_paused: {
    paused_by_player_id: string;
    paused_by_index: number;
    paused_by_name: string;
    deadline_ms: number;
    duration_ms: number;
    pause_reason: string;
    eligible_voters: number[] | null;
    kick_votes: number[] | null;
  };
  game_resumed: {
    resumed_by_player_id: string;
    resumed_by_index: number;
    resumed_by_name: string;
    resume_reason: string;
  };
  kick_vote_update: {
    eligible_voters: number[] | null;
    kick_votes: number[] | null;
  };
  player_kicked: {
    player_index: number;
    reason: string;
  };
  game_log: {
    turn: number;
    scope: string;
    message: string;
    player_index?: number | null;
  };
  investigate_result: {
    target_name: string;
    role: string;
  };
  chat_message: {
    id: string;
    senderIndex: number;
    senderName: string;
    text: string;
    timestamp: number;
  };
}

/**
 * Validator function type for payload validation
 */
export type PayloadValidator<T> = (payload: unknown) => ValidationResult<T>;

/**
 * Field validator configuration for factory
 */
export interface FieldValidator {
  /** Field name */
  name: string;
  /** Whether the field is required */
  required: boolean;
  /** Expected type */
  type: "string" | "number" | "boolean" | "array" | "object";
  /** For arrays, the item type */
  itemType?: "string" | "number" | "boolean" | "object";
  /** Custom validation function */
  validator?: (value: unknown) => boolean;
  /** For arrays, custom item validator */
  itemValidator?: (item: unknown) => boolean;
}
