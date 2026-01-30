/* eslint-disable */
/**
 * This file is auto-generated from the backend WebSocket schema.
 * Source: shared/schemas/envelope.json
 * Do not edit by hand.
 */

export type ElMakinaWebSocketEnvelope =
  | Hello
  | HelloAck
  | HelloError
  | LobbyCreate
  | LobbyList
  | LobbyJoin
  | LobbyStart
  | LobbyCreated
  | LobbyJoined
  | LobbyListResult
  | LobbyStarted
  | LobbyState
  | LobbyError
  | RequestAction
  | RequestStep
  | ChallengeWindow
  | CounterWindow
  | Action
  | Challenge
  | Counter
  | StepResult
  | GameLog
  | GameOver
  | GameState
  | GameConfig
  | PromptClosed
  | InvestigateResult
  | HandState
  | PlayerEliminated
  | TurnTimer
  | GamePaused
  | GameResumed
  | KickVote
  | KickVoteUpdate
  | PlayerKicked
  | ChatMessage;
export type Hello = EnvelopeBase & {
  payload: HelloPayload;
  type: 'hello';
  [k: string]: unknown;
};
export type HelloAck = EnvelopeBase & {
  payload: HelloAckPayload;
  type: 'hello_ack';
  [k: string]: unknown;
};
export type HelloError = EnvelopeBase & {
  payload: HelloErrorPayload;
  type: 'hello_error';
  [k: string]: unknown;
};
export type LobbyCreate = EnvelopeBase & {
  type: 'lobby_create';
  [k: string]: unknown;
};
export type LobbyList = EnvelopeBase & {
  type: 'lobby_list';
  [k: string]: unknown;
};
export type LobbyJoin = EnvelopeBase & {
  payload: LobbyJoinPayload;
  type: 'lobby_join';
  [k: string]: unknown;
};
export type LobbyStart = EnvelopeBase & {
  payload: LobbyStartPayload;
  type: 'lobby_start';
  [k: string]: unknown;
};
export type LobbyCreated = EnvelopeBase & {
  payload: LobbyCreatedPayload;
  type: 'lobby_created';
  [k: string]: unknown;
};
export type LobbyJoined = EnvelopeBase & {
  payload: LobbyJoinPayload;
  type: 'lobby_joined';
  [k: string]: unknown;
};
export type LobbyListResult = EnvelopeBase & {
  payload: LobbyListPayload;
  type: 'lobby_list_result';
  [k: string]: unknown;
};
export type LobbyStarted = EnvelopeBase & {
  payload: LobbyStartedPayload;
  type: 'lobby_started';
  [k: string]: unknown;
};
export type LobbyState = EnvelopeBase & {
  payload: LobbyStatePayload;
  type: 'lobby_state';
  [k: string]: unknown;
};
export type LobbyError = EnvelopeBase & {
  payload: LobbyErrorPayload;
  type: 'lobby_error';
  [k: string]: unknown;
};
export type RequestAction = EnvelopeBase & {
  payload: RequestActionPayload;
  type: 'request_action';
  [k: string]: unknown;
};
export type RequestStep = EnvelopeBase & {
  payload: RequestStepPayload;
  type: 'request_step';
  [k: string]: unknown;
};
export type ChallengeWindow = EnvelopeBase & {
  payload: ChallengeWindowPayload;
  type: 'challenge_window';
  [k: string]: unknown;
};
export type CounterWindow = EnvelopeBase & {
  payload: CounterWindowPayload;
  type: 'counter_window';
  [k: string]: unknown;
};
export type Action = EnvelopeBase & {
  payload: ActionPayload;
  type: 'action';
  [k: string]: unknown;
};
export type Challenge = EnvelopeBase & {
  payload: ChallengePayload;
  type: 'challenge';
  [k: string]: unknown;
};
export type Counter = EnvelopeBase & {
  payload: ActionPayload;
  type: 'counter';
  [k: string]: unknown;
};
export type StepResult = EnvelopeBase & {
  payload: StepResultPayload;
  type: 'step_result';
  [k: string]: unknown;
};
export type StepResultPayload =
  | number[]
  | {
      [k: string]: unknown;
    }
  | string
  | number
  | boolean
  | null;
export type GameLog = EnvelopeBase & {
  payload: GameLogPayload;
  type: 'game_log';
  [k: string]: unknown;
};
export type GameOver = EnvelopeBase & {
  payload: GameOverPayload;
  type: 'game_over';
  [k: string]: unknown;
};
export type GameState = EnvelopeBase & {
  payload: GameStatePayload;
  type: 'game_state';
  [k: string]: unknown;
};
export type GameConfig = EnvelopeBase & {
  payload: GameConfigPayload;
  type: 'game_config';
  [k: string]: unknown;
};
export type PromptClosed = EnvelopeBase & {
  payload: PromptClosedPayload;
  type: 'prompt_closed';
  [k: string]: unknown;
};
export type InvestigateResult = EnvelopeBase & {
  payload: InvestigateResultPayload;
  type: 'investigate_result';
  [k: string]: unknown;
};
export type HandState = EnvelopeBase & {
  payload: HandStatePayload;
  type: 'hand_state';
  [k: string]: unknown;
};
export type PlayerEliminated = EnvelopeBase & {
  payload: PlayerEliminatedPayload;
  type: 'player_eliminated';
  [k: string]: unknown;
};
export type TurnTimer = EnvelopeBase & {
  payload: TurnTimerPayload;
  type: 'turn_timer';
  [k: string]: unknown;
};
export type GamePaused = EnvelopeBase & {
  payload: GamePausedPayload;
  type: 'game_paused';
  [k: string]: unknown;
};
export type GameResumed = EnvelopeBase & {
  payload: GameResumedPayload;
  type: 'game_resumed';
  [k: string]: unknown;
};
export type KickVote = EnvelopeBase & {
  payload: KickVotePayload;
  type: 'kick_vote';
  [k: string]: unknown;
};
export type KickVoteUpdate = EnvelopeBase & {
  payload: KickVoteUpdatePayload;
  type: 'kick_vote_update';
  [k: string]: unknown;
};
export type PlayerKicked = EnvelopeBase & {
  payload: PlayerKickedPayload;
  type: 'player_kicked';
  [k: string]: unknown;
};
export type ChatMessage = EnvelopeBase & {
  payload: ChatMessagePayload;
  type: 'chat_message';
  [k: string]: unknown;
};

export interface EnvelopeBase {
  payload?: unknown;
  request_id?: string;
  type?: string;
}
export interface HelloPayload {
  avatar?: string;
  nickname: string;
  reconnect_token?: string;
}
export interface HelloAckPayload {
  player_id: string;
  token: string;
}
export interface HelloErrorPayload {
  error: string;
}
export interface LobbyJoinPayload {
  lobby_id: string;
}
export interface LobbyStartPayload {
  lobby_id: string;
}
export interface LobbyCreatedPayload {
  lobby_id: string;
}
export interface LobbyListPayload {
  lobbies: LobbySummaryPayload[];
}
export interface LobbySummaryPayload {
  id: string;
  leader_id: string;
  leader_nick: string;
  player_avatars?: string[];
  player_count: number;
  player_ids?: string[];
  player_nicks?: string[];
  status: string;
}
export interface LobbyStartedPayload {
  index_mapping: {
    [k: string]: number;
  };
  lobby_id: string;
  match_id: string;
  player_avatars?: string[];
  player_count: number;
  player_index: number;
  player_names: string[];
}
export interface LobbyStatePayload {
  leader_id: string;
  leader_nick: string;
  lobby_id: string;
  player_avatars?: string[];
  player_count: number;
  player_ids: string[];
  player_nicks: string[];
  status: string;
}
export interface LobbyErrorPayload {
  error: string;
}
export interface RequestActionPayload {
  actor_index: number;
  allowed_actions?: string[];
  prompt?: string;
}
export interface RequestStepPayload {
  prompt?: string;
  step: unknown;
}
export interface ChallengeWindowPayload {
  action_id: string;
  actor_index: number;
  claimed_role: string;
  eligible: boolean;
  kind: string;
  prompt?: string;
  target_index?: number;
  timeout_ms?: number;
}
export interface CounterWindowPayload {
  action_id: string;
  actor_index: number;
  allowed_actions?: string[];
  eligible: boolean;
  prompt?: string;
  target_index?: number;
  timeout_ms?: number;
}
export interface ActionPayload {
  guess?: string;
  id: string;
  main_action?: string;
  pass?: boolean;
  source_index: number;
  target_index?: number;
}
export interface ChallengePayload {
  actor_discard_index: number;
  actor_proving_card_index: number;
  challenger_discard_index: number;
  challenger_index: number;
  pass?: boolean;
}
export interface GameLogPayload {
  message: string;
  player_index?: number;
  scope: string;
  turn: number;
}
export interface GameOverPayload {
  winner_index: number;
  winner_name: string;
}
export interface GameStatePayload {
  active_player_index: number;
  players: PlayerStatePayload[];
  turn_number: number;
}
export interface PlayerStatePayload {
  alive: boolean;
  avatar?: string;
  card_count: number;
  coins: number;
  index: number;
  name: string;
}
export interface GameConfigPayload {
  roles: string[];
}
export interface PromptClosedPayload {
  reason: string;
}
export interface InvestigateResultPayload {
  role: string;
  target_name: string;
}
export interface HandStatePayload {
  hand: string[];
  player_index: number;
}
export interface PlayerEliminatedPayload {
  player_index: number;
  reason: string;
  turn: number;
}
export interface TurnTimerPayload {
  active_player_index: number;
  duration_ms: number;
  state: string;
  turn_number: number;
}
export interface GamePausedPayload {
  deadline_ms: number;
  duration_ms: number;
  eligible_voters: number[];
  kick_votes: number[];
  pause_reason: string;
  paused_by_index: number;
  paused_by_name: string;
  paused_by_player_id: string;
}
export interface GameResumedPayload {
  resume_reason: string;
  resumed_by_index: number;
  resumed_by_name: string;
  resumed_by_player_id: string;
}
export interface KickVotePayload {
  target_index: number;
}
export interface KickVoteUpdatePayload {
  eligible_voters: number[];
  kick_votes: number[];
}
export interface PlayerKickedPayload {
  player_index: number;
  reason: string;
}
export interface ChatMessagePayload {
  id: string;
  senderIndex: number;
  senderName: string;
  text: string;
  timestamp: number;
}
