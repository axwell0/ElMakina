export type ReplayPayload = {
    match: ReplayMatch;
    participants: ReplayParticipant[];
    events: ReplayEvent[];
    snapshots: ReplaySnapshot[];
    viewer_player_id: string;
    viewer_index: number;
};

export type ReplayMatch = {
    ID: string;
    LobbyID: string;
    RulesetVersion: string;
    ProtocolVersion: string;
    RNGSeed?: string;
    CreatedAt: string;
    EndedAt?: string | null;
};

export type ReplayParticipant = {
    MatchID: string;
    PlayerID: string;
    PlayerIndex: number;
    Nick: string;
    Avatar: string;
};

export type ReplayEvent = {
    ID: number;
    MatchID: string;
    Seq: number;
    Type: string;
    Visibility: string;
    PlayerID?: string | null;
    Payload: unknown;
    CreatedAt: string;
};

export type ReplaySnapshot = {
    ID: number;
    MatchID: string;
    Seq: number;
    Payload: ReplayGameState;
    CreatedAt: string;
};

export type ReplayGameState = {
    TurnNumber: number;
    Players: Array<{
        ID: number;
        Name: string;
        Hand: Array<{ ID: number; Role: string }>;
        Coins: number;
    }>;
    Deck: Array<{ ID: number; Role: string }>;
    CurrentPlayerIndex: number;
};

function httpBaseFromWsUrl(wsUrl: string): string | null {
    try {
        const u = new URL(wsUrl);
        const protocol = u.protocol === 'wss:' ? 'https:' : 'http:';
        return `${protocol}//${u.host}`;
    } catch {
        return null;
    }
}

function defaultHttpBase(): string {
    const wsUrl = process.env.NEXT_PUBLIC_WS_URL;
    if (wsUrl) {
        const fromEnv = httpBaseFromWsUrl(wsUrl);
        if (fromEnv) return fromEnv;
    }
    if (typeof window !== 'undefined') {
        // In development, if we are on port 3000, the backend is likely on 8080.
        // We must avoid fetching from localhost:3000/replay/... because that hits the Next.js page route,
        // returning HTML instead of JSON.
        if (window.location.hostname === 'localhost' && window.location.port === '3000') {
            return 'http://localhost:8080';
        }
        return window.location.origin;
    }
    return 'http://localhost:8080';
}

export function getReplayUrl(matchId: string, viewerId?: string) {
    const base = defaultHttpBase();
    const url = new URL(`/replay/${matchId}`, base);
    if (viewerId) {
        url.searchParams.set('viewer_id', viewerId);
    }
    return url.toString();
}

export async function fetchReplay(matchId: string, viewerId: string) {
    const url = getReplayUrl(matchId, viewerId);
    const res = await fetch(url, {
        headers: { Accept: 'application/json' },
        cache: 'no-store',
    });
    if (!res.ok) {
        const text = await res.text();
        throw new Error(text || `Failed to load replay (${res.status})`);
    }
    return (await res.json()) as ReplayPayload;
}
