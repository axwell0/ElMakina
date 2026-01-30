/**
 * Game slice hook
 *
 * Bridge hook that provides access to game slice state from the legacy GameContext.
 * This allows gradual migration to the new sliced state architecture.
 */

import { useContext } from "react";
import { GameContext } from "@/store/gameContext";
import type { GameSliceState } from "@/state/slices";
import type { PauseState, GameIdentity } from "@/state/types";

/**
 * Hook that returns the game slice state from the legacy GameContext.
 * This is a bridge to help migrate components to the new sliced state.
 *
 * @example
 * ```typescript
 * const { game } = useGameSlice();
 * // game.pause, game.identity, game.isConnected
 * ```
 */
export function useGameSlice(): { game: GameSliceState & { isConnected: boolean } } {
  const { state } = useContext(GameContext);

  // Map the legacy flat GameState to the new sliced GameSliceState structure
  const gameSlice: GameSliceState & { isConnected: boolean } = {
    // Core game state from legacy GameState
    currentMatch: state.currentMatch,
    identity: state.identity,
    players: state.players,
    roles: state.roles,
    hand: state.hand,
    activePlayerIndex: state.activePlayerIndex,
    pendingPrompt: state.pendingPrompt,
    promptClosedReason: state.promptClosedReason,
    targeting: state.targeting,
    turnTimer: state.turnTimer,
    pause: state.pause,
    gameOver: state.gameOver,
    // Include isConnected from the connection slice of legacy state
    isConnected: state.isConnected,
  };

  return { game: gameSlice };
}

/**
 * Selector hook for pause state.
 * Returns the current pause state of the game.
 */
export function usePause(): PauseState {
  const { state } = useContext(GameContext);
  return state.pause;
}

/**
 * Selector hook for game identity.
 * Returns the current player's identity in the game.
 */
export function useIdentity(): GameIdentity | null {
  const { state } = useContext(GameContext);
  return state.identity;
}

/**
 * Selector hook for connection status.
 * Returns whether the WebSocket connection is active.
 */
export function useIsConnected(): boolean {
  const { state } = useContext(GameContext);
  return state.isConnected;
}
