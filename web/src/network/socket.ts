import type {ErrorObject} from 'ajv';
import Ajv2020 from 'ajv/dist/2020';
import wsSchema from './ws-schema.json';
import type {ElMakinaWebSocketEnvelope} from './ws-contract';

export type RequestId = string;
export type WsEnvelope = ElMakinaWebSocketEnvelope;

type OutboundEnvelope = {
    type: string;
    request_id?: RequestId;
    payload?: unknown;
};

type MessageHandler = (envelope: WsEnvelope) => void;

const buildValidator = () => {
    if (process.env.NODE_ENV === 'production') {
        return null;
    }
    const ajv = new Ajv2020({ allErrors: true, strict: false });
    return ajv.compile(wsSchema);
};

interface RequestPromise {
    resolve: (payload: unknown) => void;
    reject: (error: unknown) => void;
}

interface HelloPayload {
    nickname?: string;
    reconnect_token?: string;
    avatar?: string;
}

const STORAGE_KEYS = {
    reconnectToken: "elmakina.reconnectToken",
    nickname: "elmakina.nickname",
    playerId: "elmakina.playerId",
    avatar: "elmakina.avatar",
    connectionLog: "elmakina.connectionLog",
};

type ConnectionEvent = {
    ts: number;
    type:
        | "connect_attempt"
        | "connected"
        | "disconnected"
        | "error"
        | "reconnect_scheduled"
        | "reconnect_manual";
    data?: Record<string, unknown>;
};

export class SocketManager {
    private ws: WebSocket | null = null;
    private url: string;
    private listeners: Set<MessageHandler> = new Set();
    private pendingRequests: Map<RequestId, RequestPromise> = new Map();
    private reconnectToken: string | null = null;
    private nickname: string | null = null;
    private playerId: string | null = null;
    private avatar: string | null = null;
    private pendingHello: HelloPayload | null = null;
    private handshakeComplete: boolean = false;
    private outboundQueue: OutboundEnvelope[] = [];
    private onConnect: (() => void) | null = null;
    private onDisconnect: (() => void) | null = null;
    private reconnectInterval: number = 2000;
    private reconnectMaxInterval: number = 10000;
    private shouldReconnect: boolean = true;
    private fastRetryUsed: boolean = false;
    private mockMode: boolean = false;
    private connectionLog: ConnectionEvent[] = [];
    private maxConnectionLog: number = 50;
    private validateEnvelope = buildValidator();

    constructor(url: string = process.env.NEXT_PUBLIC_WS_URL || "ws://localhost:8080/ws") {
        this.url = url;
        if (typeof window !== "undefined") {
            this.reconnectToken = localStorage.getItem(STORAGE_KEYS.reconnectToken);
            this.nickname = localStorage.getItem(STORAGE_KEYS.nickname);
            this.playerId = localStorage.getItem(STORAGE_KEYS.playerId);
            this.avatar = localStorage.getItem(STORAGE_KEYS.avatar);
            this.connectionLog = this.loadConnectionLog();
        }
    }

    public connect() {
        if (this.mockMode) {
            return;
        }
        this.logConnectionEvent("connect_attempt", { url: this.url, hasReconnectToken: !!this.reconnectToken });
        this.shouldReconnect = true;
        if (this.ws) {
            if (this.ws.readyState === WebSocket.OPEN || this.ws.readyState === WebSocket.CONNECTING) {
                return;
            }
            this.ws.close();
        }

        this.ws = new WebSocket(this.url);

        this.ws.onopen = () => {
            console.log("WebSocket connected");
            this.logConnectionEvent("connected");
            this.onConnect?.();
            this.handshakeComplete = false;
            this.fastRetryUsed = false;
            this.reconnectInterval = 2000;
            if (this.reconnectToken) {
                this.pendingHello = { reconnect_token: this.reconnectToken, avatar: this.avatar || undefined };
            } else if (this.nickname) {
                this.pendingHello = { nickname: this.nickname, avatar: this.avatar || undefined };
            }
            this.flushHello();
        };

        this.ws.onmessage = (event) => {
            try {
                const envelope = JSON.parse(event.data) as WsEnvelope;
                if (this.validateEnvelope && !this.validateEnvelope(envelope)) {
                    const errors = this.validateEnvelope.errors as ErrorObject[] | null | undefined;
                    console.error("[ws] Invalid envelope received", {
                        errors,
                        envelope,
                    });
                    return;
                }
                this.handleMessage(envelope);
            } catch (e) {
                console.error("Failed to parse message:", event.data, e);
            }
        };

        this.ws.onclose = () => {
            console.log("WebSocket disconnected");
            this.logConnectionEvent("disconnected", { shouldReconnect: this.shouldReconnect });
            this.onDisconnect?.();
            this.ws = null;
            this.handshakeComplete = false;
            if (this.shouldReconnect) {
                const delay = this.fastRetryUsed ? this.reconnectInterval : 200;
                if (!this.fastRetryUsed) {
                    this.fastRetryUsed = true;
                }
                this.reconnectInterval = Math.min(this.reconnectInterval * 1.5, this.reconnectMaxInterval);
                this.logConnectionEvent("reconnect_scheduled", { delayMs: delay });
                setTimeout(() => this.connect(), delay);
            }
        };

        this.ws.onerror = (error) => {
            const readyState = this.ws?.readyState ?? -1;
            const detail = typeof error === "object" ? JSON.stringify(error) : String(error);
            console.warn("WebSocket error", { url: this.url, readyState, error: detail });
            this.logConnectionEvent("error", { message: detail, readyState });
            // onerror usually precedes onclose
        };
    }

    public disconnect() {
        this.shouldReconnect = false;
        this.ws?.close();
        this.ws = null;
        this.handshakeComplete = false;
    }

    public reconnectNow() {
        if (this.mockMode) {
            return;
        }
        this.logConnectionEvent("reconnect_manual");
        this.shouldReconnect = true;
        this.connect();
    }

    public isOpen() {
        return !!this.ws && this.ws.readyState === WebSocket.OPEN;
    }

    public getConnectionLog() {
        return [...this.connectionLog];
    }

    public register(nickname: string) {
        if (this.mockMode) {
            return;
        }
        this.setNickname(nickname);
        if (this.handshakeComplete) {
            return;
        }
        this.pendingHello = { nickname, avatar: this.avatar || undefined };
        this.flushHello();
    }

    public setAvatar(avatar: string | null) {
        this.avatar = avatar;
        if (typeof window !== "undefined") {
            if (avatar) {
                localStorage.setItem(STORAGE_KEYS.avatar, avatar);
            } else {
                localStorage.removeItem(STORAGE_KEYS.avatar);
            }
        }
    }

    public getAvatar() {
        return this.avatar;
    }

    public resetIdentity() {
        this.setReconnectToken(null);
        this.setPlayerId(null);
        this.setNickname("");
        this.setAvatar(null);
        this.pendingHello = null;
        if (typeof window !== "undefined") {
            Object.values(STORAGE_KEYS).forEach(key => localStorage.removeItem(key));
        }
    }

    public setNickname(nickname: string) {
        this.nickname = nickname;
        if (typeof window !== "undefined") {
            localStorage.setItem(STORAGE_KEYS.nickname, nickname);
        }
    }

    public getNickname() {
        return this.nickname;
    }

    public hasReconnectToken() {
        return !!this.reconnectToken;
    }

    public getPlayerId() {
        return this.playerId;
    }

    public getHttpBaseUrl() {
        if (!this.url) {
            return "";
        }
        try {
            const wsUrl = new URL(this.url);
            const protocol = wsUrl.protocol === "wss:" ? "https:" : "http:";
            return `${protocol}//${wsUrl.host}`;
        } catch {
            return "";
        }
    }

    public send(type: string, payload: unknown = {}, requestId?: string) {
        if (this.mockMode) {
            return;
        }
        if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
            console.warn("WebSocket not open, cannot send:", type);
            return;
        }
        if (!this.handshakeComplete && type !== "hello") {
            this.outboundQueue.push({ type, payload, request_id: requestId });
            return;
        }
        const envelope: OutboundEnvelope = { type, payload };
        if (requestId) envelope.request_id = requestId;
        this.ws.send(JSON.stringify(envelope));
    }

    public request(type: string, payload: unknown = {}): Promise<unknown> {
        if (this.mockMode) {
            return Promise.resolve({});
        }
        if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
            return Promise.reject({ error: "not_connected" });
        }
        const requestId = `req-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
        return new Promise((resolve, reject) => {
            this.pendingRequests.set(requestId, { resolve, reject });
            this.send(type, payload, requestId);

            // Timeout fallback
            setTimeout(() => {
                if (this.pendingRequests.has(requestId)) {
                    this.pendingRequests.delete(requestId);
                    reject(new Error("Request timed out"));
                }
            }, 10000); // 10s timeout
        });
    }

    public onMessage(handler: MessageHandler) {
        this.listeners.add(handler);
        return () => this.listeners.delete(handler);
    }

    public setConnectionHandlers(onConnect: () => void, onDisconnect: () => void) {
        this.onConnect = onConnect;
        this.onDisconnect = onDisconnect;
    }

    public setMockMode(enabled: boolean) {
        this.mockMode = enabled;
        if (enabled) {
            this.disconnect();
        }
    }

    private handleMessage(envelope: WsEnvelope) {
        if (envelope.type === "hello_ack" && envelope.payload) {
            const payload = envelope.payload as { token?: string; player_id?: string } | undefined;
            if (payload?.token) {
                this.setReconnectToken(payload.token);
            }
            if (payload?.player_id) {
                this.setPlayerId(payload.player_id);
            }
            this.handshakeComplete = true;
            this.flushQueue();
        }

        if (envelope.type === "hello_error") {
            this.handshakeComplete = false;
            this.setReconnectToken(null);
            this.setPlayerId(null);
        }

        // 1. Handle pending requests
        if (envelope.request_id && this.pendingRequests.has(envelope.request_id)) {
            const promise = this.pendingRequests.get(envelope.request_id)!;
            this.pendingRequests.delete(envelope.request_id);

            if (envelope.type.endsWith("_error")) {
                promise.reject(envelope.payload);
            } else {
                promise.resolve(envelope.payload);
            }
        }

        // 2. Broadcast to listeners
        this.listeners.forEach(listener => listener(envelope));
    }

    private flushHello() {
        if (!this.pendingHello) {
            return;
        }
        if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
            return;
        }
        this.send("hello", this.pendingHello);
        this.pendingHello = null;
    }

    private flushQueue() {
        if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
            return;
        }
        if (!this.handshakeComplete) {
            return;
        }
        if (this.outboundQueue.length === 0) {
            return;
        }
        const queued = [...this.outboundQueue];
        this.outboundQueue = [];
        queued.forEach(env => this.send(env.type, env.payload, env.request_id));
    }

    private setReconnectToken(token: string | null) {
        this.reconnectToken = token;
        if (typeof window !== "undefined") {
            if (token) {
                localStorage.setItem(STORAGE_KEYS.reconnectToken, token);
            } else {
                localStorage.removeItem(STORAGE_KEYS.reconnectToken);
            }
        }
    }

    private setPlayerId(playerId: string | null) {
        this.playerId = playerId;
        if (typeof window !== "undefined") {
            if (playerId) {
                localStorage.setItem(STORAGE_KEYS.playerId, playerId);
            } else {
                localStorage.removeItem(STORAGE_KEYS.playerId);
            }
        }
    }

    private logConnectionEvent(type: ConnectionEvent["type"], data?: Record<string, unknown>) {
        const entry: ConnectionEvent = { ts: Date.now(), type, data };
        this.connectionLog.push(entry);
        if (this.connectionLog.length > this.maxConnectionLog) {
            this.connectionLog = this.connectionLog.slice(-this.maxConnectionLog);
        }
        if (typeof window !== "undefined") {
            try {
                localStorage.setItem(STORAGE_KEYS.connectionLog, JSON.stringify(this.connectionLog));
            } catch (err) {
                console.warn("Failed to persist connection log", err);
            }
        }
        console.info("[ws]", type, data ?? {});
    }

    private loadConnectionLog(): ConnectionEvent[] {
        if (typeof window === "undefined") {
            return [];
        }
        const stored = localStorage.getItem(STORAGE_KEYS.connectionLog);
        if (!stored) {
            return [];
        }
        try {
            const parsed = JSON.parse(stored);
            if (Array.isArray(parsed)) {
                return parsed.slice(-this.maxConnectionLog) as ConnectionEvent[];
            }
        } catch {
            return [];
        }
        return [];
    }
}

export const socket = new SocketManager();
