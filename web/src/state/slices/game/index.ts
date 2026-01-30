/**
 * Game state slice
 *
 * Manages active game state including match, players, hand, and prompts.
 */

import type {
  ActiveMatch,
  GameIdentity,
  HandCard,
  PauseState,
  PlayerSnapshot,
  Prompt,
  TargetingState,
  TurnTimerState,
} from "@/store/types";



export interface GameSliceState {
  currentMatch: ActiveMatch | null;
  identity: GameIdentity | null;
  players: PlayerSnapshot[];
  roles: string[];
  hand: HandCard[];
  activePlayerIndex: number | null;
  pendingPrompt: Prompt | null;
  promptClosedReason: string | null;
  targeting: TargetingState | null;
  turnTimer: TurnTimerState | null;
  pause: PauseState;
  gameOver: { winnerIndex: number; winnerName: string } | null;
}

export const initialGameSliceState: GameSliceState = {
  currentMatch: null,
  identity: null,
  players: [],
  roles: [],
  hand: [],
  activePlayerIndex: null,
  pendingPrompt: null,
  promptClosedReason: null,
  targeting: null,
  turnTimer: null,
  pause: { status: "inactive" },
  gameOver: null,
};

// Payload types
interface LobbyStartedPayload {
  lobby_id: string;
  match_id?: string;
  player_index: number;
  player_count: number;
  player_names: string[];
  player_avatars?: string[];
  index_mapping?: Record<string, number>;
}

interface GameStatePayload {
  players?: Array<{
    index: number;
    name: string;
    alive: boolean;
    coins: number;
    card_count: number;
    avatar?: string;
  }>;
  active_player_index: number;
}

interface GameConfigPayload {
  roles?: string[];
}

interface RequestActionPayload {
  actor_index: number;
  allowed_actions?: string[];
}

// Current lobby info needed for game start
interface CurrentLobbyInfo {
  lobbyId: string;
  playerIds?: string[];
  playerNicks: string[];
  playerCount: number;
  leaderNick?: string;
  status?: "open" | "in_game" | "closed";
}

export type GameAction =
  | { type: "LOBBY_STARTED"; payload: LobbyStartedPayload; currentPlayerId: string | null; currentLobby: CurrentLobbyInfo | null }
  | { type: "GAME_CONFIG"; payload: GameConfigPayload | undefined }
  | { type: "GAME_STATE"; payload: GameStatePayload | undefined; currentIdentity: GameIdentity | null }
  | { type: "REQUEST_ACTION"; payload: RequestActionPayload; requestId: string; currentIdentity: GameIdentity | null }
  | { type: "CLEAR_PROMPT" }
  | { type: "SET_TARGETING"; actionId: string; requestId: string }
  | { type: "SET_TARGET_SELECTED"; targetIndex: number }
  | { type: "CLEAR_TARGETING" }
  | { type: "GAME_OVER"; winnerIndex: number; winnerName: string }
  | { type: "RESET" };

export function gameReducer(
  state: GameSliceState,
  action: GameAction
): GameSliceState {
  switch (action.type) {
    case "LOBBY_STARTED": {
      const payload = action.payload;
      const avatars = payload.player_avatars || [];
      const players: PlayerSnapshot[] = (payload.player_names || []).map(
        (name: string, index: number) => ({
          index,
          name,
          alive: true,
          coins: null,
          cardCount: null,
          avatar: avatars[index] || "",
        })
      );

      const currentLobby = action.currentLobby ?? {
        lobbyId: payload.lobby_id,
        leaderNick: "",
        playerNicks: payload.player_names || [],
        playerCount: payload.player_count,
        status: "in_game" as const,
      };

      const participantIds = payload.index_mapping
        ? Object.keys(payload.index_mapping)
        : currentLobby.playerIds || [];
      const matchId = payload.match_id || payload.lobby_id;

      return {
        ...state,
        identity: {
          playerId: action.currentPlayerId || "unknown",
          playerIndex: payload.player_index,
          playerNames: payload.player_names,
        },
        players,
        currentMatch: {
          matchId,
          lobbyId: payload.lobby_id,
          playerNames: payload.player_names,
          participantIds,
        },
        activePlayerIndex: 0,
        hand: [],
        roles: [],
        pendingPrompt: null,
        promptClosedReason: null,
        targeting: null,
        gameOver: null,
      };
    }

    case "GAME_CONFIG": {
      const payload = action.payload;
      return {
        ...state,
        roles: payload?.roles || [],
      };
    }

    case "GAME_STATE": {
      const payload = action.payload;
      const incomingPlayers = payload?.players || [];
      const updatedPlayers = incomingPlayers.map((p) => {
        const previous = state.players.find((existing) => existing.index === p.index);
        const prevCoins = previous?.coins ?? p.coins ?? 0;
        return {
          index: p.index,
          name: p.name,
          alive: p.alive,
          coins: p.coins,
          prevCoins,
          cardCount: p.card_count,
          avatar: p.avatar || previous?.avatar || "",
        };
      });

      return {
        ...state,
        players: updatedPlayers,
        activePlayerIndex: payload?.active_player_index ?? state.activePlayerIndex,
      };
    }

    case "REQUEST_ACTION": {
      const payload = action.payload;
      // Only show prompt if it's for the current player
      if (action.currentIdentity && payload.actor_index === action.currentIdentity.playerIndex) {
        return {
          ...state,
          activePlayerIndex: payload.actor_index,
          pendingPrompt: {
            kind: "action",
            requestId: action.requestId,
            allowedActions: payload.allowed_actions || [],
          },
          promptClosedReason: null,
        };
      }
      return { ...state, activePlayerIndex: payload.actor_index };
    }

    case "CLEAR_PROMPT":
      return { ...state, pendingPrompt: null };

    case "SET_TARGETING":
      return {
        ...state,
        targeting: {
          active: true,
          actionId: action.actionId,
          requestId: action.requestId,
          selectedTarget: null,
        },
      };

    case "SET_TARGET_SELECTED": {
      if (!state.targeting) return state;
      return {
        ...state,
        targeting: { ...state.targeting, selectedTarget: action.targetIndex },
      };
    }

    case "CLEAR_TARGETING":
      return { ...state, targeting: null };

    case "GAME_OVER":
      return {
        ...state,
        gameOver: {
          winnerIndex: action.winnerIndex,
          winnerName: action.winnerName,
        },
        pendingPrompt: null,
        targeting: null,
      };

    case "RESET":
      return initialGameSliceState;

    default:
      return state;
  }
}

// Action creators
export const gameActions = {
  lobbyStarted: (
    payload: LobbyStartedPayload,
    currentPlayerId: string | null,
    currentLobby: { lobbyId: string; playerIds?: string[]; playerNicks: string[]; playerCount: number } | null
  ): GameAction => ({
    type: "LOBBY_STARTED",
    payload,
    currentPlayerId,
    currentLobby,
  }),
  gameConfig: (payload: GameConfigPayload | undefined): GameAction => ({
    type: "GAME_CONFIG",
    payload,
  }),
  gameState: (payload: GameStatePayload | undefined, currentIdentity: GameIdentity | null): GameAction => ({
    type: "GAME_STATE",
    payload,
    currentIdentity,
  }),
  requestAction: (
    payload: RequestActionPayload,
    requestId: string,
    currentIdentity: GameIdentity | null
  ): GameAction => ({
    type: "REQUEST_ACTION",
    payload,
    requestId,
    currentIdentity,
  }),
  clearPrompt: (): GameAction => ({ type: "CLEAR_PROMPT" }),
  setTargeting: (actionId: string, requestId: string): GameAction => ({
    type: "SET_TARGETING",
    actionId,
    requestId,
  }),
  setTargetSelected: (targetIndex: number): GameAction => ({
    type: "SET_TARGET_SELECTED",
    targetIndex,
  }),
  clearTargeting: (): GameAction => ({ type: "CLEAR_TARGETING" }),
  gameOver: (winnerIndex: number, winnerName: string): GameAction => ({
    type: "GAME_OVER",
    winnerIndex,
    winnerName,
  }),
  reset: (): GameAction => ({ type: "RESET" }),
} as const;
