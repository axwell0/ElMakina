import {ChatMessage, GameState, HandCard, initialGameState, LogEntry, PlayerSnapshot, ReplayEntry} from "./types";
import type {WsEnvelope} from "../network/socket";

let handIdCounter = 0;

const reconcileHand = (prev: HandCard[], nextRoles: string[]): HandCard[] => {
    const buckets = new Map<string, HandCard[]>();
    for (const card of prev) {
        const list = buckets.get(card.role);
        if (list) {
            list.push(card);
        } else {
            buckets.set(card.role, [card]);
        }
    }
    return nextRoles.map((role) => {
        const list = buckets.get(role);
        if (list && list.length > 0) {
            return list.shift()!;
        }
        handIdCounter += 1;
        return { id: `hand-${handIdCounter}`, role };
    });
};

export type Action =
    | { type: "CONNECT" }
    | { type: "DISCONNECT" }
    | { type: "ERROR"; error: string }
    | { type: "CLEAR_ERROR" }
    | { type: "MESSAGE"; envelope: WsEnvelope }
    | { type: "CLEAR_PROMPT" }
    | { type: "CLEAR_INVESTIGATE" }
    | { type: "CLEAR_ELIMINATION_TOAST" }
    | { type: "SET_SFX_MUTED"; muted: boolean }
    | { type: "SET_THEME"; theme: "light" | "dark" }
    | { type: "SEND_CHAT"; text: string }
    | { type: "SET_TARGETING"; actionId: string; requestId: string }
    | { type: "SET_TARGET_SELECTED"; targetIndex: number }
    | { type: "CLEAR_TARGETING" }
    | { type: "RESET" };

type HelloAckPayload = { player_id?: string };
type HelloErrorPayload = { error?: string };
type LobbyListPayload = {
    lobbies?: Array<{
        id: string;
        leader_nick: string;
        leader_id?: string;
        player_count: number;
        player_nicks?: string[];
        player_ids?: string[];
        player_avatars?: string[];
        status: "open" | "in_game" | "closed";
    }>;
};
type LobbyStatePayload = {
    lobby_id: string;
    leader_nick: string;
    leader_id?: string;
    player_nicks?: string[];
    player_ids?: string[];
    player_avatars?: string[];
    player_count: number;
    status: "open" | "in_game" | "closed";
};
type GameConfigPayload = { roles?: string[] };
type GameStatePayload = {
    players?: Array<{
        index: number;
        name: string;
        alive: boolean;
        coins: number;
        card_count: number;
        avatar?: string;
    }>;
    active_player_index: number;
};
type LobbyStartedPayload = {
    lobby_id: string;
    match_id?: string;
    player_index: number;
    player_count: number;
    player_names: string[];
    player_avatars?: string[];
    index_mapping?: Record<string, number>;
};
type RequestActionPayload = { actor_index: number; allowed_actions?: string[] };
type ChallengeWindowPayload = {
    actor_index: number;
    action_id: string;
    claimed_role: string;
    kind?: "main" | "counter";
    target_index?: number;
    eligible?: boolean;
    timeout_ms?: number;
};
type CounterWindowPayload = {
    actor_index?: number;
    allowed_actions?: string[];
    action_id: string;
    target_index?: number;
    eligible?: boolean;
    timeout_ms?: number;
};
type RequestStepPayload = { step: { context: string; count: number; options: string[] } };
type InvestigatePayload = { target_name?: string; role?: string };
type TurnTimerPayload = {
    active_player_index?: number;
    duration_ms?: number;
    state?: "start" | "pause" | "resume" | "stop";
    turn_number?: number;
};
type GamePausedPayload = {
    paused_by_player_id?: string;
    paused_by_index?: number;
    paused_by_name?: string;
    deadline_ms?: number;
    duration_ms?: number;
    pause_reason?: "disconnect";
    eligible_voters?: number[];
    kick_votes?: number[];
};
type KickVoteUpdatePayload = {
    eligible_voters?: number[];
    kick_votes?: number[];
};
type PlayerKickedPayload = { player_index?: number; reason?: string };
type LobbyErrorPayload = { error?: string };
type PromptClosedPayload = { reason?: string };
type GameOverPayload = { winner_index: number; winner_name: string };
type PlayerEliminatedPayload = { player_index?: number; reason?: string; turn?: number };

export function gameReducer(state: GameState, action: Action): GameState {
    switch (action.type) {
        case "CONNECT":
            return { ...state, isConnected: true, isHandshakeComplete: false, error: null, connectionLostAt: null };
        case "DISCONNECT":
            if (state.currentLobby?.status === "in_game") {
                return {
                    ...state,
                    isConnected: false,
                    isHandshakeComplete: false,
                    pendingPrompt: null,
                    error: null,
                    connectionLostAt: Date.now()
                };
            }
            return {
                ...state,
                isConnected: false,
                isHandshakeComplete: false,
                currentLobby: null,
                currentMatch: null,
                players: [],
                logs: [],
                hand: [],
                investigateResult: null,
                eliminationToast: null,
                turnTimer: null,
                pendingPrompt: null,
                activePlayerIndex: null,
                gameOver: null,
                pause: { status: "inactive" },
                error: null,
                connectionLostAt: Date.now()
            };
        case "ERROR":
            return { ...state, error: action.error };
        case "CLEAR_ERROR":
            return { ...state, error: null };
        case "SET_SFX_MUTED":
            return { ...state, sfxMuted: action.muted };
        case "SET_THEME":
            return { ...state, theme: action.theme };
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
        case "SET_TARGET_SELECTED":
            if (!state.targeting) {
                return state;
            }
            return {
                ...state,
                targeting: { ...state.targeting, selectedTarget: action.targetIndex },
            };
        case "CLEAR_TARGETING":
            return { ...state, targeting: null };
        case "MESSAGE": {
            const env = action.envelope;
            const processMessage = (s: GameState): GameState => {
                switch (env.type) {
                    case "hello_ack": {
                        const helloAck = env.payload as HelloAckPayload | undefined;
                        return {
                            ...state,
                            error: null,
                            playerId: helloAck?.player_id || state.playerId,
                            isHandshakeComplete: true,
                            connectionLostAt: null
                        };
                    }

                    case "hello_error": {
                        const helloErr = env.payload as HelloErrorPayload | undefined;
                        return {
                            ...state,
                            error: helloErr?.error || "Handshake failed",
                            playerId: null,
                            isHandshakeComplete: false,
                            currentLobby: null,
                            currentMatch: null,
                            players: [],
                            logs: [],
                            hand: [],
                            investigateResult: null,
                            eliminationToast: null,
                            turnTimer: null,
                            pendingPrompt: null,
                            activePlayerIndex: null,
                            gameOver: null,
                            pause: { status: "inactive" },
                            connectionLostAt: null
                        };
                    }

                    case "lobby_list_result": {
                        const payload = env.payload as LobbyListPayload | undefined;
                        const lobbies = (payload?.lobbies || []).filter((lobby) => {
                            if (typeof lobby.player_count === "number" && lobby.player_count <= 0) {
                                return false;
                            }
                            const hasListedPlayers =
                                (lobby.player_ids && lobby.player_ids.length > 0) ||
                                (lobby.player_nicks && lobby.player_nicks.length > 0);
                            return hasListedPlayers || lobby.status !== "open";
                        });
                        const mapped = lobbies.map((lobby) => ({
                            id: lobby.id,
                            leaderNick: lobby.leader_nick,
                            leaderId: lobby.leader_id,
                            playerCount: lobby.player_count,
                            playerNicks: lobby.player_nicks || [],
                            playerIds: lobby.player_ids || [],
                            playerAvatars: lobby.player_avatars || [],
                            status: lobby.status
                        }));
                        const hasCurrentLobby =
                            state.currentLobby &&
                            mapped.some((lobby) => lobby.id === state.currentLobby?.lobbyId);
                        if (state.currentLobby && !hasCurrentLobby) {
                            return {
                                ...state,
                                lobbies: mapped,
                                currentLobby: null,
                                currentMatch: null,
                                players: [],
                                logs: [],
                                hand: [],
                                investigateResult: null,
                                eliminationToast: null,
                                turnTimer: null,
                                pendingPrompt: null,
                                activePlayerIndex: null,
                                gameOver: null,
                                pause: { status: "inactive" },
                                error: null,
                            };
                        }
                        return {
                            ...state,
                            lobbies: mapped
                        };
                    }

                    case "lobby_created": // Fallthrough
                    case "lobby_joined":
                        // We rely on 'lobby_state' message (which follows immediately) to populate currentLobby.
                        // Returning state without modifying currentLobby avoids partial state crashes.
                        return { ...state, error: null };

                    case "lobby_state": {
                        const lobbyState = env.payload as LobbyStatePayload;
                        const playerIds = lobbyState.player_ids || [];
                        if (playerIds.length > 0 && state.playerId && !playerIds.includes(state.playerId)) {
                            return {
                                ...state,
                                currentLobby: null,
                                currentMatch: null,
                                players: [],
                                logs: [],
                                hand: [],
                                activePlayerIndex: null,
                                pendingPrompt: null,
                                promptClosedReason: null,
                                targeting: null,
                                gameOver: null,
                                pause: { status: "inactive" },
                                error: null,
                            };
                        }
                        return {
                            ...state,
                            currentLobby: {
                                lobbyId: lobbyState.lobby_id,
                                leaderNick: lobbyState.leader_nick,
                                leaderId: lobbyState.leader_id,
                                playerNicks: lobbyState.player_nicks || [], // Map snake to camel, ensure array
                                playerIds: lobbyState.player_ids || [],
                                playerAvatars: lobbyState.player_avatars || [],
                                playerCount: lobbyState.player_count,
                                status: lobbyState.status
                            },
                            lobbies: []
                        };
                    }

                    case "game_config": {
                        const gameCfg = env.payload as GameConfigPayload | undefined;
                        return {
                            ...state,
                            roles: gameCfg?.roles || [],
                        };
                    }

                    case "game_state": {
                        const payload = env.payload as GameStatePayload | undefined;
                        const incomingPlayers = payload?.players || [];
                        const updatedPlayers = incomingPlayers.map((p) => {
                            const previous = state.players.find(existing => existing.index === p.index);
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

                    case "lobby_started": {
                        const payload = env.payload as LobbyStartedPayload;
                        const avatars = payload.player_avatars || [];
                        const players: PlayerSnapshot[] = (payload.player_names || []).map((name: string, index: number) => ({
                            index,
                            name,
                            alive: true,
                            coins: null,
                            cardCount: null,
                            avatar: avatars[index] || ""
                        }));

                        const currentLobby = state.currentLobby ?? {
                            lobbyId: payload.lobby_id,
                            leaderNick: "",
                            playerNicks: payload.player_names || [],
                            playerCount: payload.player_count,
                            status: "in_game"
                        };
                        const participantIds =
                            payload.index_mapping ? Object.keys(payload.index_mapping) : (currentLobby.playerIds || []);
                        const matchId = payload.match_id || payload.lobby_id;

                        return {
                            ...state,
                            identity: {
                                playerId: state.playerId || "unknown",
                                playerIndex: payload.player_index,
                                playerNames: payload.player_names,
                            },
                            players,
                            currentLobby: { ...currentLobby, status: "in_game" },
                            currentMatch: {
                                matchId,
                                lobbyId: payload.lobby_id,
                                playerNames: payload.player_names,
                                participantIds,
                            },
                            activePlayerIndex: 0, // Assume index 0 starts
                            logs: [],
                            hand: [],
                            investigateResult: null,
                            eliminationToast: null,
                            turnTimer: null,
                            gameOver: null,
                            pendingPrompt: null,
                            promptClosedReason: null,
                            targeting: null,
                            error: null,
                        };
                    }

                    case "request_action": {
                        const requestAction = env.payload as RequestActionPayload;
                        // Only show prompt if it's for us
                        // The server sends request_action to the ACTOR.
                        // The payload has actor_index.
                        if (state.identity && requestAction.actor_index === state.identity.playerIndex) {
                            return {
                                ...state,
                                activePlayerIndex: requestAction.actor_index,
                                pendingPrompt: { kind: "action", requestId: env.request_id!, allowedActions: requestAction.allowed_actions || [] },
                                promptClosedReason: null
                            };
                        }
                        return { ...state, activePlayerIndex: requestAction.actor_index };
                    }

                    case "challenge_window": {
                        const payload = env.payload as ChallengeWindowPayload;
                        const challengeKind = payload.kind === "main" || payload.kind === "counter"
                            ? payload.kind
                            : undefined;
                        return {
                            ...state,
                            pendingPrompt: {
                                kind: "challenge",
                                requestId: env.request_id!,
                                actorIndex: payload.actor_index,
                                actionId: payload.action_id,
                                claimedRole: payload.claimed_role,
                                challengeKind,
                                targetIndex: typeof payload.target_index === "number" ? payload.target_index : undefined,
                                eligible: payload.eligible === true,
                                timeoutMs: payload.timeout_ms
                            },
                            promptClosedReason: null
                        };
                    }

                    case "counter_window": {
                        const payload = env.payload as CounterWindowPayload;
                        const counterActorIndex = typeof payload.actor_index === "number"
                            ? payload.actor_index
                            : (state.activePlayerIndex ?? -1);
                        return {
                            ...state,
                            pendingPrompt: {
                                kind: "counter",
                                requestId: env.request_id!,
                                actorIndex: counterActorIndex,
                                allowedActions: payload.allowed_actions || [],
                                actionId: payload.action_id,
                                targetIndex: typeof payload.target_index === "number" ? payload.target_index : undefined,
                                eligible: payload.eligible === true,
                                timeoutMs: payload.timeout_ms
                            },
                            promptClosedReason: null
                        };
                    }

                    case "request_step": {
                        const stepPayload = env.payload as RequestStepPayload;
                        return {
                            ...state,
                            pendingPrompt: { kind: "step", requestId: env.request_id!, context: stepPayload.step.context, count: stepPayload.step.count, options: stepPayload.step.options },
                            promptClosedReason: null
                        };
                    }

                    // When we respond, we should clear the prompt?
                    // Actually the server will send a new state or log eventually, but for UI responsiveness we might want to clear it.
                    // But let's wait for server state changes to be safe, or just clear it when we submit.

                    case "hand_state": {
                        const payload = env.payload as { hand?: string[]; player_index?: number } | undefined;
                        if (!payload || !Array.isArray(payload.hand)) {
                            return state;
                        }
                        if (state.identity && payload.player_index !== state.identity.playerIndex) {
                            return state;
                        }
                        return { ...state, hand: reconcileHand(state.hand, payload.hand) };
                    }

                    case "investigate_result": {
                        const investigate = env.payload as InvestigatePayload | undefined;
                        return {
                            ...state,
                            investigateResult: {
                                targetName: investigate?.target_name ?? "Unknown",
                                role: investigate?.role ?? "Unknown",
                            },
                        };
                    }

                    case "turn_timer": {
                        const payload = (env.payload as TurnTimerPayload | undefined) ?? {};
                        const durationMs = typeof payload.duration_ms === "number" ? payload.duration_ms : 0;
                        const active: number = typeof payload.active_player_index === "number" ? payload.active_player_index : -1;
                        const stateValue: string = payload.state ?? "";
                        if (stateValue === "start") {
                            return {
                                ...state,
                                turnTimer: {
                                    activePlayerIndex: active,
                                    durationMs,
                                    running: true,
                                    paused: false,
                                    key: `${payload.turn_number ?? 0}-${Date.now()}`,
                                },
                            };
                        }
                        if (!state.turnTimer) {
                            return state;
                        }
                        if (stateValue === "pause") {
                            return { ...state, turnTimer: { ...state.turnTimer, paused: true } };
                        }
                        if (stateValue === "resume") {
                            return { ...state, turnTimer: { ...state.turnTimer, paused: false } };
                        }
                        if (stateValue === "stop") {
                            return { ...state, turnTimer: { ...state.turnTimer, running: false, paused: false } };
                        }
                        return state;
                    }

                    case "game_paused": {
                        const payload = env.payload as GamePausedPayload | undefined;
                        if (!payload) {
                            return state;
                        }
                        const pausedIndex = typeof payload.paused_by_index === "number" ? payload.paused_by_index : -1;
                        const pausedName = payload.paused_by_name ?? "Unknown";
                        return {
                            ...state,
                            pause: {
                                status: "active",
                                pausedByPlayerId: payload.paused_by_player_id ?? null,
                                pausedByIndex: pausedIndex,
                                pausedByName: pausedName,
                                deadlineMs: payload.deadline_ms ?? Date.now(),
                                durationMs: payload.duration_ms ?? 60000,
                                pauseReason: payload.pause_reason ?? "disconnect",
                                eligibleVoters: payload.eligible_voters ?? [],
                                kickVotes: payload.kick_votes ?? []
                            }
                        };
                    }

                    case "kick_vote_update": {
                        if (state.pause.status !== "active") {
                            return state;
                        }
                        const payload = env.payload as KickVoteUpdatePayload | undefined;
                        if (!payload) {
                            return state;
                        }
                        return {
                            ...state,
                            pause: {
                                ...state.pause,
                                eligibleVoters: payload.eligible_voters ?? state.pause.eligibleVoters,
                                kickVotes: payload.kick_votes ?? state.pause.kickVotes
                            }
                        };
                    }

                    case "game_resumed": {
                        return {
                            ...state,
                            pause: { status: "inactive" }
                        };
                    }

                    case "player_kicked": {
                        const payload = env.payload as PlayerKickedPayload | undefined;
                        if (!payload) {
                            return { ...state, pause: { status: "inactive" } };
                        }
                        if (typeof payload.player_index === "number") {
                            return {
                                ...state,
                                pause: { status: "inactive" },
                                players: state.players.map(p =>
                                    p.index === payload.player_index ? { ...p, alive: false, cardCount: 0 } : p
                                )
                            };
                        }
                        return { ...state, pause: { status: "inactive" } };
                    }

                    case "game_log": {
                        const logEntry = env.payload as LogEntry;
                        return { ...state, logs: [...state.logs, logEntry] };
                    }

                    case "chat_message": {
                        const msg = env.payload as ChatMessage;
                        return { ...state, chat: [...(state.chat || []), msg].slice(-50) };
                    }

                    case "lobby_error": {
                        const lobbyErr = env.payload as LobbyErrorPayload | undefined;
                        return { ...state, error: lobbyErr?.error || "Lobby error" };
                    }

                    case "prompt_closed":
                        if (state.pendingPrompt && env.request_id === state.pendingPrompt.requestId) {
                            const closed = env.payload as PromptClosedPayload | undefined;
                            return {
                                ...state,
                                pendingPrompt: null,
                                promptClosedReason: closed?.reason || null,
                                targeting: null,
                            };
                        }
                        return state;

                    case "game_over": {
                        const over = env.payload as GameOverPayload;
                        let replayHistory = state.replayHistory;
                        if (state.currentMatch && state.playerId) {
                            const entry: ReplayEntry = {
                                matchId: state.currentMatch.matchId,
                                lobbyId: state.currentMatch.lobbyId,
                                playerId: state.playerId,
                                playerNames: state.currentMatch.playerNames,
                                participantIds: state.currentMatch.participantIds,
                                winnerName: over.winner_name,
                                winnerIndex: over.winner_index,
                                endedAt: Date.now(),
                            };
                            const deduped = replayHistory.filter((item) => item.matchId !== entry.matchId);
                            replayHistory = [entry, ...deduped].slice(0, 30);
                        }
                        return {
                            ...state,
                            gameOver: {
                                winnerIndex: over.winner_index,
                                winnerName: over.winner_name,
                            },
                            pendingPrompt: null,
                            targeting: null,
                            replayHistory,
                        };
                    }

                    case "player_eliminated": {
                        const payload = env.payload as PlayerEliminatedPayload | undefined;
                        const idx = payload?.player_index;
                        const reason = payload?.reason ?? "unknown";
                        const turn = payload?.turn ?? 0;
                        if (typeof idx !== "number") {
                            return state;
                        }
                        const name = state.players.find(p => p.index === idx)?.name ?? `Player ${idx + 1}`;
                        return {
                            ...state,
                            players: state.players.map(p =>
                                p.index === idx ? { ...p, alive: false, cardCount: 0 } : p
                            ),
                            eliminationToast: {
                                playerIndex: idx,
                                playerName: name,
                                reason,
                                turn,
                                id: `${idx}-${turn}-${Date.now()}`
                            }
                        };
                    }

                    default:
                        return s;
                }
            };
            return { ...processMessage(state), lastUpdateTs: Date.now() };
        }
        case "CLEAR_PROMPT":
            return { ...state, pendingPrompt: null };
        case "CLEAR_INVESTIGATE":
            return { ...state, investigateResult: null };
        case "CLEAR_ELIMINATION_TOAST":
            return { ...state, eliminationToast: null };
        case "RESET":
            return {
                ...initialGameState,
                isConnected: state.isConnected,
                sfxMuted: state.sfxMuted,
                theme: state.theme
            };
        default:
            return state;
    }
}
