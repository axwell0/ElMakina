/**
 * State management exports
 *
 * Centralized exports for all state management modules.
 */

// Persistence layer
export * from "./persistence";

// Hooks
export * from "./hooks";

// Slice-based store (new architecture)
export * from "./slices";

// Types (moved from @/store/types to @/state/types)
export * from "./types";

// Legacy store (maintained for backwards compatibility)
// Note: These use @/store/ path alias, not relative paths
export { gameReducer } from "@/store/gameReducer";
export { GameContext, useGame } from "@/store/gameContext";
export { GameProvider } from "@/store/GameProvider";
