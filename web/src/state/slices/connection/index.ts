/**
 * Connection state slice
 * 
 * Manages WebSocket connection state and identity.
 */

export interface ConnectionState {
  isConnected: boolean;
  isHandshakeComplete: boolean;
  playerId: string | null;
  error: string | null;
  connectionLostAt: number | null;
}

export const initialConnectionState: ConnectionState = {
  isConnected: false,
  isHandshakeComplete: false,
  playerId: null,
  error: null,
  connectionLostAt: null,
};

export type ConnectionAction =
  | { type: "CONNECT" }
  | { type: "DISCONNECT"; inGame: boolean }
  | { type: "ERROR"; error: string }
  | { type: "CLEAR_ERROR" }
  | { type: "HELLO_ACK"; playerId: string | null }
  | { type: "HELLO_ERROR"; error: string | null }
  | { type: "RESET" };

export function connectionReducer(
  state: ConnectionState,
  action: ConnectionAction
): ConnectionState {
  switch (action.type) {
    case "CONNECT":
      return {
        ...state,
        isConnected: true,
        isHandshakeComplete: false,
        error: null,
        connectionLostAt: null,
      };

    case "DISCONNECT":
      return {
        ...state,
        isConnected: false,
        isHandshakeComplete: false,
        error: null,
        connectionLostAt: Date.now(),
      };

    case "ERROR":
      return { ...state, error: action.error };

    case "CLEAR_ERROR":
      return { ...state, error: null };

    case "HELLO_ACK":
      return {
        ...state,
        playerId: action.playerId ?? state.playerId,
        isHandshakeComplete: true,
        connectionLostAt: null,
        error: null,
      };

    case "HELLO_ERROR":
      return {
        ...state,
        error: action.error ?? "Handshake failed",
        playerId: null,
        isHandshakeComplete: false,
        connectionLostAt: null,
      };

    case "RESET":
      return {
        ...initialConnectionState,
        // Preserve connection status during reset
        isConnected: state.isConnected,
      };

    default:
      return state;
  }
}

// Action creators
export const connectionActions = {
  connect: (): ConnectionAction => ({ type: "CONNECT" }),
  disconnect: (inGame: boolean): ConnectionAction => ({
    type: "DISCONNECT",
    inGame,
  }),
  error: (error: string): ConnectionAction => ({ type: "ERROR", error }),
  clearError: (): ConnectionAction => ({ type: "CLEAR_ERROR" }),
  helloAck: (playerId: string | null): ConnectionAction => ({
    type: "HELLO_ACK",
    playerId,
  }),
  helloError: (error: string | null): ConnectionAction => ({
    type: "HELLO_ERROR",
    error,
  }),
  reset: (): ConnectionAction => ({ type: "RESET" }),
} as const;
