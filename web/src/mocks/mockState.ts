import type {GameState} from '@/state/types';

const now = Date.now();

export type MockScenario = 'game' | 'lobby' | 'reveal' | 'gameover' | 'paused' | 'assassinate' | 'showcase';

const baseGameState: GameState = {
    isConnected: true,
    isHandshakeComplete: true,
    lobbies: [],
    currentLobby: {
        lobbyId: 'MOCK-ROOM',
        leaderNick: 'Mahdi',
        leaderId: 'mock-leader',
        playerNicks: ['Mahdi', 'Ava', 'Yara', 'Omar', 'Zayd', 'Leila', 'Karim', 'Nour', 'Sami'],
        playerIds: ['mock-0', 'mock-1', 'mock-2', 'mock-3', 'mock-4', 'mock-5', 'mock-6', 'mock-7', 'mock-8'],
        playerAvatars: ['', '', '', '', '', '', '', '', ''],
        playerCount: 9,
        status: 'in_game',
    },
    currentMatch: {
        matchId: 'match-mock-1',
        lobbyId: 'MOCK-ROOM',
        playerNames: ['Mahdi', 'Ava', 'Yara', 'Omar', 'Zayd'],
        participantIds: ['mock-0', 'mock-1', 'mock-2', 'mock-3', 'mock-4'],
    },
    playerId: 'mock-0',
    identity: {
        playerId: 'mock-0',
        playerIndex: 0,
        playerNames: ['Mahdi', 'Ava', 'Yara', 'Omar', 'Zayd'],
    },
    players: [
        { index: 0, name: 'Mahdi', alive: true, coins: 5, cardCount: 2, avatar: '' },
        { index: 1, name: 'Ava', alive: true, coins: 3, cardCount: 2, avatar: '' },
        { index: 2, name: 'Yara', alive: true, coins: 2, cardCount: 1, avatar: '' },
        { index: 3, name: 'Omar', alive: true, coins: 6, cardCount: 2, avatar: '' },
        { index: 4, name: 'Zayd', alive: false, coins: 0, cardCount: 0, avatar: '' },
        { index: 5, name: 'Leila', alive: true, coins: 4, cardCount: 2, avatar: '' },
        { index: 6, name: 'Karim', alive: true, coins: 3, cardCount: 2, avatar: '' },
        { index: 7, name: 'Nour', alive: true, coins: 5, cardCount: 2, avatar: '' },
        { index: 8, name: 'Sami', alive: true, coins: 2, cardCount: 2, avatar: '' },
    ],
    roles: ['Businesswoman', 'TaxCollector', 'Policewoman', 'Colonel', 'Terrorist', 'Thief', 'Politician'],
    logs: [
        { turn: 1, scope: 'public', message: 'Mahdi took 1 coin.' },
        { turn: 1, scope: 'public', message: 'Ava claimed Policewoman.' },
        { turn: 1, scope: 'private', message: 'You hold Businesswoman and Thief.' },
    ],
    chat: [],
    hand: [
        { id: 'hand-1', role: 'Businesswoman' },
        { id: 'hand-2', role: 'Thief' },
    ],
    investigateResult: null,
    eliminationToast: null,
    turnTimer: {
        activePlayerIndex: 1,
        durationMs: 30000,
        running: true,
        paused: false,
        key: `mock-${now}`,
    },
    pendingPrompt: {
        kind: 'action',
        requestId: `mock-${now}`,
        allowedActions: ['income', 'foreign_aid', 'steal', 'tax', 'accuse'],
    },
    promptClosedReason: null,
    targeting: null,
    activePlayerIndex: 1,
    gameOver: null,
    sfxMuted: false,
    theme: 'light',
    error: null,
    pause: { status: "inactive" },
    replayHistory: [
        {
            matchId: 'match-mock-0',
            lobbyId: 'LOBBY-B72',
            playerId: 'mock-0',
            playerNames: ['Zayd', 'Mahdi'],
            participantIds: ['mock-4', 'mock-0'],
            winnerName: 'Mahdi',
            winnerIndex: 1,
            endedAt: now - 1000 * 60 * 25,
        },
        {
            matchId: 'match-mock-1',
            lobbyId: 'MOCK-ROOM',
            playerId: 'mock-0',
            playerNames: ['Mahdi', 'Ava', 'Yara', 'Omar', 'Zayd'],
            participantIds: ['mock-0', 'mock-1', 'mock-2', 'mock-3', 'mock-4'],
            winnerName: 'Yara',
            winnerIndex: 2,
            endedAt: now - 1000 * 60 * 90,
        },
    ],
    connectionLostAt: null,
    lastUpdateTs: now,
    mockScenario: 'game',
};

const lobbyState: GameState = {
    ...baseGameState,
    currentLobby: null,
    lobbies: [
        {
            id: 'LOBBY-AX1',
            leaderNick: 'Ava',
            leaderId: 'mock-1',
            playerCount: 3,
            playerNicks: ['Ava', 'Omar', 'Yara'],
            playerIds: ['mock-1', 'mock-3', 'mock-2'],
            playerAvatars: ['', '', ''],
            status: 'open',
        },
        {
            id: 'LOBBY-B72',
            leaderNick: 'Zayd',
            leaderId: 'mock-4',
            playerCount: 2,
            playerNicks: ['Zayd', 'Mahdi'],
            playerIds: ['mock-4', 'mock-0'],
            playerAvatars: ['', ''],
            status: 'open',
        },
    ],
    currentMatch: null,
};

const revealState: GameState = {
    ...baseGameState,
    investigateResult: {
        targetName: 'Ava',
        role: 'Policewoman',
    },
};

const gameOverState: GameState = {
    ...baseGameState,
    pendingPrompt: null,
    turnTimer: null,
    gameOver: {
        winnerIndex: 0,
        winnerName: 'Mahdi',
    },
};

const pausedState: GameState = {
    ...baseGameState,
    mockScenario: 'paused',
    pause: {
        status: "active",
        pausedByPlayerId: "mock-2",
        pausedByIndex: 2,
        pausedByName: "Yara",
        deadlineMs: now + 60000,
        durationMs: 60000,
        pauseReason: "disconnect",
        eligibleVoters: [0, 1, 3],
        kickVotes: [0],
    },
    turnTimer: {
        activePlayerIndex: 1,
        durationMs: 30000,
        running: true,
        paused: true,
        key: `mock-${now}`,
    },
};

const assassinateState: GameState = {
    ...baseGameState,
    mockScenario: 'assassinate',
};

const showcaseState: GameState = {
    ...baseGameState,
    mockScenario: 'showcase',
    pendingPrompt: {
        kind: 'challenge',
        requestId: `mock-challenge-${now}`,
        actorIndex: 1,
        actionId: 'steal',
        claimedRole: 'Thief',
        challengeKind: 'main',
        targetIndex: 0,
        eligible: true,
        timeoutMs: 15000,
    },
    activePlayerIndex: 1,
    turnTimer: {
        activePlayerIndex: 1,
        durationMs: 30000,
        running: true,
        paused: false,
        key: `mock-${now}`,
    },
};

export const getMockState = (scenario: MockScenario): GameState => {
    switch (scenario) {
        case 'lobby':
            return { ...lobbyState, mockScenario: 'lobby' };
        case 'reveal':
            return { ...revealState, mockScenario: 'reveal' };
        case 'gameover':
            return { ...gameOverState, mockScenario: 'gameover' };
        case 'paused':
            return pausedState;
        case 'assassinate':
            return assassinateState;
        case 'showcase':
            return showcaseState;
        case 'game':
        default:
            return { ...baseGameState, mockScenario: scenario };
    }
};
