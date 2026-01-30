import type { GameState } from "@/store/types";
import type { WsEnvelope } from "@/network/socket";

// Import slices
import {
  type ConnectionState,
  connectionReducer,
  connectionActions,
  initialConnectionState,
} from "./connection";

import {
  type LobbySliceState,
  lobbyReducer,
  lobbyActions,
  initialLobbyState,
} from "./lobby";

import {
  type GameSliceState,
  gameReducer,
  gameActions,
  initialGameSliceState,
} from "./game";

import {
  type UISliceState,
  uiReducer,
  uiActions,
  initialUIState,
} from "./ui";

// Re-export slice types and actions
export type { ConnectionState } from "./connection";
export type { LobbySliceState } from "./lobby";
export type { GameSliceState } from "./game";
export type { UISliceState } from "./ui";

// Combined state type
export interface SlicedGameState {
  connection: ConnectionState;
  lobby: LobbySliceState;
  game: GameSliceState;
  ui: UISliceState;
}

// Initial state
export const initialSlicedState: SlicedGameState = {
  connection: initialConnectionState,
  lobby: initialLobbyState,
  game: initialGameSliceState,
  ui: initialUIState,
};

// Helper to convert sliced state back to flat GameState (for backwards compatibility)
export function toGameState(sliced: SlicedGameState): GameState {
  return {
    // Connection
    isConnected: sliced.connection.isConnected,
    isHandshakeComplete: sliced.connection.isHandshakeComplete,
    playerId: sliced.connection.playerId,
    error: sliced.connection.error,
    connectionLostAt: sliced.connection.connectionLostAt,

    // Lobby
    lobbies: sliced.lobby.lobbies,
    currentLobby: sliced.lobby.currentLobby,

    // Game
    currentMatch: sliced.game.currentMatch,
    identity: sliced.game.identity,
    players: sliced.game.players,
    roles: sliced.game.roles,
    hand: sliced.game.hand,
    activePlayerIndex: sliced.game.activePlayerIndex,
    pendingPrompt: sliced.game.pendingPrompt,
    promptClosedReason: sliced.game.promptClosedReason,
    targeting: sliced.game.targeting,
    turnTimer: sliced.game.turnTimer,
    pause: sliced.game.pause,
    gameOver: sliced.game.gameOver,

    // UI
    sfxMuted: sliced.ui.sfxMuted,
    theme: sliced.ui.theme,
    logs: sliced.ui.logs,
    chat: sliced.ui.chat,
    investigateResult: sliced.ui.investigateResult,
    eliminationToast: sliced.ui.eliminationToast,
    replayHistory: sliced.ui.replayHistory,
    lastUpdateTs: sliced.ui.lastUpdateTs,
    mockScenario: sliced.ui.mockScenario,
  };
}

// Action type (union of all slice actions + legacy actions)
export type RootAction =
  | { type: "CONNECT" }
  | { type: "DISCONNECT" }
  | { type: "MESSAGE"; envelope: WsEnvelope }
  | { type: "ERROR"; error: string }
  | { type: "CLEAR_ERROR" }
  | { type: "SET_TARGETING"; actionId: string; requestId: string }
  | { type: "SET_TARGET_SELECTED"; targetIndex: number }
  | { type: "CLEAR_TARGETING" }
  | { type: "SET_SFX_MUTED"; muted: boolean }
  | { type: "SET_THEME"; theme: "light" | "dark" }
  | { type: "CLEAR_PROMPT" }
  | { type: "CLEAR_INVESTIGATE" }
  | { type: "CLEAR_ELIMINATION_TOAST" }
  | { type: "RESET" };

// Root reducer that delegates to slice reducers
export function rootReducer(
  state: SlicedGameState = initialSlicedState,
  action: RootAction
): SlicedGameState {
  switch (action.type) {
    case "CONNECT":
      return {
        ...state,
        connection: connectionReducer(state.connection, connectionActions.connect()),
      };

    case "DISCONNECT": {
      const inGame = state.lobby.currentLobby?.status === "in_game";
      return {
        ...state,
        connection: connectionReducer(
          state.connection,
          connectionActions.disconnect(inGame)
        ),
        // If not in game, clear lobby and game state
        ...(inGame
          ? {}
          : {
              lobby: lobbyReducer(state.lobby, lobbyActions.reset()),
              game: gameReducer(state.game, gameActions.reset()),
            }),
      };
    }

    case "ERROR":
      return {
        ...state,
        connection: connectionReducer(
          state.connection,
          connectionActions.error(action.error)
        ),
      };

    case "CLEAR_ERROR":
      return {
        ...state,
        connection: connectionReducer(
          state.connection,
          connectionActions.clearError()
        ),
      };

    case "MESSAGE": {
      const envelope = action.envelope;

      // Handle different message types by delegating to appropriate slices
      switch (envelope.type) {
        case "hello_ack": {
          const payload = envelope.payload as { player_id?: string } | undefined;
          return {
            ...state,
            connection: connectionReducer(
              state.connection,
              connectionActions.helloAck(payload?.player_id ?? null)
            ),
            ui: uiReducer(state.ui, uiActions.updateTimestamp()),
          };
        }

        case "hello_error": {
          const payload = envelope.payload as { error?: string } | undefined;
          return {
            ...state,
            connection: connectionReducer(
              state.connection,
              connectionActions.helloError(payload?.error ?? null)
            ),
            lobby: lobbyReducer(state.lobby, lobbyActions.reset()),
            game: gameReducer(state.game, gameActions.reset()),
          };
        }

        case "lobby_list_result": {
          const payload = envelope.payload as Parameters<typeof lobbyActions.listResult>[0];
          return {
            ...state,
            lobby: lobbyReducer(
              state.lobby,
              lobbyActions.listResult(payload)
            ),
            ui: uiReducer(state.ui, uiActions.updateTimestamp()),
          };
        }

        case "lobby_created":
        case "lobby_joined":
          return {
            ...state,
            lobby: lobbyReducer(state.lobby, lobbyActions.joined()),
            connection: connectionReducer(
              state.connection,
              connectionActions.clearError()
            ),
          };

        case "lobby_state": {
          const payload = envelope.payload as {
            lobby_id: string;
            leader_nick: string;
            leader_id?: string;
            player_nicks?: string[];
            player_ids?: string[];
            player_avatars?: string[];
            player_count: number;
            status: "open" | "in_game" | "closed";
          };
          return {
            ...state,
            lobby: lobbyReducer(
              state.lobby,
              lobbyActions.state(payload, state.connection.playerId)
            ),
            ui: uiReducer(state.ui, uiActions.updateTimestamp()),
          };
        }

        case "lobby_started": {
          const payload = envelope.payload as {
            lobby_id: string;
            match_id?: string;
            player_index: number;
            player_count: number;
            player_names: string[];
            player_avatars?: string[];
            index_mapping?: Record<string, number>;
          };
          return {
            ...state,
            game: gameReducer(
              state.game,
              gameActions.lobbyStarted(
                payload,
                state.connection.playerId,
                state.lobby.currentLobby
              )
            ),
            ui: uiReducer(state.ui, uiActions.updateTimestamp()),
          };
        }

        case "game_config": {
          const payload = envelope.payload as { roles?: string[] } | undefined;
          return {
            ...state,
            game: gameReducer(state.game, gameActions.gameConfig(payload)),
          };
        }

        case "game_state": {
          const payload = envelope.payload as Parameters<typeof gameActions.gameState>[0];
          return {
            ...state,
            game: gameReducer(
              state.game,
              gameActions.gameState(payload, state.game.identity)
            ),
          };
        }

        case "request_action": {
          const payload = envelope.payload as {
            actor_index: number;
            allowed_actions?: string[];
          };
          return {
            ...state,
            game: gameReducer(
              state.game,
              gameActions.requestAction(
                payload,
                envelope.request_id!,
                state.game.identity
              )
            ),
          };
        }

        case "game_log": {
          const payload = envelope.payload as {
            turn: number;
            scope: "public" | "private";
            message: string;
          };
          return {
            ...state,
            ui: uiReducer(state.ui, uiActions.gameLog(payload)),
          };
        }

        case "game_over": {
          const payload = envelope.payload as {
            winner_index: number;
            winner_name: string;
          };
          // Add to replay history
          if (state.game.currentMatch && state.connection.playerId) {
            const replayEntry = {
              matchId: state.game.currentMatch.matchId,
              lobbyId: state.game.currentMatch.lobbyId,
              playerId: state.connection.playerId,
              playerNames: state.game.currentMatch.playerNames,
              participantIds: state.game.currentMatch.participantIds,
              winnerName: payload.winner_name,
              winnerIndex: payload.winner_index,
              endedAt: Date.now(),
            };
            return {
              ...state,
              game: gameReducer(
                state.game,
                gameActions.gameOver(payload.winner_index, payload.winner_name)
              ),
              ui: uiReducer(state.ui, uiActions.addReplay(replayEntry)),
            };
          }
          return {
            ...state,
            game: gameReducer(
              state.game,
              gameActions.gameOver(payload.winner_index, payload.winner_name)
            ),
          };
        }

        case "challenge_window": {
          const payload = envelope.payload as {
            action_id: string;
            actor_index: number;
            claimed_role: string;
            eligible: boolean;
            kind: string;
            prompt?: string;
            target_index?: number;
            timeout_ms?: number;
          };
          return {
            ...state,
            game: gameReducer(
              state.game,
              gameActions.challengeWindow(payload, envelope.request_id!)
            ),
            ui: uiReducer(state.ui, uiActions.updateTimestamp()),
          };
        }

        case "counter_window": {
          const payload = envelope.payload as {
            action_id: string;
            actor_index: number;
            allowed_actions?: string[];
            eligible: boolean;
            prompt?: string;
            target_index?: number;
            timeout_ms?: number;
          };
          return {
            ...state,
            game: gameReducer(
              state.game,
              gameActions.counterWindow(payload, envelope.request_id!)
            ),
            ui: uiReducer(state.ui, uiActions.updateTimestamp()),
          };
        }

        case "request_step": {
          const payload = envelope.payload as {
            prompt?: string;
            step: {
              context: string;
              count: number;
              options: string[];
            };
          };
          return {
            ...state,
            game: gameReducer(
              state.game,
              gameActions.requestStep(payload, envelope.request_id!)
            ),
            ui: uiReducer(state.ui, uiActions.updateTimestamp()),
          };
        }

        case "hand_state": {
          const payload = envelope.payload as {
            hand: string[];
            player_index: number;
          };
          return {
            ...state,
            game: gameReducer(
              state.game,
              gameActions.handState(payload, state.game.identity)
            ),
            ui: uiReducer(state.ui, uiActions.updateTimestamp()),
          };
        }

        case "prompt_closed": {
          const payload = envelope.payload as { reason: string };
          return {
            ...state,
            game: gameReducer(
              state.game,
              gameActions.promptClosed(payload, envelope.request_id!, state.game.pendingPrompt)
            ),
            ui: uiReducer(state.ui, uiActions.updateTimestamp()),
          };
        }

        case "turn_timer": {
          const payload = envelope.payload as {
            active_player_index: number;
            duration_ms: number;
            state: string;
            turn_number: number;
          };
          return {
            ...state,
            game: gameReducer(
              state.game,
              gameActions.turnTimer(payload)
            ),
          };
        }

        case "player_eliminated": {
          const payload = envelope.payload as {
            player_index: number;
            reason: string;
            turn: number;
          };
          const idx = payload.player_index;
          const turn = payload.turn;
          const playerName = state.game.players.find((p) => p.index === idx)?.name ?? "";
          const toast = {
            playerIndex: idx,
            playerName,
            reason: payload.reason,
            turn,
            id: `${idx}-${turn}-${Date.now()}`,
          };
          return {
            ...state,
            game: gameReducer(
              state.game,
              gameActions.playerEliminated(payload)
            ),
            ui: uiReducer(state.ui, uiActions.elimination(toast)),
          };
        }

        default:
          // Unknown message type - just update timestamp
          return {
            ...state,
            ui: uiReducer(state.ui, uiActions.updateTimestamp()),
          };
      }
    }

    case "SET_TARGETING":
      return {
        ...state,
        game: gameReducer(
          state.game,
          gameActions.setTargeting(action.actionId, action.requestId)
        ),
      };

    case "SET_TARGET_SELECTED":
      return {
        ...state,
        game: gameReducer(
          state.game,
          gameActions.setTargetSelected(action.targetIndex)
        ),
      };

    case "CLEAR_TARGETING":
      return {
        ...state,
        game: gameReducer(state.game, gameActions.clearTargeting()),
      };

    case "SET_SFX_MUTED":
      return {
        ...state,
        ui: uiReducer(state.ui, uiActions.setSfxMuted(action.muted)),
      };

    case "SET_THEME":
      return {
        ...state,
        ui: uiReducer(state.ui, uiActions.setTheme(action.theme)),
      };

    case "CLEAR_PROMPT":
      return {
        ...state,
        game: gameReducer(state.game, gameActions.clearPrompt()),
      };

    case "CLEAR_INVESTIGATE":
      return {
        ...state,
        ui: uiReducer(state.ui, uiActions.clearInvestigate()),
      };

    case "CLEAR_ELIMINATION_TOAST":
      return {
        ...state,
        ui: uiReducer(state.ui, uiActions.clearEliminationToast()),
      };

    case "RESET":
      return {
        connection: connectionReducer(
          state.connection,
          connectionActions.reset()
        ),
        lobby: lobbyReducer(state.lobby, lobbyActions.reset()),
        game: gameReducer(state.game, gameActions.reset()),
        ui: uiReducer(state.ui, uiActions.reset(true)),
      };

    default:
      return state;
  }
}

// Export action creators for convenience
export { connectionActions, lobbyActions, gameActions, uiActions };
