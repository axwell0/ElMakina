# Frontend Type Safety & Architecture Migration Plan

**Status**: Planning Phase  
**Last Updated**: January 2026  
**Branch**: experiment  

---

## Overview

This document outlines the roadmap for hardening the ElMakina frontend type system, completing the state management migration, and establishing robust validation boundaries. The goal is to eliminate string-typing, remove technical debt from the hybrid state system, and create a maintainable message handling architecture.

---

## Current State Analysis

### Problems Identified

1. **Type Safety Gap**: Frontend uses `string` for ActionID, Role, and other enums while backend has strict Go constants
2. **Hybrid State**: Both legacy `store/` and new `state/` exist, creating import confusion
3. **Switch Statement Hell**: 400+ line rootReducer with nested switches for message handling
4. **Validation Inconsistency**: AJV validates envelopes but payload types are loosely checked
5. **Naming Divergence**: Mix of PascalCase (Businesswoman) and camelCase (businesswoman) across codebase
6. **Missing Message Handlers**: New slice-based state doesn't handle critical message types (challenge_window, counter_window, request_step, hand_state, prompt_closed) that legacy reducer handles
7. **Incorrect Game Constants**: Frontend constants (TAX_AMOUNT=3, MAX_HAND_SIZE=2, MAX_PLAYERS=6) don't match backend rules
8. **Missing Client-Side Validation**: UI allows targeting players for steal who have <2 coins, causing server-side rejection without user feedback

### Success Criteria

- [ ] All ActionID, Role, and enum types are strict unions (not string)
- [ ] Legacy store/ directory completely removed
- [ ] Message handlers are individually testable units
- [ ] Runtime validation at SocketManager and per-handler levels
- [ ] Identical naming between Go backend and TypeScript frontend
- [ ] Zero TypeScript errors with strictest config

---

## Critical Backend-Frontend Divergences (Priority: URGENT)

These issues represent functional gaps where the frontend will break or behave incorrectly when interacting with the backend.

### Divergence 1: Missing Message Handlers in Slice Architecture

**Severity**: CRITICAL - Breaks game flow  
**Files**: `web/src/state/slices/index.ts`, `web/src/store/gameReducer.ts`, `backend/server/ws/messages.go`

#### Problem
The new slice-based architecture (Phase 2) is incomplete. While the legacy `gameReducer.ts` handles all WebSocket message types including:
- `challenge_window` - Player must decide to challenge or pass
- `counter_window` - Target must decide to counter or pass  
- `request_step` - Multi-step action needs player input (Exchange, Coup discard)
- `hand_state` - Player's current cards
- `prompt_closed` - Action window expired or was closed

The new `rootReducer` in `state/slices/index.ts` only handles:
- `hello_ack`, `lobby_list_result`, `lobby_created`, `lobby_joined`
- `lobby_state`, `lobby_started`, `game_config`, `game_state`
- `request_action`, `game_log`, `game_over`

**Impact**: Players will never see challenge/counter prompts, can't perform multi-step actions (Exchange, Coup), and won't see their hand updates.

#### Solution
Must implement handlers for all missing message types in the new slice architecture:

```typescript
// Add to state/slices/index.ts rootReducer

// In the MESSAGE switch, add cases:
case 'challenge_window':
  return handleChallengeWindow(state, envelope);
  
case 'counter_window':
  return handleCounterWindow(state, envelope);
  
case 'request_step':
  return handleRequestStep(state, envelope);
  
case 'hand_state':
  return handleHandState(state, envelope);
  
case 'prompt_closed':
  return handlePromptClosed(state, envelope);
```

Or better, use the Message Handler Registry pattern (Phase 3) and implement:
- `handlers/game.ts` - `handleChallengeWindow`, `handleCounterWindow`, `handleRequestStep`
- `handlers/ui.ts` - `handlePromptClosed`, `handleHandState`

#### Acceptance Criteria
- [ ] All message types from `ws-contract.ts` have handlers
- [ ] Challenge/counter windows display correctly
- [ ] Multi-step actions (Exchange, Coup) work end-to-end
- [ ] Hand state updates when cards are drawn/discarded
- [ ] Prompt closure reasons displayed to user

---

### Divergence 2: Incorrect Game Constants

**Severity**: HIGH - Misleading rules, potential UI bugs  
**Files**: `web/src/domain/game/rules/constants.ts`, `backend/actions/apply.go`, `backend/engine/game.go`

#### Problem
Frontend constants don't match backend implementation:

| Constant | Frontend Value | Backend Reality | Status |
|----------|---------------|-----------------|---------|
| `TAX_AMOUNT` | 3 | No +3 coins! Tax is global -1 coin for all players with 7+ coins | ❌ WRONG |
| `MAX_HAND_SIZE` | 2 | Variable: 2 cards (5+ players) or 3 cards (2-4 players) | ❌ WRONG |
| `MAX_PLAYERS` | 6 | 9 players allowed | ❌ WRONG |

**Impact**: 
- UI might show incorrect tax information
- Hand display logic may break with 3-card hands
- Lobby creation might artificially limit to 6 players

#### Solution
Update constants to match backend:

```typescript
// constants.ts - remove or fix misleading constants

// REMOVE: Tax doesn't give +3 coins to TaxCollector
// export const TAX_AMOUNT = 3; // ❌ WRONG

// REMOVE: Hand size is variable (2-3), not fixed
// export const MAX_HAND_SIZE = 2; // ❌ WRONG

// FIX: Max players is 9, not 6
export const MAX_PLAYERS = 9; // ✅ CORRECT

// ADD: Variable hand size logic
export function getStartingHandSize(playerCount: number): number {
  return playerCount <= 4 ? 3 : 2; // Matches backend
}
```

Better approach: Don't duplicate backend constants. Use generated types that include these values, or query from backend at runtime.

#### Acceptance Criteria
- [ ] `TAX_AMOUNT` removed (tax doesn't give +3 coins)
- [ ] `MAX_HAND_SIZE` removed (variable, not constant)
- [ ] `MAX_PLAYERS` updated to 9
- [ ] Add `getStartingHandSize()` helper function
- [ ] UI displays correct hand sizes based on player count

---

### Divergence 3: Missing Targeting Validation (Steal Action)

**Severity**: MEDIUM - Poor UX, unnecessary server round-trips  
**Files**: `web/src/components/game/PlayerRing.tsx`, `backend/actions/validators.go`, `web/src/components/ActionPanel.tsx`

#### Problem
When player selects "Steal" action, UI allows clicking ANY alive target, but backend validates that target must have ≥2 coins:

**Backend Validation** (`validators.go:74-78`):
```go
func validateTargetHasCoins(s *state.GameState, targetIndex int, minimum int) error {
  if s.Players[targetIndex].Coins < minimum {
    return fmt.Errorf("target %d does not have %d coins", targetIndex, minimum)
  }
  return nil
}
```

**Frontend Logic** (`PlayerRing.tsx:52-61`):
```typescript
const isTargetable = Boolean(
  identity &&
  targeting?.active &&
  targeting.requestId &&
  targeting.actionId &&
  pendingPrompt?.kind === "action" &&
  pendingPrompt.requestId === targeting.requestId &&
  player.alive &&              // ← Only checks alive
  player.index !== identity.playerIndex
  // ❌ Missing: player.coins >= 2 check
);
```

**Impact**: User can click target with 0-1 coins, action is sent to server, server rejects it, user sees error/confusion.

#### Solution
Add client-side validation for steal targets:

```typescript
// In PlayerRing.tsx
const isTargetable = Boolean(
  identity &&
  targeting?.active &&
  targeting.requestId &&
  pendingPrompt?.kind === "action" &&
  pendingPrompt.requestId === targeting.requestId &&
  player.alive &&
  player.index !== identity.playerIndex &&
  // Add action-specific validation
  (targeting.actionId !== "steal" || (player.coins ?? 0) >= 2) &&
  (targeting.actionId !== "assassinate" || player.alive) // Already checked but explicit
);
```

Better approach: Create a validation helper:

```typescript
// utils/targeting.ts
export function canTargetPlayer(
  actionId: ActionID,
  target: PlayerSnapshot
): boolean {
  if (!target.alive) return false;
  
  switch (actionId) {
    case 'steal':
      return (target.coins ?? 0) >= 2;
    case 'assassinate':
    case 'coup':
    case 'investigate':
    case 'accuse':
      return target.alive; // Just need them alive
    default:
      return true;
  }
}
```

#### Acceptance Criteria
- [ ] Steal targets with <2 coins are visually disabled/non-clickable
- [ ] Helper function `canTargetPlayer()` created
- [ ] All targeting actions use validation helper
- [ ] Visual indicator shows WHY target is invalid (e.g., grayed out with "too poor" tooltip)

---

## Phase 1: Type Generation (Priority: Critical)

### Goal
Generate strict TypeScript types directly from Go structs to ensure frontend and backend contracts never diverge.

### Tool Selection

**Primary**: `github.com/skia-dev/go2ts`
- Library-based approach (no CLI)
- Handles Go structs, enums, and interfaces
- Supports TypeScript union types
- Battle-tested by Skia team

**Alternative**: `github.com/alanshaw/go2ts` (if CLI preferred)
- Simpler setup with command-line interface
- Good for CI/CD integration
- Less flexible for complex types

### Implementation Steps

#### 1.1 Install go2ts

```bash
cd backend
go get github.com/skia-dev/go2ts
```

#### 1.2 Create Generation Script

Create `backend/scripts/generate-types/main.go`:

```go
package main

import (
    "os"
    "github.com/skia-dev/go2ts"
    "ElMakina/backend/models"
)

func main() {
    generator := go2ts.New()
    
    // Add types to generate
    generator.Add(models.ActionID)
    generator.Add(models.Role)
    generator.Add(models.ActionKind)
    generator.Add(models.PlayerAction{})
    generator.Add(models.TargetPayload{})
    generator.Add(models.AccusePayload{})
    // ... all payload types
    
    // Add envelope types
    generator.Add(RequestActionPayload{})
    generator.Add(ChallengeWindowPayload{})
    // ... all envelope payload types
    
    output, err := generator.Render()
    if err != nil {
        panic(err)
    }
    
    err = os.WriteFile("../web/src/types/generated.ts", []byte(output), 0644)
    if err != nil {
        panic(err)
    }
}
```

#### 1.3 Define Generation Targets

**Must Generate**:
- `ActionID` → `type ActionID = 'businesswoman' | 'block_foreign_aid' | ...`
- `Role` → `type Role = 'Businesswoman' | 'TaxCollector' | ...`
- `ActionKind` → `type ActionKind = 'main' | 'counter'`
- All payload structs from `backend/models/payloads.go`
- All envelope payload types from `backend/server/ws/messages.go`

**Output Strategy**:
- **Option A**: Monolithic file (`web/src/types/generated.ts`)
  - Pros: Simple imports, single source of truth
  - Cons: Large file, re-imports entire module when any type changes
- **Option B**: Split by domain (`web/src/types/actions.ts`, `roles.ts`, `envelopes.ts`)
  - Pros: Cleaner imports, granular caching
  - Cons: More complex generation script

**Recommendation**: Start with Option A (monolithic), migrate to Option B if file exceeds 1000 lines.

#### 1.4 NPM Script Integration

Add to root `package.json`:

```json
{
  "scripts": {
    "generate:types": "cd backend && go run scripts/generate-types/main.go",
    "check:types": "npm run generate:types && git diff --exit-code web/src/types/generated.ts"
  }
}
```

#### 1.5 CI/CD Integration

Add to GitHub Actions / CI pipeline:

```yaml
- name: Verify Type Generation
  run: |
    npm run generate:types
    git diff --exit-code web/src/types/generated.ts || 
      (echo "Types are out of sync with Go structs" && exit 1)
```

### Acceptance Criteria

- [ ] `go run backend/scripts/generate-types/main.go` produces `web/src/types/generated.ts`
- [ ] `ActionID` is a strict union type, not `string`
- [ ] `Role` is a strict union type, not `string`
- [ ] All payload types match Go struct field names exactly
- [ ] CI fails if generated types don't match Go structs

---

## Phase 2: Legacy Store Deletion (Priority: High)

### Goal
Complete migration from hybrid state (legacy store/ + new state/) to pure slice-based architecture.

### Current State

**Legacy Store** (`web/src/store/`):
- `types.ts` - Old GameState type (206 lines)
- `gameReducer.ts` - Legacy reducer
- `GameProvider.tsx` - Provider using legacy reducer (already migrated in experiment branch!)
- `gameContext.ts` - React context definition

**New State** (`web/src/state/`):
- `slices/` - Redux-style slice architecture
- `persistence/` - Storage abstraction
- `hooks/` - React hooks for storage sync

### Migration Strategy

#### 2.1 Audit Current Imports

Identify all files importing from legacy store:

```bash
grep -r "from '@/store'" web/src --include="*.ts" --include="*.tsx" | grep -v "state/index.ts"
```

Expected imports to migrate:
- Components using `GameState` type directly
- Utilities importing types from `store/types`
- Replay functionality using old types

#### 2.2 Create Compatibility Layer (Temporary)

Update `web/src/state/index.ts` to re-export legacy types for gradual migration:

```typescript
// Temporary compatibility exports
export type { GameState } from '@/store/types';
export { gameReducer as legacyGameReducer } from '@/store/gameReducer';
```

#### 2.3 Migrate Import Sites

Update each file importing from legacy store:

**Before**:
```typescript
import { GameState } from '@/store/types';
```

**After**:
```typescript
import type { GameState } from '@/state';
// OR use SlicedGameState directly
import type { SlicedGameState } from '@/state/slices';
```

#### 2.4 Component Updates

Key components to update:
- `ActionPanel.tsx` - Uses GameState type
- `GameView.tsx` - Uses GameState type
- `ReplaysPanel.tsx` - Uses replay types
- `ReplayGameView.tsx` - Uses game types
- All game-related components

#### 2.5 Remove Legacy Store

Once all imports migrated:

```bash
rm -rf web/src/store/
```

Update `web/src/state/index.ts`:
- Remove compatibility exports
- Only export new slice-based system

### Acceptance Criteria

- [ ] No imports from `@/store/*` remain (except via `@/state` barrel)
- [ ] `web/src/store/` directory deleted
- [ ] All components use new `SlicedGameState` or converted `GameState`
- [ ] Build passes without errors

---

## Phase 3: Message Handler Registry (Priority: High)

### Goal
Replace 400+ line switch statement with modular, testable message handlers.

### Current Problem

`web/src/state/slices/index.ts` rootReducer:
- 400+ lines
- Nested switch statements
- Hard to test individual handlers
- Merge conflicts when adding message types
- Violates single responsibility principle

### Architecture: Registry Pattern

#### 3.1 Define Handler Interface

Create `web/src/state/handlers/types.ts`:

```typescript
import type { SlicedGameState } from '@/state/slices';
import type { WsEnvelope } from '@/network/ws-contract';

export interface MessageHandlerContext {
  state: SlicedGameState;
  envelope: WsEnvelope;
}

export interface MessageHandlerResult {
  state: SlicedGameState;
  effects?: SideEffect[];
}

export type MessageHandler = (
  context: MessageHandlerContext
) => MessageHandlerResult;

export interface HandlerRegistration {
  messageType: string;
  handler: MessageHandler;
  targetSlice: 'connection' | 'lobby' | 'game' | 'ui';
}
```

#### 3.2 Create Handler Registry

Create `web/src/state/handlers/registry.ts`:

```typescript
import type { HandlerRegistration } from './types';

export const messageHandlers: Record<string, HandlerRegistration> = {
  hello_ack: {
    messageType: 'hello_ack',
    handler: handleHelloAck,
    targetSlice: 'connection',
  },
  lobby_state: {
    messageType: 'lobby_state',
    handler: handleLobbyState,
    targetSlice: 'lobby',
  },
  game_state: {
    messageType: 'game_state',
    handler: handleGameState,
    targetSlice: 'game',
  },
  // ... all message types
};

export function getHandler(messageType: string): HandlerRegistration | undefined {
  return messageHandlers[messageType];
}
```

#### 3.3 Implement Individual Handlers

Create `web/src/state/handlers/connection.ts`:

```typescript
import type { MessageHandler } from './types';
import { connectionActions } from '@/state/slices/connection';
import { uiActions } from '@/state/slices/ui';

export const handleHelloAck: MessageHandler = ({ state, envelope }) => {
  const payload = envelope.payload as { player_id?: string } | undefined;
  
  return {
    state: {
      ...state,
      connection: connectionReducer(
        state.connection,
        connectionActions.helloAck(payload?.player_id ?? null)
      ),
      ui: uiReducer(state.ui, uiActions.updateTimestamp()),
    },
  };
};
```

Similar files for:
- `handlers/lobby.ts` - lobby-related messages
- `handlers/game.ts` - game-related messages
- `handlers/ui.ts` - UI-related messages

#### 3.4 Refactor Root Reducer

New `web/src/state/slices/index.ts`:

```typescript
import { getHandler } from '@/state/handlers/registry';

export function rootReducer(
  state: SlicedGameState,
  action: RootAction
): SlicedGameState {
  // Handle non-message actions (SET_TARGETING, RESET, etc.)
  if (action.type !== 'MESSAGE') {
    return handleLocalAction(state, action);
  }
  
  // Handle WebSocket messages via registry
  const envelope = action.envelope;
  const registration = getHandler(envelope.type);
  
  if (!registration) {
    // Unknown message type - just update timestamp
    return {
      ...state,
      ui: uiReducer(state.ui, uiActions.updateTimestamp()),
    };
  }
  
  const result = registration.handler({ state, envelope });
  return result.state;
}

function handleLocalAction(state: SlicedGameState, action: RootAction): SlicedGameState {
  switch (action.type) {
    case 'CONNECT':
      return {
        ...state,
        connection: connectionReducer(state.connection, connectionActions.connect()),
      };
    // ... other local actions (much smaller switch)
    default:
      return state;
  }
}
```

#### 3.5 Testing Strategy

Each handler is independently testable:

```typescript
// handlers/__tests__/connection.test.ts
describe('handleHelloAck', () => {
  it('should set player ID and complete handshake', () => {
    const state = createMockState();
    const envelope = createMockEnvelope('hello_ack', { player_id: 'player-123' });
    
    const result = handleHelloAck({ state, envelope });
    
    expect(result.state.connection.playerId).toBe('player-123');
    expect(result.state.connection.isHandshakeComplete).toBe(true);
  });
});
```

### Acceptance Criteria

- [ ] Each message handler in separate file
- [ ] Root reducer switch statement < 100 lines
- [ ] All handlers have unit tests
- [ ] Adding new message type requires only: contract + handler + registration (3 files)
- [ ] No change to existing slice reducers (reuse connectionReducer, gameReducer, etc.)

---

## Phase 4: Validation Boundaries (Priority: Medium)

### Goal
Implement defense-in-depth validation at SocketManager (boundary) and per-handler (contextual).

### Strategy: Dual Validation

#### 4.1 SocketManager Validation (Lightweight)

Purpose: Catch malformed messages early, prevent crashes

Update `web/src/network/socket.ts`:

```typescript
// Existing AJV validation stays
// Add envelope structure validation

interface EnvelopeValidator {
  (envelope: unknown): envelope is WsEnvelope;
}

const validateEnvelopeStructure: EnvelopeValidator = (envelope): envelope is WsEnvelope => {
  if (!envelope || typeof envelope !== 'object') return false;
  if (!('type' in envelope) || typeof envelope.type !== 'string') return false;
  // payload is optional, but if present must be object
  if ('payload' in envelope && envelope.payload !== undefined) {
    if (typeof envelope.payload !== 'object' || envelope.payload === null) {
      return false;
    }
  }
  return true;
};

// In handleMessage:
private handleMessage(data: unknown) {
  if (!validateEnvelopeStructure(data)) {
    console.error('[ws] Invalid envelope structure', data);
    return;
  }
  
  // Continue with AJV validation and dispatch
}
```

#### 4.2 Per-Handler Validation (Strict)

Purpose: Ensure payload matches expected schema for specific message type

Create `web/src/state/handlers/validation.ts`:

```typescript
import type { ActionID } from '@/types/generated';
import type { Role } from '@/types/generated';

// Runtime validators for generated types
export function isValidActionID(value: string): value is ActionID {
  const validActions: ActionID[] = [
    'businesswoman', 'block_foreign_aid', 'income', 
    'foreign_aid', 'coup', 'tax', 'tax_business_woman',
    'investigate', 'block_investigate', 'accuse',
    'block_terrorist', 'assassinate', 'steal',
    'block_steal', 'exchange', 'escape', 'pass'
  ];
  return validActions.includes(value as ActionID);
}

export function isValidRole(value: string): value is Role {
  const validRoles: Role[] = [
    'Businesswoman', 'TaxCollector', 'Policewoman',
    'Colonel', 'Terrorist', 'Thief', 'Politician'
  ];
  return validRoles.includes(value as Role);
}

// Payload validators
export function validateRequestActionPayload(payload: unknown): RequestActionPayload | null {
  if (!payload || typeof payload !== 'object') return null;
  
  const p = payload as Record<string, unknown>;
  
  // Required fields
  if (typeof p.actor_index !== 'number') return null;
  
  // Optional fields with validation
  if (p.allowed_actions !== undefined) {
    if (!Array.isArray(p.allowed_actions)) return null;
    if (!p.allowed_actions.every(isValidActionID)) return null;
  }
  
  return payload as RequestActionPayload;
}
```

#### 4.3 Integrate Validation in Handlers

Example: `handleRequestAction`:

```typescript
import { validateRequestActionPayload } from './validation';

export const handleRequestAction: MessageHandler = ({ state, envelope }) => {
  const payload = validateRequestActionPayload(envelope.payload);
  
  if (!payload) {
    console.error('[handler] Invalid request_action payload', envelope.payload);
    return { state }; // Return unchanged state
  }
  
  // Continue with validated payload
  return {
    state: {
      ...state,
      game: gameReducer(
        state.game,
        gameActions.requestAction(payload, envelope.request_id!, state.game.identity)
      ),
    },
  };
};
```

#### 4.4 Error Handling Strategy

**SocketManager Level** (structure violations):
- Log error with connection ID
- Increment error counter metric
- Do not dispatch to reducers
- Connection stays alive

**Handler Level** (payload violations):
- Log error with message type and payload
- Return unchanged state
- UI shows "Invalid message received" toast (optional)
- Continue processing future messages

**Slice Level** (state invariant violations):
- Redux reducers should never crash
- Use defensive programming
- Log invariant violations in development

### Acceptance Criteria

- [ ] SocketManager validates envelope structure before AJV
- [ ] Each handler validates its specific payload type
- [ ] Invalid payloads are logged with context (message type, connection ID)
- [ ] Invalid messages don't crash the app or corrupt state
- [ ] Generated types include runtime validators (isValidActionID, isValidRole)

---

## Phase 5: Naming Standardization (Priority: Low)

### Goal
Eliminate naming confusion between Go backend and TypeScript frontend.

### Current State

| Backend (Go) | Frontend (Current) | Problem |
|--------------|-------------------|---------|
| `ActionID = "businesswoman"` | `actionId: string` | Loose typing |
| `Role = "Businesswoman"` | `role: string` | Loose typing + different case |
| `Payload structs` | `any` or hand-written | Drift risk |

### Standardization Rules

1. **Use Go naming exactly**: If Go uses snake_case, TypeScript uses snake_case
2. **Generate types directly**: No manual translation layers
3. **No mapping functions**: No `toFrontendRole()` or `toBackendAction()`

### Changes Required

#### 5.1 Update Domain Layer

`web/src/domain/game/actions/definitions.ts`:

```typescript
// Before (manual)
export const ACTION_ROLE: Record<string, CardRole> = {
  businesswoman: "Businesswoman",
  // ...
};

// After (generated)
import type { ActionID, Role } from '@/types/generated';

export const ACTION_ROLE: Record<ActionID, Role> = {
  businesswoman: "Businesswoman",
  tax: "TaxCollector",
  // ...
};
```

#### 5.2 Update Components

Remove case transformations:

```typescript
// Before
const label = actionLabel(actionId); // May transform case

// After
import { actionLabels } from '@/domain/game';
const label = actionLabels[actionId]; // Exact match
```

### Acceptance Criteria

- [ ] All ActionID usage uses generated union type
- [ ] All Role usage uses generated union type
- [ ] No case transformation functions exist
- [ ] Identical strings between Go and TypeScript

---

## Implementation Timeline

### Week 0: Critical Divergences (URGENT - Before all other phases)
- [ ] Implement missing message handlers (challenge_window, counter_window, request_step, hand_state, prompt_closed)
- [ ] Fix game constants (remove TAX_AMOUNT, fix MAX_PLAYERS=9, add variable hand size)
- [ ] Add targeting validation for steal action (disable targets with <2 coins)
- [ ] Test critical game flows: challenges, counters, exchanges, coups
- [ ] Verify all WebSocket message types are handled

### Week 1: Type Generation
- [ ] Install go2ts
- [ ] Create generation script
- [ ] Generate initial types
- [ ] Integrate with CI/CD
- [ ] Team review of generated types

### Week 2: Legacy Store Migration
- [ ] Audit all legacy imports
- [ ] Create compatibility layer
- [ ] Migrate components in batches
- [ ] Delete legacy store
- [ ] Full regression test

### Week 3: Message Handler Registry
- [ ] Create handler infrastructure
- [ ] Migrate 5 most common message types
- [ ] Write tests for handlers
- [ ] Team review of pattern
- [ ] Migrate remaining message types

### Week 4: Validation & Polish
- [ ] Implement SocketManager validation
- [ ] Add per-handler validation
- [ ] Standardize naming
- [ ] Write documentation
- [ ] Final integration test

---

## Testing Strategy

### Unit Tests
- Each message handler independently tested
- Validation functions tested with valid/invalid inputs
- Reducer integration tests

### Integration Tests
- End-to-end WebSocket message flow
- State transitions for complete game scenarios
- Replay functionality

### Type Safety
- `tsc --noEmit` with strictest config
- No `any` types (enforced via ESLint)
- Generated types must compile without errors

---

## Rollback Plan

If critical issues discovered:

1. **Type Generation Issues**: Revert to hand-written types, fix generation script
2. **State Migration Issues**: Restore legacy store from git, gradual migration
3. **Handler Registry Issues**: Keep switch statement temporarily, refactor incrementally
4. **Validation Issues**: Disable strict validation, log and monitor

---

## Success Metrics

After completion:

- **Type Coverage**: 100% of backend types have frontend equivalents
- **Test Coverage**: >80% for message handlers
- **Bundle Size**: No increase (or decrease after removing legacy code)
- **Build Time**: No significant increase
- **Developer Experience**: 
  - Autocomplete works for all ActionIDs and Roles
  - Type errors caught at compile time
  - Clear error messages for runtime validation failures

---

## Appendix A: File Structure

```
web/src/
├── types/
│   ├── generated.ts          # Auto-generated from Go
│   └── index.ts              # Barrel exports
├── state/
│   ├── handlers/
│   │   ├── types.ts          # Handler interfaces
│   │   ├── registry.ts       # Handler registration
│   │   ├── validation.ts     # Runtime validators
│   │   ├── connection.ts     # Connection handlers
│   │   ├── lobby.ts          # Lobby handlers
│   │   ├── game.ts           # Game handlers
│   │   └── ui.ts             # UI handlers
│   ├── slices/
│   │   ├── index.ts          # Root reducer (refactored)
│   │   ├── connection/
│   │   ├── lobby/
│   │   ├── game/
│   │   └── ui/
│   └── persistence/
├── domain/
│   └── game/
│       └── actions/
│           └── definitions.ts # Uses generated types
└── [store/ deleted]
```

## Appendix B: Migration Checklist

### Before Starting
- [ ] Backup current branch
- [ ] Notify team of migration
- [ ] Set up feature flags (if needed)

### During Migration
- [ ] Run type generation daily to catch drift
- [ ] Update ARCHITECTURE.md with new patterns
- [ ] Pair review for complex handler logic

### After Completion
- [ ] Update onboarding docs
- [ ] Team presentation on new patterns
- [ ] Monitor error rates for 1 week
- [ ] Collect developer feedback

---

## Questions & Decisions

**Q: Should we keep the generated types in version control?**  
A: Yes, to avoid build-time Go dependency for frontend-only developers. CI checks prevent drift.

**Q: What if go2ts doesn't support a specific Go feature?**  
A: Use struct tags or manual overrides in generation script. Document workarounds.

**Q: How do we handle breaking changes in backend types?**  
A: Backend changes trigger CI failure. Frontend team reviews before merging backend PR.

**Q: Should validation be sync or async?**  
A: Sync for performance. Async validation only if external API calls needed (rare).

---

## References

- [go2ts Documentation](https://pkg.go.dev/github.com/skia-dev/go2ts)
- [TypeScript Strict Mode](https://www.typescriptlang.org/tsconfig#strict)
- [Redux Style Guide](https://redux.js.org/style-guide/)
- ElMakina Backend Architecture (backend/ARCHITECTURE.md)
