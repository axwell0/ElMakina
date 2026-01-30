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

// WebSocket message payload types
interface ChallengeWindowPayload {
  action_id: string;
  actor_index: number;
  claimed_role: string;
  eligible: boolean;
  kind: string;
  prompt?: string;
  target_index?: number;
  timeout_ms?: number;
}

interface CounterWindowPayload {
  action_id: string;
  actor_index: number;
  allowed_actions?: string[];
  eligible: boolean;
  prompt?: string;
  target_index?: number;
  timeout_ms?: number;
}

interface RequestStepPayload {
  prompt?: string;
  step: {
    context: string;
    count: number;
    options: string[];
  };
}

interface HandStatePayload {
  hand: string[];
  player_index: number;
}

interface PromptClosedPayload {
  reason: string;
}

interface TurnTimerPayload {
  active_player_index: number;
  duration_ms: number;
  state: string;
  turn_number: number;
}

interface PlayerEliminatedPayload {
  player_index: number;
  reason: string;
  turn: number;
}

interface GamePausedPayload {
  paused_by_player_id: string;
  paused_by_index: number;
  paused_by_name: string;
  deadline_ms: number;
  duration_ms: number;
  pause_reason: string;
  eligible_voters: number[];
  kick_votes: number[];
}

interface GameResumedPayload {
  resumed_by_player_id: string;
  resumed_by_index: number;
  resumed_by_name: string;
  resume_reason: string;
}

interface KickVoteUpdatePayload {
  eligible_voters: number[];
  kick_votes: number[];
}

interface PlayerKickedPayload {
  player_index: number;
  reason: string;
}

export type GameAction =
  | { type: "LOBBY_STARTED"; payload: LobbyStartedPayload; currentPlayerId: string | null; currentLobby: CurrentLobbyInfo | null }
  | { type: "GAME_CONFIG"; payload: GameConfigPayload | undefined }
  | { type: "GAME_STATE"; payload: GameStatePayload | undefined; currentIdentity: GameIdentity | null }
  | { type: "REQUEST_ACTION"; payload: RequestActionPayload; requestId: string; currentIdentity: GameIdentity | null }
  | { type: "CHALLENGE_WINDOW"; payload: ChallengeWindowPayload; requestId: string }
  | { type: "COUNTER_WINDOW"; payload: CounterWindowPayload; requestId: string }
  | { type: "REQUEST_STEP"; payload: RequestStepPayload; requestId: string }
  | { type: "HAND_STATE"; payload: HandStatePayload; currentIdentity: GameIdentity | null }
  | { type: "PROMPT_CLOSED"; payload: PromptClosedPayload; requestId: string; currentPrompt: Prompt | null }
  | { type: "TURN_TIMER"; payload: TurnTimerPayload }
  | { type: "CLEAR_PROMPT" }
  | { type: "SET_TARGETING"; actionId: string; requestId: string }
  | { type: "SET_TARGET_SELECTED"; targetIndex: number }
  | { type: "CLEAR_TARGETING" }
  | { type: "GAME_OVER"; winnerIndex: number; winnerName: string }
  | { type: "PLAYER_ELIMINATED"; payload: PlayerEliminatedPayload }
  | { type: "GAME_PAUSED"; payload: GamePausedPayload }
  | { type: "GAME_RESUMED"; payload: GameResumedPayload }
  | { type: "KICK_VOTE_UPDATE"; payload: KickVoteUpdatePayload }
  | { type: "PLAYER_KICKED"; payload: PlayerKickedPayload }
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

    case "CHALLENGE_WINDOW": {
      const payload = action.payload;
      const challengeKind = payload.kind === "main" || payload.kind === "counter"
        ? payload.kind
        : undefined;
      return {
        ...state,
        pendingPrompt: {
          kind: "challenge",
          requestId: action.requestId,
          actorIndex: payload.actor_index,
          actionId: payload.action_id,
          claimedRole: payload.claimed_role,
          challengeKind,
          targetIndex: typeof payload.target_index === "number" ? payload.target_index : undefined,
          eligible: payload.eligible === true,
          timeoutMs: payload.timeout_ms,
        },
        promptClosedReason: null,
      };
    }

    case "COUNTER_WINDOW": {
      const payload = action.payload;
      const counterActorIndex = typeof payload.actor_index === "number"
        ? payload.actor_index
        : (state.activePlayerIndex ?? -1);
      return {
        ...state,
        pendingPrompt: {
          kind: "counter",
          requestId: action.requestId,
          actorIndex: counterActorIndex,
          allowedActions: payload.allowed_actions || [],
          actionId: payload.action_id,
          targetIndex: typeof payload.target_index === "number" ? payload.target_index : undefined,
          eligible: payload.eligible === true,
          timeoutMs: payload.timeout_ms,
        },
        promptClosedReason: null,
      };
    }

    case "REQUEST_STEP": {
      const payload = action.payload;
      return {
        ...state,
        pendingPrompt: {
          kind: "step",
          requestId: action.requestId,
          context: payload.step.context,
          count: payload.step.count,
          options: payload.step.options,
        },
        promptClosedReason: null,
      };
    }

    case "HAND_STATE": {
      const payload = action.payload;
      // Only update hand if it's for the current player
      if (action.currentIdentity && payload.player_index !== action.currentIdentity.playerIndex) {
        return state;
      }
      return {
        ...state,
        hand: payload.hand.map((role, index) => ({
          id: `${payload.player_index}-${index}-${role}`,
          role,
        })),
      };
    }

    case "PROMPT_CLOSED": {
      // Only clear if this closed message matches our current prompt
      if (action.currentPrompt && action.requestId === action.currentPrompt.requestId) {
        return {
          ...state,
          pendingPrompt: null,
          promptClosedReason: action.payload.reason || null,
          targeting: null,
        };
      }
      return state;
    }

    case "TURN_TIMER": {
      const payload = action.payload;
      return {
        ...state,
        turnTimer: {
          activePlayerIndex: payload.active_player_index,
          durationMs: payload.duration_ms,
          running: payload.state === "start",
          paused: payload.state === "paused",
          key: `${payload.turn_number ?? 0}-${Date.now()}`,
        },
      };
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

    case "PLAYER_ELIMINATED": {
      const payload = action.payload;
      return {
        ...state,
        players: state.players.map((player) =>
          player.index === payload.player_index
            ? { ...player, alive: false, cardCount: 0 }
            : player
        ),
      };
    }

    case "GAME_PAUSED": {
      const payload = action.payload;
      return {
        ...state,
        pause: {
          status: "paused",
          pausedByPlayerId: payload.paused_by_player_id,
          pausedByIndex: payload.paused_by_index,
          pausedByName: payload.paused_by_name,
          deadlineMs: payload.deadline_ms,
          durationMs: payload.duration_ms,
          pauseReason: payload.pause_reason,
          eligibleVoters: payload.eligible_voters,
          kickVotes: payload.kick_votes,
        },
      };
    }

    case "GAME_RESUMED": {
      const payload = action.payload;
      return {
        ...state,
        pause: {
          status: "resumed",
          resumedByPlayerId: payload.resumed_by_player_id,
          resumedByIndex: payload.resumed_by_index,
          resumedByName: payload.resumed_by_name,
          resumeReason: payload.resume_reason,
        },
      };
    }

    case "KICK_VOTE_UPDATE": {
      const payload = action.payload;
      if (state.pause.status !== "paused") {
        return state;
      }
      return {
        ...state,
        pause: {
          ...state.pause,
          eligibleVoters: payload.eligible_voters,
          kickVotes: payload.kick_votes,
        },
      };
    }

    case "PLAYER_KICKED": {
      const payload = action.payload;
      return {
        ...state,
        players: state.players.map((player) =>
          player.index === payload.player_index
            ? { ...player, alive: false, cardCount: 0 }
            : player
        ),
        pause: { status: "inactive" },
      };
    }

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
  challengeWindow: (
    payload: ChallengeWindowPayload,
    requestId: string
  ): GameAction => ({
    type: "CHALLENGE_WINDOW",
    payload,
    requestId,
  }),
  counterWindow: (
    payload: CounterWindowPayload,
    requestId: string
  ): GameAction => ({
    type: "COUNTER_WINDOW",
    payload,
    requestId,
  }),
  requestStep: (
    payload: RequestStepPayload,
    requestId: string
  ): GameAction => ({
    type: "REQUEST_STEP",
    payload,
    requestId,
  }),
  handState: (
    payload: HandStatePayload,
    currentIdentity: GameIdentity | null
  ): GameAction => ({
    type: "HAND_STATE",
    payload,
    currentIdentity,
  }),
  promptClosed: (
    payload: PromptClosedPayload,
    requestId: string,
    currentPrompt: Prompt | null
  ): GameAction => ({
    type: "PROMPT_CLOSED",
    payload,
    requestId,
    currentPrompt,
  }),
  turnTimer: (payload: TurnTimerPayload): GameAction => ({
    type: "TURN_TIMER",
    payload,
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
  playerEliminated: (payload: PlayerEliminatedPayload): GameAction => ({
    type: "PLAYER_ELIMINATED",
    payload,
  }),
  gamePaused: (payload: GamePausedPayload): GameAction => ({
    type: "GAME_PAUSED",
    payload,
  }),
  gameResumed: (payload: GameResumedPayload): GameAction => ({
    type: "GAME_RESUMED",
    payload,
  }),
  kickVoteUpdate: (payload: KickVoteUpdatePayload): GameAction => ({
    type: "KICK_VOTE_UPDATE",
    payload,
  }),
  playerKicked: (payload: PlayerKickedPayload): GameAction => ({
    type: "PLAYER_KICKED",
    payload,
  }),
  reset: (): GameAction => ({ type: "RESET" }),
} as const;
