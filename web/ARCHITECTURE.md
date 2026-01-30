# ElMakina Web Architecture

## Overview

This document describes the modular architecture of the ElMakina web frontend. The codebase has been refactored to follow modern TypeScript/React best practices with clear separation of concerns, testability, and maintainability.

## Directory Structure

```
web/src/
├── api/                    # External communication layer
├── domain/                 # Pure business logic
├── state/                  # State management
├── ui/                     # UI components and utilities
├── config/                 # Configuration
├── store/                  # Legacy store (backwards compatible)
├── network/                # WebSocket client
├── lib/                    # Legacy utilities (backwards compatible)
└── components/             # React components
```

## Architecture Layers

### 1. Domain Layer (`domain/`)

**Purpose**: Pure business logic with no React, WebSocket, or storage dependencies.

**Structure**:
```
domain/
├── game/
│   ├── cards/             # Card roles and images
│   ├── actions/           # Game actions and helpers
│   ├── rules/             # Game constants
│   └── index.ts           # Barrel exports
├── replay/                # Replay domain (future)
└── index.ts
```

**Key Principles**:
- Pure functions only
- No side effects
- Fully testable without mocks
- Single source of truth for game rules

**Usage**:
```typescript
import { CardRole, getCardImage, getRoleForAction } from '@/domain/game';
```

### 2. API Layer (`api/`)

**Purpose**: Abstract external communication (HTTP and WebSocket).

**Structure**:
```
api/
├── http/                  # HTTP client and endpoints
│   ├── client/           # Fetch wrapper
│   └── endpoints/        # API endpoint definitions
├── ws/                    # WebSocket abstraction
│   ├── client/           # Connection management
│   ├── handlers/         # Message handlers
│   └── validation/       # Schema validation
└── index.ts
```

**Key Principles**:
- Centralized error handling
- Type-safe request/response
- Interceptor support
- Easy to mock for testing

### 3. State Layer (`state/`)

**Purpose**: Centralized state management with slice-based architecture.

**Structure**:
```
state/
├── persistence/          # Storage abstraction
│   ├── types.ts
│   ├── keys.ts
│   ├── local-storage.ts
│   ├── typed-storage.ts
│   └── index.ts
├── slices/              # Redux-style slices
│   ├── connection/      # Connection state
│   ├── lobby/           # Lobby state
│   ├── game/            # Game state
│   ├── ui/              # UI state
│   └── index.ts         # Root reducer
├── hooks/               # React hooks
│   ├── useStorageSync.ts
│   └── index.ts
└── index.ts
```

**Slice Pattern**:
Each slice contains:
- `types.ts` - State interface and action types
- `reducer.ts` - Pure reducer function
- `actions.ts` - Action creators
- `selectors.ts` - Memoized selectors (optional)
- `index.ts` - Barrel exports

**Example**:
```typescript
// Connection slice
export interface ConnectionState {
  isConnected: boolean;
  isHandshakeComplete: boolean;
  playerId: string | null;
}

export type ConnectionAction =
  | { type: 'CONNECT' }
  | { type: 'DISCONNECT'; inGame: boolean }
  | { type: 'HELLO_ACK'; playerId: string | null };

export function connectionReducer(
  state: ConnectionState,
  action: ConnectionAction
): ConnectionState {
  // Implementation
}
```

**Root Reducer**:
The `rootReducer` in `slices/index.ts` composes all slices and handles message routing:
```typescript
export function rootReducer(
  state: SlicedGameState,
  action: RootAction
): SlicedGameState {
  switch (action.type) {
    case 'CONNECT':
      return {
        ...state,
        connection: connectionReducer(
          state.connection,
          connectionActions.connect()
        ),
      };
    // ... other cases
  }
}
```

### 4. UI Layer (`ui/`)

**Purpose**: UI-specific utilities and hooks.

**Structure**:
```
ui/
├── utils/               # UI utilities
│   ├── cn.ts           # Tailwind class merge
│   └── index.ts
└── hooks/               # React hooks
```

### 5. Storage Persistence (`state/persistence/`)

**Purpose**: Abstract browser storage with type safety.

**Key Components**:
- `StorageAdapter` interface - Abstract storage operations
- `LocalStorageAdapter` - localStorage implementation
- `TypedStorage` - Type-safe storage wrapper
- `STORAGE_KEYS` - Centralized key definitions

**Usage**:
```typescript
import { localStorageAdapter, STORAGE_KEYS } from '@/state/persistence';

// Store value
localStorageAdapter.setItem(STORAGE_KEYS.theme, 'dark');

// Retrieve value
const theme = localStorageAdapter.getItem(STORAGE_KEYS.theme);

// Type-safe storage
const replayStorage = createTypedStorage(
  localStorageAdapter,
  STORAGE_KEYS.replayHistory,
  jsonSerializer<ReplayEntry[]>()
);
```

## State Flow

### Initialization
1. `GameProvider` mounts with `initialSlicedState`
2. `loadPersistedState` hydrates from localStorage
3. Socket connection established

### Message Handling
1. WebSocket message arrives
2. `socket.onMessage` dispatches `{ type: 'MESSAGE', envelope }`
3. `rootReducer` routes to appropriate slice
4. Slice reducer updates state
5. `toGameState` converts to flat structure
6. React re-renders with new state

### Action Flow
1. Component calls `dispatch(action)`
2. `rootReducer` handles action
3. Relevant slice reducer updates state
4. Side effects (storage sync) execute in useEffect

## Migration Guide

### From Old Imports
```typescript
// Old (still works)
import { cardImageForRole } from '@/lib/cards';

// New (recommended)
import { getCardImage } from '@/domain/game';
```

### Adding New State
1. Define state interface in slice `types.ts`
2. Add actions to slice action union type
3. Implement reducer case
4. Export action creator
5. Add to root reducer if cross-cutting

### Testing Slices
```typescript
import { connectionReducer, connectionActions } from '@/state/slices';

describe('connection slice', () => {
  it('should handle CONNECT', () => {
    const state = connectionReducer(
      initialConnectionState,
      connectionActions.connect()
    );
    expect(state.isConnected).toBe(true);
  });
});
```

## Backwards Compatibility

The architecture maintains backwards compatibility:
- Old `GameState` type still exported from `@/store/types`
- `toGameState()` converts sliced state to flat structure
- All existing components work without changes
- Legacy reducers available but not used

## Best Practices

### File Organization
- Keep files under 150 lines
- One logical concern per file
- Use barrel exports (`index.ts`)
- Prefer absolute imports (`@/domain/game`)

### Type Safety
- Explicit return types on exports
- No `any` types
- Use discriminated unions for actions
- Validate at boundaries

### Testing
- Test domain logic first (pure functions)
- Test reducers with action creators
- Mock storage adapters, not localStorage
- Test component integration with real store

### Error Handling
- Use storage adapter error handling
- Log warnings for storage failures
- Graceful degradation when storage unavailable
- Type-safe error actions

## Common Patterns

### Creating a New Slice
```typescript
// 1. Define state
export interface MySliceState {
  data: string[];
}

// 2. Define actions
export type MyAction =
  | { type: 'ADD_DATA'; item: string }
  | { type: 'CLEAR_DATA' };

// 3. Create reducer
export function myReducer(
  state: MySliceState,
  action: MyAction
): MySliceState {
  switch (action.type) {
    case 'ADD_DATA':
      return { ...state, data: [...state.data, action.item] };
    default:
      return state;
  }
}

// 4. Export actions
export const myActions = {
  addData: (item: string): MyAction => ({ type: 'ADD_DATA', item }),
} as const;
```

### Adding to Root Reducer
```typescript
// In slices/index.ts
export interface SlicedGameState {
  // ... existing slices
  mySlice: MySliceState;
}

export const initialSlicedState: SlicedGameState = {
  // ... existing initial states
  mySlice: initialMyState,
};

export function rootReducer(state, action) {
  switch (action.type) {
    case 'ADD_DATA':
      return {
        ...state,
        mySlice: myReducer(state.mySlice, action),
      };
  }
}
```

## Future Improvements

### Planned
- [ ] Add unit tests for all slices
- [ ] Create React hooks for slice access
- [ ] HTTP API abstraction layer
- [ ] Component directory reorganization

### Considered
- [ ] Redux Toolkit integration
- [ ] Zustand alternative state management
- [ ] React Query for server state
- [ ] Feature-based code splitting

## Troubleshooting

### Type Errors After Migration
If you see type errors in components:
1. Check import is using `RootAction` not old `Action`
2. Ensure component props use `React.Dispatch<RootAction>`
3. Run `tsc --noEmit` to verify

### State Not Persisting
1. Check `STORAGE_KEYS` has the key
2. Verify `localStorageAdapter.isAvailable()` returns true
3. Look for errors in console

### Actions Not Working
1. Check action type is in `RootAction` union
2. Verify root reducer handles the action
3. Confirm slice reducer updates state correctly

## References

- [Redux Style Guide](https://redux.js.org/style-guide/)
- [React TypeScript Cheatsheet](https://react-typescript-cheatsheet.netlify.app/)
- [Module Pattern in TypeScript](https://khalilstemmler.com/articles/typescript-domain-driven-design/)

---

## Quick Reference

### Import Map
```typescript
// Domain
import { CardRole, getCardImage } from '@/domain/game';

// State
import { useStorageSync } from '@/state/hooks';
import { rootReducer, gameActions } from '@/state/slices';
import { localStorageAdapter } from '@/state/persistence';

// UI
import { cn } from '@/ui/utils';

// Legacy (still works)
import { GameState } from '@/store/types';
import { socket } from '@/network/socket';
```

### Action Types
- `CONNECT` / `DISCONNECT` - Connection state
- `MESSAGE` - WebSocket message wrapper
- `SET_SFX_MUTED` / `SET_THEME` - UI preferences
- `CLEAR_PROMPT` / `CLEAR_TARGETING` - UI clearing
- `RESET` - Full state reset

### Storage Keys
- `STORAGE_KEYS.reconnectToken`
- `STORAGE_KEYS.nickname`
- `STORAGE_KEYS.theme`
- `STORAGE_KEYS.sfxMuted`
- `STORAGE_KEYS.replayHistory`

---

*Last Updated: January 2026*
*Architecture Version: 2.0*
