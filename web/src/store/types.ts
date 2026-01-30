export type PlayerIndex = number;

export interface LobbySummary {
    id: string;
    leaderNick: string;
    leaderId?: string;
    playerCount: number;
    playerNicks: string[];
    playerIds?: string[];
    playerAvatars?: string[];
    status: "open" | "in_game" | "closed";
}

export interface LobbyState {
    lobbyId: string;
    leaderNick: string;
    leaderId?: string;
    playerNicks: string[];
    playerIds?: string[];
    playerAvatars?: string[];
    playerCount: number;
    status: "open" | "in_game" | "closed";
}

export interface GameIdentity {
    playerId: string;
    playerIndex: number;
    playerNames: string[];
}

export interface ActiveMatch {
    matchId: string;
    lobbyId: string;
    playerNames: string[];
    participantIds: string[];
}

export interface ReplayEntry {
    matchId: string;
    lobbyId: string;
    playerId: string;
    playerNames: string[];
    participantIds: string[];
    winnerName?: string;
    winnerIndex?: number;
    endedAt: number;
}

export interface PlayerSnapshot {
    index: number;
    name: string;
    alive: boolean;
    coins: number | null;
    prevCoins?: number; // For animation trigger
    cardCount: number | null;
    avatar?: string;
}

export interface HandCard {
    id: string;
    role: string;
}

export type Prompt =
    | { kind: "action"; requestId: string; allowedActions: string[] }
    | {
        kind: "challenge";
        requestId: string;
        actorIndex: number;
        actionId: string;
        claimedRole: string;
        challengeKind?: "main" | "counter";
        targetIndex?: number;
        eligible?: boolean;
        timeoutMs?: number;
    }
    | {
        kind: "counter";
        requestId: string;
        actorIndex: number;
        allowedActions: string[];
        actionId: string;
        targetIndex?: number;
        eligible?: boolean;
        timeoutMs?: number;
    }
    | { kind: "step"; requestId: string; context: string; count: number; options: string[] };

export interface LogEntry {
    turn: number;
    scope: "public" | "private";
    message: string;
}

export interface InvestigateResult {
    targetName: string;
    role: string;
}

export interface EliminationToast {
    playerIndex: number;
    playerName: string;
    reason: string;
    turn: number;
    id: string;
}

export interface TurnTimerState {
    activePlayerIndex: number;
    durationMs: number;
    running: boolean;
    paused: boolean;
    key: string;
}

export type PauseState =
    | { status: "inactive" }
    | {
        status: "active";
        pausedByPlayerId: string | null;
        pausedByIndex: number;
        pausedByName: string;
        deadlineMs: number;
        durationMs: number;
        pauseReason: "disconnect";
        eligibleVoters: number[];
        kickVotes: number[];
    };

export interface TargetingState {
    active: boolean;
    actionId: string | null;
    requestId: string | null;
    selectedTarget: number | null;
}

export interface ChatMessage {
    id: string;
    senderIndex: number;
    senderName: string;
    text: string;
    timestamp: number;
}

export interface GameState {
    isConnected: boolean;
    isHandshakeComplete: boolean;
    lobbies: LobbySummary[];
    currentLobby: LobbyState | null;
    currentMatch: ActiveMatch | null;
    playerId: string | null;
    identity: GameIdentity | null;
    players: PlayerSnapshot[];
    roles: string[];
    logs: LogEntry[];
    chat: ChatMessage[];
    hand: HandCard[];
    investigateResult: InvestigateResult | null;
    eliminationToast: EliminationToast | null;
    turnTimer: TurnTimerState | null;
    pendingPrompt: Prompt | null;
    promptClosedReason: string | null;
    targeting: TargetingState | null;
    activePlayerIndex: number | null; // Track whose turn it is
    gameOver: { winnerIndex: number; winnerName: string } | null;
    sfxMuted: boolean;
    theme: "light" | "dark";
    error: string | null;
    pause: PauseState;
    replayHistory: ReplayEntry[];
    connectionLostAt: number | null;
    lastUpdateTs: number;
    mockScenario?: string;
}

export const initialGameState: GameState = {
    isConnected: false,
    isHandshakeComplete: false,
    lobbies: [],
    currentLobby: null,
    currentMatch: null,
    playerId: null,
    identity: null,
    players: [],
    roles: [],
    logs: [],
    chat: [],
    hand: [],
    investigateResult: null,
    eliminationToast: null,
    turnTimer: null,
    pendingPrompt: null,
    promptClosedReason: null,
    targeting: null,
    activePlayerIndex: null,
    gameOver: null,
    sfxMuted: false,
    theme: "light",
    error: null,
    pause: { status: "inactive" },
    replayHistory: [],
    connectionLostAt: null,
    lastUpdateTs: 0,
    mockScenario: undefined,
};
