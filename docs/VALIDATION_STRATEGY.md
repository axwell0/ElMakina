# WebSocket Validation Strategy

## Executive Summary

This document defines a comprehensive validation strategy for WebSocket messages in the ElMakina frontend. The strategy implements a **dual validation approach** with four distinct validation layers, ensuring type safety, security, and robust error handling across the message processing pipeline.

**Current State:**
- SocketManager uses AJV-based JSON Schema validation (ws-schema.json) for envelope structure
- Message dispatching happens in `state/slices/index.ts` without payload validation
- Generated TypeScript types exist but lack runtime validators
- No per-handler validation exists

**Target State:**
- Layer 1-2: SocketManager validates envelope structure and message type whitelist
- Layer 3: Per-handler payload validation using type guards
- Layer 4: Business logic validation within reducers
- Comprehensive error handling and logging throughout

---

## Table of Contents

1. [Dual Validation Approach](#dual-validation-approach)
2. [Validation Layers](#validation-layers)
3. [Type-Safe Validation](#type-safe-validation)
4. [Error Handling Strategy](#error-handling-strategy)
5. [Implementation Plan](#implementation-plan)
6. [Testing Strategy](#testing-strategy)
7. [Appendix: Message Type Registry](#appendix-message-type-registry)

---

## Dual Validation Approach

### Overview

The validation strategy splits responsibility between two locations:

```
┌─────────────────────────────────────────────────────────────────┐
│                     WebSocket Message Flow                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐      │
│  │   Network    │───▶│  Validation  │───▶│   Dispatch   │      │
│  │   Layer      │    │   Layers     │    │   Layer      │      │
│  └──────────────┘    └──────────────┘    └──────────────┘      │
│         │                   │                   │               │
│         ▼                   ▼                   ▼               │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐      │
│  │ SocketManager│    │ Per-Handler  │    │    Slices    │      │
│  │ (Boundary)   │    │ (Type Guard) │    │ (Business)   │      │
│  │              │    │              │    │              │      │
│  │ • JSON parse │    │ • Payload    │    │ • State      │      │
│  │ • Envelope   │    │   validation │    │   checks     │      │
│  │   structure  │    │ • Type       │    │ • Transition │      │
│  │ • Type       │    │   narrowing  │    │   validity   │      │
│  │   whitelist  │    │              │    │              │      │
│  └──────────────┘    └──────────────┘    └──────────────┘      │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 1. Boundary Validation (SocketManager)

**Location:** `web/src/network/socket.ts`

**Responsibilities:**
- Parse raw WebSocket message data as JSON
- Validate envelope structure (has `type`, has `payload` object)
- Validate message type against whitelist
- Reject malformed messages early to prevent downstream errors

**Current Implementation:**
```typescript
// socket.ts (current)
this.ws.onmessage = (event) => {
    try {
        const envelope = JSON.parse(event.data) as WsEnvelope;
        if (this.validateEnvelope && !this.validateEnvelope(envelope)) {
            const errors = this.validateEnvelope.errors as ErrorObject[] | null | undefined;
            console.error("[ws] Invalid envelope received", { errors, envelope });
            return;
        }
        this.handleMessage(envelope);
    } catch (e) {
        console.error("Failed to parse message:", event.data, e);
    }
};
```

**Improvements Needed:**
- Always validate (not just in non-production)
- Add structured error logging
- Add metrics/telemetry hooks

### 2. Handler Validation (Per-Message)

**Location:** `web/src/state/slices/index.ts` and individual slice handlers

**Responsibilities:**
- Validate payload structure matches expected schema for message type
- Type-narrow payloads using type guards
- Validate required fields exist and have correct types
- Provide detailed validation error context

**Current Gap:**
```typescript
// slices/index.ts (current - no validation)
case "game_state": {
    const payload = envelope.payload as Parameters<typeof gameActions.gameState>[0];
    // No validation - just casts!
    return {
        ...state,
        game: gameReducer(
            state.game,
            gameActions.gameState(payload, state.game.identity)
        ),
    };
}
```

**Target Implementation:**
```typescript
// slices/index.ts (target - with validation)
case "game_state": {
    const validation = validateGameStatePayload(envelope.payload);
    if (!validation.valid) {
        logValidationError("game_state", envelope, validation.errors);
        return state; // Preserve state on invalid message
    }
    const payload = validation.data; // Type-narrowed payload
    return {
        ...state,
        game: gameReducer(
            state.game,
            gameActions.gameState(payload, state.game.identity)
        ),
    };
}
```

---

## Validation Layers

### Layer 1: Envelope Structure

**Purpose:** Ensure the message has the basic envelope structure required by all messages.

**Validation Rules:**
```typescript
interface EnvelopeStructure {
    type: string;           // Required: non-empty string
    payload?: unknown;      // Optional: any valid JSON value
    request_id?: string;    // Optional: non-empty string
}
```

**Validation Function:**
```typescript
// validation/envelope.ts
export interface ValidationResult<T> {
    valid: boolean;
    data?: T;
    errors: ValidationError[];
}

export interface ValidationError {
    path: string;
    message: string;
    code: string;
}

export function validateEnvelopeStructure(
    data: unknown
): ValidationResult<{ type: string; payload?: unknown; request_id?: string }> {
    const errors: ValidationError[] = [];

    // Must be an object
    if (typeof data !== "object" || data === null) {
        return { valid: false, errors: [{ path: "", message: "Message must be an object", code: "NOT_OBJECT" }] };
    }

    const obj = data as Record<string, unknown>;

    // Must have type property
    if (!("type" in obj)) {
        errors.push({ path: "type", message: "Missing required field 'type'", code: "MISSING_TYPE" });
    } else if (typeof obj.type !== "string" || obj.type.length === 0) {
        errors.push({ path: "type", message: "'type' must be a non-empty string", code: "INVALID_TYPE" });
    }

    // request_id must be string if present
    if ("request_id" in obj && obj.request_id !== undefined) {
        if (typeof obj.request_id !== "string") {
            errors.push({ path: "request_id", message: "'request_id' must be a string", code: "INVALID_REQUEST_ID" });
        }
    }

    // payload can be any type, including undefined

    if (errors.length > 0) {
        return { valid: false, errors };
    }

    return {
        valid: true,
        data: obj as { type: string; payload?: unknown; request_id?: string },
        errors: []
    };
}
```

### Layer 2: Message Type Whitelist

**Purpose:** Ensure the message type is one we recognize and handle.

**Message Type Registry:**
```typescript
// validation/message-types.ts

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

export type InboundMessageType = typeof INBOUND_MESSAGE_TYPES[number];

export function isValidInboundMessageType(type: string): type is InboundMessageType {
    return INBOUND_MESSAGE_TYPES.includes(type as InboundMessageType);
}
```

**Integration with SocketManager:**
```typescript
// socket.ts
private handleMessage(envelope: WsEnvelope) {
    // Layer 2: Validate message type
    if (!isValidInboundMessageType(envelope.type)) {
        logValidationError({
            layer: "L2_TYPE_WHITELIST",
            messageType: envelope.type,
            requestId: envelope.request_id,
            error: `Unknown message type: ${envelope.type}`
        });
        return; // Silently drop unknown message types
    }
    
    // ... continue processing
}
```

### Layer 3: Payload Validation

**Purpose:** Ensure the payload matches the expected schema for the specific message type.

**Type Guard Pattern:**
```typescript
// validation/payloads/game-state.ts
import type { GameStatePayload } from "@/types/generated";

export function isGameStatePayload(payload: unknown): payload is GameStatePayload {
    if (typeof payload !== "object" || payload === null) {
        return false;
    }
    
    const p = payload as Record<string, unknown>;
    
    // Required fields
    if (typeof p.turn_number !== "number") return false;
    if (typeof p.active_player_index !== "number") return false;
    if (!Array.isArray(p.players)) return false;
    
    // Validate players array
    for (const player of p.players) {
        if (!isPlayerStatePayload(player)) return false;
    }
    
    return true;
}

function isPlayerStatePayload(player: unknown): boolean {
    if (typeof player !== "object" || player === null) return false;
    const p = player as Record<string, unknown>;
    
    return (
        typeof p.index === "number" &&
        typeof p.name === "string" &&
        typeof p.coins === "number" &&
        typeof p.card_count === "number" &&
        typeof p.alive === "boolean"
    );
}

export function validateGameStatePayload(payload: unknown): ValidationResult<GameStatePayload> {
    if (isGameStatePayload(payload)) {
        return { valid: true, data: payload, errors: [] };
    }
    
    return {
        valid: false,
        errors: [{
            path: "payload",
            message: "Invalid GameState payload structure",
            code: "INVALID_GAME_STATE_PAYLOAD"
        }]
    };
}
```

**Complete Payload Validators:**

Create validators for all message types:

```typescript
// validation/payloads/index.ts
export { validateHelloAckPayload, isHelloAckPayload } from "./hello";
export { validateHelloErrorPayload, isHelloErrorPayload } from "./hello";
export { validateLobbyListResultPayload, isLobbyListResultPayload } from "./lobby";
export { validateLobbyCreatedPayload, isLobbyCreatedPayload } from "./lobby";
export { validateLobbyJoinedPayload, isLobbyJoinedPayload } from "./lobby";
export { validateLobbyStatePayload, isLobbyStatePayload } from "./lobby";
export { validateLobbyStartedPayload, isLobbyStartedPayload } from "./game";
export { validateGameConfigPayload, isGameConfigPayload } from "./game";
export { validateGameStatePayload, isGameStatePayload } from "./game-state";
export { validateRequestActionPayload, isRequestActionPayload } from "./game";
export { validateChallengeWindowPayload, isChallengeWindowPayload } from "./game";
export { validateCounterWindowPayload, isCounterWindowPayload } from "./game";
export { validateRequestStepPayload, isRequestStepPayload } from "./game";
export { validateHandStatePayload, isHandStatePayload } from "./game";
export { validatePromptClosedPayload, isPromptClosedPayload } from "./game";
export { validateTurnTimerPayload, isTurnTimerPayload } from "./game";
export { validateGameOverPayload, isGameOverPayload } from "./game";
export { validatePlayerEliminatedPayload, isPlayerEliminatedPayload } from "./game";
export { validateGamePausedPayload, isGamePausedPayload } from "./pause";
export { validateGameResumedPayload, isGameResumedPayload } from "./pause";
export { validateKickVoteUpdatePayload, isKickVoteUpdatePayload } from "./pause";
export { validatePlayerKickedPayload, isPlayerKickedPayload } from "./pause";
export { validateGameLogPayload, isGameLogPayload } from "./ui";
export { validateInvestigateResultPayload, isInvestigateResultPayload } from "./ui";
export { validateChatMessagePayload, isChatMessagePayload } from "./ui";
```

### Layer 4: Business Logic Validation

**Purpose:** Ensure the message makes sense in the current application state.

**Examples:**
```typescript
// Business logic validation within reducers

// Example 1: Ignore game_state if not in a match
case "GAME_STATE": {
    if (!state.currentMatch) {
        // Log warning but don't crash
        console.warn("[validation] Received game_state but not in a match");
        return state;
    }
    // ... process normally
}

// Example 2: Validate player index is within bounds
case "PLAYER_ELIMINATED": {
    const payload = action.payload;
    if (payload.player_index < 0 || payload.player_index >= state.players.length) {
        console.error(`[validation] Invalid player_index: ${payload.player_index}`);
        return state;
    }
    // ... process normally
}

// Example 3: Validate turn number progression
case "TURN_TIMER": {
    const payload = action.payload;
    if (state.turnTimer && payload.turn_number < state.turnTimer.turnNumber) {
        console.warn(`[validation] Received stale turn timer: ${payload.turn_number} < ${state.turnTimer.turnNumber}`);
        // Still process - might be a server correction
    }
    // ... process normally
}
```

---

## Type-Safe Validation

### Using Generated Types

The `generated.ts` file contains TypeScript interfaces auto-generated from Go structs. These should be the source of truth for payload shapes.

**Generated Type Example:**
```typescript
// types/generated.ts
export interface GameStatePayload {
    turn_number: number;
    active_player_index: number;
    players: PlayerStatePayload[] | null;
}

export interface PlayerStatePayload {
    index: number;
    name: string;
    coins: number;
    card_count: number;
    alive: boolean;
    avatar?: string;
}
```

**Mapping Generated Types to Validators:**
```typescript
// validation/types.ts
import type * as Generated from "@/types/generated";

// Map message types to their payload types
export interface PayloadTypeMap {
    "hello_ack": Generated.HelloAckPayload;
    "hello_error": Generated.HelloErrorPayload;
    "lobby_list_result": Generated.LobbyListPayload;
    "lobby_created": Generated.LobbyCreatedPayload;
    "lobby_joined": Generated.LobbyJoinPayload;
    "lobby_state": Generated.LobbyStatePayload;
    "lobby_started": Generated.LobbyStartedPayload;
    "game_config": Generated.GameConfigPayload;
    "game_state": Generated.GameStatePayload;
    "request_action": Generated.RequestActionPayload;
    "challenge_window": Generated.ChallengeWindowPayload;
    "counter_window": Generated.CounterWindowPayload;
    "request_step": Generated.RequestStepPayload;
    "hand_state": Generated.HandStatePayload;
    "prompt_closed": Generated.PromptClosedPayload;
    "turn_timer": Generated.TurnTimerPayload;
    "game_over": Generated.GameOverPayload;
    "player_eliminated": Generated.PlayerEliminatedPayload;
    "game_paused": Generated.GamePausedPayload;
    "game_resumed": Generated.GameResumedPayload;
    "kick_vote_update": Generated.KickVoteUpdatePayload;
    "player_kicked": Generated.PlayerKickedPayload;
    "game_log": Generated.GameLogPayload;
    "investigate_result": Generated.InvestigateResultPayload;
    "chat_message": Generated.ChatMessagePayload;
}

// Type-safe validator lookup
export type PayloadValidator<T extends InboundMessageType> = (
    payload: unknown
) => ValidationResult<PayloadTypeMap[T]>;
```

### Runtime Type Guards

**Factory for Creating Validators:**
```typescript
// validation/factory.ts
import type { PayloadTypeMap } from "./types";
import type { InboundMessageType, ValidationResult } from "./envelope";

interface FieldValidator {
    name: string;
    required: boolean;
    type: "string" | "number" | "boolean" | "array" | "object";
    itemType?: "string" | "number" | "boolean" | "object";
    validator?: (value: unknown) => boolean;
}

export function createPayloadValidator<T extends InboundMessageType>(
    fields: FieldValidator[]
): (payload: unknown) => ValidationResult<PayloadTypeMap[T]> {
    return (payload: unknown): ValidationResult<PayloadTypeMap[T]> => {
        const errors: ValidationError[] = [];
        
        if (typeof payload !== "object" || payload === null) {
            return {
                valid: false,
                errors: [{ path: "", message: "Payload must be an object", code: "NOT_OBJECT" }]
            };
        }
        
        const obj = payload as Record<string, unknown>;
        
        for (const field of fields) {
            const value = obj[field.name];
            
            if (field.required && !(field.name in obj)) {
                errors.push({
                    path: field.name,
                    message: `Missing required field '${field.name}'`,
                    code: "MISSING_FIELD"
                });
                continue;
            }
            
            if (value === undefined && !field.required) {
                continue;
            }
            
            // Type checking
            if (!checkType(value, field)) {
                errors.push({
                    path: field.name,
                    message: `Field '${field.name}' must be of type ${field.type}`,
                    code: "TYPE_MISMATCH"
                });
                continue;
            }
            
            // Custom validation
            if (field.validator && !field.validator(value)) {
                errors.push({
                    path: field.name,
                    message: `Field '${field.name}' failed custom validation`,
                    code: "CUSTOM_VALIDATION_FAILED"
                });
            }
        }
        
        if (errors.length > 0) {
            return { valid: false, errors };
        }
        
        return { valid: true, data: obj as PayloadTypeMap[T], errors: [] };
    };
}

function checkType(value: unknown, field: FieldValidator): boolean {
    switch (field.type) {
        case "string":
            return typeof value === "string";
        case "number":
            return typeof value === "number" && !isNaN(value);
        case "boolean":
            return typeof value === "boolean";
        case "array":
            if (!Array.isArray(value)) return false;
            if (field.itemType) {
                return value.every(item => checkItemType(item, field.itemType!));
            }
            return true;
        case "object":
            return typeof value === "object" && value !== null && !Array.isArray(value);
        default:
            return false;
    }
}

function checkItemType(value: unknown, type: string): boolean {
    switch (type) {
        case "string": return typeof value === "string";
        case "number": return typeof value === "number";
        case "boolean": return typeof value === "boolean";
        case "object": return typeof value === "object" && value !== null;
        default: return false;
    }
}
```

**Generated Validator Example:**
```typescript
// validation/payloads/generated/hello.ts
import { createPayloadValidator } from "../../factory";
import type { HelloAckPayload, HelloErrorPayload } from "@/types/generated";

export const validateHelloAckPayload = createPayloadValidator<"hello_ack">([
    { name: "player_id", required: true, type: "string" },
    { name: "token", required: true, type: "string" },
]);

export const validateHelloErrorPayload = createPayloadValidator<"hello_error">([
    { name: "error", required: true, type: "string" },
]);
```

---

## Error Handling Strategy

### Layer-Specific Error Handling

| Layer | Failure Action | Logging | User Impact |
|-------|---------------|---------|-------------|
| L1: Structure | Drop message, log error | ERROR level with raw data | None (silent) |
| L2: Type Whitelist | Drop message, log warning | WARN level with type name | None (silent) |
| L3: Payload | Drop message, log error | ERROR level with validation errors | None (silent) |
| L4: Business Logic | Process partially or skip | WARN level with context | Minimal (graceful) |

### Error Logger Interface

```typescript
// validation/logging.ts

interface ValidationErrorContext {
    layer: "L1_STRUCTURE" | "L2_TYPE_WHITELIST" | "L3_PAYLOAD" | "L4_BUSINESS_LOGIC";
    messageType?: string;
    requestId?: string;
    error: string;
    details?: unknown;
    rawEnvelope?: unknown;
}

interface ValidationLogger {
    logValidationError(context: ValidationErrorContext): void;
    logValidationWarning(context: ValidationErrorContext): void;
    logValidationMetric(messageType: string, duration: number): void;
}

// Production logger
export const productionValidationLogger: ValidationLogger = {
    logValidationError(context: ValidationErrorContext) {
        // Send to error tracking service (Sentry, etc.)
        console.error("[validation:error]", {
            layer: context.layer,
            messageType: context.messageType,
            requestId: context.requestId,
            error: context.error,
            // Don't log rawEnvelope in production (may contain sensitive data)
        });
    },
    
    logValidationWarning(context: ValidationErrorContext) {
        console.warn("[validation:warning]", {
            layer: context.layer,
            messageType: context.messageType,
            error: context.error,
        });
    },
    
    logValidationMetric(messageType: string, duration: number) {
        // Send to metrics service
        if (process.env.NODE_ENV === "development") {
            console.log(`[validation:metric] ${messageType}: ${duration}ms`);
        }
    }
};

// Development logger (more verbose)
export const developmentValidationLogger: ValidationLogger = {
    logValidationError(context: ValidationErrorContext) {
        console.error("[validation:error]", {
            ...context,
            rawEnvelope: context.rawEnvelope, // Include in dev
        });
    },
    
    logValidationWarning(context: ValidationErrorContext) {
        console.warn("[validation:warning]", context);
    },
    
    logValidationMetric(messageType: string, duration: number) {
        console.log(`[validation:metric] ${messageType}: ${duration}ms`);
    }
};

export const validationLogger = process.env.NODE_ENV === "production"
    ? productionValidationLogger
    : developmentValidationLogger;
```

### Integration Points

**SocketManager Integration:**
```typescript
// socket.ts
private handleMessage(envelope: WsEnvelope) {
    // L2: Type whitelist validation
    if (!isValidInboundMessageType(envelope.type)) {
        validationLogger.logValidationWarning({
            layer: "L2_TYPE_WHITELIST",
            messageType: envelope.type,
            requestId: envelope.request_id,
            error: `Unknown message type: ${envelope.type}`,
            rawEnvelope: envelope
        });
        return;
    }
    
    // Broadcast to listeners - validation continues in slices
    this.listeners.forEach(listener => listener(envelope));
}
```

**Slice Integration:**
```typescript
// state/slices/index.ts
import { validatePayload } from "@/validation/payloads";
import { validationLogger } from "@/validation/logging";

case "game_state": {
    const startTime = performance.now();
    const validation = validatePayload("game_state", envelope.payload);
    const duration = performance.now() - startTime;
    
    validationLogger.logValidationMetric("game_state", duration);
    
    if (!validation.valid) {
        validationLogger.logValidationError({
            layer: "L3_PAYLOAD",
            messageType: "game_state",
            requestId: envelope.request_id,
            error: validation.errors.map(e => `${e.path}: ${e.message}`).join(", "),
            details: validation.errors,
            rawEnvelope: envelope
        });
        return state;
    }
    
    // Proceed with validated payload
    const payload = validation.data;
    // ...
}
```

---

## Implementation Plan

### Step 1: Create Validation Utilities (Estimated: 2 days)

**Files to Create:**
```
web/src/validation/
├── index.ts                 # Main exports
├── types.ts                 # Type definitions
├── envelope.ts              # L1 & L2 validation
├── factory.ts               # Validator factory
├── logging.ts               # Error logging
├── payloads/
│   ├── index.ts             # Payload validator exports
│   ├── hello.ts             # hello_ack, hello_error
│   ├── lobby.ts             # Lobby-related payloads
│   ├── game.ts              # Game action payloads
│   ├── game-state.ts        # GameStatePayload
│   ├── pause.ts             # Pause/vote payloads
│   └── ui.ts                # UI-related payloads
```

**Tasks:**
1. Create `envelope.ts` with L1 structure and L2 type whitelist validation
2. Create `factory.ts` with `createPayloadValidator` helper
3. Create `logging.ts` with validation logger
4. Generate all payload validators using factory

### Step 2: Add SocketManager Validation (Estimated: 1 day)

**Changes to `web/src/network/socket.ts`:**
1. Add L1 validation before JSON parsing
2. Add L2 validation in `handleMessage`
3. Integrate validation logger
4. Remove environment-gated validation (always validate)
5. Add metrics collection

**Code Changes:**
```typescript
// Add imports
import { validateEnvelopeStructure, isValidInboundMessageType } from "@/validation/envelope";
import { validationLogger } from "@/validation/logging";

// Update onmessage
this.ws.onmessage = (event) => {
    const startTime = performance.now();
    
    try {
        const raw = JSON.parse(event.data);
        
        // L1: Structure validation
        const structureValidation = validateEnvelopeStructure(raw);
        if (!structureValidation.valid) {
            validationLogger.logValidationError({
                layer: "L1_STRUCTURE",
                error: "Invalid envelope structure",
                details: structureValidation.errors,
                rawEnvelope: raw
            });
            return;
        }
        
        const envelope = structureValidation.data as WsEnvelope;
        
        // L2: Type whitelist validation
        if (!isValidInboundMessageType(envelope.type)) {
            validationLogger.logValidationWarning({
                layer: "L2_TYPE_WHITELIST",
                messageType: envelope.type,
                error: `Unknown message type: ${envelope.type}`,
                rawEnvelope: raw
            });
            return;
        }
        
        const duration = performance.now() - startTime;
        validationLogger.logValidationMetric("envelope_validation", duration);
        
        this.handleMessage(envelope);
    } catch (e) {
        validationLogger.logValidationError({
            layer: "L1_STRUCTURE",
            error: `Failed to parse JSON: ${e instanceof Error ? e.message : String(e)}`,
            rawEnvelope: event.data
        });
    }
};
```

### Step 3: Add Per-Handler Validation (Estimated: 3 days)

**Changes to `web/src/state/slices/index.ts`:**

1. Add imports:
```typescript
import { validatePayload } from "@/validation/payloads";
import { validationLogger } from "@/validation/logging";
```

2. Create validation wrapper:
```typescript
function validateAndDispatch<T extends InboundMessageType>(
    messageType: T,
    payload: unknown,
    requestId: string | undefined,
    onValid: (data: PayloadTypeMap[T]) => SlicedGameState
): SlicedGameState {
    const validation = validatePayload(messageType, payload);
    
    if (!validation.valid) {
        validationLogger.logValidationError({
            layer: "L3_PAYLOAD",
            messageType,
            requestId,
            error: validation.errors.map(e => `${e.path}: ${e.message}`).join(", "),
            details: validation.errors
        });
        return state;
    }
    
    return onValid(validation.data);
}
```

3. Update each message handler to use validation. Example:
```typescript
case "game_state": {
    return validateAndDispatch(
        "game_state",
        envelope.payload,
        envelope.request_id,
        (payload) => ({
            ...state,
            game: gameReducer(
                state.game,
                gameActions.gameState(payload, state.game.identity)
            ),
        })
    );
}
```

### Step 4: Integration Testing (Estimated: 2 days)

**Test Files to Create:**
```
web/src/validation/__tests__/
├── envelope.test.ts         # L1 & L2 tests
├── factory.test.ts          # Validator factory tests
├── payloads/
│   ├── hello.test.ts
│   ├── lobby.test.ts
│   ├── game.test.ts
│   └── game-state.test.ts
```

**Test Scenarios:**
1. Valid envelope passes all layers
2. Invalid JSON fails L1
3. Missing type field fails L1
4. Unknown message type fails L2
5. Invalid payload structure fails L3
6. Missing required payload fields fails L3
7. Wrong payload types fail L3

### Step 5: Documentation (Estimated: 1 day)

**Update Documentation:**
1. Update `AGENTS.md` with validation guidelines
2. Add validation section to component READMEs
3. Create runbook for common validation errors

---

## Testing Strategy

### Unit Tests for Validators

**Validator Test Template:**
```typescript
// validation/__tests__/payloads/game-state.test.ts
import { describe, it, expect } from "vitest";
import { validateGameStatePayload } from "../../payloads/game-state";

describe("validateGameStatePayload", () => {
    it("should validate valid payload", () => {
        const validPayload = {
            turn_number: 1,
            active_player_index: 0,
            players: [
                { index: 0, name: "Alice", coins: 2, card_count: 2, alive: true }
            ]
        };
        
        const result = validateGameStatePayload(validPayload);
        
        expect(result.valid).toBe(true);
        expect(result.data).toEqual(validPayload);
        expect(result.errors).toHaveLength(0);
    });
    
    it("should fail when turn_number is missing", () => {
        const invalidPayload = {
            active_player_index: 0,
            players: []
        };
        
        const result = validateGameStatePayload(invalidPayload);
        
        expect(result.valid).toBe(false);
        expect(result.errors).toContainEqual(
            expect.objectContaining({ path: "turn_number", code: "MISSING_FIELD" })
        );
    });
    
    it("should fail when players is not an array", () => {
        const invalidPayload = {
            turn_number: 1,
            active_player_index: 0,
            players: "not an array"
        };
        
        const result = validateGameStatePayload(invalidPayload);
        
        expect(result.valid).toBe(false);
        expect(result.errors).toContainEqual(
            expect.objectContaining({ path: "players", code: "TYPE_MISMATCH" })
        );
    });
    
    it("should fail when player object is invalid", () => {
        const invalidPayload = {
            turn_number: 1,
            active_player_index: 0,
            players: [{ index: "not a number", name: 123 }]
        };
        
        const result = validateGameStatePayload(invalidPayload);
        
        expect(result.valid).toBe(false);
    });
    
    it("should handle null payload", () => {
        const result = validateGameStatePayload(null);
        
        expect(result.valid).toBe(false);
        expect(result.errors).toContainEqual(
            expect.objectContaining({ code: "NOT_OBJECT" })
        );
    });
    
    it("should handle undefined payload", () => {
        const result = validateGameStatePayload(undefined);
        
        expect(result.valid).toBe(false);
    });
});
```

### Integration Tests for Validation Pipeline

**SocketManager Integration Test:**
```typescript
// network/__tests__/socket-validation.test.ts
import { describe, it, expect, vi, beforeEach } from "vitest";
import { SocketManager } from "../socket";
import * as validation from "@/validation/envelope";

describe("SocketManager Validation", () => {
    let socket: SocketManager;
    let mockWebSocket: any;
    
    beforeEach(() => {
        socket = new SocketManager("ws://localhost:8080/ws");
        mockWebSocket = {
            send: vi.fn(),
            close: vi.fn(),
            readyState: WebSocket.OPEN
        };
        // Inject mock WebSocket
        (socket as any).ws = mockWebSocket;
    });
    
    it("should drop messages with invalid JSON", () => {
        const errorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
        
        mockWebSocket.onmessage?.({ data: "not valid json" });
        
        expect(errorSpy).toHaveBeenCalled();
        errorSpy.mockRestore();
    });
    
    it("should drop messages without type field", () => {
        const errorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
        const handler = vi.fn();
        socket.onMessage(handler);
        
        mockWebSocket.onmessage?.({ data: JSON.stringify({ payload: {} }) });
        
        expect(handler).not.toHaveBeenCalled();
        expect(errorSpy).toHaveBeenCalled();
        errorSpy.mockRestore();
    });
    
    it("should drop messages with unknown type", () => {
        const warnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});
        const handler = vi.fn();
        socket.onMessage(handler);
        
        mockWebSocket.onmessage?.({ 
            data: JSON.stringify({ type: "unknown_type", payload: {} }) 
        });
        
        expect(handler).not.toHaveBeenCalled();
        warnSpy.mockRestore();
    });
    
    it("should pass valid messages to handlers", () => {
        const handler = vi.fn();
        socket.onMessage(handler);
        
        mockWebSocket.onmessage?.({ 
            data: JSON.stringify({ 
                type: "game_state", 
                payload: {
                    turn_number: 1,
                    active_player_index: 0,
                    players: []
                }
            }) 
        });
        
        expect(handler).toHaveBeenCalled();
    });
});
```

### Mock Scenarios for Invalid Messages

**Invalid Message Fixtures:**
```typescript
// validation/__tests__/fixtures/invalid-messages.ts

export const invalidMessages = {
    // L1 Failures
    notAnObject: "just a string",
    nullValue: null,
    arrayInsteadOfObject: ["not", "valid"],
    numberInsteadOfObject: 42,
    
    missingType: {
        payload: { data: "test" }
    },
    
    emptyType: {
        type: "",
        payload: {}
    },
    
    numberType: {
        type: 123,
        payload: {}
    },
    
    // L2 Failures
    unknownType: {
        type: "totally_fake_type",
        payload: {}
    },
    
    // L3 Failures - game_state
    gameStateMissingTurn: {
        type: "game_state",
        payload: {
            active_player_index: 0,
            players: []
        }
    },
    
    gameStateInvalidPlayers: {
        type: "game_state",
        payload: {
            turn_number: 1,
            active_player_index: 0,
            players: "not an array"
        }
    },
    
    gameStateInvalidPlayerObject: {
        type: "game_state",
        payload: {
            turn_number: 1,
            active_player_index: 0,
            players: [
                { index: "not a number", name: 12345 }
            ]
        }
    },
    
    // L3 Failures - hello_ack
    helloAckMissingPlayerId: {
        type: "hello_ack",
        payload: {
            token: "abc123"
        }
    },
    
    helloAckInvalidTypes: {
        type: "hello_ack",
        payload: {
            player_id: 123,
            token: true
        }
    }
};
```

### Coverage Targets

| Component | Target Coverage |
|-----------|----------------|
| validation/envelope.ts | 100% |
| validation/factory.ts | 100% |
| validation/payloads/*.ts | >90% |
| SocketManager validation | >85% |
| Slice validation integration | >80% |
| **Overall validation module** | **>85%** |

**Coverage Configuration:**
```typescript
// vitest.config.ts
export default {
    test: {
        coverage: {
            reporter: ["text", "json", "html"],
            include: ["src/validation/**/*.ts"],
            exclude: ["src/validation/**/*.test.ts", "src/validation/**/__tests__/**"],
            thresholds: {
                statements: 85,
                branches: 80,
                functions: 85,
                lines: 85
            }
        }
    }
};
```

---

## Appendix: Message Type Registry

### Complete Message Type Reference

| Message Type | Direction | Has Payload | Slice | Critical |
|--------------|-----------|-------------|-------|----------|
| `hello` | Outbound | Yes | Connection | Yes |
| `hello_ack` | Inbound | Yes | Connection | Yes |
| `hello_error` | Inbound | Yes | Connection | Yes |
| `lobby_create` | Outbound | No | Lobby | No |
| `lobby_created` | Inbound | Yes | Lobby | No |
| `lobby_list` | Outbound | No | Lobby | No |
| `lobby_list_result` | Inbound | Yes | Lobby | No |
| `lobby_join` | Outbound | Yes | Lobby | No |
| `lobby_joined` | Inbound | Yes | Lobby | No |
| `lobby_start` | Outbound | Yes | Lobby | No |
| `lobby_started` | Inbound | Yes | Game | Yes |
| `lobby_state` | Inbound | Yes | Lobby | No |
| `lobby_error` | Inbound | Yes | Lobby | No |
| `game_config` | Inbound | Yes | Game | No |
| `game_state` | Inbound | Yes | Game | Yes |
| `request_action` | Inbound | Yes | Game | Yes |
| `action` | Outbound | Yes | Game | Yes |
| `challenge_window` | Inbound | Yes | Game | Yes |
| `challenge` | Outbound | Yes | Game | Yes |
| `counter_window` | Inbound | Yes | Game | Yes |
| `counter` | Outbound | Yes | Game | Yes |
| `request_step` | Inbound | Yes | Game | Yes |
| `step_result` | Outbound | Yes | Game | Yes |
| `hand_state` | Inbound | Yes | Game | No |
| `prompt_closed` | Inbound | Yes | Game | No |
| `turn_timer` | Inbound | Yes | Game | No |
| `game_log` | Inbound | Yes | UI | No |
| `game_over` | Inbound | Yes | Game | Yes |
| `player_eliminated` | Inbound | Yes | Game | No |
| `game_paused` | Inbound | Yes | Game | Yes |
| `game_resumed` | Inbound | Yes | Game | Yes |
| `pause_vote` | Outbound | Yes | Game | No |
| `kick_vote` | Outbound | Yes | Game | No |
| `kick_vote_update` | Inbound | Yes | Game | No |
| `player_kicked` | Inbound | Yes | Game | Yes |
| `investigate_result` | Inbound | Yes | UI | No |
| `chat_message` | Inbound | Yes | UI | No |
| `chat` | Outbound | Yes | UI | No |

### Payload Schema Reference

For complete payload schemas, refer to:
- `web/src/network/ws-schema.json` - JSON Schema definitions
- `web/src/types/generated.ts` - TypeScript interfaces
- `backend/docs/ws_contract.md` - Backend contract documentation

---

## Summary

This validation strategy provides a robust, multi-layered defense against malformed WebSocket messages:

1. **L1 (Structure)**: Catches JSON parse errors and missing required fields
2. **L2 (Type Whitelist)**: Prevents processing of unknown message types
3. **L3 (Payload)**: Ensures type-safe payload structures
4. **L4 (Business Logic)**: Validates messages against application state

The dual validation approach distributes responsibility appropriately:
- **SocketManager** handles boundary concerns (L1-L2)
- **Slice handlers** handle type and business concerns (L3-L4)

This architecture ensures:
- **Early rejection** of invalid messages
- **Clear error attribution** (which layer failed)
- **Type safety** throughout the pipeline
- **Graceful degradation** when messages are invalid
- **Comprehensive logging** for debugging and monitoring

---

*Document Version: 1.0*
*Last Updated: 2026-01-30*
*Author: Documentation Engineering Team*
