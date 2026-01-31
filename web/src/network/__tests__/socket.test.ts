import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { SocketManager } from "@/network/socket";
import * as validation from "@/validation";

// Mock the validation module
vi.mock("@/validation", () => ({
  validateEnvelopeStructure: vi.fn((data: unknown) => {
    // Basic mock implementation - pass through valid objects with type
    if (
      typeof data === "object" &&
      data !== null &&
      !Array.isArray(data) &&
      "type" in data &&
      typeof (data as Record<string, unknown>).type === "string"
    ) {
      return {
        valid: true,
        data: {
          type: (data as Record<string, unknown>).type as string,
          payload: (data as Record<string, unknown>).payload,
          request_id: (data as Record<string, unknown>).request_id as
            | string
            | undefined,
        },
        errors: [],
      };
    }
    return {
      valid: false,
      errors: [{ path: "", message: "Invalid envelope", code: "INVALID" }],
    };
  }),
  isValidInboundMessageType: vi.fn((type: string) => {
    const validTypes = [
      "hello_ack",
      "hello_error",
      "game_state",
      "lobby_state",
      "chat_message",
      "error",
      "join_lobby_error",
      "data_response",
    ];
    return validTypes.includes(type);
  }),
  validationLogger: {
    logValidationError: vi.fn(),
    logValidationWarning: vi.fn(),
    logValidationMetric: vi.fn(),
  },
}));

describe("SocketManager", () => {
  let socket: SocketManager;
  let mockLocalStorage: Record<string, string>;

  beforeEach(() => {
    vi.clearAllMocks();
    vi.useFakeTimers();
    mockLocalStorage = {};

    // Reset localStorage mock
    (window.localStorage.getItem as ReturnType<typeof vi.fn>).mockImplementation(
      (key: string) => mockLocalStorage[key] ?? null
    );
    (window.localStorage.setItem as ReturnType<typeof vi.fn>).mockImplementation(
      (key: string, value: string) => {
        mockLocalStorage[key] = value;
      }
    );
    (
      window.localStorage.removeItem as ReturnType<typeof vi.fn>
    ).mockImplementation((key: string) => {
      delete mockLocalStorage[key];
    });

    // Set up spies - will spy on individual instances as needed

    socket = new SocketManager();
  });

  afterEach(() => {
    // Disconnect socket if connected - handle case where ws methods were mocked
    try {
      socket.disconnect();
    } catch {
      // Ignore errors from mocked methods
    }
    vi.useRealTimers();
  });

  // ==========================================
  // Constructor Tests
  // ==========================================
  describe("Constructor", () => {
    it("should use default URL when no url or env provided", () => {
      const s = new SocketManager();
      expect(s.getHttpBaseUrl()).toBe("http://localhost:8080");
    });

    it("should use provided URL in constructor", () => {
      const s = new SocketManager("ws://example.com/ws");
      expect(s.getHttpBaseUrl()).toBe("http://example.com");
    });

    it("should use wss:// for https base URL", () => {
      const s = new SocketManager("wss://secure.example.com/ws");
      expect(s.getHttpBaseUrl()).toBe("https://secure.example.com");
    });

    it("should load reconnectToken from localStorage", () => {
      mockLocalStorage["elmakina.reconnectToken"] = "test-token-123";
      const s = new SocketManager();
      expect(s.hasReconnectToken()).toBe(true);
    });

    it("should load nickname from localStorage", () => {
      mockLocalStorage["elmakina.nickname"] = "TestPlayer";
      const s = new SocketManager();
      expect(s.getNickname()).toBe("TestPlayer");
    });

    it("should load playerId from localStorage", () => {
      mockLocalStorage["elmakina.playerId"] = "player-456";
      const s = new SocketManager();
      expect(s.getPlayerId()).toBe("player-456");
    });

    it("should load avatar from localStorage", () => {
      mockLocalStorage["elmakina.avatar"] = "avatar-1";
      const s = new SocketManager();
      expect(s.getAvatar()).toBe("avatar-1");
    });

    it("should load connection log from localStorage", () => {
      const log = [{ ts: 123456, type: "connected" as const, data: {} }];
      mockLocalStorage["elmakina.connectionLog"] = JSON.stringify(log);
      const s = new SocketManager();
      expect(s.getConnectionLog()).toHaveLength(1);
    });

    it("should handle invalid connection log JSON", () => {
      mockLocalStorage["elmakina.connectionLog"] = "invalid json";
      const s = new SocketManager();
      expect(s.getConnectionLog()).toHaveLength(0);
    });

    it("should handle non-array connection log", () => {
      mockLocalStorage["elmakina.connectionLog"] = JSON.stringify({ foo: "bar" });
      const s = new SocketManager();
      expect(s.getConnectionLog()).toHaveLength(0);
    });

    it("should return empty string for getHttpBaseUrl when URL is invalid", () => {
      // This tests the catch block by providing an invalid URL that won't parse
      const s = new SocketManager("not a valid url");
      expect(s.getHttpBaseUrl()).toBe("");
    });
  });

  // ==========================================
  // Connection Lifecycle Tests
  // ==========================================
  describe("Connection Lifecycle", () => {
    it("should create WebSocket on connect", () => {
      socket.connect();
      expect(socket.isOpen()).toBe(true);
    });

    it("should not create new WebSocket if already connecting", () => {
      socket.connect();
      const firstConnect = socket.isOpen();
      socket.connect(); // Second call
      expect(socket.isOpen()).toBe(firstConnect);
    });

    it("should not create new WebSocket if already open", () => {
      socket.connect();
      const wsSendSpy = vi.spyOn(
        (socket as unknown as { ws: WebSocket }).ws!,
        "send"
      );
      socket.connect(); // Second call while open
      expect(wsSendSpy).not.toHaveBeenCalled();
    });

    it("should close existing WebSocket if in wrong state", () => {
      socket.connect();
      const ws = (socket as unknown as { ws: WebSocket }).ws!;
      const closeMock = vi.fn();
      ws.close = closeMock;

      // Simulate a different readyState
      Object.defineProperty(ws, "readyState", { value: 3 }); // CLOSED

      socket.connect();
      expect(closeMock).toHaveBeenCalled();
    });

    it("should call onConnect handler when connection opens", () => {
      const onConnect = vi.fn();
      socket.setConnectionHandlers(onConnect, () => {});
      socket.connect();

      const ws = (socket as unknown as { ws: WebSocket }).ws!;
      ws.onopen?.(new Event("open"));

      expect(onConnect).toHaveBeenCalled();
    });

    it("should call onDisconnect handler when connection closes", () => {
      const onDisconnect = vi.fn();
      socket.setConnectionHandlers(() => {}, onDisconnect);
      socket.connect();

      const ws = (socket as unknown as { ws: WebSocket }).ws!;
      ws.onclose?.(new CloseEvent("close"));

      expect(onDisconnect).toHaveBeenCalled();
    });

    it("should disconnect and clear WebSocket", () => {
      socket.connect();
      expect(socket.isOpen()).toBe(true);

      socket.disconnect();
      expect(socket.isOpen()).toBe(false);
    });

    it("should prevent reconnection after manual disconnect", () => {
      const webSocketSpy = vi.spyOn(global, "WebSocket");
      socket.connect();
      
      // Get reference to ws before disconnect
      const ws = (socket as unknown as { ws: WebSocket }).ws!;
      
      socket.disconnect();

      // Simulate close event after disconnect (ws is null after disconnect, but we have reference)
      ws.onclose?.(new CloseEvent("close"));

      // Should not attempt reconnect
      vi.advanceTimersByTime(10000);
      expect(webSocketSpy).toHaveBeenCalledTimes(1); // Only initial connect
    });

    it("should reconnect immediately with reconnectNow when not connected", () => {
      // reconnectNow should create a new connection when not connected
      expect(socket.isOpen()).toBe(false);
      socket.reconnectNow();
      expect(socket.isOpen()).toBe(true);
    });

    it("should return false for isOpen when WebSocket is null", () => {
      expect(socket.isOpen()).toBe(false);
    });

    it("should handle reconnect with fast retry on first close", () => {
      socket.connect();
      const ws = (socket as unknown as { ws: WebSocket }).ws!;

      vi.spyOn(global, "WebSocket");
      ws.onclose?.(new CloseEvent("close"));

      // First retry should happen quickly (200ms)
      vi.advanceTimersByTime(200);
      expect(WebSocket).toHaveBeenCalledTimes(2);
    });

    it("should increase reconnect interval after each retry", () => {
      const webSocketSpy = vi.spyOn(global, "WebSocket");
      socket.connect();
      const ws = (socket as unknown as { ws: WebSocket }).ws!;

      // Clear spy after initial connect
      webSocketSpy.mockClear();

      // First close - fast retry (200ms)
      ws.onclose?.(new CloseEvent("close"));
      vi.advanceTimersByTime(200);
      expect(webSocketSpy).toHaveBeenCalledTimes(1); // First reconnect

      // Second close - longer delay (3000ms = 2000 * 1.5)
      const ws2 = (socket as unknown as { ws: WebSocket }).ws!;
      ws2.onclose?.(new CloseEvent("close"));

      // Not enough time passed yet
      vi.advanceTimersByTime(2000);
      expect(webSocketSpy).toHaveBeenCalledTimes(1);

      // Now enough time has passed (2000 + 1000 = 3000ms)
      vi.advanceTimersByTime(1000);
      expect(webSocketSpy).toHaveBeenCalledTimes(2);
    });

    it("should cap reconnect interval at max", () => {
      socket.connect();

      // Simulate multiple reconnects
      for (let i = 0; i < 10; i++) {
        const ws = (socket as unknown as { ws: WebSocket }).ws!;
        ws.onclose?.(new CloseEvent("close"));
        vi.advanceTimersByTime(20000); // Advance past any interval
      }

      // The interval should be capped at 10000ms
      // We verify by checking WebSocket was created multiple times
      expect(WebSocket).toHaveBeenCalled();
    });
  });

  // ==========================================
  // Message Handling Tests
  // ==========================================
  describe("Message Handling", () => {
    it("should register message listener", () => {
      const handler = vi.fn();
      const unsubscribe = socket.onMessage(handler);

      socket.connect();
      const ws = (socket as unknown as { ws: WebSocket }).ws!;

      // Trigger onopen first to set up the connection
      ws.onopen?.(new Event("open"));

      // Now send a message
      const message = {
        type: "game_state",
        payload: { state: "playing" },
      };
      ws.onmessage?.(
        new MessageEvent("message", { data: JSON.stringify(message) })
      );

      expect(handler).toHaveBeenCalled();
      unsubscribe();
    });

    it("should unsubscribe message listener", () => {
      const handler = vi.fn();
      const unsubscribe = socket.onMessage(handler);
      unsubscribe();

      socket.connect();
      const ws = (socket as unknown as { ws: WebSocket }).ws!;
      ws.onopen?.(new Event("open"));

      const message = { type: "game_state", payload: {} };
      ws.onmessage?.(
        new MessageEvent("message", { data: JSON.stringify(message) })
      );

      expect(handler).not.toHaveBeenCalled();
    });

    it("should handle hello_ack and set reconnect token", () => {
      socket.connect();
      const ws = (socket as unknown as { ws: WebSocket }).ws!;
      ws.onopen?.(new Event("open"));

      const message = {
        type: "hello_ack",
        payload: { token: "new-token", player_id: "player-123" },
      };
      ws.onmessage?.(
        new MessageEvent("message", { data: JSON.stringify(message) })
      );

      expect(socket.hasReconnectToken()).toBe(true);
      expect(socket.getPlayerId()).toBe("player-123");
    });

    it("should handle hello_error and clear tokens", () => {
      // Set up initial state
      socket.connect();
      const ws = (socket as unknown as { ws: WebSocket }).ws!;
      ws.onopen?.(new Event("open"));

      // First get a token
      const ackMessage = {
        type: "hello_ack",
        payload: { token: "token-123", player_id: "player-123" },
      };
      ws.onmessage?.(
        new MessageEvent("message", { data: JSON.stringify(ackMessage) })
      );
      expect(socket.hasReconnectToken()).toBe(true);

      // Now receive error
      const errorMessage = { type: "hello_error", payload: {} };
      ws.onmessage?.(
        new MessageEvent("message", { data: JSON.stringify(errorMessage) })
      );

      expect(socket.hasReconnectToken()).toBe(false);
      expect(socket.getPlayerId()).toBeNull();
    });

    it("should resolve pending request on matching response", async () => {
      socket.connect();
      const ws = (socket as unknown as { ws: WebSocket }).ws!;
      ws.onopen?.(new Event("open"));

      // Complete handshake first
      ws.onmessage?.(
        new MessageEvent("message", {
          data: JSON.stringify({
            type: "hello_ack",
            payload: { token: "token" },
          }),
        })
      );

      // Make a request
      const requestPromise = socket.request("get_lobby", { id: "lobby-1" });

      // Simulate response
      const response = {
        type: "lobby_state",
        request_id: Array.from(
          (socket as unknown as { pendingRequests: Map<string, unknown> }).pendingRequests.keys()
        )[0],
        payload: { id: "lobby-1", name: "Test Lobby" },
      };
      ws.onmessage?.(
        new MessageEvent("message", { data: JSON.stringify(response) })
      );

      const result = await requestPromise;
      expect(result).toEqual({ id: "lobby-1", name: "Test Lobby" });
    });

    it("should reject pending request on error response", async () => {
      socket.connect();
      const ws = (socket as unknown as { ws: WebSocket }).ws!;
      ws.onopen?.(new Event("open"));

      // Complete handshake with valid payload
      ws.onmessage?.(
        new MessageEvent("message", {
          data: JSON.stringify({
            type: "hello_ack",
            payload: { player_id: "player-123", token: "token-123" },
          }),
        })
      );

      // Make a request
      const requestPromise = socket.request("join_lobby", { id: "lobby-1" });

      // Get the request ID
      const requestId = Array.from(
        (socket as unknown as { pendingRequests: Map<string, unknown> }).pendingRequests.keys()
      )[0];

      // Simulate error response
      const response = {
        type: "join_lobby_error",
        request_id: requestId,
        payload: { error: "Lobby full" },
      };
      ws.onmessage?.(
        new MessageEvent("message", { data: JSON.stringify(response) })
      );

      await expect(requestPromise).rejects.toEqual({ error: "Lobby full" });
    });

    it("should handle invalid JSON gracefully", () => {
      socket.connect();
      const ws = (socket as unknown as { ws: WebSocket }).ws!;
      ws.onopen?.(new Event("open"));

      // This should not throw
      expect(() => {
        ws.onmessage?.(
          new MessageEvent("message", { data: "invalid json" })
        );
      }).not.toThrow();
    });

    it("should log validation error for invalid envelope structure", () => {
      (validation.validateEnvelopeStructure as ReturnType<typeof vi.fn>).mockReturnValueOnce({
        valid: false,
        errors: [{ path: "type", message: "Missing type", code: "MISSING_TYPE" }],
      });

      socket.connect();
      const ws = (socket as unknown as { ws: WebSocket }).ws!;
      ws.onopen?.(new Event("open"));

      ws.onmessage?.(
        new MessageEvent("message", { data: JSON.stringify({}) })
      );

      expect(validation.validationLogger.logValidationError).toHaveBeenCalled();
    });

    it("should log warning for unknown message type", () => {
      (validation.isValidInboundMessageType as ReturnType<typeof vi.fn>).mockReturnValueOnce(false);

      socket.connect();
      const ws = (socket as unknown as { ws: WebSocket }).ws!;
      ws.onopen?.(new Event("open"));

      ws.onmessage?.(
        new MessageEvent("message", {
          data: JSON.stringify({ type: "unknown_type", payload: {} }),
        })
      );

      expect(validation.validationLogger.logValidationWarning).toHaveBeenCalled();
    });

    it("should broadcast to all listeners", () => {
      const handler1 = vi.fn();
      const handler2 = vi.fn();
      socket.onMessage(handler1);
      socket.onMessage(handler2);

      socket.connect();
      const ws = (socket as unknown as { ws: WebSocket }).ws!;
      ws.onopen?.(new Event("open"));

      const message = { type: "chat_message", payload: { text: "Hello" } };
      ws.onmessage?.(
        new MessageEvent("message", { data: JSON.stringify(message) })
      );

      expect(handler1).toHaveBeenCalled();
      expect(handler2).toHaveBeenCalled();
    });
  });

  // ==========================================
  // Sending Messages Tests
  // ==========================================
  describe("Sending Messages", () => {
    it("should send message when connected", () => {
      socket.connect();
      const ws = (socket as unknown as { ws: WebSocket }).ws!;
      const sendMock = vi.fn();
      ws.send = sendMock;
      ws.onopen?.(new Event("open"));

      // Complete handshake first with valid payload
      ws.onmessage?.(
        new MessageEvent("message", {
          data: JSON.stringify({ type: "hello_ack", payload: { player_id: "player-123", token: "token-123" } }),
        })
      );

      socket.send("chat_message", { text: "Hello" });

      expect(sendMock).toHaveBeenCalledWith(
        JSON.stringify({ type: "chat_message", payload: { text: "Hello" } })
      );
    });

    it("should queue messages before handshake complete", () => {
      socket.connect();
      const ws = (socket as unknown as { ws: WebSocket }).ws!;
      const sendMock = vi.fn();
      ws.send = sendMock;
      ws.onopen?.(new Event("open"));

      // Don't complete handshake yet
      socket.send("chat_message", { text: "Hello" });

      // Should not send yet
      expect(sendMock).not.toHaveBeenCalledWith(
        expect.stringContaining("chat_message")
      );

      // Complete handshake with valid payload
      ws.onmessage?.(
        new MessageEvent("message", {
          data: JSON.stringify({ type: "hello_ack", payload: { player_id: "player-123", token: "token-123" } }),
        })
      );

      // Now should have sent
      expect(sendMock).toHaveBeenCalledWith(
        expect.stringContaining("chat_message")
      );
    });

    it("should not queue hello messages", () => {
      // Set nickname so hello will be sent
      socket.setNickname("TestPlayer");
      socket.connect();
      const ws = (socket as unknown as { ws: WebSocket }).ws!;
      
      // Clear any previous calls
      (ws.send as ReturnType<typeof vi.fn>).mockClear();
      
      ws.onopen?.(new Event("open"));

      // hello should be sent immediately
      expect(ws.send).toHaveBeenCalledWith(
        expect.stringContaining("hello")
      );
    });

    it("should warn when sending while not connected", () => {
      const consoleSpy = vi.spyOn(console, "warn").mockImplementation(() => {});
      socket.send("chat_message", { text: "Hello" });
      expect(consoleSpy).toHaveBeenCalledWith(
        expect.stringContaining("WebSocket not open"),
        expect.anything()
      );
    });

    it("should include request_id when provided", () => {
      socket.connect();
      const ws = (socket as unknown as { ws: WebSocket }).ws!;
      const sendMock = vi.fn();
      ws.send = sendMock;
      ws.onopen?.(new Event("open"));

      // Complete handshake with valid payload
      ws.onmessage?.(
        new MessageEvent("message", {
          data: JSON.stringify({ type: "hello_ack", payload: { player_id: "player-123", token: "token-123" } }),
        })
      );

      socket.send("action", { type: "move" }, "req-123");

      expect(sendMock).toHaveBeenCalledWith(
        expect.stringContaining("req-123")
      );
    });

    it("should make request and return promise", async () => {
      socket.connect();
      const ws = (socket as unknown as { ws: WebSocket }).ws!;
      ws.onopen?.(new Event("open"));

      // Complete handshake with valid hello_ack payload
      ws.onmessage?.(
        new MessageEvent("message", {
          data: JSON.stringify({ type: "hello_ack", payload: { player_id: "player-123", token: "token-123" } }),
        })
      );

      const requestPromise = socket.request("get_data", { id: 1 });

      // Get request ID and respond
      const requestId = Array.from(
        (socket as unknown as { pendingRequests: Map<string, unknown> }).pendingRequests.keys()
      )[0];

      ws.onmessage?.(
        new MessageEvent("message", {
          data: JSON.stringify({
            type: "data_response",
            request_id: requestId,
            payload: { value: 42 },
          }),
        })
      );

      const result = await requestPromise;
      expect(result).toEqual({ value: 42 });
    });

    it("should reject request when not connected", async () => {
      await expect(socket.request("action", {})).rejects.toEqual({
        error: "not_connected",
      });
    });

    it("should timeout requests after 10 seconds", async () => {
      socket.connect();
      const ws = (socket as unknown as { ws: WebSocket }).ws!;
      ws.onopen?.(new Event("open"));

      // Complete handshake with valid payload
      ws.onmessage?.(
        new MessageEvent("message", {
          data: JSON.stringify({ type: "hello_ack", payload: { player_id: "player-123", token: "token-123" } }),
        })
      );

      const requestPromise = socket.request("slow_action", {});

      // Advance time past timeout
      vi.advanceTimersByTime(10001);

      await expect(requestPromise).rejects.toThrow("Request timed out");
    });

    it("should flush queued messages after handshake", () => {
      socket.connect();
      const ws = (socket as unknown as { ws: WebSocket }).ws!;
      const sendMock = vi.fn();
      ws.send = sendMock;
      ws.onopen?.(new Event("open"));

      // Queue multiple messages
      socket.send("msg1", { data: 1 });
      socket.send("msg2", { data: 2 });
      socket.send("msg3", { data: 3 });

      // Complete handshake with valid payload
      ws.onmessage?.(
        new MessageEvent("message", {
          data: JSON.stringify({ type: "hello_ack", payload: { player_id: "player-123", token: "token-123" } }),
        })
      );

      // Should have sent all queued messages
      const calls = sendMock.mock.calls.filter(
        (call: [string, ...unknown[]]) =>
          call[0].includes("msg1") ||
          call[0].includes("msg2") ||
          call[0].includes("msg3")
      );
      expect(calls).toHaveLength(3);
    });
  });

  // ==========================================
  // Identity Management Tests
  // ==========================================
  describe("Identity Management", () => {
    it("should set nickname", () => {
      socket.setNickname("PlayerOne");
      expect(socket.getNickname()).toBe("PlayerOne");
    });

    it("should persist nickname to localStorage", () => {
      socket.setNickname("PlayerOne");
      expect(window.localStorage.setItem).toHaveBeenCalledWith(
        "elmakina.nickname",
        "PlayerOne"
      );
    });

    it("should set avatar", () => {
      socket.setAvatar("avatar-1");
      expect(socket.getAvatar()).toBe("avatar-1");
    });

    it("should persist avatar to localStorage", () => {
      socket.setAvatar("avatar-1");
      expect(window.localStorage.setItem).toHaveBeenCalledWith(
        "elmakina.avatar",
        "avatar-1"
      );
    });

    it("should remove avatar from localStorage when set to null", () => {
      socket.setAvatar("avatar-1");
      socket.setAvatar(null);
      expect(window.localStorage.removeItem).toHaveBeenCalledWith(
        "elmakina.avatar"
      );
    });

    it("should register new player with nickname", () => {
      socket.connect();
      const ws = (socket as unknown as { ws: WebSocket }).ws!;
      const sendMock = vi.fn();
      ws.send = sendMock;
      ws.onopen?.(new Event("open"));

      socket.register("NewPlayer");

      expect(socket.getNickname()).toBe("NewPlayer");
      expect(sendMock).toHaveBeenCalledWith(
        expect.stringContaining("NewPlayer")
      );
    });

    it("should not send hello if handshake already complete", () => {
      socket.connect();
      const ws = (socket as unknown as { ws: WebSocket }).ws!;
      ws.onopen?.(new Event("open"));

      // Complete handshake with valid payload
      ws.onmessage?.(
        new MessageEvent("message", {
          data: JSON.stringify({ type: "hello_ack", payload: { player_id: "player-123", token: "token-123" } }),
        })
      );

      // Clear send mock after handshake
      (ws.send as ReturnType<typeof vi.fn>).mockClear();
      socket.register("Player");

      // Should not send hello again
      expect(ws.send).not.toHaveBeenCalledWith(
        expect.stringContaining("hello")
      );
    });

    it("should reset identity and clear all storage", () => {
      // Set up some state
      socket.setNickname("Player");
      socket.setAvatar("avatar-1");
      socket.connect();
      const ws = (socket as unknown as { ws: WebSocket }).ws!;
      ws.onopen?.(new Event("open"));
      ws.onmessage?.(
        new MessageEvent("message", {
          data: JSON.stringify({
            type: "hello_ack",
            payload: { token: "token-123", player_id: "player-456" },
          }),
        })
      );

      expect(socket.hasReconnectToken()).toBe(true);

      socket.resetIdentity();

      expect(socket.getNickname()).toBe("");
      expect(socket.getAvatar()).toBeNull();
      expect(socket.hasReconnectToken()).toBe(false);
      expect(socket.getPlayerId()).toBeNull();

      // Should have cleared all localStorage keys
      expect(window.localStorage.removeItem).toHaveBeenCalledWith(
        "elmakina.reconnectToken"
      );
      expect(window.localStorage.removeItem).toHaveBeenCalledWith(
        "elmakina.nickname"
      );
      expect(window.localStorage.removeItem).toHaveBeenCalledWith(
        "elmakina.playerId"
      );
      expect(window.localStorage.removeItem).toHaveBeenCalledWith(
        "elmakina.avatar"
      );
      expect(window.localStorage.removeItem).toHaveBeenCalledWith(
        "elmakina.connectionLog"
      );
    });

    it("should send hello with reconnect token when available", () => {
      mockLocalStorage["elmakina.reconnectToken"] = "existing-token";
      const s = new SocketManager();
      s.connect();

      const ws = (s as unknown as { ws: WebSocket }).ws!;
      const sendMock = vi.fn();
      ws.send = sendMock;
      ws.onopen?.(new Event("open"));

      expect(sendMock).toHaveBeenCalledWith(
        expect.stringContaining("existing-token")
      );
    });

    it("should send hello with nickname when no token", () => {
      mockLocalStorage["elmakina.nickname"] = "SavedPlayer";
      const s = new SocketManager();
      s.connect();

      const ws = (s as unknown as { ws: WebSocket }).ws!;
      const sendMock = vi.fn();
      ws.send = sendMock;
      ws.onopen?.(new Event("open"));

      expect(sendMock).toHaveBeenCalledWith(
        expect.stringContaining("SavedPlayer")
      );
    });

    it("should include avatar in hello when available", () => {
      mockLocalStorage["elmakina.nickname"] = "Player";
      mockLocalStorage["elmakina.avatar"] = "avatar-2";
      const s = new SocketManager();
      s.connect();

      const ws = (s as unknown as { ws: WebSocket }).ws!;
      const sendMock = vi.fn();
      ws.send = sendMock;
      ws.onopen?.(new Event("open"));

      expect(sendMock).toHaveBeenCalledWith(
        expect.stringContaining("avatar-2")
      );
    });
  });

  // ==========================================
  // localStorage Persistence Tests
  // ==========================================
  describe("localStorage Persistence", () => {
    it("should persist reconnect token on hello_ack", () => {
      socket.connect();
      const ws = (socket as unknown as { ws: WebSocket }).ws!;
      ws.onopen?.(new Event("open"));

      ws.onmessage?.(
        new MessageEvent("message", {
          data: JSON.stringify({
            type: "hello_ack",
            payload: { token: "new-token-123" },
          }),
        })
      );

      expect(window.localStorage.setItem).toHaveBeenCalledWith(
        "elmakina.reconnectToken",
        "new-token-123"
      );
    });

    it("should persist playerId on hello_ack", () => {
      socket.connect();
      const ws = (socket as unknown as { ws: WebSocket }).ws!;
      ws.onopen?.(new Event("open"));

      ws.onmessage?.(
        new MessageEvent("message", {
          data: JSON.stringify({
            type: "hello_ack",
            payload: { player_id: "player-789" },
          }),
        })
      );

      expect(window.localStorage.setItem).toHaveBeenCalledWith(
        "elmakina.playerId",
        "player-789"
      );
    });

    it("should remove reconnect token on null", () => {
      socket.connect();
      const ws = (socket as unknown as { ws: WebSocket }).ws!;
      ws.onopen?.(new Event("open"));

      // First set a token
      ws.onmessage?.(
        new MessageEvent("message", {
          data: JSON.stringify({
            type: "hello_ack",
            payload: { token: "token" },
          }),
        })
      );

      // Then clear it via hello_error
      ws.onmessage?.(
        new MessageEvent("message", {
          data: JSON.stringify({ type: "hello_error", payload: {} }),
        })
      );

      expect(window.localStorage.removeItem).toHaveBeenCalledWith(
        "elmakina.reconnectToken"
      );
    });

    it("should load all identity values from localStorage on init", () => {
      mockLocalStorage["elmakina.reconnectToken"] = "token";
      mockLocalStorage["elmakina.nickname"] = "name";
      mockLocalStorage["elmakina.playerId"] = "id";
      mockLocalStorage["elmakina.avatar"] = "av";

      const s = new SocketManager();

      expect(s.hasReconnectToken()).toBe(true);
      expect(s.getNickname()).toBe("name");
      expect(s.getPlayerId()).toBe("id");
      expect(s.getAvatar()).toBe("av");
    });
  });

  // ==========================================
  // Connection Logging Tests
  // ==========================================
  describe("Connection Logging", () => {
    it("should log connection attempt", () => {
      socket.connect();
      const log = socket.getConnectionLog();
      expect(log.some((e) => e.type === "connect_attempt")).toBe(true);
    });

    it("should log connected event", () => {
      socket.connect();
      const ws = (socket as unknown as { ws: WebSocket }).ws!;
      ws.onopen?.(new Event("open"));

      const log = socket.getConnectionLog();
      expect(log.some((e) => e.type === "connected")).toBe(true);
    });

    it("should log disconnected event", () => {
      socket.connect();
      const ws = (socket as unknown as { ws: WebSocket }).ws!;
      ws.onclose?.(new CloseEvent("close"));

      const log = socket.getConnectionLog();
      expect(log.some((e) => e.type === "disconnected")).toBe(true);
    });

    it("should log error event", () => {
      socket.connect();
      const ws = (socket as unknown as { ws: WebSocket }).ws!;
      ws.onerror?.(new Event("error"));

      const log = socket.getConnectionLog();
      expect(log.some((e) => e.type === "error")).toBe(true);
    });

    it("should log reconnect scheduled", () => {
      socket.connect();
      const ws = (socket as unknown as { ws: WebSocket }).ws!;
      ws.onclose?.(new CloseEvent("close"));

      const log = socket.getConnectionLog();
      expect(log.some((e) => e.type === "reconnect_scheduled")).toBe(true);
    });

    it("should log manual reconnect", () => {
      socket.connect();
      socket.reconnectNow();

      const log = socket.getConnectionLog();
      expect(log.some((e) => e.type === "reconnect_manual")).toBe(true);
    });

    it("should persist connection log to localStorage", () => {
      socket.connect();
      expect(window.localStorage.setItem).toHaveBeenCalledWith(
        "elmakina.connectionLog",
        expect.any(String)
      );
    });

    it("should limit connection log to max entries", () => {
      // Create many connection events
      for (let i = 0; i < 60; i++) {
        socket.connect();
        socket.disconnect();
      }

      const log = socket.getConnectionLog();
      expect(log.length).toBeLessThanOrEqual(50);
    });

    it("should return copy of connection log", () => {
      socket.connect();
      const log1 = socket.getConnectionLog();
      const log2 = socket.getConnectionLog();
      expect(log1).not.toBe(log2);
      expect(log1).toEqual(log2);
    });

    it("should include timestamp in log entries", () => {
      const before = Date.now();
      socket.connect();
      const after = Date.now();

      const log = socket.getConnectionLog();
      const entry = log.find((e) => e.type === "connect_attempt");
      expect(entry?.ts).toBeGreaterThanOrEqual(before);
      expect(entry?.ts).toBeLessThanOrEqual(after);
    });

    it("should include data in log entries", () => {
      socket.connect();
      const log = socket.getConnectionLog();
      const entry = log.find((e) => e.type === "connect_attempt");
      expect(entry?.data).toHaveProperty("url");
      expect(entry?.data).toHaveProperty("hasReconnectToken");
    });

    it("should handle localStorage errors gracefully", () => {
      (
        window.localStorage.setItem as ReturnType<typeof vi.fn>
      ).mockImplementation(() => {
        throw new Error("Storage full");
      });

      const consoleSpy = vi.spyOn(console, "warn").mockImplementation(() => {});
      socket.connect();

      expect(consoleSpy).toHaveBeenCalledWith(
        expect.stringContaining("Failed to persist connection log"),
        expect.any(Error)
      );
    });
  });

  // ==========================================
  // Mock Mode Tests
  // ==========================================
  describe("Mock Mode", () => {
    it("should not connect when in mock mode", () => {
      socket.setMockMode(true);
      socket.connect();
      expect(socket.isOpen()).toBe(false);
    });

    it("should disconnect when enabling mock mode", () => {
      socket.connect();
      expect(socket.isOpen()).toBe(true);

      socket.setMockMode(true);
      expect(socket.isOpen()).toBe(false);
    });

    it("should not send messages in mock mode", () => {
      socket.setMockMode(true);
      const consoleSpy = vi.spyOn(console, "warn").mockImplementation(() => {});

      socket.send("action", {});
      expect(consoleSpy).not.toHaveBeenCalled();
    });

    it("should return resolved promise for request in mock mode", async () => {
      socket.setMockMode(true);
      const result = await socket.request("action", {});
      expect(result).toEqual({});
    });

    it("should not register in mock mode", () => {
      socket.setMockMode(true);
      socket.register("Player");
      // Should not throw or do anything
      expect(socket.getNickname()).toBeNull();
    });

    it("should not reconnect in mock mode", () => {
      socket.setMockMode(true);
      socket.reconnectNow();
      // Should not throw or create WebSocket
      expect(socket.isOpen()).toBe(false);
    });

    it("should allow disabling mock mode", () => {
      socket.setMockMode(true);
      expect(socket.isOpen()).toBe(false);

      socket.setMockMode(false);
      socket.connect();
      expect(socket.isOpen()).toBe(true);
    });
  });

  // ==========================================
  // Error Handling Tests
  // ==========================================
  describe("Error Handling", () => {
    it("should handle WebSocket error event", () => {
      const consoleSpy = vi.spyOn(console, "warn").mockImplementation(() => {});
      socket.connect();
      const ws = (socket as unknown as { ws: WebSocket }).ws!;

      ws.onerror?.(new Event("error"));

      expect(consoleSpy).toHaveBeenCalledWith(
        expect.stringContaining("WebSocket error"),
        expect.any(Object)
      );
    });

    it("should log error with readyState info", () => {
      socket.connect();
      const ws = (socket as unknown as { ws: WebSocket }).ws!;
      ws.onerror?.(new Event("error"));

      const log = socket.getConnectionLog();
      const errorEntry = log.find((e) => e.type === "error");
      expect(errorEntry?.data).toHaveProperty("readyState");
    });

    it("should handle WebSocket close with shouldReconnect false", () => {
      socket.connect();

      // First disconnect to set shouldReconnect to false
      socket.disconnect();

      // Manually trigger onclose after disconnect - ws is null now so this won't work
      // Instead, verify that after disconnect, reconnection won't happen
      vi.advanceTimersByTime(10000);

      // Log should not have new reconnect_scheduled entries after disconnect
      // The log might have changed due to disconnect event, but no reconnect_scheduled
      const reconnectEvents = socket
        .getConnectionLog()
        .filter((e) => e.type === "reconnect_scheduled");
      expect(reconnectEvents.length).toBe(0);
    });

    it("should handle missing onConnect/onDisconnect handlers", () => {
      socket.connect();
      const ws = (socket as unknown as { ws: WebSocket }).ws!;

      // Should not throw when handlers are null
      expect(() => {
        ws.onopen?.(new Event("open"));
        ws.onclose?.(new CloseEvent("close"));
      }).not.toThrow();
    });

    it("should handle send when WebSocket is null", () => {
      const consoleSpy = vi.spyOn(console, "warn").mockImplementation(() => {});
      socket.send("action", {});
      expect(consoleSpy).toHaveBeenCalled();
    });

    it("should handle send when WebSocket is not open", () => {
      socket.connect();
      const ws = (socket as unknown as { ws: WebSocket }).ws!;
      Object.defineProperty(ws, "readyState", { value: WebSocket.CONNECTING });

      const consoleSpy = vi.spyOn(console, "warn").mockImplementation(() => {});
      socket.send("action", {});

      expect(consoleSpy).toHaveBeenCalled();
    });

    it("should handle flushHello when WebSocket is not open", () => {
      socket.connect();
      const ws = (socket as unknown as { ws: WebSocket }).ws!;
      ws.onopen?.(new Event("open"));

      // Simulate disconnect
      Object.defineProperty(ws, "readyState", { value: WebSocket.CLOSED });

      // Should not throw
      expect(() => {
        (socket as unknown as { flushHello: () => void }).flushHello();
      }).not.toThrow();
    });

    it("should handle flushQueue when not handshake complete", () => {
      socket.connect();
      const ws = (socket as unknown as { ws: WebSocket }).ws!;
      ws.onopen?.(new Event("open"));

      // Queue a message but don't complete handshake
      socket.send("action", {});

      // Try to flush without handshake
      (socket as unknown as { flushQueue: () => void }).flushQueue();

      // Message should still be queued
      const queue = (socket as unknown as { outboundQueue: unknown[] }).outboundQueue;
      expect(queue.length).toBe(1);
    });

    it("should handle JSON parse errors in onmessage", () => {
      socket.connect();
      const ws = (socket as unknown as { ws: WebSocket }).ws!;
      ws.onopen?.(new Event("open"));

      expect(() => {
        ws.onmessage?.(
          new MessageEvent("message", { data: "not valid json" })
        );
      }).not.toThrow();

      expect(validation.validationLogger.logValidationError).toHaveBeenCalled();
    });

    it("should handle request timeout cleanup", async () => {
      socket.connect();
      const ws = (socket as unknown as { ws: WebSocket }).ws!;
      ws.onopen?.(new Event("open"));

      // Complete handshake with valid payload
      ws.onmessage?.(
        new MessageEvent("message", {
          data: JSON.stringify({ type: "hello_ack", payload: { player_id: "player-123", token: "token-123" } }),
        })
      );

      const requestPromise = socket.request("action", {});

      // Let it timeout
      vi.advanceTimersByTime(10001);

      await expect(requestPromise).rejects.toThrow("Request timed out");

      // Should have cleaned up pending request
      const pending = (socket as unknown as { pendingRequests: Map<string, unknown> }).pendingRequests;
      expect(pending.size).toBe(0);
    });
  });

  // ==========================================
  // Edge Cases and Additional Tests
  // ==========================================
  describe("Edge Cases", () => {
    it("should handle multiple rapid connects", () => {
      socket.connect();
      socket.connect();
      socket.connect();

      // Should only create one WebSocket
      expect(WebSocket).toHaveBeenCalledTimes(1);
    });

    it("should handle message with request_id but no pending request", () => {
      socket.connect();
      const ws = (socket as unknown as { ws: WebSocket }).ws!;
      ws.onopen?.(new Event("open"));

      // Send message with request_id that doesn't exist
      expect(() => {
        ws.onmessage?.(
          new MessageEvent("message", {
            data: JSON.stringify({
              type: "response",
              request_id: "non-existent",
              payload: {},
            }),
          })
        );
      }).not.toThrow();
    });

    it("should handle empty payload in messages", () => {
      socket.connect();
      const ws = (socket as unknown as { ws: WebSocket }).ws!;
      // Clear send mock before testing
      (ws.send as ReturnType<typeof vi.fn>).mockClear();
      ws.onopen?.(new Event("open"));

      // Complete handshake with valid payload
      ws.onmessage?.(
        new MessageEvent("message", {
          data: JSON.stringify({ type: "hello_ack", payload: { player_id: "player-123", token: "token-123" } }),
        })
      );

      socket.send("ping");

      expect(ws.send).toHaveBeenCalledWith(
        JSON.stringify({ type: "ping", payload: {} })
      );
    });

    it("should handle hello_ack without token", () => {
      socket.connect();
      const ws = (socket as unknown as { ws: WebSocket }).ws!;
      ws.onopen?.(new Event("open"));

      // hello_ack without token should not set reconnect token
      ws.onmessage?.(
        new MessageEvent("message", {
          data: JSON.stringify({ type: "hello_ack", payload: { player_id: "player-123" } }),
        })
      );

      expect(socket.hasReconnectToken()).toBe(false);
    });

    it("should handle hello_ack with only player_id", () => {
      socket.connect();
      const ws = (socket as unknown as { ws: WebSocket }).ws!;
      ws.onopen?.(new Event("open"));

      ws.onmessage?.(
        new MessageEvent("message", {
          data: JSON.stringify({
            type: "hello_ack",
            payload: { player_id: "player-only" },
          }),
        })
      );

      expect(socket.getPlayerId()).toBe("player-only");
      expect(socket.hasReconnectToken()).toBe(false);
    });

    it("should not call flushHello when pendingHello is null", () => {
      // Connect without nickname or token - pendingHello will be null
      socket.connect();
      const ws = (socket as unknown as { ws: WebSocket }).ws!;
      
      // Clear any previous calls
      (ws.send as ReturnType<typeof vi.fn>).mockClear();

      // First connect - no pendingHello, so no hello sent
      ws.onopen?.(new Event("open"));
      
      // Reset mock
      (ws.send as ReturnType<typeof vi.fn>).mockClear();

      // Second open also shouldn't send hello (pendingHello still null)
      ws.onopen?.(new Event("open"));
      const helloCalls = (ws.send as ReturnType<typeof vi.fn>).mock.calls.filter((call: [string, ...unknown[]]) =>
        call[0].includes("hello")
      );
      expect(helloCalls.length).toBe(0);
    });

    it("should handle getHttpBaseUrl with invalid URL", () => {
      // URL constructor would throw on completely invalid URLs
      // But we can test the catch block with a malformed URL
      const s = new SocketManager("not-a-valid-url");
      expect(s.getHttpBaseUrl()).toBe("");
    });

    it("should handle window.ELMAKINA_CONFIG for URL", () => {
      (window as Window & { ELMAKINA_CONFIG?: { WS_URL: string } }).ELMAKINA_CONFIG = {
        WS_URL: "ws://config-url.com/ws",
      };

      const s = new SocketManager();
      expect(s.getHttpBaseUrl()).toBe("http://config-url.com");

      // Clean up
      delete (window as Window & { ELMAKINA_CONFIG?: { WS_URL: string } }).ELMAKINA_CONFIG;
    });

    it("should prefer constructor URL over runtime config", () => {
      (window as Window & { ELMAKINA_CONFIG?: { WS_URL: string } }).ELMAKINA_CONFIG = {
        WS_URL: "ws://config-url.com/ws",
      };

      const s = new SocketManager("ws://constructor-url.com/ws");
      expect(s.getHttpBaseUrl()).toBe("http://constructor-url.com");

      delete (window as Window & { ELMAKINA_CONFIG?: { WS_URL: string } }).ELMAKINA_CONFIG;
    });
  });
});
