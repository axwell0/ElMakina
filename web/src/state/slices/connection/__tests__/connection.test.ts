/**
 * Connection slice tests
 *
 * Tests for WebSocket connection state management.
 */

import { describe, it, expect } from "vitest";
import {
  connectionReducer,
  connectionActions,
  initialConnectionState,
  type ConnectionState,
  type ConnectionAction,
} from "@/state/slices/connection";

describe("Connection Slice", () => {
  describe("Initial State", () => {
    it("should have correct initial values", () => {
      expect(initialConnectionState.isConnected).toBe(false);
      expect(initialConnectionState.isHandshakeComplete).toBe(false);
      expect(initialConnectionState.playerId).toBeNull();
      expect(initialConnectionState.error).toBeNull();
      expect(initialConnectionState.connectionLostAt).toBeNull();
    });
  });

  describe("CONNECT action", () => {
    it("should set isConnected to true", () => {
      const action = connectionActions.connect();
      const newState = connectionReducer(initialConnectionState, action);

      expect(newState.isConnected).toBe(true);
    });

    it("should reset isHandshakeComplete to false", () => {
      const state: ConnectionState = {
        ...initialConnectionState,
        isHandshakeComplete: true,
      };
      const action = connectionActions.connect();
      const newState = connectionReducer(state, action);

      expect(newState.isHandshakeComplete).toBe(false);
    });

    it("should clear any existing error", () => {
      const state: ConnectionState = {
        ...initialConnectionState,
        error: "Previous error",
      };
      const action = connectionActions.connect();
      const newState = connectionReducer(state, action);

      expect(newState.error).toBeNull();
    });

    it("should reset connectionLostAt", () => {
      const state: ConnectionState = {
        ...initialConnectionState,
        connectionLostAt: Date.now(),
      };
      const action = connectionActions.connect();
      const newState = connectionReducer(state, action);

      expect(newState.connectionLostAt).toBeNull();
    });

    it("should preserve playerId during connect", () => {
      const state: ConnectionState = {
        ...initialConnectionState,
        playerId: "player-123",
      };
      const action = connectionActions.connect();
      const newState = connectionReducer(state, action);

      expect(newState.playerId).toBe("player-123");
    });
  });

  describe("DISCONNECT action", () => {
    it("should set isConnected to false", () => {
      const state: ConnectionState = {
        ...initialConnectionState,
        isConnected: true,
      };
      const action = connectionActions.disconnect(false);
      const newState = connectionReducer(state, action);

      expect(newState.isConnected).toBe(false);
    });

    it("should reset isHandshakeComplete to false", () => {
      const state: ConnectionState = {
        ...initialConnectionState,
        isConnected: true,
        isHandshakeComplete: true,
      };
      const action = connectionActions.disconnect(false);
      const newState = connectionReducer(state, action);

      expect(newState.isHandshakeComplete).toBe(false);
    });

    it("should clear any existing error", () => {
      const state: ConnectionState = {
        ...initialConnectionState,
        isConnected: true,
        error: "Some error",
      };
      const action = connectionActions.disconnect(false);
      const newState = connectionReducer(state, action);

      expect(newState.error).toBeNull();
    });

    it("should set connectionLostAt to current timestamp", () => {
      const beforeAction = Date.now();
      const state: ConnectionState = {
        ...initialConnectionState,
        isConnected: true,
      };
      const action = connectionActions.disconnect(false);
      const newState = connectionReducer(state, action);
      const afterAction = Date.now();

      expect(newState.connectionLostAt).toBeGreaterThanOrEqual(beforeAction);
      expect(newState.connectionLostAt).toBeLessThanOrEqual(afterAction);
    });

    it("should preserve playerId during disconnect", () => {
      const state: ConnectionState = {
        ...initialConnectionState,
        isConnected: true,
        playerId: "player-456",
      };
      const action = connectionActions.disconnect(false);
      const newState = connectionReducer(state, action);

      expect(newState.playerId).toBe("player-456");
    });

    it("should handle disconnect during game", () => {
      const state: ConnectionState = {
        ...initialConnectionState,
        isConnected: true,
        isHandshakeComplete: true,
        playerId: "player-789",
      };
      const action = connectionActions.disconnect(true);
      const newState = connectionReducer(state, action);

      expect(newState.isConnected).toBe(false);
      expect(newState.isHandshakeComplete).toBe(false);
      expect(newState.connectionLostAt).not.toBeNull();
    });

    it("should handle disconnect when not in game", () => {
      const state: ConnectionState = {
        ...initialConnectionState,
        isConnected: true,
      };
      const action = connectionActions.disconnect(false);
      const newState = connectionReducer(state, action);

      expect(newState.isConnected).toBe(false);
      expect(newState.connectionLostAt).not.toBeNull();
    });
  });

  describe("ERROR action", () => {
    it("should set error message", () => {
      const action = connectionActions.error("Connection timeout");
      const newState = connectionReducer(initialConnectionState, action);

      expect(newState.error).toBe("Connection timeout");
    });

    it.each<[string]>([
      ["Network error"],
      ["WebSocket closed"],
      ["Timeout"],
      ["Invalid message"],
    ])("should handle error: %s", (errorMessage) => {
      const action = connectionActions.error(errorMessage);
      const newState = connectionReducer(initialConnectionState, action);

      expect(newState.error).toBe(errorMessage);
    });

    it("should preserve other state when setting error", () => {
      const state: ConnectionState = {
        ...initialConnectionState,
        isConnected: true,
        isHandshakeComplete: true,
        playerId: "player-123",
      };
      const action = connectionActions.error("Some error");
      const newState = connectionReducer(state, action);

      expect(newState.isConnected).toBe(true);
      expect(newState.isHandshakeComplete).toBe(true);
      expect(newState.playerId).toBe("player-123");
    });

    it("should overwrite previous error", () => {
      const state: ConnectionState = {
        ...initialConnectionState,
        error: "Old error",
      };
      const action = connectionActions.error("New error");
      const newState = connectionReducer(state, action);

      expect(newState.error).toBe("New error");
    });
  });

  describe("CLEAR_ERROR action", () => {
    it("should clear error message", () => {
      const state: ConnectionState = {
        ...initialConnectionState,
        error: "Some error",
      };
      const action = connectionActions.clearError();
      const newState = connectionReducer(state, action);

      expect(newState.error).toBeNull();
    });

    it("should handle clearing when no error exists", () => {
      const action = connectionActions.clearError();
      const newState = connectionReducer(initialConnectionState, action);

      expect(newState.error).toBeNull();
    });

    it("should preserve other state when clearing error", () => {
      const state: ConnectionState = {
        ...initialConnectionState,
        isConnected: true,
        isHandshakeComplete: true,
        playerId: "player-123",
        error: "Some error",
      };
      const action = connectionActions.clearError();
      const newState = connectionReducer(state, action);

      expect(newState.isConnected).toBe(true);
      expect(newState.isHandshakeComplete).toBe(true);
      expect(newState.playerId).toBe("player-123");
    });
  });

  describe("HELLO_ACK action", () => {
    it("should set playerId when provided", () => {
      const action = connectionActions.helloAck("player-abc");
      const newState = connectionReducer(initialConnectionState, action);

      expect(newState.playerId).toBe("player-abc");
    });

    it("should set isHandshakeComplete to true", () => {
      const action = connectionActions.helloAck("player-123");
      const newState = connectionReducer(initialConnectionState, action);

      expect(newState.isHandshakeComplete).toBe(true);
    });

    it("should clear connectionLostAt", () => {
      const state: ConnectionState = {
        ...initialConnectionState,
        connectionLostAt: Date.now(),
      };
      const action = connectionActions.helloAck("player-123");
      const newState = connectionReducer(state, action);

      expect(newState.connectionLostAt).toBeNull();
    });

    it("should clear any error", () => {
      const state: ConnectionState = {
        ...initialConnectionState,
        error: "Previous error",
      };
      const action = connectionActions.helloAck("player-123");
      const newState = connectionReducer(state, action);

      expect(newState.error).toBeNull();
    });

    it("should preserve existing playerId when null provided", () => {
      const state: ConnectionState = {
        ...initialConnectionState,
        playerId: "existing-player",
      };
      const action = connectionActions.helloAck(null);
      const newState = connectionReducer(state, action);

      expect(newState.playerId).toBe("existing-player");
    });

    it("should set isHandshakeComplete even with null playerId", () => {
      const action = connectionActions.helloAck(null);
      const newState = connectionReducer(initialConnectionState, action);

      expect(newState.isHandshakeComplete).toBe(true);
    });

    it("should preserve isConnected status", () => {
      const state: ConnectionState = {
        ...initialConnectionState,
        isConnected: true,
      };
      const action = connectionActions.helloAck("player-123");
      const newState = connectionReducer(state, action);

      expect(newState.isConnected).toBe(true);
    });
  });

  describe("HELLO_ERROR action", () => {
    it("should set error message", () => {
      const action = connectionActions.helloError("Invalid nickname");
      const newState = connectionReducer(initialConnectionState, action);

      expect(newState.error).toBe("Invalid nickname");
    });

    it("should use default error message when null provided", () => {
      const action = connectionActions.helloError(null);
      const newState = connectionReducer(initialConnectionState, action);

      expect(newState.error).toBe("Handshake failed");
    });

    it("should clear playerId", () => {
      const state: ConnectionState = {
        ...initialConnectionState,
        playerId: "player-123",
      };
      const action = connectionActions.helloError("Some error");
      const newState = connectionReducer(state, action);

      expect(newState.playerId).toBeNull();
    });

    it("should set isHandshakeComplete to false", () => {
      const state: ConnectionState = {
        ...initialConnectionState,
        isHandshakeComplete: true,
      };
      const action = connectionActions.helloError("Error");
      const newState = connectionReducer(state, action);

      expect(newState.isHandshakeComplete).toBe(false);
    });

    it("should clear connectionLostAt", () => {
      const state: ConnectionState = {
        ...initialConnectionState,
        connectionLostAt: Date.now(),
      };
      const action = connectionActions.helloError("Error");
      const newState = connectionReducer(state, action);

      expect(newState.connectionLostAt).toBeNull();
    });

    it.each<[string | null, string]>([
      ["Nickname taken", "Nickname taken"],
      ["Server error", "Server error"],
      [null, "Handshake failed"],
      ["", ""],
    ])("should handle error message: %s", (input, expected) => {
      const action = connectionActions.helloError(input);
      const newState = connectionReducer(initialConnectionState, action);

      expect(newState.error).toBe(expected);
    });
  });

  describe("RESET action", () => {
    it("should reset to initial state", () => {
      const state: ConnectionState = {
        isConnected: true,
        isHandshakeComplete: true,
        playerId: "player-123",
        error: "Some error",
        connectionLostAt: Date.now(),
      };
      const action = connectionActions.reset();
      const newState = connectionReducer(state, action);

      expect(newState.isHandshakeComplete).toBe(false);
      expect(newState.playerId).toBeNull();
      expect(newState.error).toBeNull();
      expect(newState.connectionLostAt).toBeNull();
    });

    it("should preserve isConnected during reset", () => {
      const state: ConnectionState = {
        ...initialConnectionState,
        isConnected: true,
        isHandshakeComplete: true,
        playerId: "player-123",
      };
      const action = connectionActions.reset();
      const newState = connectionReducer(state, action);

      expect(newState.isConnected).toBe(true);
    });

    it("should reset when not connected", () => {
      const state: ConnectionState = {
        isConnected: false,
        isHandshakeComplete: true,
        playerId: "player-123",
        error: "Error",
        connectionLostAt: Date.now(),
      };
      const action = connectionActions.reset();
      const newState = connectionReducer(state, action);

      expect(newState.isConnected).toBe(false);
      expect(newState.playerId).toBeNull();
      expect(newState.error).toBeNull();
    });

    it("should handle reset from initial state", () => {
      const action = connectionActions.reset();
      const newState = connectionReducer(initialConnectionState, action);

      expect(newState).toEqual(initialConnectionState);
    });
  });

  describe("Unknown actions", () => {
    it("should return current state for unknown action type", () => {
      const state: ConnectionState = {
        isConnected: true,
        isHandshakeComplete: true,
        playerId: "player-123",
        error: null,
        connectionLostAt: null,
      };
      const action = { type: "UNKNOWN_ACTION" } as unknown as ConnectionAction;
      const newState = connectionReducer(state, action);

      expect(newState).toEqual(state);
    });

    it("should handle undefined action gracefully", () => {
      const state: ConnectionState = {
        isConnected: true,
        isHandshakeComplete: false,
        playerId: null,
        error: null,
        connectionLostAt: null,
      };
      // This tests the default case in the switch statement
      const action = { type: "RANDOM_TYPE" } as unknown as ConnectionAction;
      const newState = connectionReducer(state, action);

      expect(newState).toEqual(state);
    });
  });

  describe("State Transitions", () => {
    it("should handle full connection lifecycle: disconnected → connecting → connected → handshake → disconnect", () => {
      let state = initialConnectionState;

      // Connect
      state = connectionReducer(state, connectionActions.connect());
      expect(state.isConnected).toBe(true);
      expect(state.isHandshakeComplete).toBe(false);

      // Handshake complete
      state = connectionReducer(state, connectionActions.helloAck("player-1"));
      expect(state.isHandshakeComplete).toBe(true);
      expect(state.playerId).toBe("player-1");

      // Disconnect
      state = connectionReducer(state, connectionActions.disconnect(false));
      expect(state.isConnected).toBe(false);
      expect(state.isHandshakeComplete).toBe(false);
      expect(state.connectionLostAt).not.toBeNull();
    });

    it("should handle reconnection with same playerId", () => {
      let state: ConnectionState = {
        ...initialConnectionState,
        playerId: "player-123",
      };

      // First connection
      state = connectionReducer(state, connectionActions.connect());
      state = connectionReducer(state, connectionActions.helloAck("player-123"));
      expect(state.playerId).toBe("player-123");

      // Disconnect
      state = connectionReducer(state, connectionActions.disconnect(false));

      // Reconnect
      state = connectionReducer(state, connectionActions.connect());
      expect(state.playerId).toBe("player-123"); // Preserved

      // New handshake
      state = connectionReducer(state, connectionActions.helloAck("player-123"));
      expect(state.playerId).toBe("player-123");
    });

    it("should handle failed handshake followed by successful handshake", () => {
      let state = initialConnectionState;

      // Connect
      state = connectionReducer(state, connectionActions.connect());

      // Failed handshake
      state = connectionReducer(state, connectionActions.helloError("Nickname taken"));
      expect(state.error).toBe("Nickname taken");
      expect(state.isHandshakeComplete).toBe(false);
      expect(state.playerId).toBeNull();

      // Reconnect
      state = connectionReducer(state, connectionActions.connect());
      expect(state.error).toBeNull(); // Error cleared

      // Successful handshake
      state = connectionReducer(state, connectionActions.helloAck("player-456"));
      expect(state.isHandshakeComplete).toBe(true);
      expect(state.playerId).toBe("player-456");
      expect(state.error).toBeNull();
    });

    it("should handle error during connected state", () => {
      let state: ConnectionState = {
        ...initialConnectionState,
        isConnected: true,
        isHandshakeComplete: true,
        playerId: "player-123",
      };

      // Error occurs but stay connected
      state = connectionReducer(state, connectionActions.error("Message timeout"));
      expect(state.isConnected).toBe(true);
      expect(state.isHandshakeComplete).toBe(true);
      expect(state.playerId).toBe("player-123");
      expect(state.error).toBe("Message timeout");
    });
  });

  describe("Action Creators", () => {
    it("should create CONNECT action", () => {
      const action = connectionActions.connect();
      expect(action).toEqual({ type: "CONNECT" });
    });

    it("should create DISCONNECT action with inGame flag", () => {
      const action = connectionActions.disconnect(true);
      expect(action).toEqual({ type: "DISCONNECT", inGame: true });
    });

    it("should create DISCONNECT action when not in game", () => {
      const action = connectionActions.disconnect(false);
      expect(action).toEqual({ type: "DISCONNECT", inGame: false });
    });

    it("should create ERROR action", () => {
      const action = connectionActions.error("Network failure");
      expect(action).toEqual({ type: "ERROR", error: "Network failure" });
    });

    it("should create CLEAR_ERROR action", () => {
      const action = connectionActions.clearError();
      expect(action).toEqual({ type: "CLEAR_ERROR" });
    });

    it("should create HELLO_ACK action with playerId", () => {
      const action = connectionActions.helloAck("player-xyz");
      expect(action).toEqual({ type: "HELLO_ACK", playerId: "player-xyz" });
    });

    it("should create HELLO_ACK action with null playerId", () => {
      const action = connectionActions.helloAck(null);
      expect(action).toEqual({ type: "HELLO_ACK", playerId: null });
    });

    it("should create HELLO_ERROR action with error", () => {
      const action = connectionActions.helloError("Server error");
      expect(action).toEqual({ type: "HELLO_ERROR", error: "Server error" });
    });

    it("should create HELLO_ERROR action with null", () => {
      const action = connectionActions.helloError(null);
      expect(action).toEqual({ type: "HELLO_ERROR", error: null });
    });

    it("should create RESET action", () => {
      const action = connectionActions.reset();
      expect(action).toEqual({ type: "RESET" });
    });
  });

  describe("Edge Cases", () => {
    it("should handle multiple consecutive connects", () => {
      let state = initialConnectionState;
      state = connectionReducer(state, connectionActions.connect());
      state = connectionReducer(state, connectionActions.connect());
      state = connectionReducer(state, connectionActions.connect());

      expect(state.isConnected).toBe(true);
      expect(state.isHandshakeComplete).toBe(false);
    });

    it("should handle disconnect when already disconnected", () => {
      const action = connectionActions.disconnect(false);
      const newState = connectionReducer(initialConnectionState, action);

      expect(newState.isConnected).toBe(false);
      expect(newState.connectionLostAt).not.toBeNull();
    });

    it("should handle clear error when no error exists", () => {
      const action = connectionActions.clearError();
      const newState = connectionReducer(initialConnectionState, action);

      expect(newState.error).toBeNull();
    });

    it("should handle hello_ack with empty string playerId", () => {
      const action = connectionActions.helloAck("");
      const newState = connectionReducer(initialConnectionState, action);

      expect(newState.playerId).toBe("");
      expect(newState.isHandshakeComplete).toBe(true);
    });

    it("should handle rapid state changes", () => {
      let state = initialConnectionState;

      // Connect
      state = connectionReducer(state, connectionActions.connect());
      // Error
      state = connectionReducer(state, connectionActions.error("Error 1"));
      // Clear error
      state = connectionReducer(state, connectionActions.clearError());
      // Another error
      state = connectionReducer(state, connectionActions.error("Error 2"));
      // Handshake
      state = connectionReducer(state, connectionActions.helloAck("player-1"));
      // Reset
      state = connectionReducer(state, connectionActions.reset());

      expect(state.isConnected).toBe(true);
      expect(state.isHandshakeComplete).toBe(false);
      expect(state.playerId).toBeNull();
      expect(state.error).toBeNull();
    });

    it("should maintain state reference integrity", () => {
      const state: ConnectionState = {
        ...initialConnectionState,
        isConnected: true,
      };

      const action = connectionActions.connect();
      const newState = connectionReducer(state, action);

      // New state should be a different object
      expect(newState).not.toBe(state);
    });

    it("should not mutate original state", () => {
      const originalState: ConnectionState = {
        isConnected: false,
        isHandshakeComplete: false,
        playerId: null,
        error: null,
        connectionLostAt: null,
      };

      const stateCopy = { ...originalState };
      const action = connectionActions.connect();
      connectionReducer(originalState, action);

      // Original state should remain unchanged
      expect(originalState).toEqual(stateCopy);
    });
  });
});
